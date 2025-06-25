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

func TestRegistry(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()
	workDir := "/tmp"
	ensembleID := "test-ensemble"

	t.Run("NewOrchestrator", func(t *testing.T) {
		registry := NewRegistry()

		// Test creating a new orchestrator
		o, err := registry.NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, ensembleID, o.ID())

		// Test creating an orchestrator with existing ID
		_, err = registry.NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
		assert.ErrorIs(t, err, ErrOrchestratorExists)
	})

	t.Run("GetOrchestrator", func(t *testing.T) {
		registry := NewRegistry()

		// Test getting non-existent orchestrator
		_, err := registry.GetOrchestrator(ensembleID)
		assert.ErrorIs(t, err, ErrOrchestratorNotFound)

		// Create an orchestrator
		o, err := registry.NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
		require.NoError(t, err)

		// Test getting existing orchestrator
		retrieved, err := registry.GetOrchestrator(ensembleID)
		require.NoError(t, err)
		assert.Equal(t, o, retrieved)
	})

	t.Run("Orchestrators", func(t *testing.T) {
		registry := NewRegistry()

		// Test empty registry
		orchestrators := registry.Orchestrators()
		assert.Empty(t, orchestrators)

		// Create an orchestrator
		o, err := registry.NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
		require.NoError(t, err)

		// Test registry with one orchestrator
		orchestrators = registry.Orchestrators()
		assert.Len(t, orchestrators, 1)
		assert.Equal(t, o, orchestrators[ensembleID])
	})

	t.Run("DeleteOrchestrator", func(t *testing.T) {
		registry := NewRegistry()

		// Create an orchestrator
		_, err := registry.NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
		require.NoError(t, err)

		// Delete the orchestrator
		registry.DeleteOrchestrator(ensembleID)

		// Verify orchestrator is deleted
		_, err = registry.GetOrchestrator(ensembleID)
		assert.ErrorIs(t, err, ErrOrchestratorNotFound)
	})

	t.Run("RestoreDeployment", func(t *testing.T) {
		registry := NewRegistry()

		// Create test manifest and snapshot
		manifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
					Handle: orch.handle,
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
				},
			},
		}

		snapshot := jtypes.DeploymentSnapshot{
			Expiry: time.Now().Add(time.Hour),
		}

		// Test restoring deployment
		o, err := registry.RestoreDeployment(
			orch.actor,
			ensembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusRunning,
			snapshot,
		)
		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, ensembleID, o.ID())
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

		// Test restoring deployment with existing ID
		_, err = registry.RestoreDeployment(
			orch.actor,
			ensembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusRunning,
			snapshot,
		)
		assert.ErrorIs(t, err, ErrOrchestratorExists)
	})
}
