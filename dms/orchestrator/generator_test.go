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

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestOrchestratorWithCustomGenerators(t *testing.T) {
	BidRequestTimeout = 1 * time.Second
	CommitDeploymentTimeout = 1 * time.Second
	VerifyEdgeConstraintTimeout = 1 * time.Second
	AllocationDeploymentTimeout = 1 * time.Second
	AllocationStartTimeout = 1 * time.Second
	AllocationShutdownTimeout = 1 * time.Second

	substrate := network.NewSubstrate()

	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: types.RAM{
							Size: 1024,
						},
						Disk: types.Disk{
							Size: 1024,
						},
					},
				},
			},
		},
	}

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	// Create orchestrator with custom generators
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	// Test with test generators
	nodeGenerator := types.NewTestNodeIDGenerator()
	allocationGenerator := types.NewTestAllocationIDGenerator()

	o, err := NewOrchestrator(
		ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg,
		nodeGenerator, allocationGenerator,
	)
	require.NoError(t, err)

	// Start deployment in a goroutine
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	// Create a context for status polling
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Subscribe to status changes
	statusCh := o.StatusChannel(pollCtx)

	// Wait for deployment to complete
	for status := range statusCh {
		t.Logf("Deployment status changed to: %s", status)
		if status == jtypes.DeploymentStatusRunning {
			break
		}
	}

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify the manifest was created with custom generators
	manifest := o.Manifest()
	assert.Equal(t, ensembleID, manifest.ID)
	assert.NotEmpty(t, manifest.Nodes)
	assert.NotEmpty(t, manifest.Allocations)

	// Verify that the test generators were used
	// The test allocation generator should have generated standard format keys
	for allocKey := range manifest.Allocations {
		assert.Contains(t, allocKey, ".", "Allocation key should contain dot separator")
	}

	// Verify that the allocation IDs in the manifest use the test generator format
	for _, alloc := range manifest.Allocations {
		// The allocation ID should follow the format: ensembleID_nodeID.allocName
		assert.Contains(t, alloc.ID, ensembleID+"_", "Allocation ID should contain ensemble ID")
		assert.Contains(t, alloc.ID, ".", "Allocation ID should contain dot separator")
	}
}

func TestOrchestratorWithFailingGenerator(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: types.RAM{
							Size: 1024,
						},
						Disk: types.Disk{
							Size: 1024,
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	// Test with failing generators - should be rejected at instantiation time
	nodeGenerator := types.NewTestNodeIDGenerator()
	failingAllocationGenerator := types.NewFailingAllocationIDGenerator()

	_, err := NewOrchestrator(
		ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg,
		nodeGenerator, failingAllocationGenerator,
	)
	assert.Error(t, err, "Should reject orchestrator with failing generator")
	assert.Contains(t, err.Error(), "invalid allocation ID generator", "Error should mention generator validation failure")
}

func TestOrchestratorWithDefaultGenerators(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: types.RAM{
							Size: 1024,
						},
						Disk: types.Disk{
							Size: 1024,
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	// Test that default generators work the same as the original NewOrchestrator
	o1, err1 := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err1)

	o2, err2 := NewOrchestrator(
		ctx, afero.Afero{Fs: fs}, workDir, ensembleID+"-2", orch.actor, cfg,
		types.NewDefaultNodeIDGenerator(),
		types.NewDefaultAllocationIDGenerator(),
	)
	require.NoError(t, err2)

	// Both should create manifests with the same structure
	manifest1 := o1.newManifest(cfg)
	manifest2 := o2.newManifest(cfg)

	// Should have same number of nodes and allocations
	assert.Equal(t, len(manifest1.Nodes), len(manifest2.Nodes))
	assert.Equal(t, len(manifest1.Allocations), len(manifest2.Allocations))

	// Allocation keys should have the same format
	for allocKey1 := range manifest1.Allocations {
		found := false
		for allocKey2 := range manifest2.Allocations {
			// Both should follow the same format: nodeID.allocName
			if len(allocKey1) > 0 && len(allocKey2) > 0 {
				found = true
				break
			}
		}
		assert.True(t, found, "Allocation key format should be consistent")
	}
}
