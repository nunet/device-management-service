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
	net      network.Network
	channels map[string]chan struct{}
}

func MakeDMS(t *testing.T, substrate *network.Substrate) TestDMS {
	t.Helper()
	mockActor, peer, handle, priv, pub := actor.NewMockActorForTest(t, actor.Handle{}, substrate)
	dms := TestDMS{
		priv:     priv,
		pub:      pub,
		peerID:   peer.GetHostID(),
		handle:   handle,
		actor:    mockActor,
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
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

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
		if err := bid.Sign(providerDID); err != nil {
			fmt.Printf("Failed to sign bid: %v\n", err)
			return
		}

		var opt []actor.MessageOption
		if msg.IsBroadcast() {
			opt = append(opt, actor.WithMessageSource(dms.actor.Handle()))
		}

		reply, err := actor.ReplyTo(msg, bid, opt...)
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			t.Fatalf("sending bid response: %s", err)
		}
	}, []actor.BehaviorOption{
		actor.WithBehaviorTopic(behaviors.BidRequestTopic),
	}...))

	dms.channels[behaviors.CommitDeploymentBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.CommitDeploymentBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, CommitDeploymentResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send commit deployment response: %v\n", err)
			return
		}
	}))

	dms.channels[behaviors.AllocationDeploymentBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.AllocationDeploymentBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, jtypes.AllocationDeploymentResponse{
			OK:          true,
			Allocations: map[string]actor.Handle{"alloc1": dms.handle},
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send allocation deployment response: %v\n", err)
			return
		}
	}))

	dms.channels[behaviors.AllocationStartBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.AllocationStartBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, behaviors.AllocationStartResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send allocation start response: %v\n", err)
			return
		}
	}))

	dms.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "test-ensemble")] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "test-ensemble"), func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send subnet create response: %v\n", err)
			return
		}
	}))

	dms.channels[behaviors.SubnetAddPeerBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetAddPeerBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetAddPeerResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send add peer response: %v\n", err)
			return
		}
	}))

	dms.channels[behaviors.SubnetDNSAddRecordsBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetDNSAddRecordsBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetDNSAddRecordsResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send DNS add records response: %v\n", err)
			return
		}
	}))

	dms.channels[behaviors.SubnetMapPortBehavior] = make(chan struct{})
	require.NoError(t, dms.actor.AddBehavior(behaviors.SubnetMapPortBehavior, func(msg actor.Envelope) {
		go func() {
			select {
			case dms.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, behaviors.SubnetMapPortResponse{
			OK: true,
		})
		if err != nil {
			fmt.Printf("Failed to create reply: %v\n", err)
			return
		}

		reply.To = msg.From
		reply.From = dms.handle

		if err := dms.actor.Send(reply); err != nil {
			fmt.Printf("Failed to send map port response: %v\n", err)
			return
		}
	}))

	require.NoError(t, dms.actor.Subscribe(behaviors.BidRequestTopic, func(_ string) error {
		return nil
	}))
}
