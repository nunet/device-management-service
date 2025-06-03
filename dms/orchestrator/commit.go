// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
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

// commit works with a two-commit phases:
//   - first commit the resources in all the nodes to ensure the deployment is (still)
//     feasible.
//   - then create all the allocations for provisioning
//   - if there are any failures, we need to revert this deployment and start anew

// Note: do not rely on `o.cfg` orchestrator state. A parametrized config should be used
//
// TODO: maybe create a commit struct with the following methods so that it becomes impossible
// to mess with orchestrator state
func (o *BasicOrchestrator) commit(
	cfg jtypes.EnsembleConfig,
	manifest jtypes.EnsembleManifest,
	candidate map[string]jtypes.Bid,
) (jtypes.EnsembleManifest, error) {
	o.setStatus(jtypes.DeploymentStatusCommitting)
	log.Infow("committing candidate bids",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)

	var mx sync.Mutex

	// Phase 1: commit
	var wg1 sync.WaitGroup
	ok := true
	committed := make([]string, 0, len(candidate))
	wg1.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid jtypes.Bid) {
			defer wg1.Done()
			err := o.commitDeployment(cfg, n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorw("commit resources error",
					"labels", []string{string(observability.LabelDeployment)},
					"nodeID", n,
					"error", err)
				ok = false
			} else {
				log.Debugf("committed resources for %s", n)
				committed = append(committed, n)
			}
			mx.Unlock()
		}(n, bid)
	}
	wg1.Wait()

	if !ok {
		for _, n := range committed {
			bid := candidate[n]
			o.revertNodeDeployment(cfg, n, bid.Handle())
		}
		return manifest, fmt.Errorf("failed to commit resources: %w", ErrDeploymentFailed)
	}

	// Phase 2: allocate
	var wg2 sync.WaitGroup
	allocations := make(map[string]actor.Handle)
	wg2.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid jtypes.Bid) {
			defer wg2.Done()
			allocated, err := o.allocate(cfg, n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorw("allocation error",
					"labels", []string{string(observability.LabelDeployment)},
					"nodeID", n,
					"error", err)
				ok = false
			} else {
				log.Debugf("allocating deployment for %s", n)
				for a, h := range allocated {
					allocations[a] = h
				}
			}
			mx.Unlock()
		}(n, bid)
	}
	wg2.Wait()

	if !ok {
		for n, bid := range candidate {
			o.revertNodeDeployment(cfg, n, bid.Handle())
		}
		return manifest, fmt.Errorf("failed to allocate resources: %w", ErrDeploymentFailed)
	}

	// There are certain details that are filled during provisioning, e.g. allocation
	// VPN addresses and public port mappings
	allocationNodes := make(map[string]string)
	portsByAllocation := make(map[string][]jtypes.PortConfig)
	for n, bid := range candidate {
		// update manifest only if node already exists
		if nmf, ok := manifest.Nodes[n]; ok {
			nmf.Peer = bid.Peer()
			nmf.Handle = bid.Handle()
			nmf.Location = bid.Location()
			manifest.Nodes[n] = nmf
			// TODO: remove from here on the dynamic ensemble modification PR
			// use diffs instead, after o.commit
			o.updateNodeManifest(n, nmf)
		} else {
			nmf := jtypes.NodeManifest{
				ID:       n,
				Peer:     bid.Peer(),
				Handle:   bid.Handle(),
				Location: bid.Location(),
			}
			manifest.Nodes[n] = nmf
			// TODO: remove from here on the dynamic ensemble modification PR
			// use diffs instead, after o.commit
			o.updateNodeManifest(n, nmf)
		}

		if ncfg, ok := cfg.Node(n); ok {
			for _, a := range ncfg.Allocations {
				allocationNodes[a] = n
				// TODO: optimize the manifest format and how node/alloc data is
				//       being passed around. A bit messy at the moment. see #825
				for _, portMap := range ncfg.Ports {
					if portMap.Allocation == a {
						portsByAllocation[a] = append(portsByAllocation[a], portMap)
					}
				}
			}
		}
	}

	for name := range cfg.Allocations() {
		allocPorts := make(map[int]int)
		if ports, ok := portsByAllocation[name]; ok {
			for _, pc := range ports {
				allocPorts[pc.Public] = pc.Private
			}
		}
		if alloc, ok := manifest.Allocations[name]; ok {
			alloc.NodeID = allocationNodes[name]
			alloc.Handle = allocations[name]
			alloc.Ports = allocPorts
			manifest.Allocations[name] = alloc
			// TODO: remove from here on the dynamic ensemble modification PR
			// use diffs instead, after o.commit
			o.updateAllocationManifest(name, alloc)
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

func (o *BasicOrchestrator) commitDeployment(cfg jtypes.EnsembleConfig, n string, h actor.Handle) error {
	ncfg, ok := cfg.Node(n)
	if !ok {
		return fmt.Errorf("node %s not found", n)
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
				o.actor.Handle(),
				h,
				behaviors.CommitDeploymentBehavior,
				CommitDeploymentRequest{
					EnsembleID:     o.id,
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

			replyCh, err := o.actor.Invoke(msg)
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

func (o *BasicOrchestrator) allocate(cfg jtypes.EnsembleConfig, n string, h actor.Handle) (map[string]actor.Handle, error) {
	allocs := make(map[string]jtypes.AllocationDeploymentConfig)
	ncfg, _ := cfg.Node(n)
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
			Volume:           acfg.Volume,
		}
	}

	aggregatedTimeout := time.Duration(len(allocs)) * AllocationDeploymentTimeout
	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.AllocationDeploymentBehavior,
		jtypes.AllocationDeploymentRequest{
			EnsembleID:  o.id,
			NodeID:      n,
			Allocations: allocs,
		},
		actor.WithMessageTimeout(aggregatedTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation message for %s: %w", n, err)
	}

	log.Debugf("Invoking allocation for node: %s", n)
	replyCh, err := o.actor.Invoke(msg)
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
		if err := o.grantOrchestratorCaps(a.DID); err != nil {
			return nil, err
		}
	}

	log.Debugf("Allocation successful for node: %s", n)
	return response.Allocations, nil
}

func (o *BasicOrchestrator) grantOrchestratorCaps(alloc did.DID) error {
	oDID, err := did.FromID(o.actor.Handle().ID)
	if err != nil {
		return fmt.Errorf("failed to parse orchestrator DID: %w", err)
	}

	err = o.actor.Security().Grant(
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
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			err := o.actor.Security().Grant(
				alloc,
				o.actor.Handle().DID,
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

func (o *BasicOrchestrator) updateNodeManifest(node string, manifest jtypes.NodeManifest) {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.manifest.Nodes[node] = manifest
}

func (o *BasicOrchestrator) updateAllocationManifest(allocation string, manifest jtypes.AllocationManifest) {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.manifest.Allocations[allocation] = manifest
}
