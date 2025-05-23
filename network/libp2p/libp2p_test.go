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

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/observability"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
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
				Scheduler:               backgroundtasks.NewScheduler(1, time.Second),
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
	hosts := newNetwork(t, 2, false)
	require.Len(t, hosts, 2)
	alice := hosts[0]
	bob := hosts[1]

	pingResult, err := alice.Ping(context.TODO(), bob.Host.ID().String(), 500*time.Millisecond)
	require.NoError(t, err)
	require.True(t, pingResult.Success)
	zeroMicro, err := time.ParseDuration("0µs")
	assert.NoError(t, err)
	assert.Greater(t, pingResult.RTT, zeroMicro)

	addresses, err := alice.ResolveAddress(context.Background(), bob.Host.ID().String())
	require.NoError(t, err)
	require.Greater(t, len(addresses), 0)
	assert.Contains(t, addresses[0], bob.Host.ID().String())

	ip := alice.GetPeerIP(bob.Host.ID())
	assert.NotEmpty(t, ip)
}

func TestAdvertiseUnadvertiseQuery(t *testing.T) {
	hosts := newNetwork(t, 2, false)
	require.Len(t, hosts, 2)
	alice := hosts[0]
	bob := hosts[1]

	const (
		whoAmIKey      = "who_am_i"
		aliceValueJSON = `{"peer":"alice"}`
		bobValueJSON   = `{"peer":"bob"}`
		carolValueJSON = `{"peer":"carol"}`
	)

	require.Eventually(t, func() bool {
		err := alice.Advertise(context.TODO(), whoAmIKey, []byte(aliceValueJSON))
		return err == nil
	}, 5*time.Second, 500*time.Millisecond, "failed to advertise")

	require.Eventually(t, func() bool {
		err := bob.Advertise(context.TODO(), whoAmIKey, []byte(bobValueJSON))
		return err == nil
	}, 5*time.Second, 500*time.Millisecond, "failed to advertise")

	advertisements := make([]*commonproto.Advertisement, 0, 2)
	require.Eventually(t, func() bool {
		advertisements, err := alice.Query(context.TODO(), whoAmIKey)
		assert.NoError(t, err)

		return len(advertisements) == 2
	}, 5*time.Second, 500*time.Millisecond, "Failed to find all 2 advertisements within timeout")

	// check if all peers have returned the correct data
	for _, v := range advertisements {
		switch v.PeerId {
		case alice.Host.ID().String():
			{
				assert.Equal(t, []byte(aliceValueJSON), v.Data)
			}
		case bob.Host.ID().String():
			{
				assert.Equal(t, []byte(bobValueJSON), v.Data)
			}
		}
	}

	// test unadvertise
	err := bob.Unadvertise(context.TODO(), whoAmIKey)
	assert.NoError(t, err)

	// get the values again, it should be 1 now since one was unadvertised
	require.Eventually(t, func() bool {
		advertisements, err = alice.Query(context.TODO(), whoAmIKey)
		assert.NoError(t, err)

		return len(advertisements) == 1
	}, 5*time.Second, 500*time.Millisecond, "Failed to confirm peer unadvertised within timeout")
}

func TestPublishSubscribeUnsubscribe(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]
	// test publish/subscribe
	// in this test gossipsub messages are not always delivered from peer1 to other peers
	// and this makes the test flaky. To solve the flakiness we use the peer one for
	// both publishing and subscribing.
	subscribeData := make(chan []byte)
	subID, err := alice.Subscribe(context.TODO(), "blocks", func(data []byte) {
		subscribeData <- data
	}, nil)
	assert.NoError(t, err)

	// calling one time publish doesnt guarantee that the message has been sent.
	// without this we might not get a message in the subscribe and we would get
	// flaky test which will fail
	go func() {
		for {
			err = alice.Publish(context.TODO(), "blocks", []byte(`{"block":"1"}`))
		}
	}()

	receivedData := <-subscribeData
	assert.EqualValues(t, []byte(`{"block":"1"}`), receivedData)

	// unsubscribe
	err = alice.Unsubscribe("blocks", subID)
	assert.NoError(t, err)
}

// if we connect to ourselves we deliver the message properlly to the right handler.
func TestSelfDial(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	messageType := types.MessageType("/custom_bytes_callback/1.1.4")
	messageChannel := make(chan string)
	const testData = "testing 123"

	err := alice.RegisterBytesMessageHandler(messageType, func(dt []byte, _ peer.ID) {
		messageChannel <- string(dt)
	})
	require.NoError(t, err)
	err = alice.SendMessage(context.Background(), alice.Host.ID().String(),
		types.MessageEnvelope{
			Type: messageType,
			Data: []byte(testData),
		},
		time.Now().Add(readTimeout),
	)
	require.NoError(t, err)

	// check if we received the data properlly
	received := <-messageChannel
	require.Equal(t, testData, received)
}

func TestSendMessageAndHandlers(t *testing.T) {
	hosts := newNetwork(t, 2, true)
	require.Len(t, hosts, 2)
	alice := hosts[0]
	bob := hosts[1]
	aliceP2pAddrs, err := alice.GetMultiaddr()
	assert.NoError(t, err)

	// Protocol constants
	const customMessageProtocol = types.MessageType("/chat/1.1.1")
	const streamProtocol = types.MessageType("/custom_stream/1.2.3")
	const bytesCallbackProtocol = types.MessageType("/custom_bytes_callback/1.1.4")

	// test sendmessage and stream functionality
	// use different test cases to check if communication can be established to remote peers
	// 1. peer1 is not handling any message types so we try to send a message and get an error
	// The semantics of send have changed; this test is not applicable

	// Using the constant defined above
	helloWorlPayload := "hello world"

	// 2. alice registers the message
	payloadReceived := make(chan string)
	err = alice.RegisterStreamMessageHandler(customMessageProtocol, func(stream network.Stream) {
		bytesToRead := make([]byte, len([]byte(helloWorlPayload))+8)
		_, err = stream.Read(bytesToRead)
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, helloWorlPayload, string(bytesToRead[8:]))
		payloadReceived <- string(bytesToRead[8:])
	})
	require.NoError(t, err)

	// 3. re-register should be error
	err = alice.RegisterStreamMessageHandler(customMessageProtocol, func(_ network.Stream) {})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStreamRegistered)

	// 4. send message from bob and wait to get the response from alice
	err = bob.SendMessage(context.TODO(), alice.Host.ID().String(),
		types.MessageEnvelope{Type: customMessageProtocol, Data: []byte(helloWorlPayload)},
		time.Now().Add(readTimeout))
	require.NoError(t, err)
	aliceMessageContent := <-payloadReceived
	assert.Equal(t, helloWorlPayload, aliceMessageContent)

	// 5. open stream functionality
	streamPayloadMessage := "nunet world"
	streamPayloadReceived := make(chan string)
	err = alice.RegisterStreamMessageHandler(streamProtocol,
		func(stream network.Stream) {
			bytesToRead := make([]byte, len([]byte(streamPayloadMessage)))
			_, err := stream.Read(bytesToRead)
			assert.NoError(t, err)
			streamPayloadReceived <- string(bytesToRead)
		})
	require.NoError(t, err)
	openedStream, err := bob.OpenStream(context.TODO(), aliceP2pAddrs[0].String(),
		streamProtocol)
	require.NoError(t, err)
	_, err = openedStream.Write([]byte("nunet world"))
	require.NoError(t, err)
	aliceMessageContent = <-streamPayloadReceived
	openedStream.Close()
	assert.Equal(t, "nunet world", aliceMessageContent)

	// 6. register message with bytes handler function
	secondPayloadReceived := make(chan string)
	err = alice.RegisterBytesMessageHandler(
		bytesCallbackProtocol, func(dt []byte, _ peer.ID) {
			secondPayloadReceived <- string(dt)
		})
	require.NoError(t, err)
	err = bob.SendMessage(context.TODO(),
		alice.Host.ID().String(),
		types.MessageEnvelope{
			Type: bytesCallbackProtocol,
			Data: []byte(helloWorlPayload),
		},
		time.Now().Add(readTimeout))
	assert.NoError(t, err)
	messageFromBytesHandler := <-secondPayloadReceived
	assert.Equal(t, helloWorlPayload, messageFromBytesHandler)
}

func TestStop(t *testing.T) {
	hosts := newNetwork(t, 1, false)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	// Stop the peer and check for errors
	err := alice.Stop()
	require.NoError(t, err)

	// Ensure that the DHT and Host are closed
	require.NoError(t, alice.DHT.Close())
	require.NoError(t, alice.Host.Close())
}

func TestStat(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	// Get the network stats
	stats := alice.Stat()

	// Check if the ID is correct
	require.Equal(t, alice.Host.ID().String(), stats.ID)

	// Check if the ListenAddr is correct
	expectedAddrs := make([]string, 0, len(alice.Host.Addrs()))
	for _, addr := range alice.Host.Addrs() {
		expectedAddrs = append(expectedAddrs, addr.String())
	}

	expectedListenAddr := strings.Join(expectedAddrs, ", ")
	require.Equal(t, expectedListenAddr, stats.ListenAddr)
}

func TestSetupBroadcastTopic(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	// Test case: topic not subscribed
	err := alice.SetupBroadcastTopic("nonexistent_topic", func(_ *Topic) error {
		return nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSubscribed)

	// Test case: topic subscribed and setup function succeeds
	topicName := "test_topic"
	topicHandler, err := alice.getOrJoinTopicHandler(topicName)
	require.NoError(t, err)
	alice.pubsubTopics[topicName] = topicHandler

	err = alice.SetupBroadcastTopic(topicName, func(_ *Topic) error {
		// Simulate setup logic
		return nil
	})
	require.NoError(t, err)

	// Test case: topic subscribed and setup function fails
	errorMsg := "setup failed"
	err = alice.SetupBroadcastTopic(topicName, func(_ *Topic) error {
		// Simulate setup logic failure
		return fmt.Errorf("%s", errorMsg)
	})
	require.ErrorContains(t, err, errorMsg)
}

func TestKnownPeers(t *testing.T) {
	hosts := newNetwork(t, 3, true)
	require.Len(t, hosts, 3)
	alice := hosts[0]
	bob := hosts[1]
	carol := hosts[2]

	// Add peers to the peerstore
	aliceID := alice.Host.ID()
	bobID := bob.Host.ID()
	carolID := carol.Host.ID()

	alice.Host.Peerstore().AddAddrs(bobID, bob.Host.Addrs(), peerstore.PermanentAddrTTL)
	alice.Host.Peerstore().AddAddrs(carolID, carol.Host.Addrs(), peerstore.PermanentAddrTTL)

	// Test KnownPeers method
	peers, err := alice.KnownPeers()
	require.NoError(t, err)
	require.Len(t, peers, 3)

	expectedPeers := []peer.ID{aliceID, bobID, carolID}
	for _, p := range peers {
		require.Contains(t, expectedPeers, p.ID)
	}
}

func TestVisiblePeers(t *testing.T) {
	hosts := newNetwork(t, 3, true)
	require.Len(t, hosts, 3)
	alice := hosts[0]
	bob := hosts[1]
	carol := hosts[2]

	// Add discovered peers to alice
	alice.discoveredPeers = []peer.AddrInfo{
		{ID: bob.Host.ID(), Addrs: bob.Host.Addrs()},
		{ID: carol.Host.ID(), Addrs: carol.Host.Addrs()},
	}

	// Test VisiblePeers method
	visiblePeers := alice.VisiblePeers()
	require.Len(t, visiblePeers, 2)

	expectedPeers := []peer.ID{bob.Host.ID(), carol.Host.ID()}
	for _, p := range visiblePeers {
		require.Contains(t, expectedPeers, p.ID)
	}
}

func TestGetBroadcastScore(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	// Initialize some dummy scores
	dummyScores := map[peer.ID]*PeerScoreSnapshot{
		alice.Host.ID(): {
			Score: 10.0,
		},
	}

	// Set the dummy scores
	alice.broadcastScoreInspect(dummyScores)

	// Retrieve the broadcast scores
	scores := alice.GetBroadcastScore()

	// Check if the scores match the dummy scores
	require.Equal(t, dummyScores, scores)
	require.Equal(t, 10.0, scores[alice.Host.ID()].Score)
}

func TestBroadcastScoreInspect(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	// Initialize some dummy scores
	dummyScores := map[peer.ID]*PeerScoreSnapshot{
		alice.Host.ID(): {
			Score: 10.0,
		},
	}

	// Call broadcastScoreInspect with the dummy scores
	alice.broadcastScoreInspect(dummyScores)

	// Retrieve the broadcast scores
	scores := alice.GetBroadcastScore()

	// Check if the scores match the dummy scores
	require.Equal(t, dummyScores, scores)
	require.Equal(t, 10.0, scores[alice.Host.ID()].Score)
}

// TestHostPublicIP uses internal variables to inject a public IP address
// If decided to make all libp2p tests black boxed, this method shall not be tested.
func TestHostPublicIP(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	publicIP := "203.0.113.1"
	publicMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/1234", publicIP))
	require.NoError(t, err)

	// Inject the observed address to simulate a peer observing our address
	select {
	case alice.observedAddrCh <- publicMultiaddr:
		// Successfully sent the address
	default:
		// Channel full, using alternative method
		alice.mx.Lock()
		alice.observedAddr = publicMultiaddr
		alice.mx.Unlock()
	}

	ip, err := alice.HostPublicIP()
	require.NoError(t, err)
	require.Equal(t, publicIP, ip.String())
}

func TestNotify(t *testing.T) {
	hosts := newNetwork(t, 3, true)
	require.Len(t, hosts, 3)
	alice := hosts[0]
	bob := hosts[1]

	// Create channels to record callback invocations
	preconnectedCh := make(chan struct{}, 10)
	connectedCh := make(chan peer.ID, 10)
	disconnectedCh := make(chan peer.ID, 10)
	identifiedCh := make(chan peer.ID, 10)
	updatedCh := make(chan peer.ID, 10)

	// Setup callback functions
	preconnectedFn := func(_ peer.ID, _ []protocol.ID, connCount int) {
		assert.Greater(t, connCount, 0)
		preconnectedCh <- struct{}{}
	}

	connectedFn := func(p peer.ID) {
		connectedCh <- p
	}

	disconnectedFn := func(p peer.ID) {
		disconnectedCh <- p
	}

	identifiedFn := func(p peer.ID, _ []protocol.ID) {
		identifiedCh <- p
	}

	updatedFn := func(p peer.ID, _ []protocol.ID) {
		updatedCh <- p
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := alice.Notify(ctx, preconnectedFn, connectedFn, disconnectedFn, identifiedFn, updatedFn)
	require.NoError(t, err)

	// Verify preconnected callbacks were made
	// Since our network is already connected, we should receive preconnected callbacks
	require.Eventually(t, func() bool {
		return len(preconnectedCh) > 0
	}, 5*time.Second, 100*time.Millisecond, "No preconnected callbacks received")

	// Disconnect bob from alice
	bobConns := alice.Host.Network().ConnsToPeer(bob.Host.ID())
	require.NotEmpty(t, bobConns)
	for _, conn := range bobConns {
		err := conn.Close()
		require.NoError(t, err)
	}

	// Verify disconnected callback was called
	require.Eventually(t, func() bool {
		select {
		case peerID := <-disconnectedCh:
			return peerID == bob.Host.ID()
		default:
			return false
		}
	}, 5*time.Second, 100*time.Millisecond, "Disconnected callback not called with correct peer ID")

	// Reconnect bob to alice
	err = alice.Host.Connect(ctx, peer.AddrInfo{
		ID:    bob.Host.ID(),
		Addrs: bob.Host.Addrs(),
	})
	require.NoError(t, err)

	// Verify connected callback was called
	require.Eventually(t, func() bool {
		select {
		case peerID := <-connectedCh:
			return peerID == bob.Host.ID()
		default:
			return false
		}
	}, 5*time.Second, 100*time.Millisecond, "Connected callback not called with correct peer ID")

	// Verify identified callback was called
	require.Eventually(t, func() bool {
		select {
		case peerID := <-identifiedCh:
			return peerID == bob.Host.ID()
		default:
			return false
		}
	}, 5*time.Second, 100*time.Millisecond, "Identified callback not called with correct peer ID")
}

func TestSetBroadcastAppScore(t *testing.T) {
	hosts := newNetwork(t, 1, true)
	require.Len(t, hosts, 1)
	alice := hosts[0]

	scoreFn := func(_ peer.ID) float64 { return 20.0 }

	alice.SetBroadcastAppScore(scoreFn)

	// Call the broadcastAppScore method to see if our function is used
	score := alice.broadcastAppScore(alice.Host.ID())
	require.Equal(t, 20.0, score)

	// Test with a different value to ensure it's dynamic
	newScoreFn := func(_ peer.ID) float64 {
		return 30.0
	}
	alice.SetBroadcastAppScore(newScoreFn)
	score = alice.broadcastAppScore(alice.Host.ID())
	require.Equal(t, 30.0, score)
}
