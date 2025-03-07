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

func (o *BasicOrchestrator) Shutdown() {
	o.setStatus(jtypes.DeploymentStatusShuttingDown)
	o.lock.Lock()

	defer func() {
		o.lock.Unlock()
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
				return
			}

			// invoke the subnet destroy message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s/%s: %s", o.manifest.ID, id, err)
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
					return
				}
				if !resp.OK {
					log.Errorf("failed to destroy subnet %s/%s: %v", o.manifest.ID, id, resp.Error)
					return
				}

			case <-time.After(SubnetDestroyTimeout):
				log.Errorf("timeout destroying subnet %s", o.manifest.ID)
				return
			}

			log.Infof("subnet %s destroyed", o.manifest.ID)
		}(handle, id)
	}

	wg.Wait()

	wg = sync.WaitGroup{}
	for _, alloc := range o.manifest.Allocations {
		wg.Add(1)
		go func(h actor.Handle, allocID string) {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				fmt.Sprintf(behaviors.AllocationShutdownBehavior, o.manifest.ID),
				AllocationStopRequest{
					AllocationID: allocID,
				},
				actor.WithMessageExpiry(actor.MakeExpiry(AllocationShutdownTimeout)),
			)
			if err != nil {
				log.Errorf("error creating stop message for alloc: %s: %v", allocID, err)
				return
			}

			// invoke the stop message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s: %v", allocID, err)
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
					return
				}
				if !resp.OK {
					log.Errorf("failed to stop allocation %s", allocID)
					return
				}
			case <-time.After(AllocationShutdownTimeout):
				log.Errorf("timeout stopping allocation %s", allocID)
				return
			}
			log.Infof("allocation %s stopped", allocID)
		}(o.manifest.Nodes[alloc.NodeID].Handle, alloc.ID)
	}

	wg.Wait()
}

type RevertDeploymentMessage struct {
	EnsembleID   string
	AllocsByName []string
}

func (o *BasicOrchestrator) revertDeployment(n string, h actor.Handle) {
	ncfg, _ := o.cfg.Node(n)

	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.RevertDeploymentBehavior,
		RevertDeploymentMessage{
			EnsembleID:   o.id,
			AllocsByName: ncfg.Allocations,
		},
	)
	if err != nil {
		log.Debugf("failed to create revert message for %s: %s", n, err)
		return
	}

	if err := o.actor.Send(msg); err != nil {
		log.Debugf("failed to send revert message for %s: %s", n, err)
	}
}

func (o *BasicOrchestrator) revert(mf jtypes.EnsembleManifest) {
	log.Infof("reverting ensemble manifest: %+v", mf)
	for n, nmf := range mf.Nodes {
		o.revertDeployment(n, nmf.Handle)
	}
}
