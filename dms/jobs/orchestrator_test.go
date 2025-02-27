// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func getMockEnsembleConfig(t *testing.T) jobtypes.EnsembleConfig {
	t.Helper()

	return jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"allocation1": {
					Type:     "service",
					Executor: "docker",
					Resources: types.Resources{
						CPU: types.CPU{
							ClockSpeed: 2.4,
							Cores:      2,
							Model:      "Intel Core i7",
							Vendor:     "Intel",
						},
						GPUs: []types.GPU{
							{
								Model:      "NVIDIA GeForce GTX 1080",
								Vendor:     "NVIDIA",
								VRAM:       8,
								Index:      0,
								PCIAddress: "0000:01:00.0",
							},
						},
						RAM: types.RAM{
							Size:       16,
							ClockSpeed: 2400,
						},
						Disk: types.Disk{
							Size:       256,
							Model:      "Samsung 970 EVO",
							Vendor:     "Samsung",
							Type:       "SSD",
							Interface:  "NVMe",
							ReadSpeed:  3500,
							WriteSpeed: 2500,
						},
					},
					Execution: types.SpecConfig{
						Type: "docker",
						Params: map[string]interface{}{
							"image": "nginx",
						},
					},
				},
			},
			Nodes: map[string]NodeConfig{
				"node1": {
					Allocations: []string{"allocation1"},
				},
			},
		},
	}
}

func withTimeout(t *testing.T, name string, duration time.Duration, testFunc func(ctx context.Context, t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		t.Helper()

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		done := make(chan struct{})
		go func() {
			testFunc(ctx, t)
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			t.Fatalf("test %q timed out after %s", name, duration)
		}
	})
}

func TestOrchestratorProvider(t *testing.T) {
	t.Parallel()

	t.Run("must be able to create a new orchestrator from the provider", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		oProvider := NewOrchestratorProvider()
		orchestrator, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator)
	})

	t.Run("must be able to restore an orchestrator", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)

		oProvider := NewOrchestratorProvider()
		orchestrator, err := oProvider.RestoreDeployment(mockActor, "id", getMockEnsembleConfig(t),
			EnsembleManifest{}, DeploymentStatusPreparing, DeploymentSnapshot{})
		require.NoError(t, err)
		require.NotNil(t, orchestrator)
	})

	t.Run("must not be able to create duplicate orchestrators", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any()).Times(1)

		oProvider := NewOrchestratorProvider()
		orchestrator, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator)

		_, err = oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id", mockActor, getMockEnsembleConfig(t),
		)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrchestratorExists)
	})

	t.Run("must be able to get orchestrator by id", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		oProvider := NewOrchestratorProvider()
		orchestrator, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator)

		orchestrator, err = oProvider.GetOrchestrator("id")
		require.NoError(t, err)
		require.NotNil(t, orchestrator)
	})

	t.Run("must be able to delete orchestrator by id", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		oProvider := NewOrchestratorProvider()
		orchestrator, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator)

		oProvider.DeleteOrchestrator("id")

		_, err = oProvider.GetOrchestrator("id")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrchestratorNotFound)
	})

	t.Run("must be able to list orchestrators", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any()).Times(2)

		oProvider := NewOrchestratorProvider()
		orchestrator1, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id1", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator1)

		orchestrator2, err := oProvider.NewOrchestrator(
			context.Background(), af, workDir, "id2", mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator2)

		orchestrators := oProvider.Orchestrators()
		require.Len(t, orchestrators, 2)

		_, ok := orchestrators["id1"]
		require.True(t, ok)
		_, ok = orchestrators["id2"]
		require.True(t, ok)
	})
}

func TestOrchestrator(t *testing.T) {
	t.Parallel()

	ensembleID := "ensembleID"
	t.Run("must be able to create an orchestrator", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)
		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privKey)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		ctrl := gomock.NewController(t)
		mockPeer := NewMockNetwork(ctrl)
		mockPeer.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)
		mockPeer.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		actr := actor.CreateActor(t, mockPeer, capContext)
		require.NoError(t, actr.Start())

		orchActor, err := actr.CreateChild("orchestratorID", actr.Handle())
		require.NoError(t, err)

		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, "id", orchActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)
		require.NotNil(t, orchestrator)
	})

	t.Run("must be able to set and get status", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)
		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privKey)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		ctrl := gomock.NewController(t)
		mockPeer := NewMockNetwork(ctrl)
		mockPeer.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)
		mockPeer.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		actr := actor.CreateActor(t, mockPeer, capContext)
		require.NoError(t, actr.Start())

		orchActor, err := actr.CreateChild("orchestratorID", actr.Handle())
		require.NoError(t, err)

		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, "id", orchActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)

		orchestrator.setStatus(DeploymentStatusPreparing)
		assert.Equal(t, DeploymentStatusPreparing, orchestrator.Status())
	})

	t.Run("must be able to get ensemble config", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)
		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privKey)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		ctrl := gomock.NewController(t)
		mockPeer := NewMockNetwork(ctrl)
		mockPeer.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)
		mockPeer.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		actr := actor.CreateActor(t, mockPeer, capContext)
		require.NoError(t, actr.Start())

		orchActor, err := actr.CreateChild("orchestratorID", actr.Handle())
		require.NoError(t, err)

		ensembleConfig := getMockEnsembleConfig(t)
		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, "id", orchActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)

		actualConfig := orchestrator.Config()
		require.Equal(t, ensembleConfig, actualConfig)

		// Ensure that the config is returned safely
		actualConfig.V1.Allocations = nil
		newConfig := orchestrator.Config()
		require.Equal(t, ensembleConfig, newConfig)
	})

	t.Run("must be able to get id", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)
		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privKey)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		ctrl := gomock.NewController(t)
		mockPeer := NewMockNetwork(ctrl)
		mockPeer.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)
		mockPeer.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		actr := actor.CreateActor(t, mockPeer, capContext)
		require.NoError(t, actr.Start())

		orchActor, err := actr.CreateChild("orchestratorID", actr.Handle())
		require.NoError(t, err)

		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, "id", orchActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)

		assert.Equal(t, "id", orchestrator.ID())
	})

	t.Run("must be able to deploy an ensemble", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		var (
			bidReplyHandler actor.Behavior
			didReply        atomic.Bool
			wg              sync.WaitGroup
		)

		privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)

		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privK)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		rootSec, err := actor.NewBasicSecurityContext(pubK, privK, capContext)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testDID := did.FromPublicKey(pubK)

		provider, err := did.ProviderFromPrivateKey(privK)
		require.NoError(t, err, "provider from public key")

		mockActor.EXPECT().Security().Return(rootSec).AnyTimes()

		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, ensembleID, mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)

		wg.Add(1)
		mockActor.EXPECT().AddBehavior(behaviors.BidReplyBehavior, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, continuation actor.Behavior, _ ...actor.BehaviorOption) error {
				if !didReply.Load() {
					bidReplyHandler = continuation
				}
				return nil
			}).AnyTimes()

		mockActor.EXPECT().Handle().Return(actor.Handle{
			ID:  rootSec.ID(),
			DID: testDID,
			Address: actor.Address{
				HostID:       "hostID",
				InboxAddress: "inboxAddress",
			},
		}).AnyTimes()
		mockActor.EXPECT().Publish(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
			// bid only once
			if !didReply.Load() {
				didReply.Store(true)
				go func() {
					defer wg.Done()

					bid := jobtypes.Bid{
						V1: &jobtypes.BidV1{
							EnsembleID: ensembleID,
							NodeID:     "node1",
							Peer:       peerID.String(),
							Handle: actor.Handle{
								ID:  rootSec.ID(),
								DID: testDID,
								Address: actor.Address{
									HostID:       "hostID",
									InboxAddress: "inboxAddress",
								},
							},
							Location: Location{},
						},
					}
					err = bid.Sign(provider)
					require.NoError(t, err)

					env, err := actor.ReplyTo(msg, bid)
					require.NoError(t, err)

					bidReplyHandler(env)
				}()
			}
			return nil
		}).AnyTimes()
		mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
			respChan := make(chan actor.Envelope, 1)
			go func() {
				switch msg.Behavior {
				case behaviors.CommitDeploymentBehavior:
					reply := CommitDeploymentResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.CommitDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationDeploymentBehavior:
					reply := AllocationDeploymentResponse{
						OK: true,
						Allocations: map[string]actor.Handle{
							"allocation1": {},
						},
					}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"):
					reply := SubnetDestroyResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"), reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.SubnetAddPeerBehavior:
					reply := SubnetAddPeerResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetAddPeerBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.SubnetDNSAddRecordsBehavior:
					reply := SubnetDNSAddRecordsResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetDNSAddRecordsBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationStartBehavior:
					reply := AllocationStartResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationStartBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.AllocationLogsBehavior, ensembleID):
					reply := AllocationLogsResponse{
						Stdout: []byte{1, 2, 3, 4, 5},
						Stderr: []byte{1, 2, 3, 4, 5},
						Error:  "",
					}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationLogsBehavior, reply)
					require.NoError(t, err)
					respChan <- env

				default:
					t.Errorf("unexpected behavior: %s", msg.Behavior)
				}
			}()

			return respChan, nil
		}).AnyTimes()
		err = orchestrator.Deploy(time.Now().Add(10 * time.Second))
		require.NoError(t, err)

		wg.Wait()

		// assert the manifest
		manifest := orchestrator.Manifest()
		require.NotNil(t, manifest)
		require.Len(t, manifest.Allocations, 1)
		require.Len(t, manifest.Nodes, 1)

		// get the allocation logs
		logs, err := orchestrator.GetAllocationLogs("allocation1")
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3, 4, 5}, logs.Stdout)
		require.Equal(t, []byte{1, 2, 3, 4, 5}, logs.Stderr)
	})

	t.Run("must be able to revert a deployment", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		var (
			bidReplyHandler actor.Behavior
			didReply        atomic.Bool
			wg              sync.WaitGroup
		)

		privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		id, err := pubK.Raw()
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)
		testDID := did.FromPublicKey(pubK)
		provider := did.NewProvider(testDID, privK)

		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, ensembleID, mockActor, getMockEnsembleConfig(t),
		)
		require.NoError(t, err)

		wg.Add(2) // publish and send
		mockActor.EXPECT().AddBehavior(behaviors.BidReplyBehavior, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, continuation actor.Behavior, _ ...actor.BehaviorOption) error {
				if !didReply.Load() {
					bidReplyHandler = continuation
				}
				return nil
			}).AnyTimes()
		mockActor.EXPECT().Handle().Return(actor.Handle{
			ID:  actor.ID{PublicKey: id},
			DID: testDID,
			Address: actor.Address{
				HostID:       "hostID",
				InboxAddress: "inboxAddress",
			},
		}).AnyTimes()
		mockActor.EXPECT().Publish(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
			// bid only once
			if !didReply.Load() {
				didReply.Store(true)
				go func() {
					defer wg.Done()

					bid := jobtypes.Bid{
						V1: &jobtypes.BidV1{
							EnsembleID: ensembleID,
							NodeID:     "node1",
							Peer:       peerID.String(),
							Handle: actor.Handle{
								ID:  actor.ID{PublicKey: id},
								DID: testDID,
								Address: actor.Address{
									HostID:       "hostID",
									InboxAddress: "inboxAddress",
								},
							},
							Location: Location{},
						},
					}
					err = bid.Sign(provider)
					require.NoError(t, err)

					env, err := actor.ReplyTo(msg, bid)
					require.NoError(t, err)

					bidReplyHandler(env)
				}()
			}
			return nil
		}).AnyTimes()
		mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
			respChan := make(chan actor.Envelope, 1)
			go func() {
				switch msg.Behavior {
				case behaviors.CommitDeploymentBehavior:
					reply := CommitDeploymentResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.CommitDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationDeploymentBehavior:
					reply := AllocationDeploymentResponse{
						OK: true,
						Allocations: map[string]actor.Handle{
							"allocation1": {},
						},
					}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID):
					reply := SubnetCreateResponse{OK: false, Error: "subnet create failed"}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"), reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationStartBehavior:
					reply := AllocationStartResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationStartBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID):
					reply := AllocationStopResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID), reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID):
					reply := SubnetDestroyResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), reply)
					require.NoError(t, err)
					respChan <- env

				default:
					t.Errorf("unexpected behavior: %s", msg.Behavior)
				}
			}()

			return respChan, nil
		}).AnyTimes()
		mockActor.EXPECT().Send(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
			if msg.Behavior == behaviors.RevertDeploymentBehavior {
				wg.Done()
			}
			return nil
		}).Times(1)
		err = orchestrator.Deploy(time.Now().Add(10 * time.Second))
		require.Error(t, err)

		orchestrator.Shutdown()
		wg.Wait()
	})

	t.Run("must be able to request bid from a specific peer", func(t *testing.T) {
		t.Parallel()
		af := afero.Afero{Fs: afero.NewMemMapFs()}
		workDir := t.TempDir()

		ctrl := gomock.NewController(t)
		mockActor := NewMockActor(ctrl)
		mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())

		var (
			bidReplyHandler actor.Behavior
			didReply        atomic.Bool
			wg              sync.WaitGroup
		)

		privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		assert.NoError(t, err)

		rootDID, root := actor.MakeRootTrustContext(t)
		actorDID, trust := actor.MakeTrustContext(t, privK)
		capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

		rootSec, err := actor.NewBasicSecurityContext(pubK, privK, capContext)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testDID := did.FromPublicKey(pubK)

		provider, err := did.ProviderFromPrivateKey(privK)
		require.NoError(t, err, "provider from public key")

		mockActor.EXPECT().Security().Return(rootSec).AnyTimes()

		ensemble := getMockEnsembleConfig(t)
		ensemble.V1.Nodes = map[string]jobtypes.NodeConfig{
			"node1": {
				Peer:        peerID.String(),
				Allocations: []string{"allocation1"},
			},
		}
		orchestrator, err := NewOrchestrator(
			context.Background(), af, workDir, ensembleID, mockActor, ensemble,
		)
		require.NoError(t, err)

		wg.Add(1) // bid event
		mockActor.EXPECT().AddBehavior(behaviors.BidReplyBehavior, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, continuation actor.Behavior, _ ...actor.BehaviorOption) error {
				if !didReply.Load() {
					bidReplyHandler = continuation
				}
				return nil
			}).AnyTimes()

		mockActor.EXPECT().Handle().Return(actor.Handle{
			ID:  rootSec.ID(),
			DID: testDID,
			Address: actor.Address{
				HostID:       "hostID",
				InboxAddress: "inboxAddress",
			},
		}).AnyTimes()
		mockActor.EXPECT().Publish(gomock.Any()).DoAndReturn(func(_ actor.Envelope) error {
			// we do not bid on broadcast
			return nil
		}).AnyTimes()
		mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
			respChan := make(chan actor.Envelope, 1)
			go func() {
				switch msg.Behavior {
				case behaviors.CommitDeploymentBehavior:
					reply := CommitDeploymentResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.CommitDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationDeploymentBehavior:
					reply := AllocationDeploymentResponse{
						OK: true,
						Allocations: map[string]actor.Handle{
							"allocation1": {},
						},
					}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationDeploymentBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"):
					reply := SubnetCreateResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"), reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.SubnetAddPeerBehavior:
					reply := SubnetAddPeerResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetAddPeerBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.SubnetDNSAddRecordsBehavior:
					reply := SubnetDNSAddRecordsResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetDNSAddRecordsBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case behaviors.AllocationStartBehavior:
					reply := AllocationStartResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationStartBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID):
					reply := SubnetDestroyResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), reply)
					require.NoError(t, err)
					respChan <- env
				case fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID):
					reply := AllocationStopResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID), reply)
					require.NoError(t, err)
					respChan <- env
				default:
					t.Errorf("unexpected behavior: %s", msg.Behavior)
				}
			}()

			return respChan, nil
		}).Times(8)
		mockActor.EXPECT().Send(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
			if msg.Behavior == behaviors.BidRequestBehavior {
				go func() {
					defer wg.Done()

					bid := jobtypes.Bid{
						V1: &jobtypes.BidV1{
							EnsembleID: ensembleID,
							NodeID:     "node1",
							Peer:       peerID.String(),
							Handle: actor.Handle{
								ID:  rootSec.ID(),
								DID: testDID,
								Address: actor.Address{
									HostID:       "hostID",
									InboxAddress: "inboxAddress",
								},
							},
							Location: Location{},
						},
					}
					err = bid.Sign(provider)
					require.NoError(t, err)

					env, err := actor.ReplyTo(msg, bid)
					require.NoError(t, err)

					bidReplyHandler(env)
				}()
			}
			return nil
		})
		err = orchestrator.Deploy(time.Now().Add(10 * time.Second))
		require.NoError(t, err)
		wg.Wait()

		orchestrator.Shutdown()
	})
}

func TestOrchestrator_Supervisor(t *testing.T) {
	ensembleID := "ensembleID"
	withTimeout(t,
		"must be able to escalate failure if healthcheck fails",
		3*time.Minute,
		func(_ context.Context, t *testing.T) {
			t.Parallel()
			af := afero.Afero{Fs: afero.NewMemMapFs()}
			workDir := t.TempDir()

			ctrl := gomock.NewController(t)
			mockActor := NewMockActor(ctrl)

			var (
				bidReplyHandler actor.Behavior
				didReply        atomic.Bool
				wg              sync.WaitGroup
			)

			privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
			assert.NoError(t, err)

			rootDID, root := actor.MakeRootTrustContext(t)
			actorDID, trust := actor.MakeTrustContext(t, privK)
			capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

			rootSec, err := actor.NewBasicSecurityContext(pubK, privK, capContext)
			require.NoError(t, err)

			peerID, err := peer.IDFromPublicKey(pubK)
			require.NoError(t, err)

			testDID := did.FromPublicKey(pubK)

			provider, err := did.ProviderFromPrivateKey(privK)
			require.NoError(t, err, "provider from public key")

			mockActor.EXPECT().Security().Return(rootSec).AnyTimes()
			mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())
			mockActor.EXPECT().AddBehavior(behaviors.BidReplyBehavior, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ string, continuation actor.Behavior, _ ...actor.BehaviorOption) error {
					if !didReply.Load() {
						bidReplyHandler = continuation
					}
					return nil
				}).AnyTimes()
			mockActor.EXPECT().Handle().Return(actor.Handle{
				ID:  rootSec.ID(),
				DID: testDID,
				Address: actor.Address{
					HostID:       "hostID",
					InboxAddress: "inboxAddress",
				},
			}).AnyTimes()
			orchestrator, err := NewOrchestrator(
				context.Background(), af, workDir, ensembleID, mockActor, getMockEnsembleConfig(t),
			)
			require.NoError(t, err)

			wg.Add(1)
			mockActor.EXPECT().Publish(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
				// bid only once
				if !didReply.Load() {
					didReply.Store(true)
					go func() {
						defer wg.Done()

						bid := jobtypes.Bid{
							V1: &jobtypes.BidV1{
								EnsembleID: ensembleID,
								NodeID:     "node1",
								Peer:       peerID.String(),
								Handle: actor.Handle{
									ID:  rootSec.ID(),
									DID: testDID,
									Address: actor.Address{
										HostID:       "hostID",
										InboxAddress: "inboxAddress",
									},
								},
								Location: Location{},
							},
						}
						err = bid.Sign(provider)
						require.NoError(t, err)

						env, err := actor.ReplyTo(msg, bid)
						require.NoError(t, err)

						bidReplyHandler(env)
					}()
				}
				return nil
			}).AnyTimes()
			mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
				respChan := make(chan actor.Envelope, 1)
				go func() {
					switch msg.Behavior {
					case behaviors.CommitDeploymentBehavior:
						reply := CommitDeploymentResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.CommitDeploymentBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.AllocationDeploymentBehavior:
						reply := AllocationDeploymentResponse{
							OK: true,
							Allocations: map[string]actor.Handle{
								"allocation1": {},
							},
						}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationDeploymentBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID):
						reply := SubnetCreateResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"), reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.AllocationStartBehavior:
						reply := AllocationStartResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationStartBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID):
						reply := AllocationStopResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID):
						reply := SubnetDestroyResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.SubnetAddPeerBehavior:
						reply := SubnetAddPeerResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetAddPeerBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.SubnetDNSAddRecordsBehavior:
						reply := SubnetDNSAddRecordsResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetDNSAddRecordsBehavior, reply)
						require.NoError(t, err)
						respChan <- env

					default:
						t.Errorf("unexpected behavior: %s", msg.Behavior)
					}
				}()

				return respChan, nil
			}).Times(6)

			err = orchestrator.Deploy(time.Now().Add(60 * time.Second))
			require.NoError(t, err)
			wg.Wait() // wait until escalation happens

			// supervisor checks
			wg.Add(4) // we need 3 healthchecks to fail and a restart to happen
			mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
				respChan := make(chan actor.Envelope, 1)
				go func() {
					switch msg.Behavior {
					case behaviors.RegisterHealthcheckBehavior: // supervisor
						reply := RegisterHealthcheckResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.RegisterHealthcheckBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case actor.HealthCheckBehavior:
						defer wg.Done()
						reply := HealthCheckResponse{OK: false}
						env, err := actor.Message(actor.Handle{}, msg.From, actor.HealthCheckBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.AllocationRestartBehavior:
						defer wg.Done()
						reply := AllocationRestartResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationRestartBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID):
						reply := SubnetDestroyResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID):
						reply := AllocationStopResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					default:
						t.Errorf("unexpected behavior: %s", msg.Behavior)
					}
				}()

				return respChan, nil
			}).Times(6)
			wg.Wait() // wait until escalation happens

			orchestrator.Shutdown()
		},
	)
	withTimeout(t,
		"must be able to escalate failure if supervisor fails",
		3*time.Minute,
		func(_ context.Context, t *testing.T) {
			t.Parallel()
			af := afero.Afero{Fs: afero.NewMemMapFs()}
			workDir := t.TempDir()

			ctrl := gomock.NewController(t)
			mockActor := NewMockActor(ctrl)

			var (
				bidReplyHandler actor.Behavior
				didReply        atomic.Bool
				wg              sync.WaitGroup
			)

			privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
			assert.NoError(t, err)

			rootDID, root := actor.MakeRootTrustContext(t)
			actorDID, trust := actor.MakeTrustContext(t, privK)
			capContext := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)

			rootSec, err := actor.NewBasicSecurityContext(pubK, privK, capContext)
			require.NoError(t, err)

			peerID, err := peer.IDFromPublicKey(pubK)
			require.NoError(t, err)

			testDID := did.FromPublicKey(pubK)

			provider, err := did.ProviderFromPrivateKey(privK)
			require.NoError(t, err, "provider from public key")

			mockActor.EXPECT().Security().Return(rootSec).AnyTimes()
			mockActor.EXPECT().AddBehavior(behaviors.NotifyTaskTerminationBehavior, gomock.Any(), gomock.Any())
			mockActor.EXPECT().AddBehavior(behaviors.BidReplyBehavior, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ string, continuation actor.Behavior, _ ...actor.BehaviorOption) error {
					if !didReply.Load() {
						bidReplyHandler = continuation
					}
					return nil
				}).AnyTimes()
			mockActor.EXPECT().Handle().Return(actor.Handle{
				ID:  rootSec.ID(),
				DID: testDID,
				Address: actor.Address{
					HostID:       "hostID",
					InboxAddress: "inboxAddress",
				},
			}).AnyTimes()
			orchestrator, err := NewOrchestrator(
				context.Background(), af, workDir, ensembleID, mockActor, getMockEnsembleConfig(t),
			)
			require.NoError(t, err)

			wg.Add(1)
			mockActor.EXPECT().Publish(gomock.Any()).DoAndReturn(func(msg actor.Envelope) error {
				// bid only once
				if !didReply.Load() {
					didReply.Store(true)
					go func() {
						defer wg.Done()

						bid := jobtypes.Bid{
							V1: &jobtypes.BidV1{
								EnsembleID: ensembleID,
								NodeID:     "node1",
								Peer:       peerID.String(),
								Handle: actor.Handle{
									ID:  rootSec.ID(),
									DID: testDID,
									Address: actor.Address{
										HostID:       "hostID",
										InboxAddress: "inboxAddress",
									},
								},
								Location: Location{},
							},
						}
						err = bid.Sign(provider)
						require.NoError(t, err)

						env, err := actor.ReplyTo(msg, bid)
						require.NoError(t, err)

						bidReplyHandler(env)
					}()
				}
				return nil
			}).AnyTimes()
			mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
				respChan := make(chan actor.Envelope, 1)
				go func() {
					switch msg.Behavior {
					case behaviors.CommitDeploymentBehavior:
						reply := CommitDeploymentResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.CommitDeploymentBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.AllocationDeploymentBehavior:
						reply := AllocationDeploymentResponse{
							OK: true,
							Allocations: map[string]actor.Handle{
								"allocation1": {},
							},
						}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationDeploymentBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"):
						reply := SubnetCreateResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "ensembleID"), reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.SubnetAddPeerBehavior:
						reply := SubnetAddPeerResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetAddPeerBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.SubnetDNSAddRecordsBehavior:
						reply := SubnetDNSAddRecordsResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.SubnetDNSAddRecordsBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case behaviors.AllocationStartBehavior:
						reply := AllocationStartResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationStartBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					default:
						t.Errorf("unexpected behavior: %s", msg.Behavior)
					}
				}()

				return respChan, nil
			}).Times(6)
			err = orchestrator.Deploy(time.Now().Add(60 * time.Second))
			require.NoError(t, err)
			wg.Wait()

			// supervisor checks
			// we do not respond to healthcheck to simulate a failure
			wg.Add(4) // we need 3 healthchecks to fail and a restart to happen
			mockActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
				respChan := make(chan actor.Envelope, 1)
				go func() {
					switch msg.Behavior {
					case behaviors.RegisterHealthcheckBehavior: // supervisor
						reply := RegisterHealthcheckResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.RegisterHealthcheckBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case actor.HealthCheckBehavior:
						defer wg.Done()
						// do nothing to simulate a failure
					case behaviors.AllocationRestartBehavior:
						defer wg.Done()
						reply := AllocationRestartResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationRestartBehavior, reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID):
						reply := SubnetDestroyResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					case fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID):
						reply := AllocationStopResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID), reply)
						require.NoError(t, err)
						respChan <- env
					default:
						t.Errorf("unexpected behavior: %s", msg.Behavior)
					}
				}()

				return respChan, nil
			}).Times(6)
			wg.Wait() // wait until escalation happens

			orchestrator.Shutdown()
		},
	)
}
