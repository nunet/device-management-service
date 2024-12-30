// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/avast/retry-go"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNew(t *testing.T) {
	// Set observability to no-op mode for testing
	observability.SetNoOpMode(true)

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

	ip := peer1.GetPeerIP(peer2.Host.ID())
	assert.NotEmpty(t, ip)
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

	time.Sleep(400 * time.Millisecond)

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
	subID, err := peer1.Subscribe(context.TODO(), "blocks", func(data []byte) {
		subscribeData <- data
	}, nil)
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
	err = peer1.Unsubscribe("blocks", subID)
	assert.NoError(t, err)
}

// if we connect to ourselves we deliver the message properlly to the right handler.
func TestSelfDial(t *testing.T) {
	peer1, _, _ := createPeers(t, 65524, 65525, 65526)
	messageType := types.MessageType("/custom_bytes_callback/1.1.4")
	messageChannel := make(chan string)

	err := peer1.RegisterBytesMessageHandler(messageType, func(dt []byte, _ peer.ID) {
		messageChannel <- string(dt)
	})
	assert.NoError(t, err)
	err = peer1.SendMessage(context.Background(), peer1.Host.ID().String(),
		types.MessageEnvelope{
			Type: messageType,
			Data: []byte("testing 123"),
		},
		time.Now().Add(readTimeout),
	)
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
	// The semantics of send have changed; this test is not applicable

	customMessageProtocol := types.MessageType("/chat/1.1.1")
	helloWorlPayload := "hello world"

	// 2. peer1 registers the message
	payloadReceived := make(chan string)
	err = peer1.RegisterStreamMessageHandler(customMessageProtocol, func(stream network.Stream) {
		bytesToRead := make([]byte, len([]byte(helloWorlPayload))+8)
		_, err = stream.Read(bytesToRead)
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, helloWorlPayload, string(bytesToRead[8:]))
		payloadReceived <- string(bytesToRead[8:])
	})
	assert.NoError(t, err)
	// 3. re-register should be error
	err = peer1.RegisterStreamMessageHandler(customMessageProtocol, func(_ network.Stream) {})
	assert.ErrorContains(t, err, "stream with this protocol is already registered")

	// 4. send message from peer2 and wait to get the response from peer1
	err = peer2.SendMessage(context.TODO(), peer1.Host.ID().String(),
		types.MessageEnvelope{Type: customMessageProtocol, Data: []byte(helloWorlPayload)},
		time.Now().Add(readTimeout))
	assert.NoError(t, err)
	peer1MessageContent := <-payloadReceived
	assert.Equal(t, helloWorlPayload, peer1MessageContent)

	// 5. open stream functionality
	streamPayloadMessage := "nunet world"
	streamPayloadReceived := make(chan string)
	err = peer1.RegisterStreamMessageHandler(types.MessageType("/custom_stream/1.2.3"),
		func(stream network.Stream) {
			bytesToRead := make([]byte, len([]byte(streamPayloadMessage)))
			_, err := stream.Read(bytesToRead)
			assert.NoError(t, err)
			streamPayloadReceived <- string(bytesToRead)
		})
	assert.NoError(t, err)
	openedStream, err := peer2.OpenStream(context.TODO(), peer1p2pAddrs[0].String(),
		types.MessageType("/custom_stream/1.2.3"))
	assert.NoError(t, err)
	_, err = openedStream.Write([]byte("nunet world"))
	assert.NoError(t, err)
	peer1MessageContent = <-streamPayloadReceived
	openedStream.Close()
	assert.Equal(t, "nunet world", peer1MessageContent)

	// 6. register message with bytes handler function
	secondPayloadReceived := make(chan string)
	err = peer1.RegisterBytesMessageHandler(
		types.MessageType("/custom_bytes_callback/1.1.4"), func(dt []byte, _ peer.ID) {
			secondPayloadReceived <- string(dt)
		})
	assert.NoError(t, err)
	err = peer2.SendMessage(context.TODO(),
		peer1.Host.ID().String(),
		types.MessageEnvelope{
			Type: types.MessageType("/custom_bytes_callback/1.1.4"),
			Data: []byte(helloWorlPayload),
		},
		time.Now().Add(readTimeout))
	assert.NoError(t, err)
	messageFromBytesHandler := <-secondPayloadReceived
	assert.Equal(t, helloWorlPayload, messageFromBytesHandler)
}

func TestStop(t *testing.T) {
	peer1, _, _ := createPeers(t, 65527, 65528, 65529)

	// Stop the peer and check for errors
	err := peer1.Stop()
	require.NoError(t, err)

	// Ensure that the DHT and Host are closed
	require.NoError(t, peer1.DHT.Close())
	require.NoError(t, peer1.Host.Close())
}

func TestStat(t *testing.T) {
	peer1, _, _ := createPeers(t, 65530, 65531, 65532)

	// Get the network stats
	stats := peer1.Stat()

	// Check if the ID is correct
	require.Equal(t, peer1.Host.ID().String(), stats.ID)

	// Check if the ListenAddr is correct
	expectedAddrs := make([]string, 0, len(peer1.Host.Addrs()))
	for _, addr := range peer1.Host.Addrs() {
		expectedAddrs = append(expectedAddrs, addr.String())
	}
	expectedListenAddr := strings.Join(expectedAddrs, ", ")
	require.Equal(t, expectedListenAddr, stats.ListenAddr)
}

func TestSetupBroadcastTopic(t *testing.T) {
	peer1, _, _ := createPeers(t, 65533, 65534, 65535)

	// Test case: topic not subscribed
	err := peer1.SetupBroadcastTopic("nonexistent_topic", func(_ *Topic) error {
		return nil
	})
	require.ErrorContains(t, err, "not subscribed to nonexistent_topic")

	// Test case: topic subscribed and setup function succeeds
	topicName := "test_topic"
	topicHandler, err := peer1.getOrJoinTopicHandler(topicName)
	require.NoError(t, err)
	peer1.pubsubTopics[topicName] = topicHandler

	err = peer1.SetupBroadcastTopic(topicName, func(_ *Topic) error {
		// Simulate setup logic
		return nil
	})
	require.NoError(t, err)

	// Test case: topic subscribed and setup function fails
	err = peer1.SetupBroadcastTopic(topicName, func(_ *Topic) error {
		// Simulate setup logic failure
		return fmt.Errorf("setup failed")
	})
	require.ErrorContains(t, err, "setup failed")
}

func TestKnownPeers(t *testing.T) {
	peer1, peer2, peer3 := createPeers(t, 65200, 65201, 65202)

	// Add peers to the peerstore
	peer1ID := peer1.Host.ID()
	peer2ID := peer2.Host.ID()
	peer3ID := peer3.Host.ID()

	peer1.Host.Peerstore().AddAddrs(peer2ID, peer2.Host.Addrs(), peerstore.PermanentAddrTTL)
	peer1.Host.Peerstore().AddAddrs(peer3ID, peer3.Host.Addrs(), peerstore.PermanentAddrTTL)

	// Test KnownPeers method
	peers, err := peer1.KnownPeers()
	require.NoError(t, err)
	require.Len(t, peers, 3)

	expectedPeers := []peer.ID{peer1ID, peer2ID, peer3ID}
	for _, p := range peers {
		require.Contains(t, expectedPeers, p.ID)
	}
}

func TestVisiblePeers(t *testing.T) {
	peer1, peer2, peer3 := createPeers(t, 65203, 65204, 65205)

	// Add discovered peers to peer1
	peer1.discoveredPeers = []peer.AddrInfo{
		{ID: peer2.Host.ID(), Addrs: peer2.Host.Addrs()},
		{ID: peer3.Host.ID(), Addrs: peer3.Host.Addrs()},
	}

	// Test VisiblePeers method
	visiblePeers := peer1.VisiblePeers()
	require.Len(t, visiblePeers, 2)

	expectedPeers := []peer.ID{peer2.Host.ID(), peer3.Host.ID()}
	for _, p := range visiblePeers {
		require.Contains(t, expectedPeers, p.ID)
	}
}

func TestGetBroadcastScore(t *testing.T) {
	peer1, _, _ := createPeers(t, 65136, 65137, 65138)

	// Initialize some dummy scores
	dummyScores := map[peer.ID]*PeerScoreSnapshot{
		peer1.Host.ID(): {
			Score: 10.0,
		},
	}

	// Set the dummy scores
	peer1.broadcastScoreInspect(dummyScores)

	// Retrieve the broadcast scores
	scores := peer1.GetBroadcastScore()

	// Check if the scores match the dummy scores
	require.Equal(t, dummyScores, scores)
	require.Equal(t, 10.0, scores[peer1.Host.ID()].Score)
}

func TestBroadcastScoreInspect(t *testing.T) {
	peer1, _, _ := createPeers(t, 65139, 65140, 65141)

	// Initialize some dummy scores
	dummyScores := map[peer.ID]*PeerScoreSnapshot{
		peer1.Host.ID(): {
			Score: 10.0,
		},
	}

	// Call broadcastScoreInspect with the dummy scores
	peer1.broadcastScoreInspect(dummyScores)

	// Retrieve the broadcast scores
	scores := peer1.GetBroadcastScore()

	// Check if the scores match the dummy scores
	require.Equal(t, dummyScores, scores)
	require.Equal(t, 10.0, scores[peer1.Host.ID()].Score)
}

func createPeers(t *testing.T, port1, port2, port3 int) (*Libp2p, *Libp2p, *Libp2p) {
	// setup peer1
	cfg := &config.Config{}

	peer1Config := setupPeerConfig(t, port1, []multiaddr.Multiaddr{})
	peer1, err := New(peer1Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer1)
	err = peer1.Init(cfg)
	assert.NoError(t, err)
	err = peer1.Start()
	assert.NoError(t, err)

	// peers of peer1 should be one (itself)
	assert.Equal(t, peer1.Host.Peerstore().Peers().Len(), 1)

	// setup peer2 to connect to peer 1
	peer1p2pAddrs, err := peer1.GetMultiaddr()
	assert.NoError(t, err)
	peer2Config := setupPeerConfig(t, port2, peer1p2pAddrs)
	peer2, err := New(peer2Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer2)

	err = peer2.Init(cfg)
	assert.NoError(t, err)
	err = peer2.Start()
	assert.NoError(t, err)

	// setup a new peer and advertise specs
	// peer3 will connect to peer2 in a ring setup.
	peer2p2pAddrs, err := peer2.GetMultiaddr()
	assert.NoError(t, err)
	peer3Config := setupPeerConfig(t, port3, peer2p2pAddrs)
	peer3, err := New(peer3Config, afero.NewMemMapFs())
	assert.NoError(t, err)
	assert.NotNil(t, peer3)

	err = peer3.Init(cfg)
	assert.NoError(t, err)
	err = peer3.Start()
	assert.NoError(t, err)

	// ensure that the peers are connected
	err = retry.Do(
		func() error {
			if peer1.Host.Network().Connectedness(peer2.Host.ID()) != network.Connected {
				return fmt.Errorf("peer1 is not connected to peer2")
			}

			if peer2.Host.Network().Connectedness(peer3.Host.ID()) != network.Connected {
				return fmt.Errorf("peer2 is not connected to peer3")
			}
			return nil
		},
		retry.Attempts(5),
		retry.Delay(200*time.Millisecond),
	)
	require.NoErrorf(t, err, "could not connect peers")

	return peer1, peer2, peer3
}

func setupPeerConfig(t *testing.T, libp2pPort int, bootstrapPeers []multiaddr.Multiaddr) *types.Libp2pConfig {
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	return &types.Libp2pConfig{
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-randevouz",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10),
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           []string{fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1/", libp2pPort)},
		PeerCountDiscoveryLimit: 40,
	}
}
