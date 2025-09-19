// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
)

type TestDMS struct {
	priv             crypto.PrivKey
	pub              crypto.PubKey
	peerID           peer.ID
	handle           actor.Handle
	actor            actor.Actor
	super            actor.Actor // orchestrator actor is the child of node actor. This is the parent actor in the test.
	net              network.Network
	channels         map[string]chan struct{}
	allocationActors map[string]actor.Actor // Keep allocation actors in memory
}

func MakeProvider(t *testing.T, substrate *network.Substrate) TestDMS {
	t.Helper()
	mockActor, peer, handle, priv, pub := actor.NewMockActorForTest(t, actor.Handle{}, substrate)
	dms := TestDMS{
		priv:             priv,
		pub:              pub,
		peerID:           peer.GetHostID(),
		handle:           handle,
		actor:            mockActor,
		super:            nil,
		net:              peer,
		channels:         make(map[string]chan struct{}),
		allocationActors: make(map[string]actor.Actor),
	}
	return dms
}

func MakeOrchestrator(t *testing.T, substrate *network.Substrate) TestDMS {
	t.Helper()
	mockActor, peer, handle, priv, pub := actor.NewMockActorForTest(t, actor.Handle{}, substrate)
	childActor, err := mockActor.CreateChild("test-orch-child", handle)
	require.NoError(t, err)
	require.NoError(t, childActor.Start())
	dms := TestDMS{
		priv:             priv,
		pub:              pub,
		peerID:           peer.GetHostID(),
		handle:           handle,
		actor:            childActor,
		super:            mockActor,
		net:              peer,
		channels:         make(map[string]chan struct{}),
		allocationActors: make(map[string]actor.Actor),
	}
	return dms
}

func (dms *TestDMS) MockOrchestratorBehaviors(t *testing.T, ensembleID string) {
	t.Helper()

	dms.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
	require.NoError(t, dms.super.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
	require.NoError(t, dms.super.AddBehavior(fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Add SubnetDestroy behavior for revert operations
	dms.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
	require.NoError(t, dms.super.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))
}

func (dms *TestDMS) MockDeploymentBehaviors(t *testing.T, ensembleID string, bidBehavior func(msg actor.Envelope), orchestratorActor ...actor.Actor) {
	t.Helper()

	defaultBidBehavior := func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		var request jtypes.EnsembleBidRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal bid request: %s", err)
		}

		// send bid response
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: request.ID,
				NodeID:     request.Request[0].V1.NodeID,
				Peer:       dms.handle.Address.HostID,
				Location:   jtypes.Location{Country: "US"},
				Handle:     dms.handle,
			},
		}

		// sign the bid using the provider's private key
		// Create DID provider for signing
		providerDID := did.NewProvider(dms.actor.Handle().DID, dms.priv)

		// Sign the bid
		require.NoError(t, bid.Sign(providerDID))

		var opt []actor.MessageOption
		if msg.IsBroadcast() {
			opt = append(opt, actor.WithMessageSource(dms.actor.Handle()))
		}

		reply, err := actor.ReplyTo(msg, bid, opt...)
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}
	// Add compute provider behaviors
	dms.channels[behaviors.BidRequestBehavior] = make(chan struct{})
	if bidBehavior == nil {
		bidBehavior = defaultBidBehavior
	}
	require.NoError(t, dms.actor.AddBehavior(behaviors.BidRequestBehavior, bidBehavior, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	dms.channels[behaviors.CommitDeploymentBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.CommitDeploymentBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, CommitDeploymentResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[behaviors.AllocationDeploymentBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.AllocationDeploymentBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		var request jtypes.AllocationDeploymentRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal allocation deployment request: %s", err)
		}

		allocs := request.Allocations
		// Create actual allocation actors for each allocation
		allocationHandles := make(map[string]actor.Handle)

		// Create allocation actor for alloc1 if it doesn't exist
		for alloc := range allocs {
			if _, exists := dms.allocationActors[alloc]; !exists {
				allocationActor, err := dms.actor.CreateChild(alloc, dms.actor.Handle())
				require.NoError(t, err)

				// Set up subnet behaviors on the allocation actor
				require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetAddPeerBehavior, func(msg actor.Envelope) {
					defer msg.Discard()

					reply, err := actor.ReplyTo(msg, behaviors.SubnetAddPeerResponse{
						OK: true,
					})
					require.NoError(t, err)

					reply.To = msg.From
					reply.From = allocationActor.Handle()

					require.NoError(t, allocationActor.Send(reply))
				}))

				require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetMapPortBehavior, func(msg actor.Envelope) {
					defer msg.Discard()

					reply, err := actor.ReplyTo(msg, behaviors.SubnetMapPortResponse{
						OK: true,
					})
					require.NoError(t, err)

					reply.To = msg.From
					reply.From = allocationActor.Handle()

					require.NoError(t, allocationActor.Send(reply))
				}))

				require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetDNSAddRecordsBehavior, func(msg actor.Envelope) {
					defer msg.Discard()

					reply, err := actor.ReplyTo(msg, behaviors.SubnetDNSAddRecordsResponse{
						OK: true,
					})
					require.NoError(t, err)

					reply.To = msg.From
					reply.From = allocationActor.Handle()

					require.NoError(t, allocationActor.Send(reply))
				}))
				require.NoError(t, allocationActor.AddBehavior(behaviors.AllocationStartBehavior, func(msg actor.Envelope) {
					defer msg.Discard()

					reply, err := actor.ReplyTo(msg, behaviors.AllocationStartResponse{
						OK: true,
					})
					require.NoError(t, err)

					reply.To = msg.From
					reply.From = allocationActor.Handle()

					require.NoError(t, allocationActor.Send(reply))
				}))

				require.NoError(t, allocationActor.Start())

				// Store the allocation actor in the TestDMS struct
				dms.allocationActors[alloc] = allocationActor

				// Grant capabilities between allocation actor and orchestrator
				// This simulates what happens in the commit phase
				if len(orchestratorActor) > 0 {
					// Grant capabilities from orchestrator to allocation actor (OrchestratorNamespace)
					err = orchestratorActor[0].Security().Grant(
						allocationActor.Handle().DID,
						orchestratorActor[0].Handle().DID,
						[]ucan.Capability{behaviors.OrchestratorNamespace},
						5*time.Minute,
					)
					require.NoError(t, err)

					// Grant capabilities from allocation actor to orchestrator (AllocationNamespace)
					err = allocationActor.Security().Grant(
						orchestratorActor[0].Handle().DID,
						allocationActor.Handle().DID,
						[]ucan.Capability{behaviors.AllocationNamespace},
						5*time.Minute,
					)
					require.NoError(t, err)
				}
			}
			allocationHandles[alloc] = dms.allocationActors[alloc].Handle()
		}

		reply, err := actor.ReplyTo(msg, jtypes.AllocationDeploymentResponse{
			OK:          true,
			Allocations: allocationHandles,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[behaviors.SubnetAddPeerBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetAddPeerBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetAddPeerResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[behaviors.SubnetDNSAddRecordsBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetDNSAddRecordsBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetDNSAddRecordsResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	dms.channels[behaviors.SubnetMapPortBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetMapPortBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			go func() { dms.channels[msg.Behavior] <- struct{}{} }()
		}()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetMapPortResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	require.NoError(t, dms.actor.Subscribe(behaviors.BidRequestTopic, func(_ string) error {
		return nil
	}))
}

// MockCommittingStateBehaviors sets up behaviors specific to committing state restoration
func (dms *TestDMS) MockCommittingStateBehaviors(t *testing.T, ensembleID string) {
	t.Helper()

	// Mock behavior for deployment revert (called before redeploying)
	dms.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		// Send reply for invoke-style messaging
		reply, err := actor.ReplyTo(msg, DeploymentRevertResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock behavior for reverting node deployment
	shutdownBehavior := fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)
	dms.channels[shutdownBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(shutdownBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, behaviors.AllocationRestartResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock subnet behaviors for provisioning phase
	// Note: These need to match the ensemble ID used in the test

	// Mock SubnetCreateBehavior
	createBehavior := fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)
	dms.channels[createBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(createBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock SubnetJoinBehavior
	joinBehavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
	dms.channels[joinBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(joinBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, SubnetJoinResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock AllocationStartBehavior
	dms.channels[behaviors.AllocationStartBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.AllocationStartBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, behaviors.AllocationStartResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock behavior for allocation start
	dms.channels[behaviors.AllocationStartBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.AllocationStartBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, behaviors.AllocationStartResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

	// Mock behavior for health check registration
	dms.channels[behaviors.RegisterHealthcheckBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.RegisterHealthcheckBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, behaviors.RegisterHealthcheckResponse{
			OK: true,
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))
}
