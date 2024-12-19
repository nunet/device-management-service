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
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
)

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
	ensembleID, err := jobs.EnsembleIDFromBehavior(msg.Behavior)
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

// monitorEnsembleAllocations monitors the provided allocations. It handles allocations termination.
//
// TODO: if, for some reason, the allocation is pending indefinitely, we should consider that
// some bug occurred on the orchestrator side and the deployment was not reverted. In this scenario,
// we should do an ensemble clean up (e.g.: destroy subnet)
func (n *Node) monitorEnsembleAllocations(ensembleID string, allocationIDs []string) {
	// track allocations status
	allocationsDone := make(map[string]bool)
	checkInterval := 10 * time.Second

	// Initialize the maps
	for _, allocID := range allocationIDs {
		allocationsDone[allocID] = false
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		allDone := true

		for _, allocID := range allocationIDs {
			// Skip if already marked as done
			if allocationsDone[allocID] {
				continue
			}

			// Retrieve the allocation
			alloc, err := n.GetAllocation(allocID)
			if err != nil {
				log.Warnf("Monitor Ensemble: Allocation %s not found: %v", allocID, err)
				allocationsDone[allocID] = true // Consider it done to avoid infinite loop
				continue
			}

			status := alloc.Status(context.TODO()).Status
			if status == jobs.Completed || status == jobs.Stopped {
				allocationsDone[allocID] = true
				continue
			}

			// Allocation is still running or pending
			allDone = false
		}

		if allDone {
			// All allocations are done; do the necessary ensemble allocs cleanups
			log.Infof("All allocations for ensemble %s are completed or stopped. Cleaning up ensemble.", ensembleID)
			n.cleanupFinishedEnsemble(ensembleID)
			break
		}
	}
}

func (n *Node) cleanupFinishedEnsemble(ensembleID string) {
	if err := n.network.DestroySubnet(ensembleID); err != nil {
		log.Errorf("failed to destroy subnet %s: %v", ensembleID, err)
	}
}

func (n *Node) constructAllocationID(ensembleID, allocName string) string {
	return ensembleID + "_" + allocName
}
