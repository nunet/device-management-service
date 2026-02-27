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
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
)

type SubnetDestroyRequest struct {
	SubnetID string
}

type SubnetDestroyResponse struct {
	OK    bool
	Error string
}

type AllocationStopRequest struct {
	AllocationID string
}

type AllocationStopResponse struct {
	OK    bool
	Error string
}

func (o *BasicOrchestrator) Shutdown() error {
	allocStatuses := make(map[string]jtypes.AllocationStatus)

	if o.status == jtypes.DeploymentStatusCompleted || o.status == jtypes.DeploymentStatusShuttingDown {
		log.Error("orchestrator already shutting down or completed")
		return nil
	}

	o.setStatus(jtypes.DeploymentStatusShuttingDown)
	o.lock.Lock()

	log.Infow("orchestrator_shutdown_initiated",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)

	defer func() {
		// set statuses on alloc manifest
		for allocName, status := range allocStatuses {
			err := o.manifest.UpdateAllocation(allocName, func(alloc *jtypes.AllocationManifest) {
				alloc.Status = status
			})
			if err != nil {
				log.Errorf("failed to update allocation manifest %s status: %v", allocName, err)
			}
		}

		o.lock.Unlock()

		log.Infow("status updated",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", o.id,
			"status", jtypes.DeploymentStatusCompleted.String(),
		)

		// set orchestrator status
		o.setStatus(jtypes.DeploymentStatusCompleted)
		if o.cancel != nil {
			o.cancel()
		}

		for _, v := range o.contracts {
			evt := events.DeploymentStop{
				EventBase:       events.EventBase{Type: events.DeploymentStopEvent},
				DeploymentID:    o.manifest.ID,
				OrchestratorID:  o.id,
				HeadContractDID: v.DID, // treat contrat as if head of contract chain, won't be taken into consideration in billing if contract is p2p
			}
			o.contractEventHandler.Push(eventhandler.Event{
				ContractHostDID: v.Host,
				ContractDID:     v.DID,
				Payload:         evt,
			})
		}

		o.UpdateAllocationStatus()
	}()

	destroyHandles := map[string]actor.Handle{}
	for _, node := range o.manifest.Nodes {
		destroyHandles[node.ID] = node.Handle
	}

	if o.manifest.Subnet.Join {
		destroyHandles["orchestrator"] = o.actor.Supervisor()
	}

	errCh1 := make(chan error, len(destroyHandles))
	wg := sync.WaitGroup{}
	for id, handle := range destroyHandles {
		wg.Add(1)

		go func(h actor.Handle, id string) {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, o.manifest.ID),
				SubnetDestroyRequest{
					SubnetID: o.manifest.ID,
				},
				actor.WithMessageExpiry(actor.MakeExpiry(5*time.Second)),
			)
			if err != nil {
				log.Errorf("error creating stop message for %s/%s: %s", o.manifest.ID, id, err)
				errCh1 <- err
				return
			}

			// invoke the subnet destroy message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s/%s: %s", o.manifest.ID, id, err)
				errCh1 <- err
				return
			}

			var reply actor.Envelope
			// wait for the reply
			select {
			case reply = <-replyCh:
				defer reply.Discard()
				var resp SubnetDestroyResponse
				if err := json.Unmarshal(reply.Message, &resp); err != nil {
					log.Errorf("error unmarshalling subnet destroy response: %v", err)
					errCh1 <- err
					return
				}
				if !resp.OK {
					log.Errorf("failed to destroy subnet %s/%s: %v", o.manifest.ID, id, resp.Error)
					errCh1 <- fmt.Errorf("failed to destroy subnet %s/%s: %v", o.manifest.ID, id, resp.Error)
					return
				}

			case <-time.After(SubnetDestroyTimeout):
				log.Errorf("timeout destroying subnet %s", o.manifest.ID)
				errCh1 <- fmt.Errorf("timeout destroying subnet %s", o.manifest.ID)
				return
			}

			log.Infof("subnet %s destroyed", o.manifest.ID)
		}(handle, id)
	}

	wg.Wait()
	close(errCh1)

	errCh2 := make(chan error, len(o.manifest.Allocations))
	wg = sync.WaitGroup{}
	for allocName, alloc := range o.manifest.Allocations {
		wg.Add(1)
		go func(h actor.Handle, allocID string, allocName string) {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, o.manifest.ID),
				AllocationStopRequest{
					AllocationID: allocID,
				},
				actor.WithMessageExpiry(actor.MakeExpiry(AllocationShutdownTimeout)),
			)
			if err != nil {
				log.Errorf("error creating stop message for alloc: %s: %v", allocID, err)
				errCh2 <- err
				return
			}

			// invoke the stop message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s: %v", allocID, err)
				errCh2 <- err
				return
			}

			// wait for the reply
			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()
				var resp AllocationStopResponse
				if err := json.Unmarshal(reply.Message, &resp); err != nil {
					log.Errorf("error unmarshalling stop allocation response: %s", err)
					errCh2 <- err
					return
				}
				if !resp.OK {
					log.Errorf("failed to stop allocation %s", allocID)
					errCh2 <- fmt.Errorf("failed to stop allocation %s", allocID)
					return
				}
			case <-time.After(AllocationShutdownTimeout):
				log.Errorf("timeout stopping allocation %s", allocID)
				errCh2 <- fmt.Errorf("timeout stopping allocation %s", allocID)
				return
			}
			log.Infof("allocation %s stopped", allocID)
			allocStatuses[allocName] = jtypes.AllocationCompleted
		}(o.manifest.Nodes[alloc.NodeID].Handle, alloc.ID, allocName)
	}
	wg.Wait()
	log.Infow("orchestrator_shutdown_complete",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)

	close(errCh2)

	err1 := aggregateErrors(errCh1)
	err2 := aggregateErrors(errCh2)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("errors occurred during shutdown: %w, %w", err1, err2)
	}

	return nil
}

type DeploymentRevertRequest struct {
	EnsembleID   string
	AllocsByName []string
}

type DeploymentRevertResponse struct {
	OK    bool
	Error string
}

func (o *BasicOrchestrator) revertNodeDeployment(
	cfg jtypes.EnsembleConfig, n string, h actor.Handle,
) {
	ncfg, ok := cfg.Node(n)
	if !ok {
		log.Warnf("revert node: failed to find node config for %s", n)
		return
	}

	// Instead of sending the allocation names with nodeID prefix,
	// we should send the complete allocation IDs as created by types.ConstructAllocationID
	// to match exactly how they're stored in the allocator
	allocIDs := make([]string, 0, len(ncfg.Allocations))
	for _, allocName := range ncfg.Allocations {
		// Generate full allocation ID using the generator
		allocID, err := o.allocationIDGenerator.GenerateFullAllocationID(o.id, n, allocName)
		if err != nil {
			log.Errorf("failed to generate full allocation ID for %s.%s: %v", n, allocName, err)
			continue
		}
		allocIDs = append(allocIDs, allocID)
	}

	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.DeploymentRevertBehavior,
		DeploymentRevertRequest{
			EnsembleID:   o.id,
			AllocsByName: allocIDs,
		},
	)
	if err != nil {
		log.Debugw("revert_message_create_failure",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n,
			"error", err)
		return
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		log.Debugw("revert_message_invoke_failure",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n,
			"error", err)
		return
	}

	// Wait for revert response with timeout
	select {
	case reply := <-replyCh:
		defer reply.Discard()

		var response DeploymentRevertResponse
		if err := json.Unmarshal(reply.Message, &response); err != nil {
			log.Errorw("failed to unmarshal revert response",
				"labels", []string{string(observability.LabelDeployment)},
				"nodeID", n,
				"error", err)
			return
		}

		if !response.OK {
			log.Errorw("revert failed on node",
				"labels", []string{string(observability.LabelDeployment)},
				"nodeID", n,
				"error", response.Error)
			return
		}

		log.Debugw("revert completed successfully",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n)

	case <-time.After(30 * time.Second):
		log.Errorw("revert timeout on node",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n)
		return
	}

	o.removeNodeFromManifest(n)
	log.Debugw("revert message sent successfully",
		"labels", []string{string(observability.LabelDeployment)},
		"nodeID", n)
}

func (o *BasicOrchestrator) revert(cfg jtypes.EnsembleConfig, mf jtypes.EnsembleManifest) {
	log.Infow("reverting manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", mf.ID)

	// Log the manifest content to see what nodes are being reverted
	log.Infow("manifest nodes being reverted",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", mf.ID,
		"manifestNodes", len(mf.Nodes),
		"nodeIDs", func() []string {
			var nodeIDs []string
			for nodeID := range mf.Nodes {
				nodeIDs = append(nodeIDs, nodeID)
			}
			return nodeIDs
		}(),
	)

	// BUG FIX: Collect all nodes first to avoid modifying map during iteration
	// The original code was calling removeNodeFromManifest during iteration,
	// which modified mf.Nodes and caused the iteration to stop prematurely
	nodesToRevert := make([]struct {
		nodeID string
		handle actor.Handle
	}, len(mf.Nodes))

	for n, nmf := range mf.Nodes {
		nodesToRevert = append(nodesToRevert, struct {
			nodeID string
			handle actor.Handle
		}{n, nmf.Handle})
	}

	for i, node := range nodesToRevert {
		log.Infow("attempting to revert node",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", mf.ID,
			"nodeID", node.nodeID,
			"handle", node.handle.String(),
			"iteration", i+1,
			"total", len(nodesToRevert),
		)
		o.revertNodeDeployment(cfg, node.nodeID, node.handle)
		log.Infow("completed reverting node",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", mf.ID,
			"nodeID", node.nodeID,
		)
	}

	// If subnet was being joined, destroy it during revert
	if mf.Subnet.Join {
		// IMPORTANT
		// For the deployment restoration (#1166) to work properly
		// We need to cleanup the subnetManifest state for the orchestrator
		// to trigger the code path that will result in re-creating the subnet on the orchestrator
		// during redeployment (during restoration)
		// without this line, the redeployment will fail and will continually try to redeploy without succes
		// because the orchestrator will try to join the subnet where it wasn't created (i.e: subnet doesnt exist error)
		o.revertSubnetManifest()

		log.Infow("destroying subnet during revert",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", mf.ID,
		)

		// Send subnet destroy request to the supervisor
		msg, err := actor.Message(
			o.actor.Handle(),
			o.actor.Supervisor(),
			fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, mf.ID),
			SubnetDestroyRequest{
				SubnetID: mf.ID,
			},
		)
		if err != nil {
			log.Errorw("failed to create subnet destroy message",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", mf.ID,
				"error", err,
			)
			return
		}
		// Send the message and wait for response
		replyCh, err := o.actor.Invoke(msg)
		if err != nil {
			log.Errorw("failed to invoke subnet destroy message during revert",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", mf.ID,
				"error", err,
			)
			return
		}

		// Wait for subnet destroy response with timeout
		select {
		case reply := <-replyCh:
			defer reply.Discard()

			var response SubnetDestroyResponse
			if err := json.Unmarshal(reply.Message, &response); err != nil {
				log.Errorw("failed to unmarshal subnet destroy response",
					"labels", []string{string(observability.LabelDeployment)},
					"orchestratorID", mf.ID,
					"error", err,
				)
				return
			} else if !response.OK {
				log.Errorw("subnet destroy failed during revert",
					"labels", []string{string(observability.LabelDeployment)},
					"orchestratorID", mf.ID,
					"error", response.Error,
				)
				return
			} else {
				log.Infow("subnet destroy completed successfully during revert",
					"labels", []string{string(observability.LabelDeployment)},
					"orchestratorID", mf.ID,
				)
			}

		case <-time.After(30 * time.Second):
			log.Errorw("subnet destroy timeout during revert",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", mf.ID,
			)
		}
	}
}

// revertSubnetManifest reverts the subnet manifest
// only as far as the orchestrator is concerned
// we retain the same subnet state for the rest of the peers
// to avoid re-shaping the subnet with new IPs and routing tables.
func (o *BasicOrchestrator) revertSubnetManifest() {
	orchestratorIP := o.subnetManifest.IndexRoutingTable[orchSubnetName] // IMPORTANT: to re-trigger subnet creation during restoration
	delete(o.subnetManifest.IndexRoutingTable, orchSubnetName)
	delete(o.subnetManifest.RoutingTable, orchestratorIP)
	delete(o.subnetManifest.UsedIPs, orchestratorIP)
	delete(o.subnetManifest.DNSRecords, orchSubnetName)

	log.Infow("reverted subnet manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id,
	)
}

// removeNodeFromManifest removes the node from the manifest and its allocations
func (o *BasicOrchestrator) removeNodeFromManifest(name string) {
	log.Infof("removing node %s from manifest", name)

	o.lock.Lock()
	defer o.lock.Unlock()

	n, ok := o.manifest.Node(name)
	if !ok {
		return
	}

	// Remove allocations from subnet (inline the logic to avoid nested locking)
	for _, allocName := range n.Allocations {
		allocManifest, ok := o.manifest.Allocations[allocName]
		if !ok {
			log.Warnf(
				"skipping subnet removal: allocation %s not found in manifest",
				allocName,
			)
			continue
		}

		ip, ok := o.subnetManifest.IndexRoutingTable[allocName]
		if !ok {
			log.Warnf(
				"skipping subnet removal: allocation %s not found in index routing table",
				allocName,
			)
			continue
		}

		// Remove the IP from the used IPs map
		delete(o.subnetManifest.UsedIPs, ip)

		// Remove the routing table entry
		delete(o.subnetManifest.RoutingTable, ip)

		// Remove the index routing table entry
		delete(o.subnetManifest.IndexRoutingTable, allocName)

		// Remove any DNS records associated with this allocation
		if allocManifest.DNSName != "" {
			delete(o.subnetManifest.DNSRecords, allocManifest.DNSName)
		}

		log.Debugf("Removed allocation %s with IP %s from subnet", allocName, ip)
	}

	// Clean up allocations and remove node
	for _, a := range n.Allocations {
		//  be careful with redundant allocations
		alloc := o.manifest.Allocations[a]
		alloc.Status = jtypes.AllocationTerminated
		o.manifest.Allocations[a] = alloc
		// XXX we're setting the status and then removing the allocs
		//     the status is irrelevant at this point but keeping it
		//     since we will most likely need to move to keeping with
		//     a removed status to keep a history of the deployment.
		delete(o.manifest.Allocations, a)
	}
	delete(o.manifest.Nodes, name)
}
