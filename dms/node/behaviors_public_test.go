package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/observability"
)

func TestPublicHelloBehavior(t *testing.T) {
	t.Parallel()

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PublicHelloBehavior)
		node.peers = make(map[peer.ID]*peerState)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PublicHelloBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp HelloResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().DID, resp.DID)
	})
}

func TestPublicStatusBehavior(t *testing.T) {
	t.Parallel()

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PublicStatusBehavior)
		node.peers = make(map[peer.ID]*peerState)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PublicStatusBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PublicStatusResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "OK", resp.Status)
		machineResources, err := node.hardware.GetMachineResources()
		require.NoError(t, err)
		assert.Equal(t, machineResources.Resources, resp.Resources)
	})
}

func TestHandleStatusDiscoveryBehavior(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	t.Run("invoke status discovery behavior", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.StatusDiscoveryBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.StatusDiscoveryBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DiscoveryStatusResponse
		err = json.Unmarshal(reply.Message, &resp)

		assert.NotEmpty(t, resp)

		// filter for self peerID
		peerResp := resp[node.network.GetHostID().String()]

		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().DID.String(), peerResp.DID)
		assert.Empty(t, peerResp.Errors)
	})

	t.Run("expect a reply with introduced errors", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.StatusDiscoveryBehavior)

		// introduce errors
		node.hardware = nil
		node.onboarding = nil

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.StatusDiscoveryBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		// we should still get a reply with errors even if all information can't be fetched
		var resp DiscoveryStatusResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NotEmpty(t, resp)

		// filter for self peerID
		peerResp := resp[node.network.GetHostID().String()]

		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().DID.String(), peerResp.DID)
		assert.Equal(t, 2, len(peerResp.Errors))
	})
}
