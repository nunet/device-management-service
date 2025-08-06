package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandlePeerPing(t *testing.T) {
	t.Parallel()

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerPingBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerPingBehavior,
			"invalid-request", // invalid request
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PingResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), resp.RTT)
		assert.EqualError(t, types.ErrUnmarshal, resp.Error)
	})
	t.Run("empty request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerPingBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerPingBehavior,
			PingRequest{}, // Empty request
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PingResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), resp.RTT)
		assert.Contains(t, resp.Error, ErrHostEmpty.Error())
	})

	t.Run("successful ping", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerPingBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerPingBehavior,
			PingRequest{
				Host: sActor.Handle().Address.HostID,
			}, // Empty request
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PingResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.RTT)
		assert.Empty(t, resp.Error)
	})
}

func TestHandlePeerList(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeersListBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeersListBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PeersListResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Peers))
		assert.Equal(t, sActor.Handle().Address.HostID, resp.Peers[0].String())
	})
}

func TestHandlePeerSelf(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerAddrInfoBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerAddrInfoBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PeerAddrInfoResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().Address.HostID, resp.ID)
	})
}

func TestHandlePeerConnect(t *testing.T) {
	t.Parallel()

	t.Run("empty request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerConnectBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerConnectBehavior,
			PeerConnectRequest{},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PeerConnectResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Status)
		assert.NotEmpty(t, resp.Error)
	})
	t.Run("successful with libp2p", func(t *testing.T) {
		t.Parallel()

		// node, sActor, _ := newMockNodeWithSender(t, behaviors.PeerConnectBehavior)
		nodeNet, nPriv := newLibp2pNetwork(t, afero.NewMemMapFs(), nil, backgroundtasks.NewScheduler(1, time.Second))
		nActor, nActorCap, nRootTrust, nRootDID := newActor(t, nPriv, nodeNet)
		node := &Node{}
		node.network = nodeNet
		node.actor = nActor

		nodeAddr, err := nodeNet.GetMultiaddr()
		require.NoError(t, err)

		// sender actor
		sNet, sPriv := newLibp2pNetwork(
			t,
			afero.NewMemMapFs(),
			nodeAddr,
			backgroundtasks.NewScheduler(1, time.Second),
		)
		sActor, sActorCap, sRootTrust, sRootDID := newActor(t, sPriv, sNet)

		actor.AllowReciprocal(t, nActorCap, nRootTrust, nRootDID, sActor.Handle().DID, behaviors.PeerConnectBehavior)
		actor.AllowReciprocal(t, sActorCap, sRootTrust, sRootDID, nActor.Handle().DID, behaviors.PeerConnectBehavior)

		require.NoError(t, sActor.Start())
		require.NoError(t, nActor.Start())

		// Add behavior to test
		err = node.actor.AddBehavior(behaviors.PeerConnectBehavior, node.handlePeerConnect)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.PeerConnectBehavior,
			PeerConnectRequest{
				Address: "/p2p/" + sNet.GetHostID().String(),
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp PeerConnectResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "CONNECTED", resp.Status)
		assert.Empty(t, resp.Error)

		require.NoError(t, sActor.Stop())
		require.NoError(t, nActor.Stop())
		require.NoError(t, sNet.Stop())
		require.NoError(t, nodeNet.Stop())
	})
}
