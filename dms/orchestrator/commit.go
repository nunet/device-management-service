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
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

type Committer struct {
	ctx   context.Context
	eid   string // ensemble id
	actor actor.Actor
}

func NewCommitter(ctx context.Context, eid string, act actor.Actor) *Committer {
	return &Committer{
		ctx:   ctx,
		eid:   eid,
		actor: act,
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
			log.Debugf("committing deployment for %s", n)
			err = updateNodeManifest(manifest.Nodes, n, func(n *jtypes.NodeManifest) {
				n.Handle = bid.Handle()
			})
			if err != nil {
				log.Errorw("committing: update node error",
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
				return
			}
			log.Debugf("allocating deployment for %s", n)
			for a, h := range allocated {
				allocations[a] = h
			}
			mx.Unlock()
		}(n, bid)
	}
	wg2.Wait()

	if !ok {
		return manifest, fmt.Errorf("failed to allocate resources: %w", ErrDeploymentFailed)
	}

	// There are certain details that are filled during provisioning, e.g. allocation
	// VPN addresses and public port mappings
	for n, bid := range candidate {
		// update manifest only if node already exists
		if nmf, ok := manifest.Nodes[n]; ok {
			nmf.Peer = bid.Peer()
			nmf.Handle = bid.Handle()
			nmf.Location = bid.Location()
			manifest.Nodes[n] = nmf
			// TODO: manifest partial updates
		} else {
			nmf := jtypes.NodeManifest{
				ID:       n,
				Peer:     bid.Peer(),
				Handle:   bid.Handle(),
				Location: bid.Location(),
			}
			manifest.Nodes[n] = nmf
			// TODO: manifest partial updates
		}

		if ncfg, ok := cfg.Node(n); ok {
			for _, a := range ncfg.Allocations {
				allocPorts := make(map[int]int)
				for i := range ncfg.Ports {
					if ncfg.Ports[i].Allocation == a {
						pc := ncfg.Ports[i]
						allocPorts[pc.Public] = pc.Private
					}
				}

				if alloc, ok := manifest.Allocations[a]; ok {
					alloc.NodeID = n
					alloc.Handle = allocations[a]
					alloc.Ports = allocPorts
					manifest.Allocations[a] = alloc
					// TODO: manifest partial updates
				}
			}
		}
	}

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
	ncfg, ok := cfg.Node(n)
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
			msg, err := actor.Message(
				c.actor.Handle(),
				h,
				behaviors.CommitDeploymentBehavior,
				CommitDeploymentRequest{
					EnsembleID:     c.eid,
					AllocationName: allocName,
					NodeID:         n,
					Resources:      types.CommittedResources{Resources: allocation.Resources},
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
	ncfg, ok := cfg.Node(n)
	if !ok {
		return nil, fmt.Errorf("node %s not found", n)
	}

	if len(ncfg.Allocations) == 0 {
		return nil, nil
	}

	for _, a := range ncfg.Allocations {
		acfg, _ := cfg.Allocation(a)

		provisionScripts := make(map[string][]byte)
		for _, p := range acfg.Provision {
			provisionScripts[p] = cfg.V1.Scripts[p]
		}

		allocs[a] = jtypes.AllocationDeploymentConfig{
			Type:             acfg.Type,
			Executor:         acfg.Executor,
			Resources:        acfg.Resources,
			Execution:        acfg.Execution,
			ProvisionScripts: provisionScripts,
			Keys:             acfg.Keys,
			Volumes:          acfg.Volumes,
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

	for _, a := range response.Allocations {
		if err := c.grantOrchestratorCaps(a.DID); err != nil {
			return nil, err
		}
	}

	log.Debugf("Allocation successful for node: %s", n)
	return response.Allocations, nil
}

func (c *Committer) grantOrchestratorCaps(alloc did.DID) error {
	oDID, err := did.FromID(c.actor.Handle().ID)
	if err != nil {
		return fmt.Errorf("failed to parse orchestrator DID: %w", err)
	}

	err = c.actor.Security().Grant(
		alloc,
		oDID,
		[]ucan.Capability{behaviors.OrchestratorNamespace},
		grantOrchestratorCapsFrequency,
	)
	if err != nil {
		return fmt.Errorf(
			"granting orchestrator caps to alloc %s: %w",
			alloc.String(), err)
	}

	// TODO: create helper func to periodically grant caps as
	// it's being used here and on createAllocations()
	go func() {
		ticker := time.NewTicker(grantOrchestratorCapsFrequency)
		defer ticker.Stop()

		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			err := c.actor.Security().Grant(
				alloc,
				c.actor.Handle().DID,
				[]ucan.Capability{},
				grantOrchestratorCapsFrequency,
			)
			if err != nil {
				log.Errorf(
					"periodic grant orchestrator caps to alloc %s: %w",
					alloc.String(), err)
			}
			return
		}
	}()
	return nil
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
