// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

type Committer struct {
	ctx                   context.Context
	eid                   string // ensemble id
	actor                 actor.Actor
	allocationIDGenerator types.AllocationIDGenerator
	nodeIDGenerator       types.NodeIDGenerator
}

func NewCommitter(ctx context.Context, eid string, act actor.Actor, allocationIDGenerator types.AllocationIDGenerator, nodeIDGenerator types.NodeIDGenerator) *Committer {
	return &Committer{
		ctx:                   ctx,
		eid:                   eid,
		actor:                 act,
		allocationIDGenerator: allocationIDGenerator,
		nodeIDGenerator:       nodeIDGenerator,
	}
}

// parseStandbyNode parses a node name and returns (isStandby, primaryNode, standbyIndex)
func parseStandbyNode(nodeName string) (bool, string, int) {
	return types.ParseNodeName(nodeName)
}

// processNodeAllocations processes allocations and port mappings for a node
func (c *Committer) processNodeAllocations(
	cfg jtypes.EnsembleConfig,
	nodeID string,
	isStandby bool,
	primaryNode string,
	allocationNodes map[string]string,
	portsByAllocation map[string][]jtypes.PortConfig,
) {
	// For standby nodes, we need to get config from primary
	var nodeConfig jtypes.NodeConfig
	var ok bool

	if isStandby {
		nodeConfig, ok = cfg.NodeWithGenerator(primaryNode, c.nodeIDGenerator)
	} else {
		nodeConfig, ok = cfg.NodeWithGenerator(nodeID, c.nodeIDGenerator)
	}

	if !ok {
		return
	}

	for _, allocName := range nodeConfig.Allocations {
		// Generate manifest key using generator
		manifestKey, err := c.allocationIDGenerator.GenerateManifestKey(nodeID, allocName)
		if err != nil {
			log.Errorf("failed to generate manifest key for %s.%s: %v", nodeID, allocName, err)
			continue
		}

		// Track the node that will deploy this allocation using manifest key
		allocationNodes[manifestKey] = nodeID

		// TODO: optimize the manifest format and how node/alloc data is
		//       being passed around. A bit messy at the moment. see #825
		for _, portMap := range nodeConfig.Ports {
			if portMap.Allocation == allocName {
				portsByAllocation[allocName] = append(portsByAllocation[allocName], portMap)
			}
		}
	}
}

// updateManifestAllocations updates manifest allocations with node and port information
func (c *Committer) updateManifestAllocations(
	manifest jtypes.EnsembleManifest,
	allocationNodes map[string]string,
	portsByAllocation map[string][]jtypes.PortConfig,
	allocations map[string]actor.Handle,
) {
	for _, nodeManifest := range manifest.Nodes {
		for _, allocName := range nodeManifest.Allocations {
			// Parse the manifest key to get allocation details
			allocID, err := types.ParseManifestKey(allocName, c.eid)
			if err != nil {
				log.Warnf("failed to parse manifest key %s: %v", allocName, err)
				continue
			}

			allocPorts := make(map[int]int)
			if ports, ok := portsByAllocation[allocID.ConfigName()]; ok {
				for _, pc := range ports {
					allocPorts[pc.Public] = pc.Private
				}
			}
			if alloc, ok := manifest.Allocations[allocName]; ok {
				alloc.NodeID = allocationNodes[allocName]
				alloc.Handle = allocations[allocID.String()]
				alloc.Ports = allocPorts
				alloc.IsStandby = nodeManifest.RedundancyRole == jtypes.RoleStandby
				alloc.RedundancyGroup = allocID.ConfigName()
				manifest.Allocations[allocName] = alloc
				log.Infow("adding allocation to manifest", "labels", []string{string(observability.LabelDeployment)},
					// TODO allocationID?
					"allocation", allocName,
					"nodeID", alloc.NodeID,
					"handle", alloc.Handle,
					"isStandby", alloc.IsStandby)
			}
		}
	}
}

// commit works with a two-commit phases:
//   - first commit the resources in all the nodes to ensure the deployment is (still)
//     feasible.
//   - then create all the allocations for provisioning
//   - if there are any failures, we need to revert this deployment and start anew
func (c *Committer) commit(
	cfgReader jtypes.EnsembleCfgReader,
	manifestReader jtypes.ManifestReader,
	candidate map[string]jtypes.Bid,
) (jtypes.EnsembleManifest, error) {
	var mx sync.Mutex

	cfg := cfgReader.Read()
	manifest := manifestReader.Read()

	// Phase 1: commit
	var wg1 sync.WaitGroup
	ok := true
	wg1.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid jtypes.Bid) {
			defer wg1.Done()
			err := c.commitDeployment(cfg, n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorw("commit resources error",
					"labels", []string{string(observability.LabelDeployment)},
					"nodeID", n,
					"error", err)
				ok = false
				return
			}
			log.Infow("committing deployment",
				"nodeID", n,
			)
			err = updateNodeManifest(manifest.Nodes, n, func(n *jtypes.NodeManifest) {
				n.Handle = bid.Handle()
			})
			if err != nil {
				log.Debugw("committing: update node error",
					"labels", []string{string(observability.LabelDeployment)},
					"nodeID", n,
					"error", err)
				ok = false
			}

			mx.Unlock()
		}(n, bid)
	}
	wg1.Wait()

	if !ok {
		return manifest, fmt.Errorf("failed to commit resources: %w", ErrDeploymentFailed)
	}

	// Phase 2: allocate
	var wg2 sync.WaitGroup
	allocations := make(map[string]actor.Handle)
	wg2.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid jtypes.Bid) {
			defer wg2.Done()
			allocated, err := c.allocate(cfg, n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorw("allocation error",
					"labels", []string{string(observability.LabelDeployment)},
					"nodeID", n,
					"error", err)
				ok = false
			} else {
				log.Debugw("allocating deployment", "nodeID", n)
				for a, h := range allocated {
					allocations[a] = h
				}
			}
			mx.Unlock()
		}(n, bid)
	}
	wg2.Wait()

	if !ok {
		return manifest, fmt.Errorf("failed to allocate resources: %w", ErrDeploymentFailed)
	}

	allocationNodes := make(map[string]string)
	portsByAllocation := make(map[string][]jtypes.PortConfig)
	// There are certain details that are filled during provisioning, e.g. allocation
	// VPN addresses and public port mappings
	for n, bid := range candidate {
		// Extract node role information
		var role jtypes.RedundancyRole
		var primaryNode string
		var standbyIndex int

		isStandby, parsedPrimary, parsedIndex := parseStandbyNode(n)
		if isStandby {
			role = jtypes.RoleStandby
			primaryNode = parsedPrimary
			standbyIndex = parsedIndex
		} else {
			role = jtypes.RolePrimary
			primaryNode = n
			standbyIndex = 0
		}

		// update manifest node
		if nmf, ok := manifest.Nodes[n]; ok {
			nmf.Peer = bid.Peer()
			nmf.PubAddress = append(nmf.PubAddress, bid.PubAddress())
			nmf.Handle = bid.Handle()
			nmf.Location = bid.Location()
			nmf.RedundancyRole = role
			nmf.PrimaryNode = primaryNode
			nmf.StandbyIndex = standbyIndex
			manifest.Nodes[n] = nmf

			// TODO: remove from here on the dynamic ensemble modification PR
			// use diffs instead, after o.commit
		} else {
			nmf := jtypes.NodeManifest{
				ID:             n,
				Peer:           bid.Peer(),
				PubAddress:     []string{bid.PubAddress()},
				Handle:         bid.Handle(),
				Location:       bid.Location(),
				RedundancyRole: role,
				PrimaryNode:    primaryNode,
				StandbyIndex:   standbyIndex,
				StandbyNodes:   make([]string, 0),
			}
			if role == jtypes.RoleStandby {
				nmf.StandbyNodes = make([]string, 0)
			} else {
				for i := 0; i < standbyIndex; i++ {
					nmf.StandbyNodes = append(nmf.StandbyNodes, fmt.Sprintf("%s-standby-%d", primaryNode, i+1))
				}
			}

			manifest.Nodes[n] = nmf
			// TODO: manifest partial updates
		}

		// Process node allocations and port mappings
		c.processNodeAllocations(cfg, n, isStandby, primaryNode, allocationNodes, portsByAllocation)
	}

	// Update manifest allocations with node and port information
	c.updateManifestAllocations(manifest, allocationNodes, portsByAllocation, allocations)

	return manifest, nil
}

type CommitDeploymentRequest struct {
	EnsembleID     string
	AllocationName string
	NodeID         string
	Resources      types.CommittedResources
	PortMapping    map[int]int
}

type CommitDeploymentResponse struct {
	OK    bool
	Error string
}

func (c *Committer) commitDeployment(cfg jtypes.EnsembleConfig, n string, h actor.Handle) error {
	// Check if this is a standby node and get the primary node config
	isStandby, primaryNode, _ := parseStandbyNode(n)
	var ncfg jtypes.NodeConfig
	var ok bool

	if isStandby {
		ncfg, ok = cfg.NodeWithGenerator(primaryNode, c.nodeIDGenerator)
	} else {
		ncfg, ok = cfg.NodeWithGenerator(n, c.nodeIDGenerator)
	}

	if !ok {
		return fmt.Errorf("node %s not found", n)
	}

	if len(ncfg.Allocations) == 0 {
		return nil
	}

	getAllocPortMapping := func(allocName string) map[int]int {
		ports := make(map[int]int)
		for _, pc := range ncfg.Ports {
			if pc.Allocation == allocName {
				ports[pc.Public] = pc.Private
			}
		}
		return ports
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(ncfg.Allocations))
	aggregatedTimeout := time.Duration(len(ncfg.Allocations)) * CommitDeploymentTimeout
	for _, allocName := range ncfg.Allocations {
		wg.Add(1)
		go func(allocName string) {
			defer wg.Done()
			allocation, ok := cfg.Allocation(allocName)
			if !ok {
				errCh <- fmt.Errorf("allocation %s not found: %w", allocName, ErrDeploymentFailed)
				return
			}

			allocPorts := getAllocPortMapping(allocName)

			// Generate full allocation ID using generator
			fullAllocID, err := c.allocationIDGenerator.GenerateFullAllocationID(c.eid, n, allocName)
			if err != nil {
				errCh <- fmt.Errorf("failed to generate full allocation ID for %s.%s: %w", n, allocName, err)
				return
			}

			msg, err := actor.Message(
				c.actor.Handle(),
				h,
				behaviors.CommitDeploymentBehavior,
				CommitDeploymentRequest{
					EnsembleID:     c.eid,
					AllocationName: fullAllocID,
					NodeID:         n,
					Resources:      types.CommittedResources{Resources: allocation.Resources, AllocationID: fullAllocID},
					PortMapping:    allocPorts,
				},
				actor.WithMessageTimeout(aggregatedTimeout),
			)
			if err != nil {
				errCh <- fmt.Errorf("failed to create commit message for %s: %w", n, err)
				return
			}

			replyCh, err := c.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("failed to invoke commit for %s: %w", n, err)
				return
			}

			ticker := time.NewTicker(aggregatedTimeout)
			defer ticker.Stop()

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()
			case <-ticker.C:
				errCh <- fmt.Errorf("timeout committing for %s: %w", n, ErrDeploymentFailed)
				return
			}

			var response CommitDeploymentResponse
			if err := json.Unmarshal(reply.Message, &response); err != nil {
				errCh <- fmt.Errorf("error unmarshalling commit response for %s: %w", n, err)
				return
			}

			if !response.OK {
				errCh <- fmt.Errorf("error committing for %s: %s: %w", n, response.Error, ErrDeploymentFailed)
				return
			}
		}(allocName)
	}

	wg.Wait()
	close(errCh)

	var aggErr error
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	return nil
}

func (c *Committer) allocate(cfg jtypes.EnsembleConfig, n string, h actor.Handle) (map[string]actor.Handle, error) {
	allocs := make(map[string]jtypes.AllocationDeploymentConfig)

	// Check if this is a standby node and get the primary node config
	isStandby, primaryNode, _ := parseStandbyNode(n)
	var ncfg jtypes.NodeConfig
	var ok bool

	if isStandby {
		ncfg, ok = cfg.NodeWithGenerator(primaryNode, c.nodeIDGenerator)
	} else {
		ncfg, ok = cfg.NodeWithGenerator(n, c.nodeIDGenerator)
	}

	if !ok {
		return nil, fmt.Errorf("node not found for %s", n)
	}

	if len(ncfg.Allocations) == 0 {
		log.Warnf("no allocations found for %s, won't allocate (ensemble: %s)", n, c.eid)
		return nil, nil
	}
	contracts := cfg.Contracts()
	for _, a := range ncfg.Allocations {
		acfg, _ := cfg.Allocation(a)

		provisionScripts := make(map[string][]byte)
		for _, p := range acfg.Provision {
			provisionScripts[p] = cfg.V1.Scripts[p]
		}

		// Generate full allocation ID using generator
		fullAllocID, err := c.allocationIDGenerator.GenerateFullAllocationID(c.eid, n, a)
		if err != nil {
			return nil, fmt.Errorf("failed to generate full allocation ID for %s.%s: %w", n, a, err)
		}

		fmt.Println("fullAllocID", fullAllocID)
		allocs[fullAllocID] = jtypes.AllocationDeploymentConfig{
			Type:             acfg.Type,
			Executor:         acfg.Executor,
			Resources:        acfg.Resources,
			Execution:        acfg.Execution,
			ProvisionScripts: provisionScripts,
			Keys:             acfg.Keys,
			Volume:           acfg.Volume,
			Contracts:        contracts,
		}
	}

	aggregatedTimeout := time.Duration(len(allocs)) * AllocationDeploymentTimeout
	msg, err := actor.Message(
		c.actor.Handle(),
		h,
		behaviors.AllocationDeploymentBehavior,
		jtypes.AllocationDeploymentRequest{
			EnsembleID:  c.eid,
			NodeID:      n,
			Allocations: allocs,
		},
		actor.WithMessageTimeout(aggregatedTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation message for %s: %w", n, err)
	}

	log.Debugf("Invoking allocation for node: %s", n)
	replyCh, err := c.actor.Invoke(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke allocate for %s: %w", n, err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(aggregatedTimeout):
		return nil, fmt.Errorf("timeout in allocation for %s: %w", n, err)
	}
	defer reply.Discard()

	var response jtypes.AllocationDeploymentResponse
	if err := json.Unmarshal(reply.Message, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling allocation response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("allocation for %s failed: %s: %w", n, response.Error, ErrDeploymentFailed)
	}

	// verify that the allocation map has all the allocations
	for a := range allocs {
		if _, ok := response.Allocations[a]; !ok {
			return nil, fmt.Errorf("missing allocation %s for %s: %w", a, n, ErrDeploymentFailed)
		}
	}

	log.Infow("Allocation successful", "nodeID", n)
	return response.Allocations, nil
}

func updateNodeManifest(
	m map[string]jtypes.NodeManifest,
	nodeName string, fn func(*jtypes.NodeManifest),
) error {
	if node, ok := m[nodeName]; ok {
		fn(&node)
		m[nodeName] = node
		return nil
	}
	return fmt.Errorf("node %s not found", nodeName)
}
