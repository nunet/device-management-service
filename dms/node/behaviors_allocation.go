// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
)

func (n *Node) handleSubnetCreate(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetCreateRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.SubnetCreateResponse{}
	err := n.network.CreateSubnet(context.Background(), request.SubnetID, request.RoutingTable)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleSubnetDestroy(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetDestroyRequest
	resp := jobs.SubnetDestroyResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err := n.network.DestroySubnet(request.SubnetID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleAllocationDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.AllocationDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.AllocationDeploymentResponse{}
	if err := n.addEnsembleBehaviors(request.EnsembleID); err != nil {
		err = fmt.Errorf("failed to register dynamic behaviors: %w", err)
		log.Error(err)
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	allocations, err := n.createAllocations(
		msg.From.DID,
		request.EnsembleID,
		request.Allocations,
		msg.From,
	)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	resp.Allocations = allocations
	n.sendReply(msg, resp)
}

type AllocationShutdownRequest struct {
	AllocationID string
}

type AllocationShutdownResponse struct {
	OK    bool
	Error string
}

func (n *Node) handleAllocationShutdown(msg actor.Envelope) {
	log.Debugf("handling allocation shutdown request from %s", msg.From.DID)
	defer msg.Discard()

	var request AllocationShutdownRequest
	resp := AllocationShutdownResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err := n.releaseAllocation(request.AllocationID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

// TODO: instead of having a behavior to get logs from a FINISHED allocation,
// we may allow only getting logs from ONGOING allocations.
// For finite allocations, it could send the logs back to the orchestrator
// if specified on the ensemble.
func (n *Node) handleAllocationLogs(msg actor.Envelope) {
	defer msg.Discard()
	log.Infof("behavior get logs invoked by: %+v", msg.From)

	handleErr := func(err error) {
		log.Errorf("error getting allocation logs: %s", err)
		n.sendReply(msg, jobs.AllocationLogsResponse{Error: err.Error()})
	}

	var resp jobs.AllocationLogsResponse
	ensembleID, err := ensembleIDFromBehavior(msg.Behavior)
	if err != nil {
		handleErr(fmt.Errorf("error getting ensemble ID from behavior %s: %s", msg.Behavior, err))
		return
	}

	var req jobs.AllocationLogsRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("error unmarshalling allocation logs request: %s", err))
		return
	}

	allocID := n.constructAllocationID(ensembleID, req.AllocName)
	resultsDir := filepath.Join(n.dmsConfig.WorkDir, "jobs", allocID)

	stdout, err := n.fs.ReadFile(filepath.Join(resultsDir, "stdout.log"))
	if err != nil {
		if err == os.ErrNotExist {
			log.Warnf("stdout file for allocation %s does not exist (ensemble: %s)", req.AllocName, ensembleID)
		} else {
			handleErr(fmt.Errorf("failed to read results file: %s", err))
			return
		}
	}

	stderr, err := n.fs.ReadFile(filepath.Join(resultsDir, "stderr.log"))
	if err != nil {
		if err == os.ErrNotExist {
			log.Debugf("stderr file for allocation %s does not exist (ensemble: %s)", req.AllocName, ensembleID)
		} else {
			handleErr(fmt.Errorf("failed to read results file: %s", err))
			return
		}
	}

	if len(stdout) == 0 && len(stderr) == 0 {
		handleErr(
			fmt.Errorf("stdout and stderr files for allocation %s are empty (ensemble: %s)",
				req.AllocName, ensembleID),
		)
		return
	}

	log.Info("sending logs for allocation: ", allocID)

	resp.Stdout = stdout
	resp.Stderr = stderr
	n.sendReply(msg, resp)
}
