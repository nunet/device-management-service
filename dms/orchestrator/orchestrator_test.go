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

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	workDir    = "/tmp"
	ensembleID = "test-ensemble"
)

func TestOrchestratorDeploy(t *testing.T) {
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

	// Create orchestrator with orchestrator mock
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
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
	alloc, ok := manifest.Allocations["node1.alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
}

func TestOrchestratorDeployWithRedundancy(t *testing.T) {
	BidRequestTimeout = 1 * time.Second
	CommitDeploymentTimeout = 1 * time.Second
	VerifyEdgeConstraintTimeout = 1 * time.Second
	AllocationDeploymentTimeout = 1 * time.Second
	AllocationStartTimeout = 1 * time.Second
	AllocationShutdownTimeout = 1 * time.Second

	substrate := network.NewSubstrate()

	orch := MakeOrchestrator(t, substrate)
	provider1 := MakeProvider(t, substrate)
	provider2 := MakeProvider(t, substrate)
	provider3 := MakeProvider(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations:     []string{"alloc1"},
					Redundancy:      2,
					FailureRecovery: jtypes.NodeFailureRecoveryStayDown,
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

	// Set up behaviors for all providers
	provider1.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)
	provider2.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)
	provider3.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	mx := sync.Mutex{}
	nodesPeerIDs := make(map[string]string)

	// Override provider2 to return a bid response for the first standby node
	require.NoError(t, provider1.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case provider1.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		var request jtypes.EnsembleBidRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal bid request: %s", err)
		}

		mx.Lock()
		nodesPeerIDs[request.Request[0].V1.NodeID] = provider1.handle.Address.HostID
		mx.Unlock()

		// send bid response
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: request.ID,
				NodeID:     request.Request[0].V1.NodeID,
				Peer:       provider1.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider1.handle,
			},
		}

		// sign the bid using the provider's private key
		// Create DID provider for signing
		providerDID := did.NewProvider(provider1.actor.Handle().DID, provider1.priv)

		// Sign the bid
		if err := bid.Sign(providerDID); err != nil {
			fmt.Printf("Failed to sign bid: %v\n", err)
			return
		}

		var opt []actor.MessageOption
		if msg.IsBroadcast() {
			opt = append(opt, actor.WithMessageSource(provider1.actor.Handle()))
		}

		reply, err := actor.ReplyTo(msg, bid, opt...)
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = provider1.handle

		if err := provider1.actor.Send(reply); err != nil {
			t.Fatalf("sending bid response: %s", err)
		}
	}, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	// Override provider2 to return a bid response for the first standby node
	require.NoError(t, provider2.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case provider2.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		var request jtypes.EnsembleBidRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal bid request: %s", err)
		}

		mx.Lock()
		nodesPeerIDs[request.Request[1].V1.NodeID] = provider2.handle.Address.HostID
		mx.Unlock()
		// send bid response
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: request.ID,
				NodeID:     request.Request[1].V1.NodeID,
				Peer:       provider2.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider2.handle,
			},
		}

		// sign the bid using the provider's private key
		// Create DID provider for signing
		providerDID := did.NewProvider(provider2.actor.Handle().DID, provider2.priv)

		// Sign the bid
		if err := bid.Sign(providerDID); err != nil {
			fmt.Printf("Failed to sign bid: %v\n", err)
			return
		}

		var opt []actor.MessageOption
		if msg.IsBroadcast() {
			opt = append(opt, actor.WithMessageSource(provider2.actor.Handle()))
		}

		reply, err := actor.ReplyTo(msg, bid, opt...)
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = provider2.handle

		if err := provider2.actor.Send(reply); err != nil {
			t.Fatalf("sending bid response: %s", err)
		}
	}, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	// Override provider3 to return a bid response for the second standby node
	require.NoError(t, provider3.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case provider3.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		var request jtypes.EnsembleBidRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal bid request: %s", err)
		}

		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: request.ID,
				NodeID:     request.Request[2].V1.NodeID,
				Peer:       provider3.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     provider3.handle,
			},
		}

		mx.Lock()
		nodesPeerIDs[request.Request[2].V1.NodeID] = provider3.handle.Address.HostID
		mx.Unlock()

		// sign the bid using the provider's private key
		// Create DID provider for signing
		providerDID := did.NewProvider(provider3.actor.Handle().DID, provider3.priv)

		// Sign the bid
		if err := bid.Sign(providerDID); err != nil {
			fmt.Printf("Failed to sign bid: %v\n", err)
			return
		}

		var opt []actor.MessageOption
		if msg.IsBroadcast() {
			opt = append(opt, actor.WithMessageSource(provider3.actor.Handle()))
		}

		reply, err := actor.ReplyTo(msg, bid, opt...)
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = provider3.handle

		if err := provider3.actor.Send(reply); err != nil {
			t.Fatalf("sending bid response: %s", err)
		}
	}, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	// Create orchestrator with orchestrator mock
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Start deployment in a goroutine
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		t.Helper()
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

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

	// Verify nodes were deployed with redundancy
	node, ok := manifest.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, nodesPeerIDs["node1"], node.Peer)
	assert.Equal(t, jtypes.RolePrimary, node.RedundancyRole)
	assert.Len(t, node.StandbyNodes, 2)

	// Verify standby nodes were created
	standby1, ok := manifest.Nodes["node1-standby-1"]
	assert.True(t, ok)
	assert.Equal(t, jtypes.RoleStandby, standby1.RedundancyRole)
	assert.Equal(t, "node1", standby1.PrimaryNode)
	assert.Equal(t, 1, standby1.StandbyIndex)
	assert.Equal(t, nodesPeerIDs["node1-standby-1"], standby1.Peer)

	standby2, ok := manifest.Nodes["node1-standby-2"]
	assert.True(t, ok)
	assert.Equal(t, jtypes.RoleStandby, standby2.RedundancyRole)
	assert.Equal(t, "node1", standby2.PrimaryNode)
	assert.Equal(t, 2, standby2.StandbyIndex)
	assert.Equal(t, nodesPeerIDs["node1-standby-2"], standby2.Peer)

	// Verify allocation was deployed
	alloc, ok := manifest.Allocations["node1.alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
	assert.False(t, alloc.IsStandby)
	assert.Equal(t, "alloc1", alloc.RedundancyGroup)

	// Verify standby allocations were created
	standbyAlloc1, ok := manifest.Allocations["node1-standby-1.alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1-standby-1", standbyAlloc1.NodeID)
	assert.True(t, standbyAlloc1.IsStandby)
	assert.Equal(t, "alloc1", standbyAlloc1.RedundancyGroup)

	standbyAlloc2, ok := manifest.Allocations["node1-standby-2.alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1-standby-2", standbyAlloc2.NodeID)
	assert.True(t, standbyAlloc2.IsStandby)
	assert.Equal(t, "alloc1", standbyAlloc2.RedundancyGroup)
}

func TestOrchestratorID(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	assert.Equal(t, ensembleID, o.ID())
}

func TestOrchestratorConfig(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	config := o.Config()
	assert.Equal(t, cfg.V1.Nodes["node1"].Allocations, config.V1.Nodes["node1"].Allocations)
	assert.Equal(t, cfg.V1.Allocations["alloc1"].Type, config.V1.Allocations["alloc1"].Type)
}

func TestOrchestratorStatus(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Initial status should be Preparing
	assert.Equal(t, jtypes.DeploymentStatusPreparing, o.status)

	statusCh := o.StatusChannel(context.Background())

	go func() {
		err := o.Shutdown()
		require.NoError(t, err)
	}()

	// Track expected status transitions
	expectedStatuses := []jtypes.DeploymentStatus{
		jtypes.DeploymentStatusPreparing,
		jtypes.DeploymentStatusShuttingDown,
		jtypes.DeploymentStatusCompleted,
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
}

func TestOrchestratorManifest(t *testing.T) {
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
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

	// Manifest assertions after deployment is successful
	manifest := o.Manifest()
	assert.Equal(t, ensembleID, manifest.ID)
	assert.Equal(t, orch.actor.Handle(), manifest.Orchestrator)
	assert.NotEmpty(t, manifest.Nodes)
	assert.NotEmpty(t, manifest.Allocations)

	// Verify node was deployed
	node, ok := manifest.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, provider.peerID.String(), node.Peer)

	// Verify allocation was deployed
	alloc, ok := manifest.Allocations["node1.alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
}

func TestOrchestratorActorPrivateKey(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	privKey := o.ActorPrivateKey()
	assert.NotNil(t, privKey)
}

func TestOrchestratorDeploymentSnapshot(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	snapshot := o.DeploymentSnapshot()
	assert.Empty(t, snapshot.Candidates)
}

func TestOrchestratorStatusChannel(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx, cancel := context.WithCancel(context.Background())
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	statusCh := o.StatusChannel(ctx)

	// Should receive initial status
	status := <-statusCh
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status)

	// Cancel context and verify channel is closed
	cancel()
	_, ok := <-statusCh
	assert.False(t, ok, "Status channel should be closed after context cancellation")
	// Test multiple subscribers
	ctx = context.Background()
	statusCh1 := o.StatusChannel(ctx)
	statusCh2 := o.StatusChannel(ctx)

	// Both channels should receive initial status
	status1 := <-statusCh1
	status2 := <-statusCh2
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status1)
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status2)
}

func TestOrchestratorGetAllocationLogs(t *testing.T) {
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

	behavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, "test-ensemble")
	require.NoError(t, provider.actor.AddBehavior(behavior, func(msg actor.Envelope) {
		go func() {
			select {
			case provider.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, AllocationLogsResponse{
			Stdout: []byte("ok"),
			Stderr: []byte{},
			Error:  "",
		})
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = provider.handle

		if err := provider.actor.Send(reply); err != nil {
			t.Fatalf("sending allocation logs response: %s", err)
		}
	}))

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
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
	go func() {
		for status := range statusCh {
			t.Logf("Deployment status changed to: %s", status)
		}
	}()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify final state
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	// Test GetAllocationLogs
	logs, err := o.GetAllocationLogs("node1.alloc1")
	require.NoError(t, err)
	assert.Equal(t, "ok", string(logs.Stdout))
}

func TestHandleTaskTermination(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

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
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Deploy the ensemble
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	tests := []struct {
		name         string
		notification behaviors.TaskTerminationNotification
		checkStatus  func(t *testing.T, o *BasicOrchestrator)
	}{
		{
			name: "successful task termination",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_node1.alloc1",
				Status:       string(jtypes.AllocationCompleted),
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["node1.alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationCompleted {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationCompleted, alloc.Status)
				}
			},
		},
		{
			name: "task termination with error",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_node1.alloc1",
				Status:       string(jtypes.AllocationFailed),
				Error: behaviors.TerminationError{
					ExitCode: 1,
					Err:      "execution exit code != 0, exit code: 1",
				},
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["node1.alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationFailed {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationFailed, alloc.Status)
				}
			},
		},
		{
			name: "task termination with logs",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_node1.alloc1",
				Status:       string(jtypes.AllocationCompleted),
				Stdout:       []byte("test stdout"),
				Stderr:       []byte("test stderr"),
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["node1.alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationCompleted {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationCompleted, alloc.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := actor.Message(
				provider.handle,
				orch.actor.Handle(),
				behaviors.NotifyTaskTerminationBehavior,
				tt.notification,
				actor.WithMessageSource(provider.handle),
				actor.WithMessageReplyTo("replyto/123"),
			)
			if err != nil {
				t.Fatalf("failed to create message: %v", err)
			}

			o.handleTaskTermination(msg)

			if tt.checkStatus != nil {
				tt.checkStatus(t, o)
			}
		})
	}
}

func TestWriteAllocationLogs(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	o, err := NewOrchestrator(ctx, fs, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Write allocation logs
	stdout := []byte("test stdout")
	stderr := []byte("test stderr")
	allocDir, err := o.WriteAllocationLogs("node1.alloc1", stdout, stderr)
	require.NoError(t, err)
	assert.NotEmpty(t, allocDir)

	stdoutContent, err := fs.ReadFile(fmt.Sprintf("/tmp/deployments/%s/node1.alloc1/stdout.log", ensembleID))
	require.NoError(t, err)
	stderrContent, err := fs.ReadFile(fmt.Sprintf("/tmp/deployments/%s/node1.alloc1/stderr.log", ensembleID))
	require.NoError(t, err)
	assert.Equal(t, stdout, stdoutContent)
	assert.Equal(t, stderr, stderrContent)
}

func TestAllocNameFromID(t *testing.T) {
	// Test allocation name extraction
	allocID := "test-ensemble_node1.alloc1"
	allocIDStruct, err := types.ParseAllocationID(allocID)
	require.NoError(t, err)
	allocName := allocIDStruct.ConfigName()
	assert.Equal(t, "alloc1", allocName)

	// Test invalid allocation ID
	invalidID := "invalid-id"
	_, err = types.ParseAllocationID(invalidID)
	assert.Error(t, err)
}

func TestVerifyEdgeConstraints(t *testing.T) {
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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "UK"},
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
			Edges: []jtypes.EdgeConstraint{
				{
					S:   "node1",
					T:   "node2",
					RTT: 100,
					BW:  1000,
				},
			},
		},
	}

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Create test bids
	bid1 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bid2 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node2",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "UK"},
			Handle:     provider.handle,
		},
	}

	// Test valid edge constraints
	candidate := map[string]jtypes.Bid{
		"node1": bid1,
		"node2": bid2,
	}

	t.Run("valid edge constraints", func(t *testing.T) {
		// Mock the provider's behavior for edge constraint verification
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			reply, err := actor.ReplyTo(msg, VerifyEdgeConstraintResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		// Test valid edge constraints
		result := bidC.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.True(t, result)
	})

	t.Run("invalid edge constraints (timeout)", func(t *testing.T) {
		// Test invalid edge constraints (timeout)
		VerifyEdgeConstraintTimeout = 1 * time.Second
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
			// Don't send a reply to simulate timeout
		}))

		result := bidC.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.False(t, result)
	})

	t.Run("invalid edge constraints (error response)", func(t *testing.T) {
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			reply, err := actor.ReplyTo(msg, VerifyEdgeConstraintResponse{
				OK:    false,
				Error: "constraint violation",
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		result := bidC.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.False(t, result)
	})
}

func TestRevertNodeDeployment(t *testing.T) {
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Set up test manifest
	o.manifest = jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_node1.alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Handle:      provider.handle,
			},
		},
	}

	// Test successful revert
	t.Run("successful revert", func(t *testing.T) {
		provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			// Send reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK: true,
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider.handle

			require.NoError(t, provider.actor.Send(reply))
		}))

		o.revertNodeDeployment(cfg, "node1", provider.handle)
		<-provider.channels[behaviors.DeploymentRevertBehavior]

		// Verify node was removed from manifest
		_, ok := o.manifest.Nodes["node1"]
		assert.False(t, ok)
		_, ok = o.manifest.Allocations["test-ensemble_node1.alloc1"]
		assert.False(t, ok)
	})

	// Test revert failure
	t.Run("revert failure", func(t *testing.T) {
		// Reset manifest
		o.manifest = jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_node1.alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Handle:      provider.handle,
				},
			},
		}

		fmt.Println("manifest", o.manifest)
		provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			// Send failure reply for invoke-style messaging
			reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
				OK:    false,
				Error: "simulated revert failure",
			})
			require.NoError(t, err)

			reply.To = msg.From
			reply.From = provider.handle

			require.NoError(t, provider.actor.Send(reply))
		}))

		o.revertNodeDeployment(cfg, "node1", provider.handle)
		<-provider.channels[behaviors.DeploymentRevertBehavior]

		// Verify node wasn't removed from manifest cause of failure
		_, ok := o.manifest.Nodes["node1"]
		assert.True(t, ok)
		_, ok = o.manifest.Allocations["alloc1"]
		assert.True(t, ok)
	})

	// Test non-existent node
	t.Run("non-existent node", func(t *testing.T) {
		o.manifest = jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_node1.alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
					Handle:      provider.handle,
				},
			},
		}
		o.revertNodeDeployment(cfg, "non-existent", provider.handle)

		_, ok := o.manifest.Nodes["non-existent"]
		assert.False(t, ok)
		_, ok = o.manifest.Nodes["node1"]
		assert.True(t, ok)
		_, ok = o.manifest.Allocations["alloc1"]
		assert.True(t, ok)
	})
}

func TestRevert(t *testing.T) {
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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "UK"},
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Set up test manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
			"alloc2": {
				ID:     "test-ensemble_alloc2",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Handle:      provider.handle,
			},
			"node2": {
				ID:          "node2",
				Allocations: []string{"alloc2"},
				Handle:      provider.handle,
			},
		},
	}
	o.manifest = manifest

	// Mock the provider's behavior for deployment revert
	provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 2)
	require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
		defer msg.Discard()
		defer func() {
			provider.channels[msg.Behavior] <- struct{}{}
		}()

		// Send reply for invoke-style messaging
		reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = provider.handle

		require.NoError(t, provider.actor.Send(reply))
	}))

	// Test successful revert
	o.revert(cfg, manifest)

	// Wait for both nodes to be reverted
	<-provider.channels[behaviors.DeploymentRevertBehavior]
	<-provider.channels[behaviors.DeploymentRevertBehavior]

	// Verify all nodes and allocations were removed
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)
}

func TestRemoveNodeFromManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1", "alloc2"},
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Set up test manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
			"alloc2": {
				ID:     "test-ensemble_alloc2",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1", "alloc2"},
			},
		},
	}
	o.manifest = manifest

	// Test removing existing node
	o.removeNodeFromManifest("node1")
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)

	// Test removing non-existent node
	o.removeNodeFromManifest("non-existent")
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)
}

func TestShutdown(t *testing.T) {
	t.Skip("Skipping testShutdown due to a bug with consecutive deployments causing a deadlock. To be investigated.")
	SubnetDestroyTimeout = time.Second * 1
	AllocationShutdownTimeout = time.Second * 1

	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	deploy := func() {
		t.Helper()
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

		// Wait for status changes
		go func() {
			for status := range statusCh {
				t.Logf("Deployment status changed to: %s", status)
			}
		}()

		select {
		case err := <-deployDone:
			require.NoError(t, err)
		case <-time.After(60 * time.Second):
			t.Fatal("Timeout waiting for deployment to complete")
		}

		// Verify final state
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())
	}

	manifest := o.Manifest()

	t.Run("happy path", func(t *testing.T) {
		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, AllocationStopResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		deploy()

		// Test shutdown from running state
		o.setStatus(jtypes.DeploymentStatusRunning)
		require.NoError(t, o.Shutdown())
		<-provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)]
		<-provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)]
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())

		t.Run("already shutdown", func(t *testing.T) {
			o.setStatus(jtypes.DeploymentStatusCompleted)
			o.manifest = manifest
			require.NoError(t, o.Shutdown())
			assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())

			o.setStatus(jtypes.DeploymentStatusShuttingDown)
			o.manifest = manifest
			require.NoError(t, o.Shutdown())
		})
	})

	t.Run("failed state", func(t *testing.T) {
		// Test shutdown from failed state
		o.setStatus(jtypes.DeploymentStatusFailed)
		o.manifest = manifest

		o.cancel = func() {
		}
		require.NoError(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// Test shutdown with subnet destroy failure
	t.Run("subnet destroy failure", func(t *testing.T) {
		CommitDeploymentTimeout = time.Second * 1
		AllocationShutdownTimeout = time.Second * 1
		SubnetCreateTimeout = time.Second * 1

		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			go func() {
				provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
				OK:    false,
				Error: "subnet destroy failed",
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		deploy()
		require.Error(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// Test shutdown with subnet destroy timeout
	t.Run("subnet destroy timeout", func(t *testing.T) {
		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			// Don't send a reply to simulate timeout
		}))

		deploy()

		require.Error(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})
}

func TestContainsExecutor(t *testing.T) {
	// Test with executor in list
	executors := []jtypes.AllocationExecutor{jtypes.ExecutorDocker, jtypes.ExecutorFirecracker, jtypes.ExecutorNull}
	assert.True(t, containsExecutor(executors, jtypes.ExecutorDocker))
	assert.True(t, containsExecutor(executors, jtypes.ExecutorFirecracker))
	assert.True(t, containsExecutor(executors, jtypes.ExecutorNull))

	// Test with executor not in list
	assert.False(t, containsExecutor([]jtypes.AllocationExecutor{
		jtypes.ExecutorFirecracker,
		jtypes.ExecutorNull,
	}, jtypes.ExecutorDocker))

	// Test with empty list
	assert.False(t, containsExecutor([]jtypes.AllocationExecutor{}, jtypes.ExecutorDocker))
}

func TestRequestBidPeer(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
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

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Create test bid request
	bidRequest := jtypes.EnsembleBidRequest{
		ID:    ensembleID,
		Nonce: bidC.getNonce(),
		Request: []jtypes.BidRequest{
			{
				V1: &jtypes.BidRequestV1{
					NodeID: "node1",
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
				},
			},
		},
	}

	// Test successful bid request
	t.Run("successful bid request", func(t *testing.T) {
		provider.channels[behaviors.BidRequestBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			var request jtypes.EnsembleBidRequest
			if err := json.Unmarshal(msg.Message, &request); err != nil {
				t.Fatalf("unmarshal bid request: %s", err)
			}

			// Create and sign bid response
			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node1",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}

			reply, err := actor.Message(provider.handle, msg.From, msg.Options.ReplyTo, bid)
			if err != nil {
				t.Fatalf("creating reply: %s", err)
			}

			if err := provider.actor.Send(reply); err != nil {
				t.Fatalf("sending bid response: %s", err)
			}
		}))

		err := bidC.requestBidPeer(bidRequest, cfg.V1.Nodes["node1"], uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.NoError(t, err)
		<-provider.channels[behaviors.BidRequestBehavior]
	})

	// Test invalid peer ID
	t.Run("invalid peer ID", func(t *testing.T) {
		invalidNodeConfig := jtypes.NodeConfig{
			Peer: "invalid-peer-id",
		}

		err := bidC.requestBidPeer(bidRequest, invalidNodeConfig, uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting handle of selected peer")
	})

	// Test message sending error
	t.Run("message sending error", func(t *testing.T) {
		provider.channels[behaviors.BidRequestBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
			// Don't send a reply to simulate sending error
		}))

		err := bidC.requestBidPeer(bidRequest, cfg.V1.Nodes["node1"], uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.NoError(t, err) // The function itself doesn't return an error for sending failures
		<-provider.channels[behaviors.BidRequestBehavior]
	})
}

func TestMakeCandidateDeploymentBig(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1", "alloc2"},
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

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Create test bids
	bid := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bids := map[string][]jtypes.Bid{
		"node1": {bid},
	}

	// Test successful deployment creation
	t.Run("successful deployment creation", func(t *testing.T) {
		nextCandidate, ok := bidC.makeCandidateDeploymentBig(cfg, bids)
		require.True(t, ok)
		require.NotNil(t, nextCandidate)

		candidate, ok := nextCandidate()
		require.True(t, ok)
		assert.NotNil(t, candidate)
		assert.Equal(t, bid, candidate["node1"])
	})

	// Test deployment creation with invalid bid
	t.Run("invalid bid", func(t *testing.T) {
		invalidBids := map[string][]jtypes.Bid{
			"node1": {{
				V1: &jtypes.BidV1{
					EnsembleID: "invalid-ensemble",
					NodeID:     "node1",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}},
		}

		nextCandidate, ok := bidC.makeCandidateDeploymentBig(cfg, invalidBids)
		require.True(t, ok)
		require.NotNil(t, nextCandidate)

		candidate, ok := nextCandidate()
		require.True(t, ok)
		assert.NotNil(t, candidate)
		assert.Equal(t, invalidBids["node1"][0], candidate["node1"])
	})
}

func TestMakeResidualBidRequest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
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

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Create test bids
	bid := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bids := map[string][]jtypes.Bid{
		"node1": {bid},
	}

	// Test successful residual bid request creation
	t.Run("successful residual bid request", func(t *testing.T) {
		rmBid := func(_ jtypes.Bid) {
			// This is a no-op for testing
		}

		request, err := bidC.makeResidualBidRequest(cfg, bids, rmBid)
		require.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, ensembleID, request.ID)
		assert.Equal(t, 1, len(request.Request))
		assert.Equal(t, "node2", request.Request[0].V1.NodeID)
	})

	// Test residual bid request with complete candidate
	t.Run("complete candidate", func(t *testing.T) {
		completeBids := map[string][]jtypes.Bid{
			"node1": {bid},
			"node2": {{
				V1: &jtypes.BidV1{
					EnsembleID: ensembleID,
					NodeID:     "node2",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}},
		}

		rmBid := func(_ jtypes.Bid) {
			// This is a no-op for testing
		}

		request, err := bidC.makeResidualBidRequest(cfg, completeBids, rmBid)
		require.NoError(t, err)
		assert.Empty(t, request.Request)
	})
}

func TestMonitorOnlyTaskManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
					Type: jtypes.AllocationTypeTask,
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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Test with only task manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeTask,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
			},
		},
	}
	o.manifest = manifest

	monitorOnlyTaskManifestInterval = time.Millisecond * 200
	// Test successful task termination
	t.Run("successful task termination", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel
		defer cancel()

		go o.monitorOnlyTaskManifest()
		time.Sleep(250 * time.Millisecond)
		assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())

		o.lock.Lock()
		alloc, ok := o.manifest.Allocations["alloc1"]
		require.True(t, ok)
		alloc.Status = jtypes.AllocationCompleted
		o.manifest.Allocations["alloc1"] = alloc
		o.lock.Unlock()

		time.Sleep(250 * time.Millisecond)
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// reset the allocation status and orchestrator status
	o.lock.Lock()
	alloc, ok := o.manifest.Allocations["alloc1"]
	require.True(t, ok)
	alloc.Status = jtypes.AllocationRunning
	o.manifest.Allocations["alloc1"] = alloc
	o.lock.Unlock()

	o.setStatus(jtypes.DeploymentStatusRunning)

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel

		done := make(chan struct{})
		go func() {
			o.monitorOnlyTaskManifest()
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-done:
			assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for task monitoring to stop")
		}
	})

	// Test with non-task manifest
	t.Run("non-task manifest", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel
		defer cancel()

		// Create a new manifest with service type
		serviceManifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
				},
			},
		}

		o.manifest = serviceManifest
		go o.monitorOnlyTaskManifest()

		time.Sleep(250 * time.Millisecond)
		assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())
	})
}

func TestOrchestratorJoinSubnet(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeTask,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				},
			},
			Subnet: jtypes.SubnetConfig{Join: true},
		},
	}

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	p := NewProvisioner(ctx, o.cancel, orch.actor, o.subnetManifest, types.NewDefaultAllocationIDGenerator())

	// Prepare routing table and DNS records
	indexRoutingTable := map[string]string{"orchestrator": "10.0.0.2"}
	routingTable := map[string]string{"10.0.0.2": p.actor.Handle().Address.HostID}
	dnsRecords := map[string]string{"orchestrator": "10.0.0.2"}

	t.Run("success", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		ch := make(chan struct{}, 1)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			ch <- struct{}{}
			resp := SubnetJoinResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		o.manifest = jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
					Type:   jtypes.AllocationTypeTask,
					Status: jtypes.AllocationRunning,
				},
			},
		}

		err = p.orchestratorJoinSubnet(ensembleID, indexRoutingTable, routingTable, dnsRecords)
		assert.NoError(t, err)
		<-ch
	})

	t.Run("error response", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			resp := SubnetJoinResponse{OK: false, Error: "join failed"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		err := p.orchestratorJoinSubnet(ensembleID, indexRoutingTable, routingTable, dnsRecords)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "join failed")
	})

	t.Run("timeout", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			// Do not reply to simulate timeout
		}))
		orchestratorJoinTimeout = 1 * time.Second

		err := p.orchestratorJoinSubnet(ensembleID, indexRoutingTable, routingTable, dnsRecords)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout joining orchestrator to subnet")
	})
}

func TestEscalateFailure(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeTask,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
					HealthCheck: types.HealthCheckManifest{
						Type:     "http",
						Endpoint: "/health",
						Response: types.HealthCheckResponse{
							Type:  "string",
							Value: "OK",
						},
						Interval: time.Second,
					},
				},
			},
		},
	}

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Prepare allocation manifest
	allocManifest := jtypes.AllocationManifest{
		ID:     "test-ensemble_alloc1",
		Type:   jtypes.AllocationTypeTask,
		Status: jtypes.AllocationRunning,
		Handle: provider.handle,
		Healthcheck: types.HealthCheckManifest{
			Type:     "http",
			Endpoint: "/health",
			Response: types.HealthCheckResponse{
				Type:  "string",
				Value: "OK",
			},
			Interval: time.Second,
		},
	}
	o.supervisor.manifest.Allocations["alloc1"] = allocManifest

	t.Run("success", func(t *testing.T) {
		ch := make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			ch <- struct{}{}
			resp := behaviors.AllocationRestartResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		err := o.supervisor.escalateFailure(allocManifest)
		assert.NoError(t, err)
		<-ch
		assert.Equal(t, 1, o.supervisor.escalations[allocManifest.ID])
	})

	t.Run("error response", func(t *testing.T) {
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			resp := behaviors.AllocationRestartResponse{OK: false, Error: "restart failed"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		err := o.supervisor.escalateFailure(allocManifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "restart failed")
	})

	t.Run("timeout", func(t *testing.T) {
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			// Do not reply to simulate timeout
		}))

		FailureEscalationTimeout = 1 * time.Second
		err := o.supervisor.escalateFailure(allocManifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for supervisor reply")
	})
}

func TestCheckPermutationEdgeConstraints(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration with edge constraints
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
			Edges: []jtypes.EdgeConstraint{
				{
					S:   "node1",
					T:   "node2",
					RTT: 100,
					BW:  1000,
				},
			},
		},
	}

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Create mock GeoLocator
	mockGeo := geolocation.NewMockGeoLocator()
	// Add test locations
	mockGeo.AddLocation("US", "Los Angeles", 34.0522, -118.2437) // Los Angeles coordinates
	mockGeo.AddLocation("US", "New York", 40.7128, -74.0060)     // New York coordinates
	bidC.geo = mockGeo

	// Create test bids
	bid1 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US", City: "Los Angeles"},
			Handle:     provider.handle,
		},
	}
	bid2 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node2",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US", City: "New York"},
			Handle:     provider.handle,
		},
	}

	t.Run("successful constraint verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		result := bidC.checkPermutationEdgeConstraints(cfg, candidate)
		assert.True(t, result)
	})

	t.Run("failed constraint verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}
		cfg.V1.Edges = []jtypes.EdgeConstraint{
			{
				S:   "node1",
				T:   "node2",
				RTT: 1,
				BW:  1000,
			},
		}

		result := bidC.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})

	t.Run("missing node in candidate", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
		}

		result := bidC.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})

	t.Run("no edge constraints", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		cfgNoEdges := jtypes.EnsembleConfig{
			V1: &jtypes.EnsembleConfigV1{
				Nodes:       cfg.V1.Nodes,
				Allocations: cfg.V1.Allocations,
			},
		}

		result := bidC.checkPermutationEdgeConstraints(cfgNoEdges, candidate)
		assert.True(t, result)
	})

	t.Run("timeout during verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		result := bidC.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})
}

func TestGetNonce(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	bidC, err := NewBidCoordinator(ensembleID, orch.actor)
	require.NoError(t, err)

	// Test getting nonce
	nonce1 := bidC.getNonce()
	nonce2 := bidC.getNonce()
	assert.NotEqual(t, nonce1, nonce2)
	assert.Greater(t, nonce2, nonce1)
}

func TestProvisionSubnet(t *testing.T) {
	SubnetCreateTimeout = 1 * time.Second
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	ctx, cancel := context.WithCancel(context.Background())

	subnetManifest, err := newSubnetManifest()
	require.NoError(t, err)
	p := NewProvisioner(ctx, cancel, orch.actor, subnetManifest, types.NewDefaultAllocationIDGenerator())

	const ensembleID = "test-ensemble"

	routingTable := map[string]string{
		"10.0.0.2": provider.peerID.String(),
	}
	subCreateHandles := []actor.Handle{provider.handle}
	err = p.createSubnet(ensembleID, routingTable, subCreateHandles)
	require.NoError(t, err)
	<-provider.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)]
}

func TestIsOnlyTaskManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
					Type: jtypes.AllocationTypeTask,
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
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test manifest with only tasks
	assert.True(t, isOnlyTaskManifest(o.manifest))

	// Add a service allocation
	cfg.V1.Allocations["alloc2"] = jtypes.AllocationConfig{
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
	}
	nodeConfig := cfg.V1.Nodes["node1"]
	nodeConfig.Allocations = append(nodeConfig.Allocations, "alloc2")
	cfg.V1.Nodes["node1"] = nodeConfig
	o.manifest = o.newManifest(o.cfg)

	// Test manifest with mixed allocations
	assert.False(t, isOnlyTaskManifest(o.manifest))
}

func TestSubnetAddPeer(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	subnetManifest, err := newSubnetManifest()
	require.NoError(t, err)
	p := NewProvisioner(ctx, cancel, orch.actor, subnetManifest, types.NewDefaultAllocationIDGenerator())

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test adding peer to subnet
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	err = p.subnetAddPeer(o.manifest.ID, subReqs)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetAddPeerBehavior]
}

func TestAddDNSRecords(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	p := NewProvisioner(ctx, o.cancel, orch.actor, o.subnetManifest, types.NewDefaultAllocationIDGenerator())

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test adding DNS records
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	dnsRecords := map[string]string{
		"alloc1.internal": "10.0.0.2",
	}
	err = p.addDNSRecords(o.manifest.ID, subReqs, dnsRecords)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetDNSAddRecordsBehavior]
}

func TestMapPorts(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator())
	require.NoError(t, err)

	p := NewProvisioner(ctx, o.cancel, orch.actor, o.subnetManifest, types.NewDefaultAllocationIDGenerator())

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test mapping ports
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	err = p.mapPorts(o.manifest.ID, subReqs)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetMapPortBehavior]
}
