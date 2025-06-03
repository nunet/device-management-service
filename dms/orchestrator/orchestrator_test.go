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

func TestOrchestratorDeploy(t *testing.T) {
	substrate := network.NewSubstrate()

	orch := MakeDMS(t, substrate)
	provider := MakeDMS(t, substrate)

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

	provider.MockDeploymentBehaviors(t)

	// Create orchestrator with orchestrator mock
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	workDir := "/tmp"
	ensembleID := "test-ensemble"

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
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

	// Track expected status transitions
	expectedStatuses := []jtypes.DeploymentStatus{
		jtypes.DeploymentStatusPreparing,
		jtypes.DeploymentStatusGenerating,
		jtypes.DeploymentStatusCommitting,
		jtypes.DeploymentStatusProvisioning,
		jtypes.DeploymentStatusRunning,
	}
	statusIndex := 0

	// Wait for status changes
	for status := range statusCh {
		t.Logf("Deployment status changed to: %s", status)
		if statusIndex < len(expectedStatuses) {
			assert.Equal(t, expectedStatuses[statusIndex], status)
			statusIndex++
		}
	}

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify final state
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	manifest := o.Manifest()
	assert.NotEmpty(t, manifest.Nodes)
	assert.NotEmpty(t, manifest.Allocations)

	// Verify node was deployed
	node, ok := manifest.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, provider.peerID.String(), node.Peer)

	// Verify allocation was deployed
	alloc, ok := manifest.Allocations["alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
}
