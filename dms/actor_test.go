package dms

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewActorFactory(t *testing.T) {
	factory := NewActorFactory("host1", &libp2p.Libp2p{}, &ActorParams{
		HeartbeatInterval:      time.Second,
		HeartbeatCheckInterval: time.Second,
		Threshold:              3,
		Action:                 func() {},
	})

	assert.Equal(t, "host1", factory.hostID)
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
				hostID:  "123",
				network: &libp2p.Libp2p{},
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
	net, err := libp2p.New(&types.Libp2pConfig{
		Scheduler: &backgroundtasks.Scheduler{},
	}, nil)
	assert.NoError(t, err)
	err = net.Init(context.Background())
	assert.NoError(t, err)
	factory := NewActorFactory("host1", net, &ActorParams{
		HeartbeatInterval:      time.Second * 5,
		HeartbeatCheckInterval: time.Second * 8,
		Threshold:              3,
		Action:                 func() {},
	})

	root, err := factory.NewActor()
	assert.NoError(t, err)

	newActor, err := root.CreateActor()
	assert.NoError(t, err)
	err = newActor.Start()
	assert.NoError(t, err)

	assert.Equal(t, "host1", newActor.hostID)
	assert.NotEmpty(t, newActor.Address().InboxAddress)
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
	err = actor1.Start()
	assert.NoError(t, err)

	// send a message from root 1 to newly created actor which are on the same dms
	// both have same host id but different inbox address
	msgToSend := &Message{
		Type:   "generic",
		Sender: actor1.Address().InboxAddress,
		Data:   []byte("hello world"),
	}
	err = root1.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.hostID, InboxAddress: actor1.Address().InboxAddress}, msgToSend)
	assert.NoError(t, err)

	// create another root actor from factory 2 and start it
	root2AnotherMachine, err := factory2.NewActor()
	assert.NoError(t, err)
	err = root2AnotherMachine.Start()
	assert.NoError(t, err)

	// from root 2, send a message to root 1
	msgToSend2 := &Message{
		Type:   "generic",
		Sender: root2AnotherMachine.address,
		Data:   []byte("hello from remote actor"),
	}
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: root1.hostID, InboxAddress: root1.address}, msgToSend2)
	assert.NoError(t, err)

	// from root 2, send a message to newly created actor 1 by root 1
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.hostID, InboxAddress: actor1.Address().InboxAddress}, msgToSend2)
	assert.NoError(t, err)

	// send invalid message
	err = root2AnotherMachine.SendMessage(context.Background(), &ActorAddrInfo{HostID: actor1.hostID, InboxAddress: actor1.Address().InboxAddress}, nil)
	assert.EqualError(t, err, "message is invalid")
}

func TestActorHeartbeat(t *testing.T) {
	heartbeatInterval := time.Second * 5
	heartbeatCheckInterval := time.Second * 8

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

	actionCalled := make(chan bool)
	root1.factory.params.Action = func() {
		actionCalled <- true
	}
	// create another actor from root 1 and get its address
	actor1, err := root1.CreateActor()
	assert.NoError(t, err)
	err = actor1.Start()
	assert.NoError(t, err)

	// send a message from root 1 to newly created actor which are on the same dms
	// both have same host id but different inbox address
	// msgToSend := &Message{
	// 	msgType: "heartbeat",
	// 	sender:  actor1.Address().InboxAddress,
	// 	data:    []byte("hello world"),
	// }
	// err = root1.SendHeartbeat(context.Background(), &ActorAddrInfo{HostID: actor1.HostID, InboxAddress: actor1.InboxAddress}, msgToSend)
	// assert.NoError(t, err)

	// create another root actor from factory 2 and start it
	root2AnotherMachine, err := factory2.NewActor()
	assert.NoError(t, err)
	err = root2AnotherMachine.Start()
	assert.NoError(t, err)

	{ // actor to actor heartbeat, no parent-child relationship
		// from root 2, send a message to root 1
		msgToSend2 := &Message{
			Type:   "heartbeat",
			Sender: root2AnotherMachine.address,
			Data:   []byte("hello from remote actor"),
		}
		err = root2AnotherMachine.SendHeartbeat(context.Background(), &ActorAddrInfo{HostID: root1.hostID, InboxAddress: root1.address}, msgToSend2)
		assert.NoError(t, err)

		// from root 2, send a message to newly created actor 1 by root 1
		err = root2AnotherMachine.SendHeartbeat(context.Background(), &ActorAddrInfo{HostID: actor1.Address().HostID, InboxAddress: actor1.Address().InboxAddress}, msgToSend2)
		assert.NoError(t, err)
	}

	{ // parent-child relationship
		<-time.After(heartbeatCheckInterval) // wait for heartbeat to be sent from actor1 (child) to root1 (parent)
		hbt, ok := root1.heartbeatTracker[actor1.Address().InboxAddress]
		assert.True(t, ok)
		assert.NotNil(t, hbt)
		assert.Equal(t, 0, hbt.missed)
		assert.True(t, hbt.lastHeartbeatMS >= heartbeatCheckInterval.Milliseconds()-heartbeatInterval.Milliseconds())
	}

	{ // child fails to heartbeat, parent performs action
		require.NoError(t, actor1.Stop())
		<-time.After(heartbeatCheckInterval*3 + jitter(heartbeatCheckInterval, 0.5)) // wait for 3 missed heartbeats
		assert.True(t, <-actionCalled)

		hbt, ok := root1.heartbeatTracker[actor1.Address().InboxAddress]
		assert.True(t, ok)
		assert.NotNil(t, hbt)
		assert.Equal(t, 3, hbt.missed)
		assert.True(t, hbt.lastHeartbeatMS > 0)
		assert.True(t, time.Now().UnixMilli() >= (heartbeatInterval*3+time.Second).Milliseconds())
	}
}

func newActorFactory(t *testing.T, port string, bootstrap []multiaddr.Multiaddr) (*ActorFactory, []multiaddr.Multiaddr, *libp2p.Libp2p) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
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
	return NewActorFactory(libp2pInstance.DHT.PeerID().String(), net, &ActorParams{
		HeartbeatInterval:      time.Second * 5,
		HeartbeatCheckInterval: time.Second * 10,
		Threshold:              2,
		Action:                 func() {},
	}), multi, libp2pInstance
}
