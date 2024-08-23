package libp2p

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"

	"gitlab.com/nunet/device-management-service/types"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		config *types.Libp2pConfig
		expErr string
	}{
		"no config": {
			config: nil,
			expErr: "config is nil",
		},
		"no scheduler": {
			config: &types.Libp2pConfig{
				PrivateKey:              &crypto.Secp256k1PrivateKey{},
				BootstrapPeers:          []multiaddr.Multiaddr{},
				Rendezvous:              "nunet-randevouz",
				Server:                  false,
				Scheduler:               nil,
				CustomNamespace:         "/nunet-dht-1/",
				ListenAddress:           []string{"/ip4/localhost/tcp/10209"},
				PeerCountDiscoveryLimit: 40,
				PrivateNetwork: types.PrivateNetworkConfig{
					WithSwarmKey: false,
				},
			},
			expErr: "scheduler is nil",
		},
		"success": {
			config: &types.Libp2pConfig{
				PrivateKey:              &crypto.Secp256k1PrivateKey{},
				BootstrapPeers:          []multiaddr.Multiaddr{},
				Rendezvous:              "nunet-randevouz",
				Server:                  false,
				Scheduler:               backgroundtasks.NewScheduler(1),
				CustomNamespace:         "/nunet-dht-1/",
				ListenAddress:           []string{"/ip4/localhost/tcp/10209"},
				PeerCountDiscoveryLimit: 40,
				PrivateNetwork: types.PrivateNetworkConfig{
					WithSwarmKey: false,
				},
			},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			libp2p, err := New(tt.config, afero.NewMemMapFs())
			if tt.expErr != "" {
				assert.Nil(t, libp2p)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, libp2p)
			}
		})
	}
}

func TestPingResolveAddress(t *testing.T) {
	peer1, peer2, _ := createPeers(t, 65512, 65513, 65514)
	pingResult, err := peer1.Ping(context.TODO(), peer2.Host.ID().String(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, pingResult.Success)
	zeroMicro, err := time.ParseDuration("0µs")
	assert.NoError(t, err)
	assert.Greater(t, pingResult.RTT, zeroMicro)

	addresses, err := peer1.ResolveAddress(context.Background(), peer2.Host.ID().String())
	assert.NoError(t, err)
	assert.Greater(t, len(addresses), 0)
	assert.Contains(t, addresses[0], peer2.Host.ID().String())
}

func TestAdvertiseUnadvertiseQuery(t *testing.T) {
	peer1, peer2, peer3 := createPeers(t, 65515, 65516, 65517)
	// advertise key
	err := peer1.Advertise(context.TODO(), "who_am_i", []byte(`{"peer":"peer1"}`))
	assert.NoError(t, err)
	err = peer2.Advertise(context.TODO(), "who_am_i", []byte(`{"peer":"peer2"}`))
	assert.NoError(t, err)
	err = peer3.Advertise(context.TODO(), "who_am_i", []byte(`{"peer":"peer3"}`))
	assert.NoError(t, err)

	time.Sleep(40 * time.Millisecond)

	// get the peers who have the who_am_i key
	advertisements, err := peer1.Query(context.TODO(), "who_am_i")
	assert.NoError(t, err)
	assert.NotNil(t, advertisements)
	assert.Len(t, advertisements, 3)

	// check if all peers have returned the correct data
	for _, v := range advertisements {
		switch v.PeerId {
		case peer1.Host.ID().String():
			{
				assert.Equal(t, []byte(`{"peer":"peer1"}`), v.Data)
			}
		case peer2.Host.ID().String():
			{
				assert.Equal(t, []byte(`{"peer":"peer2"}`), v.Data)
			}
		case peer3.Host.ID().String():
			{
				assert.Equal(t, []byte(`{"peer":"peer3"}`), v.Data)
			}
		}
	}

	// peer3 unadvertises
	err = peer3.Unadvertise(context.TODO(), "who_am_i")
	assert.NoError(t, err)
	time.Sleep(40 * time.Millisecond)

	// get the values again, it should be 2 peers only
	advertisements, err = peer1.Query(context.TODO(), "who_am_i")
	assert.NoError(t, err)
	assert.NotNil(t, advertisements)
	assert.Len(t, advertisements, 2)
}

func TestPublishSubscribeUnsubscribe(t *testing.T) {
	peer1, _, _ := createPeers(t, 65518, 65519, 65520)
	// test publish/subscribe
	// in this test gossipsub messages are not always delivered from peer1 to other peers
	// and this makes the test flaky. To solve the flakiness we use the peer one for
	// both publishing and subscribing.
	subscribeData := make(chan []byte)
	err := peer1.Subscribe(context.TODO(), "blocks", func(data []byte) {
		subscribeData <- data
	})
	assert.NoError(t, err)

	// calling one time publish doesnt guarantee that the message has been sent.
	// without this we might not get a message in the subscribe and we would get
	// flaky test which will fail
	go func() {
		for {
			err = peer1.Publish(context.TODO(), "blocks", []byte(`{"block":"1"}`))
		}
	}()

	receivedData := <-subscribeData
	assert.EqualValues(t, []byte(`{"block":"1"}`), receivedData)

	// unsubscribe
	err = peer1.Unsubscribe("blocks")
	assert.NoError(t, err)
}

// if we connect to ourselves we deliver the message properlly to the right handler.
func TestSelfDial(t *testing.T) {
	peer1, _, _ := createPeers(t, 65524, 65525, 65526)
	messageType := types.MessageType("/custom_bytes_callback/1.1.4")
	messageChannel := make(chan string)

	err := peer1.RegisterBytesMessageHandler(messageType, func(dt []byte) {
		messageChannel <- string(dt)
	})
	assert.NoError(t, err)
	peer1p2pAddrs, err := peer1.GetMultiaddr()
	assert.NoError(t, err)
	err = peer1.SendMessage(context.Background(), []string{peer1p2pAddrs[0].String()}, types.MessageEnvelope{
		Type: messageType,
		Data: []byte("testing 123"),
	})
	assert.NoError(t, err)
	// check if we received the data properlly
	received := <-messageChannel
	assert.Equal(t, "testing 123", received)
}

func TestSendMessageAndHandlers(t *testing.T) {
	peer1, peer2, _ := createPeers(t, 65521, 65522, 65523)
	peer1p2pAddrs, err := peer1.GetMultiaddr()
	assert.NoError(t, err)

	// test sendmessage and stream functionality
	// use different test cases to check if communication can be established to remote peers
	// 1. peer1 is not handling any message types so we try to send a message and get an error

	customMessageProtocol := types.MessageType("/chat/1.1.1")

	helloWorlPayload := "hello world"
	err = peer2.SendMessage(context.TODO(), []string{peer1p2pAddrs[0].String()}, types.MessageEnvelope{Type: customMessageProtocol, Data: []byte(helloWorlPayload)})
	assert.ErrorContains(t, err, "protocols not supported: [/chat/1.1.1]")

	// 2. peer1 registers the message
	payloadReceived := make(chan string)
	err = peer1.RegisterStreamMessageHandler(customMessageProtocol, func(stream network.Stream) {
		bytesToRead := make([]byte, len([]byte(helloWorlPayload))+8)
		_, err = stream.Read(bytesToRead)
		assert.NoError(t, err)
		payloadReceived <- string(bytesToRead[8:])
	})
	assert.NoError(t, err)
	// 3. re-register should be error
	err = peer1.RegisterStreamMessageHandler(customMessageProtocol, func(_ network.Stream) {})
	assert.ErrorContains(t, err, "stream with this protocol is already registered")

	// 4. send message from peer2 and wait to get the response from peer1
	err = peer2.SendMessage(context.TODO(), []string{peer1p2pAddrs[0].String()}, types.MessageEnvelope{Type: customMessageProtocol, Data: []byte(helloWorlPayload)})
	assert.NoError(t, err)
	peer1MessageContent := <-payloadReceived
	assert.Equal(t, helloWorlPayload, peer1MessageContent)

	// 5. open stream functionality
	streamPayloadMessage := "nunet world"
	streamPayloadReceived := make(chan string)
	err = peer1.RegisterStreamMessageHandler(types.MessageType("/custom_stream/1.2.3"), func(stream network.Stream) {
		bytesToRead := make([]byte, len([]byte(streamPayloadMessage)))
		_, err := stream.Read(bytesToRead)
		assert.NoError(t, err)
		streamPayloadReceived <- string(bytesToRead)
	})
	assert.NoError(t, err)
	openedStream, err := peer2.OpenStream(context.TODO(), peer1p2pAddrs[0].String(), types.MessageType("/custom_stream/1.2.3"))
	assert.NoError(t, err)
	_, err = openedStream.Write([]byte("nunet world"))
	assert.NoError(t, err)
	peer1MessageContent = <-streamPayloadReceived
	openedStream.Close()
	assert.Equal(t, "nunet world", peer1MessageContent)

	// 6. register message with bytes handler function
	secondPayloadReceived := make(chan string)
	err = peer1.RegisterBytesMessageHandler(types.MessageType("/custom_bytes_callback/1.1.4"), func(dt []byte) {
		secondPayloadReceived <- string(dt)
	})
	assert.NoError(t, err)
	err = peer2.SendMessage(context.TODO(), []string{peer1p2pAddrs[0].String()}, types.MessageEnvelope{Type: types.MessageType("/custom_bytes_callback/1.1.4"), Data: []byte(helloWorlPayload)})
	assert.NoError(t, err)
	messageFromBytesHandler := <-secondPayloadReceived
	assert.Equal(t, helloWorlPayload, messageFromBytesHandler)
}

func createPeers(t *testing.T, port1, port2, port3 int) (*Libp2p, *Libp2p, *Libp2p) {
	// setup peer1
	peer1Config := setupPeerConfig(t, port1, []multiaddr.Multiaddr{}, false)
	peer1, err := New(peer1Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer1)
	err = peer1.Init(context.TODO())
	assert.NoError(t, err)
	err = peer1.Start(context.TODO())
	assert.NoError(t, err)

	// peers of peer1 should be one (itself)
	assert.Equal(t, peer1.Host.Peerstore().Peers().Len(), 1)

	// setup peer2 to connect to peer 1
	peer1p2pAddrs, err := peer1.GetMultiaddr()
	assert.NoError(t, err)
	peer2Config := setupPeerConfig(t, port2, peer1p2pAddrs, false)
	peer2, err := New(peer2Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer2)

	err = peer2.Init(context.TODO())
	assert.NoError(t, err)
	err = peer2.Start(context.TODO())
	assert.NoError(t, err)

	// sleep until the inernals of the host get updated
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, peer2.Host.Peerstore().Peers().Len())
	assert.Equal(t, 2, peer1.Host.Peerstore().Peers().Len())

	// setup a new peer and advertise specs
	// peer3 will connect to peer2 in a ring setup.
	peer2p2pAddrs, err := peer2.GetMultiaddr()
	assert.NoError(t, err)
	peer3Config := setupPeerConfig(t, port3, peer2p2pAddrs, false)
	peer3, err := New(peer3Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer3)

	err = peer3.Init(context.TODO())
	assert.NoError(t, err)
	err = peer3.Start(context.TODO())
	assert.NoError(t, err)

	return peer1, peer2, peer3
}
