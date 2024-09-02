package actor

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDispatch(t *testing.T) {
	sc := generateSecurityContext(t)
	d := NewDispatch(sc, WithDispatchWorkers(5), WithDispatchGCInterval(60*time.Second))
	require.Equal(t, 5, d.options.Workers)
	require.Equal(t, 60*time.Second, d.options.GCInterval)
}

func TestDispatchStart(t *testing.T) {
	sc := generateSecurityContext(t)
	d := NewDispatch(sc, WithDispatchWorkers(3))
	d.Start()
	assert.True(t, d.started)
}

func TestDispatchAddBehavior(t *testing.T) {
	sc := generateSecurityContext(t)
	d := NewDispatch(sc)
	d.Start()

	behavior := func(_ Envelope) {}

	err := d.AddBehavior("test", behavior)
	assert.NoError(t, err)
	assert.Len(t, d.behaviors, 1)

	d.RemoveBehavior("test")
	assert.Len(t, d.behaviors, 0)
}

func TestDispatchReceive(t *testing.T) {
	sc := generateSecurityContext(t)
	d := NewDispatch(sc)
	d.Start()

	behaviorExecuted := make(chan bool)

	behavior := func(_ Envelope) {
		behaviorExecuted <- true
	}

	err := d.AddBehavior("test", behavior)
	assert.NoError(t, err)

	me := Handle{
		ID:  sc.ID(),
		DID: sc.DID(),
		Address: Address{
			HostID:       "123",
			InboxAddress: "111",
		},
	}

	msg, err := Message(me, me, "test", nil, WithMessageSignature(sc))
	assert.NoError(t, err)

	err = d.Receive(msg)
	assert.NoError(t, err)

	select {
	case <-behaviorExecuted:
	case <-time.After(2 * time.Second):
		t.Fatal("behavior was not executed")
	}
}

func TestDispatchGC(t *testing.T) {
	sc := generateSecurityContext(t)
	d := NewDispatch(sc, WithDispatchGCInterval(10*time.Millisecond))
	d.Start()

	behavior := func(_ Envelope) {}
	expireTime := uint64(time.Now().Add(10 * time.Millisecond).UnixNano())
	err := d.AddBehavior("test", behavior, WithBehaviorExpiry(expireTime))
	assert.NoError(t, err)
	time.Sleep(20 * time.Millisecond)
	assert.Len(t, d.behaviors, 0)
}

func generateSecurityContext(t *testing.T) *BasicSecurityContext {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	assert.NoError(t, err)
	pubKeyBytes, err := priv.GetPublic().Raw()
	assert.NoError(t, err)
	sc, err := NewBasicSecurityContext(priv.GetPublic(), priv, DID{PublicKey: pubKeyBytes})
	assert.NoError(t, err)
	return sc
}
