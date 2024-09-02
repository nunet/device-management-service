package actor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/crypto"

	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		dispatch  *Dispatch
		scheduler *backgroundtasks.Scheduler
		net       network.Network
		security  *BasicSecurityContext
		params    BasicActorParams
		self      Handle
		expErr    string
	}{
		"nil dispatch": {
			expErr: "dispatch is nil",
		},
		"nil scheduler": {
			dispatch: &Dispatch{},
			expErr:   "scheduler is nil",
		},
		"nil network": {
			dispatch:  &Dispatch{},
			scheduler: &backgroundtasks.Scheduler{},
			expErr:    "network is nil",
		},
		"nil security": {
			dispatch:  &Dispatch{},
			scheduler: &backgroundtasks.Scheduler{},
			net:       &libp2p.Libp2p{},
			expErr:    "security is nil",
		},
		"success": {
			dispatch:  &Dispatch{},
			scheduler: &backgroundtasks.Scheduler{},
			net:       &libp2p.Libp2p{},
			security:  &BasicSecurityContext{},
			params:    BasicActorParams{},
			self:      Handle{},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			act, err := New(tt.dispatch, tt.scheduler, tt.net, tt.security, tt.params, tt.self)
			if tt.expErr != "" {
				assert.Nil(t, act)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, act)
			}
		})
	}
}

func TestActorMessaging(t *testing.T) {
	addrs1, peer1 := newLibp2pNetwork(t, "15219", []multiaddr.Multiaddr{})
	_, peer2 := newLibp2pNetwork(t, "15220", addrs1)

	res, err := peer2.Ping(context.Background(), peer1.Host.ID().String(), time.Second)
	assert.NoError(t, err)
	assert.True(t, res.Success)

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 2, peer2.Host.Peerstore().Peers().Len())
	assert.Equal(t, 2, peer1.Host.Peerstore().Peers().Len())

	// create actors
	actor1 := createActor(t, peer1)
	err = actor1.Start()
	assert.NoError(t, err)
	actor2 := createActor(t, peer2)
	err = actor2.Start()
	assert.NoError(t, err)

	const registeredBehaviour = "/test/someBehaviour"

	envChan := make(chan Envelope)

	err = actor1.AddBehavior(registeredBehaviour, func(msg Envelope) {
		envChan <- msg
	})
	assert.NoError(t, err)

	type examplePayload struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	msg, err := Message(actor2.self, actor1.self, registeredBehaviour, examplePayload{Name: "random name", Type: "x"}, WithMessageSignature(actor2.security))

	go func() {
		assert.NoError(t, err)
		err = actor2.Send(msg)
		assert.NoError(t, err)
	}()

	received := <-envChan
	assert.Equal(t, string(received.Message), "{\"name\":\"random name\",\"type\":\"x\"}")

	// invoke
	_, err = actor1.Invoke(msg)
	assert.NoError(t, err)
}

func createActor(t *testing.T, peer *libp2p.Libp2p) *BasicActor {
	pubKey := peer.Host.Peerstore().PrivKey(peer.Host.ID()).GetPublic()
	sctx1, err := NewBasicSecurityContext(pubKey, peer.PS.PrivKey(peer.Host.ID()), DID{})

	assert.NoError(t, err)
	actor1Params := BasicActorParams{}

	id, err := uuid.NewUUID()
	assert.NoError(t, err)

	peerInfo := Handle{
		ID: sctx1.id,
		Address: Address{
			HostID:       peer.Host.ID().String(),
			InboxAddress: id.String(),
		},
	}
	actor1, err := New(NewDispatch(sctx1, WithDispatchLimiter(NoDispatchLimiter{})), backgroundtasks.NewScheduler(1), peer, sctx1, actor1Params, peerInfo)
	assert.NoError(t, err)
	assert.NotNil(t, actor1)

	return actor1
}

func newLibp2pNetwork(t *testing.T, port string, bootstrap []multiaddr.Multiaddr) ([]multiaddr.Multiaddr, *libp2p.Libp2p) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	assert.NoError(t, err)
	net, err := network.NewNetwork(&types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrap,
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               backgroundtasks.NewScheduler(1),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/127.0.0.1/tcp/" + port},
			PeerCountDiscoveryLimit: 40,
			PrivateNetwork: types.PrivateNetworkConfig{
				WithSwarmKey: false,
			},
		},
	}, afero.NewMemMapFs())
	assert.NoError(t, err)
	err = net.Init(context.Background())
	assert.NoError(t, err)

	err = net.Start(context.Background())
	assert.NoError(t, err)

	libp2pInstance, _ := net.(*libp2p.Libp2p)

	multi, err := libp2pInstance.GetMultiaddr()
	assert.NoError(t, err)
	return multi, libp2pInstance
}
