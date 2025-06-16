package orchestrator

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
)

type TestDMS struct {
	priv     crypto.PrivKey
	pub      crypto.PubKey
	peerID   peer.ID
	handle   actor.Handle
	actor    actor.Actor
	super    actor.Actor // orchestrator actor is the child of node actor. This is the parent actor in the test.
	net      network.Network
	channels map[string]chan struct{}
}

func MakeProvider(t *testing.T, substrate *network.Substrate) TestDMS {
	t.Helper()
	mockActor, peer, handle, priv, pub := actor.NewMockActorForTest(t, actor.Handle{}, substrate)
	dms := TestDMS{
		priv:     priv,
		pub:      pub,
		peerID:   peer.GetHostID(),
		handle:   handle,
		actor:    mockActor,
		super:    nil,
		net:      peer,
		channels: make(map[string]chan struct{}),
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
		priv:     priv,
		pub:      pub,
		peerID:   peer.GetHostID(),
		handle:   handle,
		actor:    childActor,
		super:    mockActor,
		net:      peer,
		channels: make(map[string]chan struct{}),
	}
	return dms
}

func (dms *TestDMS) MockDeploymentBehaviors(t *testing.T) {
	t.Helper()

	// Add compute provider behaviors
	dms.channels[behaviors.BidRequestBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		var request jtypes.EnsembleBidRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			t.Fatalf("unmarshal bid request: %s", err)
		}

		// send bid response
		bid := jtypes.Bid{
			V1: &jtypes.BidV1{
				EnsembleID: request.ID,
				NodeID:     "node1",
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
	}, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	dms.channels[behaviors.CommitDeploymentBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.CommitDeploymentBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
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
			dms.channels[msg.Behavior] <- struct{}{}
		}()

		reply, err := actor.ReplyTo(msg, jtypes.AllocationDeploymentResponse{
			OK:          true,
			Allocations: map[string]actor.Handle{"alloc1": dms.handle},
		})
		require.NoError(t, err)

		reply.To = msg.From
		reply.From = dms.handle

		require.NoError(t, dms.actor.Send(reply))
	}))

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

	dms.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "test-ensemble")] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "test-ensemble"), func(msg actor.Envelope) {
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

	dms.channels[behaviors.SubnetAddPeerBehavior] = make(chan struct{}, 1)
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetAddPeerBehavior, func(msg actor.Envelope) {
		defer func() {
			msg.Discard()
			dms.channels[msg.Behavior] <- struct{}{}
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
			dms.channels[msg.Behavior] <- struct{}{}
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
			dms.channels[msg.Behavior] <- struct{}{}
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
