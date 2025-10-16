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
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

// TestOrchestratorHandlesStatusUpdate tests that orchestrator receives
// and processes status change notifications
func TestOrchestratorHandlesStatusUpdate(t *testing.T) {
	t.Parallel()

	// Setup
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"service1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"service1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
					Execution: types.SpecConfig{Type: "docker"},
				},
			},
		},
	}

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	ctx := context.Background()
	fs := afero.NewMemMapFs()
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID,
		orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Deploy orchestrator (registers the real status update handler)
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	pollCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	statusCh := o.StatusChannel(pollCtx)
	for status := range statusCh {
		if status == jtypes.DeploymentStatusRunning {
			break
		}
	}

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Minute):
		t.Fatal("Timeout waiting for deployment")
	}

	// Verify initial allocation state
	initialManifest := o.Manifest()
	initialAlloc, exists := initialManifest.Allocations["node1.service1"]
	require.True(t, exists, "allocation should exist in manifest after deployment")
	require.Equal(t, jtypes.AllocationRunning, initialAlloc.Status,
		"allocation should initially be in running state")

	// Get deployed allocation actor
	allocationFullID := ensembleID + "_node1.service1"
	allocActor, exists := provider.allocationActors[allocationFullID]
	require.True(t, exists, "allocation actor should exist after deployment")

	// Test Case 1: Send status update notification (running -> stopping)
	statusUpdate1 := behaviors.AllocationStatusUpdate{
		AllocationID: allocationFullID,
		OldStatus:    "running",
		NewStatus:    "stopping",
		Timestamp:    time.Now().Unix(),
		Reason:       "user requested stop",
	}

	msg1, err := actor.Message(
		allocActor.Handle(), // FROM: allocation actor
		orch.actor.Handle(), // TO: orchestrator actor
		behaviors.NotifyAllocationStatusBehavior,
		statusUpdate1,
		actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Minute).UnixNano())),
	)
	require.NoError(t, err, "should create status update message")
	require.NoError(t, allocActor.Send(msg1), "should send status update")

	// Give time for handler to process
	time.Sleep(200 * time.Millisecond)

	// ASSERTIONS: Verify handler updated the manifest

	// 1. Check manifest was updated to new status
	manifest1 := o.Manifest()
	alloc1, exists := manifest1.Allocations["node1.service1"]
	require.True(t, exists, "allocation should still exist in manifest")
	assert.Equal(t, jtypes.AllocationStatus("stopping"), alloc1.Status,
		"handler should have updated manifest status to 'stopping'")

	// 2. Verify orchestrator is still running
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status(),
		"orchestrator should still be running after status update")

	// Test Case 2: Send another status update (stopping -> stopped)
	statusUpdate2 := behaviors.AllocationStatusUpdate{
		AllocationID: allocationFullID,
		OldStatus:    "stopping",
		NewStatus:    "stopped",
		Timestamp:    time.Now().Unix(),
		Reason:       "allocation stopped successfully",
	}

	msg2, err := actor.Message(
		allocActor.Handle(),
		orch.actor.Handle(),
		behaviors.NotifyAllocationStatusBehavior,
		statusUpdate2,
		actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Minute).UnixNano())),
	)
	require.NoError(t, err)
	require.NoError(t, allocActor.Send(msg2))

	time.Sleep(200 * time.Millisecond)

	// 3. Verify second status update was processed
	manifest2 := o.Manifest()
	alloc2, exists := manifest2.Allocations["node1.service1"]
	require.True(t, exists, "allocation should still exist in manifest")
	assert.Equal(t, jtypes.AllocationStatus("stopped"), alloc2.Status,
		"handler should have updated manifest status to 'stopped'")

	// 4. Verify orchestrator handled multiple status updates without issues
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status(),
		"orchestrator should remain healthy after multiple status updates")

	// Test Case 3: Send status update with error reason
	statusUpdate3 := behaviors.AllocationStatusUpdate{
		AllocationID: allocationFullID,
		OldStatus:    "stopped",
		NewStatus:    "failed",
		Timestamp:    time.Now().Unix(),
		Reason:       "container crashed with exit code 137",
	}

	msg3, err := actor.Message(
		allocActor.Handle(),
		orch.actor.Handle(),
		behaviors.NotifyAllocationStatusBehavior,
		statusUpdate3,
		actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Minute).UnixNano())),
	)
	require.NoError(t, err)
	require.NoError(t, allocActor.Send(msg3))

	time.Sleep(200 * time.Millisecond)

	// 5. Verify error status was recorded
	manifest3 := o.Manifest()
	alloc3, exists := manifest3.Allocations["node1.service1"]
	require.True(t, exists, "allocation should still exist in manifest")
	assert.Equal(t, jtypes.AllocationStatus("failed"), alloc3.Status,
		"handler should have updated manifest status to 'failed'")

	// 6. Final verification: orchestrator is still operational
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status(),
		"orchestrator should handle failure status updates gracefully")

	t.Log("Successfully tested status update handler - all state transitions processed correctly")
}
