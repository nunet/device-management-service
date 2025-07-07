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
	"testing"

	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloverDB "gitlab.com/nunet/device-management-service/db/clover"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/dms/resources"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNode_createOrchestrator(t *testing.T) {
	t.Parallel()

	t.Run("empty ensembleconfig", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		// use the real orchestrator for this test
		node.orchestratorRegistry = orchestrator.NewRegistry()

		ctx := context.Background()
		orch, err := node.createOrchestrator(ctx, jobtypes.EnsembleConfig{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty ensemble config")
		assert.Nil(t, orch)
	})

	t.Run("valid ensembleconfig", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		// use the real orchestrator for this test
		node.orchestratorRegistry = orchestrator.NewRegistry()

		ctx := context.Background()
		ensembleConfig := jobtypes.EnsembleConfig{
			V1: &jobtypes.EnsembleConfigV1{
				Allocations: map[string]jobtypes.AllocationConfig{
					"alloc1": {
						Resources: types.Resources{
							CPU:  types.CPU{Cores: 1},
							RAM:  types.RAM{Size: 1},
							Disk: types.Disk{Size: 1},
						},
					},
				},
				Nodes: map[string]jobtypes.NodeConfig{
					"node1": {
						Allocations: []string{"alloc1"},
					},
				},
				Subnet: jobtypes.SubnetConfig{
					Join: true,
				},
			},
		}

		orch, err := node.createOrchestrator(ctx, ensembleConfig)
		assert.NoError(t, err)
		require.NotNil(t, orch)

		// confirm orch details
		assert.Equal(t, orch.Status(), jobtypes.DeploymentStatusPreparing)
		assert.Equal(t, 1, len(orch.Config().V1.Allocations))
		assert.Equal(t, 1, len(orch.Config().V1.Nodes))
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	vNet, err := network.NewMemoryNetHost()
	require.NoError(t, err)
	privK, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	require.NotNil(t, privK)
	_, basicCap, _, _ := newActor(t, privK, vNet)
	// basicCap := ucan.NewCapabilityContext()

	// dcfg := config.DefaultConfig
	// dcfg.Observability.ElasticsearchEnabled = false

	db, err := cloverDB.NewMemDB([]string{})
	require.NoError(t, err)
	repos := resources.ManagerRepos{
		OnboardedResources: cloverDB.NewGenericEntityRepository[types.OnboardedResources](db),
		ResourceAllocation: cloverDB.NewGenericRepository[types.ResourceAllocation](db),
	}
	mockHardwareManager := hardware.NewMockHardwareManager(
		types.MachineResources{},
		types.Resources{},
		types.Resources{},
	)
	mockResourceManager, err := resources.NewResourceManager(repos, mockHardwareManager)
	require.NoError(t, err)
	require.NotNil(t, mockResourceManager)
	// onboardR := dmsclover.NewOnboardingConfig(db)
	orchestR := cloverDB.NewGenericRepository[jobtypes.OrchestratorView](db)
	// onboardingManager, err := onboarding.New(context.Background(), mockResourceManager, mockHardwareManager, onboardR)
	// require.NoError(t, err)
	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	require.NoError(t, err)
	require.NotNil(t, geoip2db)

	t.Run("nil onboarding", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			config.Config{},
			afero.Afero{},
			nil, // onboarding is nil
			basicCap,
			"hostID",
			vNet,
			mockResourceManager,
			&bt.Scheduler{},
			mockHardwareManager,
			orchestR,
			geoip2db,
			geolocation.Geolocation{},
			PortConfig{},
			&storage.VolumeTracker{},
			&controller.GlusterController{},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "onboarding is nil")
	})

	t.Run("nil rootCap", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			config.Config{},
			afero.Afero{},
			&onboarding.Onboarding{},
			nil, // rootCap is nil
			"hostID",
			vNet,
			mockResourceManager,
			&bt.Scheduler{},
			mockHardwareManager,
			orchestR,
			geoip2db,
			geolocation.Geolocation{},
			PortConfig{},
			&storage.VolumeTracker{},
			&controller.GlusterController{},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "root capability context is nil")
	})
	t.Run("nil rootCap", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			config.Config{},
			afero.Afero{},
			&onboarding.Onboarding{},
			basicCap,
			"", // hostID is empty
			vNet,
			mockResourceManager,
			&bt.Scheduler{},
			mockHardwareManager,
			orchestR,
			geoip2db,
			geolocation.Geolocation{},
			PortConfig{},
			&storage.VolumeTracker{},
			&controller.GlusterController{},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hostID is empty")
	})

	t.Run("nil network", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			config.Config{},
			afero.Afero{},
			&onboarding.Onboarding{},
			basicCap,
			"hostID",
			nil, // network is nil
			mockResourceManager,
			&bt.Scheduler{},
			mockHardwareManager,
			orchestR,
			geoip2db,
			geolocation.Geolocation{},
			PortConfig{},
			&storage.VolumeTracker{},
			&controller.GlusterController{},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network is nil")
	})

	t.Run("valid inputs", func(t *testing.T) {
		t.Parallel()

		node, err := New(
			config.Config{},
			afero.Afero{},
			&onboarding.Onboarding{},
			basicCap,
			"hostID",
			vNet,
			mockResourceManager,
			&bt.Scheduler{},
			mockHardwareManager,
			orchestR,
			geoip2db,
			geolocation.Geolocation{},
			PortConfig{},
			&storage.VolumeTracker{},
			&controller.GlusterController{},
		)
		assert.NoError(t, err)
		assert.NotNil(t, node)
	})
}
