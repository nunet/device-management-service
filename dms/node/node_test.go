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
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/avast/retry-go"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/jobs"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	repo "gitlab.com/nunet/device-management-service/db/repositories/clover"

	"gitlab.com/nunet/device-management-service/dms/onboarding"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func createRootCapabilityContext(t *testing.T) ucan.CapabilityContext {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err, "generate key")

	provider, err := did.ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")

	trustCtx := did.NewTrustContext()
	trustCtx.AddProvider(provider)

	capCtx, err := ucan.NewCapabilityContext(trustCtx, provider.DID(), nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	require.NoError(t, err, "make capability context")

	return capCtx
}

func getMockNode(t *testing.T, ctrl *gomock.Controller) *Node {
	t.Helper()

	mockResourceManager := NewMockResourceManager(ctrl)
	mockHardwareManager := NewMockHardwareManager(ctrl)
	mockNetwork := NewMockNetwork(ctrl)
	mockGeoIPLocator := NewMockGeoIPLocator(ctrl)
	mockActor := NewMockActor(ctrl)
	mockOrchestratorFactory := NewMockOrchestratorProvider(ctrl)
	mockAllocator := NewMockAllocator(ctrl)

	rootCap := createRootCapabilityContext(t)

	collections := []string{"orchestrator_view", "contract"}
	db, err := repo.NewMemDB(collections)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	t.Cleanup(func() {
		db.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	node := Node{
		rootCap:              rootCap,
		actor:                mockActor,
		scheduler:            bt.NewScheduler(1),
		network:              mockNetwork,
		resourceManager:      mockResourceManager,
		hardware:             mockHardwareManager,
		onboarding:           nil,
		executors:            make(map[string]executorMetadata),
		hostID:               "hostID",
		geoIP:                mockGeoIPLocator,
		hostLocation:         Geolocation{},
		orchestratorRepo:     repo.NewOrchestratorView(db),
		fs:                   afero.Afero{Fs: afero.NewMemMapFs()},
		ctx:                  ctx,
		cancel:               cancel,
		orchestratorProvider: mockOrchestratorFactory,
		bids:                 make(map[string]*bidState),
		allocator:            mockAllocator,
	}

	return &node
}

func TestNode(t *testing.T) {
	t.Parallel()

	t.Run("must be able to create a new node", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockResourceManager := NewMockResourceManager(ctrl)
		mockHardwareManager := NewMockHardwareManager(ctrl)
		mockNetwork := NewMockNetwork(ctrl)
		mockGeoIPLocator := NewMockGeoIPLocator(ctrl)

		rootCap := createRootCapabilityContext(t)

		collections := []string{"orchestrator_view", "contract"}
		db, err := repo.NewMemDB(collections)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		t.Cleanup(func() {
			db.Close()
		})

		node, err := New(
			*config.GetConfig(),
			afero.Afero{Fs: afero.NewMemMapFs()},
			&onboarding.Onboarding{},
			rootCap,
			"hostID",
			mockNetwork,
			mockResourceManager,
			bt.NewScheduler(1),
			mockHardwareManager,
			repo.NewOrchestratorView(db),
			mockGeoIPLocator,
			Geolocation{},
			PortConfig{AvailableRangeFrom: 49152, AvailableRangeTo: 65535},
		)
		require.NoError(t, err)
		require.NotNil(t, node)
	})

	t.Run("must be able to start and stop a node", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		node := getMockNode(t, ctrl)
		subscriptionID := uint64(1984)

		node.network.(*MockNetwork).EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(subscriptionID, nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().SetupBroadcastTopic(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().SetBroadcastAppScore(gomock.Any()).Return().AnyTimes()
		node.network.(*MockNetwork).EXPECT().Notify(gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.actor.(*MockActor).EXPECT().Start().Return(nil).Times(1)
		node.actor.(*MockActor).EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.actor.(*MockActor).EXPECT().Context().Return(context.Background()).AnyTimes()
		node.allocator.(*MockAllocator).EXPECT().Run().Return(nil).Times(1)
		err := node.Start()
		require.NoError(t, err)
		require.True(t, node.running.Load())

		node.orchestratorProvider.(*MockOrchestratorProvider).EXPECT().Orchestrators().Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().Unsubscribe(gomock.Any(), subscriptionID).Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().UnregisterMessageHandler(gomock.Any()).Return().AnyTimes()
		node.actor.(*MockActor).EXPECT().Stop().Return(nil).Times(1)
		node.allocator.(*MockAllocator).EXPECT().Stop(gomock.Any()).Return(nil).Times(1)
		err = node.Stop()
		require.NoError(t, err)
		require.False(t, node.running.Load())
	})

	t.Run("must delete expired bid requests", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		node := getMockNode(t, ctrl)
		subscriptionID := uint64(1984)
		node.bids["test"] = &bidState{
			expire: time.Now().Add(-1),
		}

		node.network.(*MockNetwork).EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(subscriptionID, nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().SetupBroadcastTopic(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.network.(*MockNetwork).EXPECT().SetBroadcastAppScore(gomock.Any()).Return().AnyTimes()
		node.network.(*MockNetwork).EXPECT().Notify(gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.actor.(*MockActor).EXPECT().Start().Return(nil).Times(1)
		node.actor.(*MockActor).EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.actor.(*MockActor).EXPECT().Context().Return(context.Background()).AnyTimes()
		node.allocator.(*MockAllocator).EXPECT().Run().Return(nil).Times(1)
		err := node.Start()
		require.NoError(t, err)

		err = retry.Do(func() error {
			if len(node.GetBidRequests()) > 0 {
				return fmt.Errorf("bids not deleted")
			}
			return nil
		},
			retry.Attempts(3),
			retry.Delay(bidStateGCInterval),
		)
		require.NoError(t, err)
	})
}

func TestDeployment(t *testing.T) {
	t.Parallel()

	t.Run("must be able to create an ensemble ID", func(t *testing.T) {
		t.Parallel()

		ensembleID, err := createEnsembleID("test")
		require.NoError(t, err)
		require.NotEmpty(t, ensembleID)
	})

	t.Run("must be able to create and store a new deployment", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		node := getMockNode(t, ctrl)
		msg := getMockDeploymentRequest(t)
		var wg sync.WaitGroup

		mockOrchestrator := NewMockOrchestratorAPI(ctrl)
		wg.Add(1)
		node.actor.(*MockActor).EXPECT().Send(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
			var response NewDeploymentResponse
			err := json.Unmarshal(msg.Message, &response)
			require.NoError(t, err)
			require.Equal(t, "OK", response.Status)
			require.NotEmpty(t, response.EnsembleID)

			wg.Done()
			return nil
		}).AnyTimes()
		node.actor.(*MockActor).EXPECT().Handle().DoAndReturn(func() actor.Handle {
			return getMockActorHandle(t)
		}).AnyTimes()
		node.actor.(*MockActor).EXPECT().CreateChild(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ actor.Handle, _ actor.BasicActorParams) (actor.Actor, error) {
				return NewMockActor(gomock.NewController(t)), nil
			},
		).AnyTimes()
		node.orchestratorProvider.(*MockOrchestratorProvider).EXPECT().NewOrchestrator(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ actor.Actor, _ jobs.EnsembleConfig) (jobs.OrchestratorAPI, error) {
				return mockOrchestrator, nil
			}).Times(1)
		mockOrchestrator.EXPECT().ID().Return("test").Times(3)
		mockOrchestrator.EXPECT().Deploy(gomock.Any()).Return(nil).Times(1)
		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
		require.NoError(t, err)
		mockOrchestrator.EXPECT().ActorPrivateKey().Return(privKey).Times(1)
		mockOrchestrator.EXPECT().Config().Return(jobs.EnsembleConfig{}).Times(1)
		mockOrchestrator.EXPECT().Manifest().Return(jobs.EnsembleManifest{}).Times(1)
		mockOrchestrator.EXPECT().Status().Return(jobs.DeploymentStatusRunning).Times(1)
		mockOrchestrator.EXPECT().DeploymentSnapshot().Return(jobs.DeploymentSnapshot{}).Times(1)
		node.handleNewDeployment(msg)
		wg.Wait()

		// Ensure that the orchestrator is stored in the db
		query := node.orchestratorRepo.GetQuery()
		query.Conditions = append(
			query.Conditions,
			repositories.LTE("Status", jobtypes.DeploymentStatusRunning),
		)
		dbOrchestrators, err := node.orchestratorRepo.FindAll(context.Background(), query)
		require.NoError(t, err)
		require.Len(t, dbOrchestrators, 1)
		require.Equal(t, "test", dbOrchestrators[0].OrchestratorID)
	})

	t.Run("must be able to restore a deployment", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		node := getMockNode(t, ctrl)

		// store the deployment in db
		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
		require.NoError(t, err)
		marshalledPrivKey, err := crypto.MarshalPrivateKey(privKey)
		require.NoError(t, err)
		_, err = node.orchestratorRepo.Create(context.Background(), jobtypes.OrchestratorView{
			OrchestratorID: "test",
			Status:         jobtypes.DeploymentStatusCommitting,
			PrivKey:        marshalledPrivKey,
		})
		require.NoError(t, err)

		// restore the deployment
		mockOrchestrator := NewMockOrchestratorAPI(ctrl)
		node.actor.(*MockActor).EXPECT().Limiter().Return(actor.NewRateLimiter(actor.DefaultRateLimiterConfig())).AnyTimes()
		node.network.(*MockNetwork).EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		node.orchestratorProvider.(*MockOrchestratorProvider).EXPECT().RestoreDeployment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(mockOrchestrator, nil).Times(1)
		mockOrchestrator.EXPECT().ID().Return("test").AnyTimes()
		err = node.restoreDeployments()
		require.NoError(t, err)
	})

	t.Run("must restore only committed deployments", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		node := getMockNode(t, ctrl)

		// store the deployment in db
		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
		require.NoError(t, err)
		marshalledPrivKey, err := crypto.MarshalPrivateKey(privKey)
		require.NoError(t, err)
		_, err = node.orchestratorRepo.Create(context.Background(), jobtypes.OrchestratorView{
			OrchestratorID: "test",
			Status:         jobtypes.DeploymentStatusPreparing,
			PrivKey:        marshalledPrivKey,
		})
		require.NoError(t, err)

		// No expected calls since the deployment is not committed
		err = node.restoreDeployments()
		require.NoError(t, err)
	})
}
