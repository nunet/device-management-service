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
		o.lock.Unlock()
		// set alloc statuses
		for allocName, status := range allocStatuses {
			err := o.manifest.UpdateAllocation(allocName, func(alloc *jtypes.AllocationManifest) {
				alloc.Status = status
			})
			if err != nil {
				log.Errorf("failed to update allocation manifest %s status: %v", allocName, err)
			}
		}
		// set orchestrator status
		o.setStatus(jtypes.DeploymentStatusCompleted)
		if o.cancel != nil {
			o.cancel()
		}
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
		go func(h actor.Handle, allocID string) {
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
		}(o.manifest.Nodes[alloc.NodeID].Handle, alloc.ID)
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

func (o *BasicOrchestrator) revertNodeDeployment(
	cfg jtypes.EnsembleConfig, n string, h actor.Handle,
) {
	ncfg, ok := cfg.Node(n)
	if !ok {
		log.Warnf("revert node: failed to find node config for %s", n)
		return
	}

	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.DeploymentRevertBehavior,
		DeploymentRevertRequest{
			EnsembleID:   o.id,
			AllocsByName: ncfg.Allocations,
		},
	)
	if err != nil {
		log.Debugw("revert_message_create_failure",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n,
			"error", err)
		return
	}

	if err := o.actor.Send(msg); err != nil {
		log.Debugw("revert_message_send_failure",
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", n,
			"error", err)
	}

	// QUESTION: do we have to update ensemble config too?
	o.removeNodeFromManifest(n)
	log.Debugf("revert message sent to node %s", n)
}

func (o *BasicOrchestrator) revert(cfg jtypes.EnsembleConfig, mf jtypes.EnsembleManifest) {
	log.Infow("reverting manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", mf.ID)
	for n, nmf := range mf.Nodes {
		o.revertNodeDeployment(cfg, n, nmf.Handle)
	}
}

// removeNodeFromManifest removes the node from the manifest and its allocations
func (o *BasicOrchestrator) removeNodeFromManifest(name string) {
	log.Infof("removing node %s from manifest", name)
	o.lock.Lock()
	n, ok := o.manifest.Node(name)
	if !ok {
		return
	}
	o.lock.Unlock()

	err := o.removeAllocationsFromSubnet(o.manifest, n.Allocations)
	if err != nil {
		log.Errorf(
			"removeNodeFromManifest: error removing allocations from subnet: %v",
			err)
	}

	o.lock.Lock()
	defer o.lock.Unlock()
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
