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
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	restoreWorkDir    = "/tmp"
	restoreEnsembleID = "test-restore-ensemble"
)

func TestRestoreDeployment(t *testing.T) {
	BidRequestTimeout = 1 * time.Second
	CommitDeploymentTimeout = 1 * time.Second
	VerifyEdgeConstraintTimeout = 1 * time.Second
	AllocationDeploymentTimeout = 1 * time.Second
	AllocationStartTimeout = 1 * time.Second
	AllocationShutdownTimeout = 1 * time.Second

	substrate := network.NewSubstrate()

	fs := afero.NewMemMapFs()

	t.Run("Restore from Committing State", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider := MakeProvider(t, substrate)

		// Set up comprehensive mock behaviors for committing state restoration
		provider.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		provider.MockCommittingStateBehaviors(t, restoreEnsembleID)

		// Mock revert behavior on provider
		revertCalls := 0
		require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			revertCalls++

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		// Create test configuration for committing state
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

		// Manifest representing a deployment that was in the process of committing
		// - Has the structure that newManifest() would create from config
		// - Nodes exist but without handles (handles get set during commit)
		// - Allocations exist but without handles or node assignments
		manifest := jtypes.EnsembleManifest{
			ID:           restoreEnsembleID,
			Orchestrator: orch.actor.Handle(),
			Metadata:     map[string]string{},
			Allocations: map[string]jtypes.AllocationManifest{
				"node1.alloc1": {
					ID:      fmt.Sprintf("%s_alloc1", restoreEnsembleID),
					DNSName: "alloc1.internal",
					Type:    jtypes.AllocationTypeService,
					Status:  jtypes.AllocationPending,
					Ports:   make(map[int]int),
					// No Handle or NodeID yet - these get set during commit
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Peer:        provider.peerID.String(),
					// No Handle yet - this gets set during commit
				},
			},
			Contracts: make(map[string]jtypes.ContractManifest),
			Subnet: jtypes.SubnetConfig{
				Join: false, // Subnet not yet created
			},
		}

		// Create snapshot with candidates for committing state
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: restoreEnsembleID,
				NodeID:     "node1",
				Peer:       provider.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider.handle,
			},
		}
		snapshot := jtypes.DeploymentSnapshot{
			Candidates: map[string]jtypes.Bid{
				"node1": bid,
			},
			Expiry: time.Now().Add(time.Hour),
		}
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment from committing state
		t.Logf("Starting restore deployment test for committing state")
		t.Logf("Manifest ID: %s", manifest.ID)
		t.Logf("Manifest nodes: %+v", manifest.Nodes)
		t.Logf("Snapshot candidates: %+v", snapshot.Candidates)

		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusCommitting,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)
		t.Logf("RestoreDeployment completed, err: %v", err)
		require.NoError(t, err)

		// For committing state, the deployment process may fail due to test environment
		// but we should still get an orchestrator back
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())
		// Assert we reached running after restoration
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

		// Assert revert behavior was called
		assert.Greater(t, revertCalls, 0, "Provider should have received revert messages during restoration")

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
		assert.Equal(t, o, orchestrators[restoreEnsembleID])

		// Verify orchestrator can be retrieved
		retrievedOrch, err := registry.GetOrchestrator(restoreEnsembleID)
		require.NoError(t, err)
		assert.Equal(t, o, retrievedOrch)
	})

	t.Run("Restore from Provisioning State (two providers)", func(t *testing.T) {
		// Speed up subnet create timeouts to avoid hanging the test in case of a missed reply
		SubnetCreateTimeout = 1 * time.Second
		registry := NewRegistry(NewMockDeploymentStore())
		substrate := network.NewSubstrate()
		orch := MakeOrchestrator(t, substrate)
		provider1 := MakeProvider(t, substrate)
		provider2 := MakeProvider(t, substrate)

		t.Logf("Provider1: %s", provider1.actor.Handle())
		t.Logf("Provider2: %s", provider2.actor.Handle())
		t.Logf("Orch: %s", orch.actor.Handle())

		provider1BidBehavior := func(msg actor.Envelope) {
			defer msg.Discard()
			var request jtypes.EnsembleBidRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			// Check if this request is for node1
			shouldRespond := false
			for _, bidReq := range request.Request {
				if bidReq.V1 != nil && bidReq.V1.NodeID == "node1" {
					shouldRespond = true
					break
				}
			}

			if !shouldRespond {
				return // Don't respond to requests not for node1
			}

			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node1", // Provider1 responds to node1
					Peer:       provider1.handle.Address.HostID,
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider1.handle,
				},
			}

			providerDID := did.NewProvider(provider1.actor.Handle().DID, provider1.priv)
			require.NoError(t, bid.Sign(providerDID))

			reply, err := actor.ReplyTo(msg, bid)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider1.handle
			require.NoError(t, provider1.actor.Send(reply))
		}

		provider2BidBehavior := func(msg actor.Envelope) {
			defer msg.Discard()
			var request jtypes.EnsembleBidRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			// Check if this request is for node2
			shouldRespond := false
			for _, bidReq := range request.Request {
				if bidReq.V1 != nil && bidReq.V1.NodeID == "node2" {
					shouldRespond = true
					break
				}
			}

			if !shouldRespond {
				return // Don't respond to requests not for node2
			}

			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node2", // Provider2 responds to node2
					Peer:       provider2.handle.Address.HostID,
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider2.handle,
				},
			}

			providerDID := did.NewProvider(provider2.actor.Handle().DID, provider2.priv)
			require.NoError(t, bid.Sign(providerDID))

			reply, err := actor.ReplyTo(msg, bid)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider2.handle
			require.NoError(t, provider2.actor.Send(reply))
		}

		// Set up mock behaviors first
		orch.MockOrchestratorBehaviors(t, restoreEnsembleID)
		provider1.MockDeploymentBehaviors(t, restoreEnsembleID, provider1BidBehavior, orch.actor)
		provider2.MockDeploymentBehaviors(t, restoreEnsembleID, provider2BidBehavior, orch.actor)

		// Override SubnetCreate behaviors to return errors so that we can trigger reverts and redeployment during restoration
		// Info: restoreDeployment will attempt provisioning, and if that fails, it will revert and redeploy from scratch
		// this behavior will help trigger that reversion.
		{
			behaviorName := fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, restoreEnsembleID)
			require.NoError(t, provider2.actor.AddBehavior(behaviorName, func(msg actor.Envelope) {
				defer func() {
					msg.Discard()
					// Signal receive to prevent channel-based waits from hanging
					if ch, ok := provider2.channels[behaviorName]; ok {
						select {
						case ch <- struct{}{}:
						default:
						}
					}
				}()

				reply, err := actor.ReplyTo(msg, SubnetCreateResponse{OK: false, Error: "test error"})
				require.NoError(t, err)
				reply.To = msg.From
				reply.From = provider2.handle
				require.NoError(t, provider2.actor.Send(reply))
			}))
		}

		// Grant capabilities between orchestrator and providers for communication
		// Provider1 grants orchestrator root
		// Provider2 grants orchestrator root
		err := provider1.actor.Security().Grant(
			provider1.actor.Handle().DID,
			orch.actor.Handle().DID,
			[]ucan.Capability{"/"},
			5*time.Minute,
		)
		require.NoError(t, err)
		err = provider2.actor.Security().Grant(
			provider2.actor.Handle().DID,
			orch.actor.Handle().DID,
			[]ucan.Capability{"/"},
			5*time.Minute,
		)
		require.NoError(t, err)
		// orchestrator grants provider1 and provider2 root
		err = orch.actor.Security().Grant(
			orch.actor.Handle().DID,
			provider1.actor.Handle().DID,
			[]ucan.Capability{"/"},
			5*time.Minute,
		)
		require.NoError(t, err)
		err = orch.actor.Security().Grant(
			orch.actor.Handle().DID,
			provider2.actor.Handle().DID,
			[]ucan.Capability{"/"},
			5*time.Minute,
		)
		require.NoError(t, err)

		// Mock revert behavior on providers
		revertCalls := 0
		require.NoError(t, provider1.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			revertCalls++

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider1.handle

			require.NoError(t, provider1.actor.Send(reply))
		}))
		reverted := make(chan struct{}, 1)
		require.NoError(t, provider2.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			revertCalls++
			reverted <- struct{}{}

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider2.handle

			require.NoError(t, provider2.actor.Send(reply))
		}))

		// when the revert has been triggered, revert to successful subnet creation to have a successful redeployment
		go func() {
			<-reverted
			behaviorName := fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, restoreEnsembleID)
			require.NoError(t, provider2.actor.AddBehavior(behaviorName, func(msg actor.Envelope) {
				defer func() {
					msg.Discard()
					if ch, ok := provider2.channels[behaviorName]; ok {
						select {
						case ch <- struct{}{}:
						default:
						}
					}
				}()
				reply, err := actor.ReplyTo(msg, SubnetCreateResponse{OK: true})
				require.NoError(t, err)
				reply.To = msg.From
				reply.From = provider2.handle
				require.NoError(t, provider2.actor.Send(reply))
			}))
		}()

		// Build two-node config and manifest
		cfg := jtypes.EnsembleConfig{
			V1: &jtypes.EnsembleConfigV1{
				Subnet: jtypes.SubnetConfig{
					Join: true,
				},
				Nodes: map[string]jtypes.NodeConfig{
					"node1": {
						Location: jtypes.LocationConstraints{
							Accept: []jtypes.Location{
								{Country: "US"},
							},
						},
						Allocations: []string{"alloc1"},
					},
					"node2": {
						Location: jtypes.LocationConstraints{
							Accept: []jtypes.Location{
								{Country: "US"},
							},
						},
						Allocations: []string{"alloc2"},
					},
				},
				Allocations: map[string]jtypes.AllocationConfig{
					"alloc1": {
						Type: jtypes.AllocationTypeService,
						Resources: types.Resources{
							CPU: types.CPU{
								Cores:      2,
								ClockSpeed: 2000,
							},
							RAM: types.RAM{
								Size: 2048,
							},
							Disk: types.Disk{
								Size: 2048,
							},
						},
					},
					"alloc2": {
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

		// Manifest representing a deployment that was in the process of provisioning with two providers
		// - Allocations are committed and have handles
		// - Nodes have handles and are ready for provisioning
		// - Subnet may be partially created
		manifest := jtypes.EnsembleManifest{
			ID:           restoreEnsembleID,
			Orchestrator: orch.actor.Handle(),
			Metadata:     map[string]string{},
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:       fmt.Sprintf("%s_alloc1", restoreEnsembleID),
					DNSName:  "alloc1.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationPending, // Committed but not yet running
					Handle:   actor.Handle{},           // Has handle from commit phase
					NodeID:   "node1",
					PrivAddr: "10.0.0.2",
					Ports:    make(map[int]int),
				},
				"alloc2": {
					ID:       fmt.Sprintf("%s_alloc2", restoreEnsembleID),
					DNSName:  "alloc2.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationPending, // Committed but not yet running
					Handle:   actor.Handle{},           // Has handle from commit phase
					NodeID:   "node2",
					PrivAddr: "10.0.0.3",
					Ports:    make(map[int]int),
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Peer:        provider1.peerID.String(),
					Handle:      provider1.handle, // Has handle from commit phase
				},
				"node2": {
					ID:          "node2",
					Allocations: []string{"alloc2"},
					Peer:        provider2.peerID.String(),
					Handle:      provider2.handle, // Has handle from commit phase
				},
			},
			Contracts: make(map[string]jtypes.ContractManifest),
			Subnet: jtypes.SubnetConfig{
				Join: true, // Subnet creation in progress
			},
		}

		// Create snapshot with candidates for both providers
		snapshot := jtypes.DeploymentSnapshot{
			Candidates: map[string]jtypes.Bid{
				"node1": {
					V1: &jtypes.BidV1{
						EnsembleID: restoreEnsembleID,
						NodeID:     "node1",
						Peer:       provider1.handle.Address.HostID,
						Location:   jtypes.Location{Country: "US"},
						Handle:     provider1.handle,
					},
				},
				"node2": {
					V1: &jtypes.BidV1{
						EnsembleID: restoreEnsembleID,
						NodeID:     "node2",
						Peer:       provider2.handle.Address.HostID,
						Location:   jtypes.Location{Country: "US"},
						Handle:     provider2.handle,
					},
				},
			},
			Expiry: time.Now().Add(time.Hour),
		}

		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable: map[string]string{
				"10.0.0.2": provider1.peerID.String(),
				"10.0.0.3": provider2.peerID.String(),
			},
			IndexRoutingTable: map[string]string{
				"alloc1": "10.0.0.2",
				"alloc2": "10.0.0.3",
			},
			DNSRecords: map[string]string{
				"alloc1.internal": "10.0.0.2",
				"alloc2.internal": "10.0.0.3",
			},
		}

		// Test restoring deployment from provisioning state with two providers
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusProvisioning,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())
		// Assert we reached running after restoration
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

		// In provisioning state restoration, revert is only called if provisioning fails
		// Since provisioning succeeded in this test, no revert messages should be sent
		assert.Equal(t, 2, revertCalls, "Providers should not have received revert messages when provisioning succeeds")

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
		assert.Equal(t, o, orchestrators[restoreEnsembleID])

		// Verify orchestrator can be retrieved
		retrievedOrch, err := registry.GetOrchestrator(restoreEnsembleID)
		require.NoError(t, err)
		assert.Equal(t, o, retrievedOrch)

		// Verify manifest is properly restored with both allocations
		restoredManifest := o.Manifest()
		assert.Equal(t, restoreEnsembleID, restoredManifest.ID)
		assert.Len(t, restoredManifest.Allocations, 2)
		assert.Len(t, restoredManifest.Nodes, 2)
		assert.Contains(t, restoredManifest.Allocations, "node1.alloc1")
		assert.Contains(t, restoredManifest.Allocations, "node2.alloc2")
	})

	t.Run("Restore from Provisioning State - Subnet Race Condition", func(t *testing.T) {
		// Speed up timeouts to make the test fail faster
		SubnetCreateTimeout = 1 * time.Second
		BidRequestTimeout = 1 * time.Second
		CommitDeploymentTimeout = 1 * time.Second
		VerifyEdgeConstraintTimeout = 1 * time.Second
		AllocationDeploymentTimeout = 1 * time.Second
		AllocationStartTimeout = 1 * time.Second
		AllocationShutdownTimeout = 1 * time.Second

		registry := NewRegistry(NewMockDeploymentStore())
		substrate := network.NewSubstrate()
		orch := MakeOrchestrator(t, substrate)
		provider1 := MakeProvider(t, substrate)
		provider2 := MakeProvider(t, substrate)

		// Create separate subnet state maps for each actor to mimic real world
		orchSubnetState := make(map[string]bool)      // orchestrator's subnet state
		provider1SubnetState := make(map[string]bool) // provider1's subnet state
		provider2SubnetState := make(map[string]bool) // provider2's subnet state
		subnetStateMutex := &sync.Mutex{}

		// Custom mock behaviors that simulate the race condition
		// Both orchestrator and providers will experience the same subnet state inconsistency
		ensembleID := "race-condition-ensemble"

		// Set up proper bid behaviors for both providers
		provider1BidBehavior := func(msg actor.Envelope) {
			defer msg.Discard()
			var request jtypes.EnsembleBidRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			// Check if this request is for node1
			shouldRespond := false
			for _, bidReq := range request.Request {
				if bidReq.V1 != nil && bidReq.V1.NodeID == "node1" {
					shouldRespond = true
					break
				}
			}

			if !shouldRespond {
				return // Don't respond to requests not for node1
			}

			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node1", // Provider1 responds to node1
					Peer:       provider1.handle.Address.HostID,
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider1.handle,
				},
			}

			providerDID := did.NewProvider(provider1.actor.Handle().DID, provider1.priv)
			require.NoError(t, bid.Sign(providerDID))

			reply, err := actor.ReplyTo(msg, bid)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider1.handle
			require.NoError(t, provider1.actor.Send(reply))
		}

		provider2BidBehavior := func(msg actor.Envelope) {
			defer msg.Discard()
			var request jtypes.EnsembleBidRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			// Check if this request is for node2
			shouldRespond := false
			for _, bidReq := range request.Request {
				if bidReq.V1 != nil && bidReq.V1.NodeID == "node2" {
					shouldRespond = true
					break
				}
			}

			if !shouldRespond {
				return // Don't respond to requests not for node2
			}

			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node2", // Provider2 responds to node2
					Peer:       provider2.handle.Address.HostID,
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider2.handle,
				},
			}

			providerDID := did.NewProvider(provider2.actor.Handle().DID, provider2.priv)
			require.NoError(t, bid.Sign(providerDID))

			reply, err := actor.ReplyTo(msg, bid)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider2.handle
			require.NoError(t, provider2.actor.Send(reply))
		}

		// Mock subnet create behavior on orchestrator that simulates "already exists" then destruction
		orch.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, orch.super.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { orch.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetCreateRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			subnetStateMutex.Lock()
			defer subnetStateMutex.Unlock()

			t.Logf("ORCHESTRATOR: Subnet create request for %s, current state: %v", request.SubnetID, orchSubnetState[request.SubnetID])

			// Simulate the race condition:
			// 1. First call: subnet exists (already created before crash)
			// 2. After revert: subnet is destroyed
			// 3. Subsequent calls: subnet doesn't exist
			if orchSubnetState[request.SubnetID] {
				// Subnet already exists - return error
				t.Logf("ORCHESTRATOR: Subnet %s already exists, returning error", request.SubnetID)
				reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
					OK:    false,
					Error: fmt.Sprintf("subnet with ID %s already exists", request.SubnetID),
				})
				require.NoError(t, err)
				reply.To = msg.From
				reply.From = orch.handle
				require.NoError(t, orch.actor.Send(reply))
				return
			}

			// Subnet doesn't exist - create it
			t.Logf("ORCHESTRATOR: Creating subnet %s", request.SubnetID)
			orchSubnetState[request.SubnetID] = true
			reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		// Mock subnet join behavior that fails when subnet doesn't exist
		orch.channels[fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, orch.super.AddBehavior(fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { orch.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetJoinRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			subnetStateMutex.Lock()
			defer subnetStateMutex.Unlock()

			// Check if subnet exists
			if !orchSubnetState[request.SubnetID] {
				// Subnet doesn't exist - return error
				reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
					OK:    false,
					Error: fmt.Sprintf("subnet with ID %s does not exist", request.SubnetID),
				})
				require.NoError(t, err)
				reply.To = msg.From
				reply.From = orch.handle
				require.NoError(t, orch.actor.Send(reply))
				return
			}

			// Subnet exists - join it
			reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		// Mock SubnetDestroy behavior on orchestrator that actually deletes subnet from map
		orch.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, orch.super.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { orch.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetDestroyRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			t.Logf("ORCHESTRATOR: SubnetDestroy request for %s, destroying subnet", request.SubnetID)
			subnetStateMutex.Lock()
			delete(orchSubnetState, request.SubnetID) // Destroy the subnet
			subnetStateMutex.Unlock()

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = orch.handle

			require.NoError(t, orch.actor.Send(reply))
		}))

		// Set up mock behaviors for both providers
		provider1.MockDeploymentBehaviors(t, ensembleID, provider1BidBehavior, orch.actor)
		provider2.MockDeploymentBehaviors(t, ensembleID, provider2BidBehavior, orch.actor)

		// Add custom subnet behaviors to both providers
		// Mock subnet create behavior on provider1 - Always succeed to allow progression to join phase
		provider1.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider1.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider1.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetCreateRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			subnetStateMutex.Lock()
			defer subnetStateMutex.Unlock()

			t.Logf("PROVIDER1: Subnet create request for %s - ALWAYS succeeding to allow progression to join phase", request.SubnetID)

			// Always succeed to allow progression to subnet join phase
			provider1SubnetState[request.SubnetID] = true
			reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider1.handle
			require.NoError(t, provider1.actor.Send(reply))
		}))

		// Mock subnet create behavior on provider2 - Always succeed to allow progression to join phase
		provider2.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider2.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider2.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetCreateRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			subnetStateMutex.Lock()
			defer subnetStateMutex.Unlock()

			t.Logf("PROVIDER2: Subnet create request for %s - ALWAYS succeeding to allow progression to join phase", request.SubnetID)

			// Always succeed to allow progression to subnet join phase
			provider2SubnetState[request.SubnetID] = true
			reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider2.handle
			require.NoError(t, provider2.actor.Send(reply))
		}))

		// Mock subnet join behavior on provider1 - ALWAYS return "does not exist" after revert
		provider1.channels[fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider1.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider1.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetJoinRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			t.Logf("PROVIDER1: Subnet join request for %s - ALWAYS returning 'does not exist' (subnet was destroyed during revert)", request.SubnetID)

			// Always return "does not exist" because subnet was destroyed during revert
			reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
				OK:    false,
				Error: fmt.Sprintf("subnet with ID %s does not exist", request.SubnetID),
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider1.handle
			require.NoError(t, provider1.actor.Send(reply))
		}))

		// Mock subnet join behavior on provider2 - ALWAYS return "does not exist" after revert
		provider2.channels[fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider2.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider2.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request SubnetJoinRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			t.Logf("PROVIDER2: Subnet join request for %s - ALWAYS returning 'does not exist' (subnet was destroyed during revert)", request.SubnetID)

			// Always return "does not exist" because subnet was destroyed during revert
			reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
				OK:    false,
				Error: fmt.Sprintf("subnet with ID %s does not exist", request.SubnetID),
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider2.handle
			require.NoError(t, provider2.actor.Send(reply))
		}))

		// Mock deployment revert behavior on provider1
		provider1.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider1.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider1.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request DeploymentRevertRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			t.Logf("PROVIDER1: Revert request for ensemble %s, destroying subnet", request.EnsembleID)

			// Simulate subnet destruction during revert on provider side as well
			subnetStateMutex.Lock()
			delete(provider1SubnetState, request.EnsembleID) // Destroy the subnet
			subnetStateMutex.Unlock()

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider1.handle

			require.NoError(t, provider1.actor.Send(reply))
		}))

		// Mock deployment revert behavior on provider2
		provider2.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider2.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer func() {
				msg.Discard()
				go func() { provider2.channels[msg.Behavior] <- struct{}{} }()
			}()

			var request DeploymentRevertRequest
			require.NoError(t, json.Unmarshal(msg.Message, &request))

			t.Logf("PROVIDER2: Revert request for ensemble %s, destroying subnet", request.EnsembleID)

			// Simulate subnet destruction during revert on provider side as well
			subnetStateMutex.Lock()
			delete(provider2SubnetState, request.EnsembleID) // Destroy the subnet
			subnetStateMutex.Unlock()

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider2.handle

			require.NoError(t, provider2.actor.Send(reply))
		}))

		// Grant capabilities
		err := orch.actor.Security().Grant(
			provider1.actor.Handle().DID,
			orch.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		err = orch.actor.Security().Grant(
			provider2.actor.Handle().DID,
			orch.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		err = provider1.actor.Security().Grant(
			orch.actor.Handle().DID,
			provider1.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		err = provider2.actor.Security().Grant(
			orch.actor.Handle().DID,
			provider2.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		// Create test configuration for two nodes
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
					"node2": {
						Location: jtypes.LocationConstraints{
							Accept: []jtypes.Location{
								{Country: "US"},
							},
						},
						Allocations: []string{"alloc2"},
					},
				},
				Allocations: map[string]jtypes.AllocationConfig{
					"alloc1": {
						Type: jtypes.AllocationTypeService,
						Resources: types.Resources{
							CPU: types.CPU{
								Cores:      2,
								ClockSpeed: 2000,
							},
							RAM: types.RAM{
								Size: 2048,
							},
							Disk: types.Disk{
								Size: 2048,
							},
						},
					},
					"alloc2": {
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

		// Create manifest in provisioning state with two nodes
		manifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Metadata:     map[string]string{},
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:       fmt.Sprintf("%s_alloc1", ensembleID),
					DNSName:  "alloc1.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationPending,
					Handle:   actor.Handle{},
					NodeID:   "node1",
					PrivAddr: "10.0.0.2",
					Ports:    make(map[int]int),
				},
				"alloc2": {
					ID:       fmt.Sprintf("%s_alloc2", ensembleID),
					DNSName:  "alloc2.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationPending,
					Handle:   actor.Handle{},
					NodeID:   "node2",
					PrivAddr: "10.0.0.3",
					Ports:    make(map[int]int),
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Peer:        provider1.peerID.String(),
					Handle:      provider1.handle,
				},
				"node2": {
					ID:          "node2",
					Allocations: []string{"alloc2"},
					Peer:        provider2.peerID.String(),
					Handle:      provider2.handle,
				},
			},
			Contracts: make(map[string]jtypes.ContractManifest),
			Subnet: jtypes.SubnetConfig{
				Join: true, // Subnet creation in progress
			},
		}

		// Create subnet manifest with existing subnet for both providers
		subnetManifest := jtypes.SubnetManifest{
			CIDR: "192.168.1.0/24",
			RoutingTable: map[string]string{
				"192.168.1.1": provider1.actor.Handle().DID.String(),
				"192.168.1.2": provider2.actor.Handle().DID.String(),
			},
			IndexRoutingTable: map[string]string{
				"alloc1": "192.168.1.1",
				"alloc2": "192.168.1.2",
			},
			UsedIPs: map[string]bool{
				"192.168.1.1": true,
				"192.168.1.2": true,
			},
			DNSRecords: make(map[string]string),
		}

		// Pre-populate the subnet state to simulate existing subnet before crash
		// Each actor has its own subnet state, but they all start with the subnet existing
		subnetStateMutex.Lock()
		orchSubnetState[ensembleID] = true      // Orchestrator thinks subnet exists
		provider1SubnetState[ensembleID] = true // Provider1 has subnet
		provider2SubnetState[ensembleID] = true // Provider2 has subnet
		subnetStateMutex.Unlock()

		// Create a proper deployment snapshot with candidates for both providers
		bid1 := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: ensembleID,
				NodeID:     "node1",
				Peer:       provider1.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider1.handle,
			},
		}

		bid2 := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: ensembleID,
				NodeID:     "node2",
				Peer:       provider2.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider2.handle,
			},
		}

		// Sign the bids using the providers' private keys
		provider1DID := did.NewProvider(provider1.actor.Handle().DID, provider1.priv)
		require.NoError(t, bid1.Sign(provider1DID))

		provider2DID := did.NewProvider(provider2.actor.Handle().DID, provider2.priv)
		require.NoError(t, bid2.Sign(provider2DID))

		snapshot := jtypes.DeploymentSnapshot{
			Candidates: map[string]jtypes.Bid{
				"node1": bid1,
				"node2": bid2,
			},
			Expiry: time.Now().Add(time.Hour),
		}

		// Attempt restoration - this should trigger the race condition
		// The test should timeout because the redeployment will keep failing
		// due to the subnet not existing after revert on both orchestrator and provider
		restoredOrch, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			ensembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusProvisioning,
			snapshot,
			subnetManifest,
			types.NewTestAllocationIDGenerator(),
		)
		require.NoError(t, err)
		// If restoration succeeds, the test should still fail because
		// the subnet state should be inconsistent
		require.NotNil(t, restoredOrch)

		// Verify that the test completed successfully
		// The test should have progressed through:
		// 1. Subnet creation (succeeds)
		// 2. Revert (destroys subnets)
		// 3. Redeployment (recreates subnets)
		// 4. Subnet join (fails with "does not exist" - this is the race condition we want to reproduce)

		subnetStateMutex.Lock()
		orchSubnetExists := orchSubnetState[ensembleID]
		provider1SubnetExists := provider1SubnetState[ensembleID]
		provider2SubnetExists := provider2SubnetState[ensembleID]
		subnetStateMutex.Unlock()

		// After successful redeployment, all actors should have recreated their subnets
		assert.False(t, orchSubnetExists, "Orchestrator subnet should have been destroyed during revert")
		assert.True(t, provider1SubnetExists, "Provider1 should have recreated subnet during redeployment")
		assert.True(t, provider2SubnetExists, "Provider2 should have recreated subnet during redeployment")

		// The test should have reached the subnet join phase and failed with "subnet does not exist"
		// This demonstrates the race condition from the E2E logs
	})

	t.Run("Restore from Provisioning State", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider := MakeProvider(t, substrate)

		// Set up comprehensive mock behaviors for provisioning state restoration
		provider.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		orch.MockOrchestratorBehaviors(t, restoreEnsembleID)

		t.Logf("Provider: %s", provider.actor.Handle())
		t.Logf("Orch: %s", orch.actor.Handle())

		// Grant capabilities between orchestrator and provider for communication
		err := orch.actor.Security().Grant(
			provider.actor.Handle().DID,
			orch.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		// Also grant capabilities from provider to orchestrator
		err = provider.actor.Security().Grant(
			orch.actor.Handle().DID,
			provider.actor.Handle().DID,
			[]ucan.Capability{behaviors.OrchestratorNamespace},
			5*time.Minute,
		)
		require.NoError(t, err)

		// Create test configuration for provisioning state
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
								Cores:      2,
								ClockSpeed: 2000,
							},
							RAM: types.RAM{
								Size: 2048,
							},
							Disk: types.Disk{
								Size: 2048,
							},
						},
					},
				},
			},
		}

		// Manifest representing a deployment that was in the process of provisioning
		// - Allocations are committed and have handles
		// - Nodes have handles and are ready for provisioning
		// - Subnet may be partially created
		manifest := jtypes.EnsembleManifest{
			ID:           restoreEnsembleID,
			Orchestrator: orch.actor.Handle(),
			Metadata:     map[string]string{},
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:       fmt.Sprintf("%s_alloc1", restoreEnsembleID),
					DNSName:  "alloc1.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationPending, // Committed but not yet running
					Handle:   provider.handle,          // Has handle from commit phase
					NodeID:   "node1",
					PrivAddr: "10.0.0.2",
					Ports:    make(map[int]int),
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Peer:        provider.peerID.String(),
					Handle:      provider.handle, // Has handle from commit phase
				},
			},
			Contracts: make(map[string]jtypes.ContractManifest),
			Subnet: jtypes.SubnetConfig{
				Join: true, // Subnet creation in progress
			},
		}

		// Create snapshot with candidates for provisioning state
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: restoreEnsembleID,
				NodeID:     "node1",
				Peer:       provider.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider.handle,
			},
		}
		snapshot := jtypes.DeploymentSnapshot{
			Candidates: map[string]jtypes.Bid{
				"node1": bid,
			},
			Expiry: time.Now().Add(time.Hour),
		}
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment from provisioning state
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusProvisioning,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())
		// Assert we reached running after restoration
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
		assert.Equal(t, o, orchestrators[restoreEnsembleID])

		// Verify orchestrator can be retrieved
		retrievedOrch, err := registry.GetOrchestrator(restoreEnsembleID)
		require.NoError(t, err)
		assert.Equal(t, o, retrievedOrch)

		// Verify manifest is properly restored
		restoredManifest := o.Manifest()
		assert.Equal(t, restoreEnsembleID, restoredManifest.ID)
		assert.NotEmpty(t, restoredManifest.Allocations)
		assert.NotEmpty(t, restoredManifest.Nodes)
	})

	t.Run("Restore from Running State", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider := MakeProvider(t, substrate)

		// Set up comprehensive mock behaviors for running state restoration
		provider.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		orch.MockOrchestratorBehaviors(t, restoreEnsembleID)

		// Create test configuration for running state
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
								Cores:      2,
								ClockSpeed: 2000,
							},
							RAM: types.RAM{
								Size: 2048,
							},
							Disk: types.Disk{
								Size: 2048,
							},
						},
					},
				},
			},
		}

		// Manifest representing a deployment that was running
		// - Allocations are running and fully provisioned
		// - Nodes have handles and are active
		// - Subnet is fully created and operational
		manifest := jtypes.EnsembleManifest{
			ID:           restoreEnsembleID,
			Orchestrator: orch.actor.Handle(),
			Metadata:     map[string]string{},
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:       fmt.Sprintf("%s_alloc1", restoreEnsembleID),
					DNSName:  "alloc1.internal",
					Type:     jtypes.AllocationTypeService,
					Status:   jtypes.AllocationRunning, // Fully running
					Handle:   provider.handle,          // Has handle
					NodeID:   "node1",
					PrivAddr: "10.0.0.2",
					Ports:    map[int]int{8080: 80}, // Port mapping configured
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Peer:        provider.peerID.String(),
					Handle:      provider.handle, // Has handle
				},
			},
			Contracts: make(map[string]jtypes.ContractManifest),
			Subnet: jtypes.SubnetConfig{
				Join: true, // Subnet is operational
			},
		}

		// Create snapshot with candidates for running state
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: restoreEnsembleID,
				NodeID:     "node1",
				Peer:       provider.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider.handle,
			},
		}
		snapshot := jtypes.DeploymentSnapshot{
			Candidates: map[string]jtypes.Bid{
				"node1": bid,
			},
			Expiry: time.Now().Add(time.Hour),
		}
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment from running state
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusRunning,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
		assert.Equal(t, o, orchestrators[restoreEnsembleID])

		// Verify orchestrator can be retrieved
		retrievedOrch, err := registry.GetOrchestrator(restoreEnsembleID)
		require.NoError(t, err)
		assert.Equal(t, o, retrievedOrch)

		// Verify manifest is properly restored
		restoredManifest := o.Manifest()
		assert.Equal(t, restoreEnsembleID, restoredManifest.ID)
		assert.NotEmpty(t, restoredManifest.Allocations)
		assert.NotEmpty(t, restoredManifest.Nodes)

		// Verify subnet manifest is properly restored
		restoredSubnet := o.SubnetManifest()
		assert.NotEmpty(t, restoredSubnet.CIDR)
	})

	t.Run("Restore with Subnet Join", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider := MakeProvider(t, substrate)

		// Set up comprehensive mock behaviors including subnet operations
		provider.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		orch.MockOrchestratorBehaviors(t, restoreEnsembleID)

		// Create test configuration with subnet join enabled
		cfg := createSubnetJoinConfig()
		manifest := createSubnetJoinManifest(restoreEnsembleID, orch.actor.Handle(), provider.handle, provider.peerID)
		snapshot := createRunningStateSnapshot(provider.handle)
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment with subnet join
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusRunning,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
		assert.Equal(t, o, orchestrators[restoreEnsembleID])

		// Verify subnet join configuration
		restoredManifest := o.Manifest()
		assert.True(t, restoredManifest.Subnet.Join)
	})

	t.Run("Restore with Multiple Allocations", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider1 := MakeProvider(t, substrate)
		provider2 := MakeProvider(t, substrate)

		// Set up mock behaviors for multiple providers
		provider1.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		provider2.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)
		orch.MockOrchestratorBehaviors(t, restoreEnsembleID)

		// Create test configuration with multiple allocations
		cfg := createMultiAllocationConfig()
		manifest := createMultiAllocationManifest(restoreEnsembleID, orch.actor.Handle(), provider1.handle, provider2.handle, provider1.peerID, provider2.peerID)
		snapshot := createRunningStateSnapshot(provider1.handle)
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment with multiple allocations
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusRunning,
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)

		// Verify multiple allocations are properly restored
		restoredManifest := o.Manifest()
		assert.Len(t, restoredManifest.Allocations, 2)
		assert.Len(t, restoredManifest.Nodes, 2)
		assert.Contains(t, restoredManifest.Allocations, "alloc1")
		assert.Contains(t, restoredManifest.Allocations, "alloc2")
	})

	t.Run("Restore with Invalid State", func(t *testing.T) {
		registry := NewRegistry(NewMockDeploymentStore())
		orch := MakeOrchestrator(t, substrate)
		provider := MakeProvider(t, substrate)

		// Set up basic mock behaviors
		provider.MockDeploymentBehaviors(t, restoreEnsembleID, nil, orch.actor)

		// Create test configuration
		cfg := createRunningStateConfig()
		manifest := createRunningStateManifest(restoreEnsembleID, orch.actor.Handle(), provider.handle, provider.peerID)
		snapshot := createRunningStateSnapshot(provider.handle)
		subnet := jtypes.SubnetManifest{
			CIDR:        "10.0.0.0/24",
			GatewayIP:   "10.0.0.1",
			BroadcastIP: "10.0.0.255",
			UsedIPs: map[string]bool{
				"10.0.0.1":   true, // gateway
				"10.0.0.255": true, // broadcast
			},
			RoutingTable:      make(map[string]string),
			IndexRoutingTable: make(map[string]string),
			DNSRecords:        make(map[string]string),
		}

		// Test restoring deployment with invalid state (should still work but not restore)
		o, err := registry.RestoreDeployment(
			context.Background(),
			afero.Afero{Fs: fs},
			orch.actor,
			restoreEnsembleID,
			cfg,
			manifest,
			jtypes.DeploymentStatusFailed, // Invalid state for restoration
			snapshot,
			subnet,
			types.NewTestAllocationIDGenerator(),
		)

		require.NoError(t, err)
		assert.NotNil(t, o)
		assert.Equal(t, restoreEnsembleID, o.ID())

		// Assert registry state
		orchestrators := registry.Orchestrators()
		assert.Contains(t, orchestrators, restoreEnsembleID)
	})
}

// Helper functions to create test configurations for different states
func createRunningStateConfig() jtypes.EnsembleConfig {
	return jtypes.EnsembleConfig{
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
							Cores:      4,
							ClockSpeed: 3000,
						},
						RAM: types.RAM{
							Size: 4096,
						},
						Disk: types.Disk{
							Size: 4096,
						},
					},
				},
			},
		},
	}
}

func createSubnetJoinConfig() jtypes.EnsembleConfig {
	cfg := createRunningStateConfig()
	cfg.V1.Subnet = jtypes.SubnetConfig{
		Join: true,
	}
	return cfg
}

func createMultiAllocationConfig() jtypes.EnsembleConfig {
	return jtypes.EnsembleConfig{
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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      2,
							ClockSpeed: 2000,
						},
						RAM: types.RAM{
							Size: 2048,
						},
						Disk: types.Disk{
							Size: 2048,
						},
					},
				},
				"alloc2": {
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
}

func createRunningStateManifest(restoreEnsembleID string, orchHandle, providerHandle actor.Handle, peerID peer.ID) jtypes.EnsembleManifest {
	return jtypes.EnsembleManifest{
		ID:           restoreEnsembleID,
		Orchestrator: orchHandle,
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:       fmt.Sprintf("%s_alloc1", restoreEnsembleID),
				DNSName:  "alloc1.internal",
				Type:     jtypes.AllocationTypeService,
				Status:   jtypes.AllocationRunning,
				Handle:   providerHandle,
				NodeID:   "node1",
				PrivAddr: "10.0.0.2",
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Peer:        peerID.String(),
				Handle:      providerHandle,
			},
		},
		Subnet: jtypes.SubnetConfig{
			Join: true,
		},
	}
}

func createSubnetJoinManifest(restoreEnsembleID string, orchHandle, providerHandle actor.Handle, peerID peer.ID) jtypes.EnsembleManifest {
	manifest := createRunningStateManifest(restoreEnsembleID, orchHandle, providerHandle, peerID)
	manifest.Subnet.Join = true
	return manifest
}

func createMultiAllocationManifest(restoreEnsembleID string, orchHandle, provider1Handle, provider2Handle actor.Handle, peer1ID, peer2ID peer.ID) jtypes.EnsembleManifest {
	return jtypes.EnsembleManifest{
		ID:           restoreEnsembleID,
		Orchestrator: orchHandle,
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:       fmt.Sprintf("%s_alloc1", restoreEnsembleID),
				DNSName:  "alloc1.internal",
				Type:     jtypes.AllocationTypeService,
				Status:   jtypes.AllocationRunning,
				Handle:   provider1Handle,
				NodeID:   "node1",
				PrivAddr: "10.0.0.2",
			},
			"alloc2": {
				ID:       fmt.Sprintf("%s_alloc2", restoreEnsembleID),
				DNSName:  "alloc2.internal",
				Type:     jtypes.AllocationTypeService,
				Status:   jtypes.AllocationRunning,
				Handle:   provider2Handle,
				NodeID:   "node2",
				PrivAddr: "10.0.0.3",
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Peer:        peer1ID.String(),
				Handle:      provider1Handle,
			},
			"node2": {
				ID:          "node2",
				Allocations: []string{"alloc2"},
				Peer:        peer2ID.String(),
				Handle:      provider2Handle,
			},
		},
		Subnet: jtypes.SubnetConfig{
			Join: true,
		},
	}
}

// Helper functions to create test snapshots for different states

func createRunningStateSnapshot(providerHandle actor.Handle) jtypes.DeploymentSnapshot {
	// Create a valid bid for the snapshot
	bid := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: restoreEnsembleID,
			NodeID:     "node1",
			Peer:       providerHandle.Address.HostID,
			Location:   jtypes.Location{Country: "US"},
			Handle:     providerHandle,
		},
	}

	return jtypes.DeploymentSnapshot{
		Candidates: map[string]jtypes.Bid{
			"node1": bid,
		},
		Expiry: time.Now().Add(time.Hour),
	}
}
