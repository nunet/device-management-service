package dms

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

func TestNewActorFactory(t *testing.T) {
	actorRegistry := NewActorRegistry()

	factory := NewActorFactory("host1", &libp2p.Libp2p{}, actorRegistry)

	assert.Equal(t, "host1", factory.hostID)
	assert.Equal(t, actorRegistry, factory.actorRegistry)
}

func TestNewActor(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		af     *ActorFactory
		expErr string
	}{
		"host id empty": {
			af:     &ActorFactory{},
			expErr: "host id is empty",
		},
		"network is nil": {
			af: &ActorFactory{
				hostID: "123",
			},
			expErr: "network is nil",
		},
		"success": {
			af: &ActorFactory{
				hostID:        "123",
				network:       &libp2p.Libp2p{},
				actorRegistry: NewActorRegistry(),
			},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			act, err := tt.af.NewActor()
			if tt.expErr != "" {
				assert.Nil(t, act)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, act)
				assert.Len(t, act.actorRegistry.actors, 1)
			}
		})
	}
}

func TestCreateActor(t *testing.T) {
	actorRegistry := NewActorRegistry()
	net, err := libp2p.New(&types.Libp2pConfig{
		Scheduler: &background_tasks.Scheduler{},
	}, nil)
	assert.NoError(t, err)
	err = net.Init(context.Background())
	assert.NoError(t, err)
	factory := NewActorFactory("host1", net, actorRegistry)

	root, err := factory.NewActor()
	assert.NoError(t, err)

	newActor, err := root.CreateActor()
	assert.NoError(t, err)
	assert.Equal(t, "host1", newActor.HostID)
	assert.NotEmpty(t, newActor.InboxAddress)
}

func TestActorMessaging(t *testing.T) {
	// create 2 peers with 2 different factories
	factory1, multiAddr, peer1 := newActorFactory(t, "15219", []multiaddr.Multiaddr{})
	factory2, _, peer2 := newActorFactory(t, "15220", multiAddr)

	res, err := peer2.Ping(context.Background(), peer1.Host.ID().String(), time.Second)
	assert.NoError(t, err)
	assert.True(t, res.Success)

	// check if both peers are connected
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 2, peer2.Host.Peerstore().Peers().Len())
	assert.Equal(t, 2, peer1.Host.Peerstore().Peers().Len())

	// create a root 1 actor and start it
	root1, err := factory1.NewActor()
	assert.NoError(t, err)
	err = root1.Start()
	assert.NoError(t, err)

	// create another actor from root 1 and get its address
	actor1, err := root1.CreateActor()
	assert.NoError(t, err)

	// send a message from root 1 to newly created actor which are on the same dms
	// both have same host id but different inbox address
	msgToSend := &Message{
		msgType: "generic",
		sender:  actor1.InboxAddress,
		data:    []byte("hello world"),
	}
	err = root1.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.HostID, InboxAddress: actor1.InboxAddress}, msgToSend)
	assert.NoError(t, err)

	// create another root actor from factory 2 and start it
	root2AnotherMachine, err := factory2.NewActor()
	assert.NoError(t, err)
	err = root2AnotherMachine.Start()
	assert.NoError(t, err)

	// from root 2, send a message to root 1
	msgToSend2 := &Message{
		msgType: "generic",
		sender:  root2AnotherMachine.address,
		data:    []byte("hello from remote actor"),
	}
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: root1.hostID, InboxAddress: root1.address}, msgToSend2)
	assert.NoError(t, err)

	// from root 2, send a message to newly created actor 1 by root 1
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.HostID, InboxAddress: actor1.InboxAddress}, msgToSend2)
	assert.NoError(t, err)

	// send invalid message
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.HostID, InboxAddress: actor1.InboxAddress}, nil)
	assert.EqualError(t, err, "message is invalid")
}

func newActorFactory(t *testing.T, port string, bootstrap []multiaddr.Multiaddr) (*ActorFactory, []multiaddr.Multiaddr, *libp2p.Libp2p) {
	actorRegistry := NewActorRegistry()
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	net, err := network.NewNetwork(&types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrap,
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               background_tasks.NewScheduler(1),
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
	return NewActorFactory(libp2pInstance.DHT.PeerID().String(), net, actorRegistry), multi, libp2pInstance
}
