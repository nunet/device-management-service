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
	"time"

	"gitlab.com/nunet/device-management-service/dms/jobs"
)

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
