package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/observability"
)

func TestHandleCapList(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	t.Run("wrong payload type", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CapListBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.CapListBehavior,
			"wrong message",
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)
		reply := <-replyChan
		require.NotNil(t, reply)

		var resp CapListResponse
		err = json.Unmarshal(reply.Message, &resp)
		require.NoError(t, err)
		require.False(t, resp.OK)
		require.Contains(t, resp.Error, "cannot unmarshal")
	})

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CapListBehavior)

		// mock the root capability with actor cap
		node.rootCap = sActor.Security().Capability()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.CapListBehavior,
			CapListRequest{},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)
		reply := <-replyChan
		require.NotNil(t, reply)

		var resp CapListResponse
		err = json.Unmarshal(reply.Message, &resp)
		require.NoError(t, err)
		require.True(t, resp.OK)
		require.Empty(t, resp.Error)

		eRoots, _, _, _ := sActor.Security().Capability().ListRoots()
		assert.Equal(t, eRoots, resp.Roots)
	})
}

func TestHandleAnchorCap(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	t.Run("wrong payload type", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CapAnchorBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.CapAnchorBehavior,
			"wrong message",
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)
		reply := <-replyChan
		require.NotNil(t, reply)

		var resp CapAnchorResponse
		err = json.Unmarshal(reply.Message, &resp)
		require.NoError(t, err)
		require.False(t, resp.OK)
		require.Contains(t, resp.Error, "cannot unmarshal")
	})

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		testAnchorCap := "/test/anchor/cap"
		testAnchorTopic := "/test/anchor/topic"

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CapAnchorBehavior)

		// make sure there's only one root from the beginning
		roots, _, _, _ := node.rootCap.ListRoots()
		require.Len(t, roots, 1)

		// another actor to generate the token for the test
		priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)
		tRootDID, tRootTrust := actor.MakeRootTrustContext(t)
		tActorDID, tActorTrust := actor.MakeTrustContext(t, priv)
		tActorCap := actor.MakeCapabilityContext(t, tActorDID, tRootDID, tActorTrust, tRootTrust)

		token, err := tActorCap.Grant(
			ucan.Delegate,
			node.actor.Handle().DID,
			did.DID{},
			[]string{testAnchorTopic},
			uint64(time.Now().Add(time.Minute).UnixNano()),
			0,
			[]ucan.Capability{
				ucan.Capability(testAnchorCap),
			},
		)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.CapAnchorBehavior,
			CapAnchorRequest{
				Root:    []did.DID{tRootDID}, // will not be anchored
				Require: ucan.TokenList{},
				Provide: token,
				Revoke:  ucan.TokenList{},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)
		reply := <-replyChan
		require.NotNil(t, reply)

		var resp CapAnchorResponse
		err = json.Unmarshal(reply.Message, &resp)
		require.NoError(t, err)
		require.True(t, resp.OK)
		require.Empty(t, resp.Error)

		roots, _, provide, _ := node.rootCap.ListRoots()

		// make sure there's still only one root (do not accept root anchoring)
		assert.Len(t, roots, 1)

		// one from the beginning (reciprocal with sender actor) and one we just added
		assert.Len(t, provide.Tokens, 2)

		var iToken *ucan.Token
		for _, t := range provide.Tokens {
			if t.Issuer().Equal(tActorDID) {
				iToken = t
				break
			}
		}

		assert.True(t, iToken.Issuer().Equal(tActorDID))
		assert.True(t, iToken.Subject().Equal(node.actor.Handle().DID))
		assert.Equal(t, testAnchorCap, string(iToken.Capability()[0]))
		assert.Equal(t, testAnchorTopic, string(iToken.Topic()[0]))
	})
}
