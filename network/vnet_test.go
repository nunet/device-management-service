package network_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	msgType   = "msg"
	topicName = "topic"
	pubMsg1   = "msg1"
	pubMsg2   = "msg2"
	advKey    = "key"
	advData   = "data"

	// IP addresses for subnet tests
	aliceIP   = "10.0.0.1"
	bobIP     = "10.0.0.2"
	charlieIP = "10.0.0.3"
	daveIP    = "10.0.0.4"
)

// createEntity creates a new PeerID based on a generated public/private key pair
func createEntity(t *testing.T) peer.ID {
	t.Helper()
	_, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	peerID, err := peer.IDFromPublicKey(pubKey)
	require.NoError(t, err)
	return peerID
}

// TestSendMessage Alice sends message to Bob
func TestSendMessage(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	bobID := createEntity(t)
	alice := substrate.AddWiredPeer(aliceID)
	bob := substrate.AddWiredPeer(bobID)

	var wg sync.WaitGroup
	wg.Add(1)
	msgContent := "hello alice"
	err := bob.HandleMessage(msgType, func(data []byte, from peer.ID) {
		defer wg.Done()
		assert.Equal(t, msgContent, string(data))
		assert.Equal(t, aliceID, from)
	})
	require.NoError(t, err)

	// note: SendMessageSync == SendMessage on the current implementation
	err = alice.SendMessageSync(
		context.Background(), bobID.String(),
		types.MessageEnvelope{Type: msgType, Data: []byte(msgContent)}, time.Time{})
	require.NoError(t, err)
	wg.Wait()

	// now unregister handler and send message again
	bob.UnregisterMessageHandler(msgType)
	err = alice.SendMessage(
		context.Background(), bobID.String(),
		types.MessageEnvelope{Type: msgType, Data: []byte(msgContent)}, time.Time{})
	require.Error(t, err)
}

func TestPublishSubscribe(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	bobID := createEntity(t)
	alice := substrate.AddWiredPeer(aliceID)
	bob := substrate.AddWiredPeer(bobID)

	var mu sync.Mutex
	var received []string

	// Alice subscribes to the topic, and Bob publishes to it
	id, err := alice.Subscribe(context.Background(), topicName, func(data []byte) {
		mu.Lock()
		received = append(received, string(data))
		mu.Unlock()
	}, nil)
	require.NoError(t, err)

	err = bob.Publish(context.Background(), topicName, []byte(pubMsg1))
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Contains(t, received, pubMsg1)
	mu.Unlock()

	// now Alice unsubscribes and she shouldn't receive more msgs from the topic
	require.NoError(t, alice.Unsubscribe(topicName, id))

	err = bob.Publish(context.Background(), topicName, []byte(pubMsg2))
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.NotContains(t, received, pubMsg2)
	mu.Unlock()
}

func TestBasicNetworkMethods(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	bobID := createEntity(t)
	carolID := createEntity(t)
	daveID := createEntity(t) // no one connects to dave
	alice := substrate.AddWiredPeer(aliceID)
	_ = substrate.AddWiredPeer(bobID)

	t.Run("ping", func(t *testing.T) {
		t.Parallel()
		result, err := alice.Ping(context.Background(), bobID.String(), time.Second)
		assert.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("peers", func(t *testing.T) {
		t.Parallel()
		peers := alice.Peers()
		assert.NotNil(t, peers)
		assert.Contains(t, peers, bobID)
		assert.NotContains(t, peers, daveID)
	})

	t.Run("peer connection", func(t *testing.T) {
		t.Parallel()
		assert.True(t, alice.PeerConnected(bobID))
		randomPeerID := createEntity(t)
		assert.False(t, alice.PeerConnected(randomPeerID))
	})

	t.Run("peer connection with Connect", func(t *testing.T) {
		t.Parallel()
		assert.False(t, alice.PeerConnected(carolID))
		err := alice.Connect(context.Background(), fmt.Sprintf("invalid-%s", carolID.String()))
		assert.Error(t, err)
		err = alice.Connect(context.Background(), carolID.String())
		assert.NoError(t, err)
		assert.True(t, alice.PeerConnected(carolID))
	})
}

func TestAdvertiseQuery(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	alice := substrate.AddWiredPeer(aliceID)

	t.Run("advertise/unadvertise and query", func(t *testing.T) {
		t.Parallel()
		// advertise and success query
		require.NoError(t, alice.Advertise(context.Background(), advKey, []byte(advData)))
		adverts, err := alice.Query(context.Background(), advKey)
		require.NoError(t, err)
		assert.Len(t, adverts, 1)
		assert.Equal(t, aliceID.String(), adverts[0].PeerId)
		assert.Equal(t, []byte(advData), adverts[0].Data)

		// unadvertise and failure query
		require.NoError(t, alice.Unadvertise(context.Background(), advKey))
		adverts, err = alice.Query(context.Background(), advKey)
		require.NoError(t, err)
		assert.Empty(t, adverts)
	})

	t.Run("query non-existent key", func(t *testing.T) {
		t.Parallel()
		adverts, err := alice.Query(context.Background(), "nokey")
		require.NoError(t, err)
		assert.Empty(t, adverts)
	})
}

func TestNewMemoryNetHost(t *testing.T) {
	t.Parallel()

	// Create a single host network
	host, err := network.NewMemoryNetHost()
	require.NoError(t, err)
	require.NotNil(t, host)

	// Verify the host has a valid ID
	hostID := host.GetHostID()
	require.NotEmpty(t, hostID)

	// Test sending message to self
	msgContent := "hello self"
	var wg sync.WaitGroup
	wg.Add(1)

	err = host.HandleMessage(msgType, func(data []byte, from peer.ID) {
		defer wg.Done()
		assert.Equal(t, msgContent, string(data))
		assert.Equal(t, hostID, from)
	})
	require.NoError(t, err)

	err = host.SendMessage(
		context.Background(),
		hostID.String(),
		types.MessageEnvelope{Type: msgType, Data: []byte(msgContent)},
		time.Time{})
	require.NoError(t, err)
	wg.Wait()

	// Test basic operations work
	require.NoError(t, host.Init(nil))
	require.NoError(t, host.Start())
	require.NoError(t, host.Stop())
}

func TestConcurrency(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	bobID := createEntity(t)
	charlieID := createEntity(t)
	alice := substrate.AddWiredPeer(aliceID)
	bob := substrate.AddWiredPeer(bobID)
	charlie := substrate.AddWiredPeer(charlieID)

	// register handlers
	err := bob.HandleMessage(msgType, func([]byte, peer.ID) {})
	require.NoError(t, err)
	err = charlie.HandleMessage(msgType, func([]byte, peer.ID) {})
	require.NoError(t, err)

	// Subscribe to topics
	_, err = alice.Subscribe(context.Background(), topicName, func([]byte) {}, nil)
	require.NoError(t, err)
	_, err = bob.Subscribe(context.Background(), topicName, func([]byte) {}, nil)
	require.NoError(t, err)
	_, err = charlie.Subscribe(context.Background(), topicName, func([]byte) {}, nil)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent Ping operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			targetID := bobID.String()
			if idx%2 == 0 {
				targetID = charlieID.String()
			}
			_, _ = alice.Ping(context.Background(), targetID, time.Second)
		}(i)
	}

	// Concurrent Publish operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := []byte("test-msg-" + string(rune('0'+idx%10)))
			_ = alice.Publish(context.Background(), topicName, msg)
			_ = bob.Publish(context.Background(), topicName, msg)
		}(i)
	}

	// Concurrent SendMessage operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := types.MessageEnvelope{
				Type: msgType,
				Data: []byte("msg-" + string(rune('0'+idx%10))),
			}
			_ = alice.SendMessage(context.Background(), bobID.String(), msg, time.Time{})
			_ = alice.SendMessage(context.Background(), charlieID.String(), msg, time.Time{})
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

func TestMiscNetworkMethods(t *testing.T) {
	t.Parallel()
	substrate := network.NewSubstrate()
	aliceID := createEntity(t)
	bobID := createEntity(t)
	alice := substrate.AddWiredPeer(aliceID)
	_ = substrate.AddWiredPeer(bobID)

	t.Run("init and lifecycle", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, alice.Init(nil))
		assert.NoError(t, alice.Start())
		assert.NoError(t, alice.Stop())
	})

	t.Run("address resolution", func(t *testing.T) {
		t.Parallel()
		addrs, err := alice.ResolveAddress(context.Background(), "example.com")
		assert.NoError(t, err)
		assert.Nil(t, addrs)
	})

	t.Run("broadcast", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, alice.SetupBroadcastTopic("test-topic", func(_ *network.Topic) error {
			return nil
		}))

		alice.SetBroadcastAppScore(func(_ network.PeerID) float64 {
			return 1.0
		})

		scores := alice.GetBroadcastScore()
		assert.NotNil(t, scores)
	})

	t.Run("host public ip", func(t *testing.T) {
		t.Parallel()
		ip, err := alice.HostPublicIP()
		assert.NoError(t, err)
		assert.Nil(t, ip)
	})

	t.Run("notification", func(t *testing.T) {
		t.Parallel()
		err := alice.Notify(
			context.Background(),
			func(_ network.PeerID, _ []network.ProtocolID, _ int) {},
			func(_ network.PeerID) {},
			func(_ network.PeerID) {},
			func(_ network.PeerID, _ []network.ProtocolID) {},
			func(_ network.PeerID, _ []network.ProtocolID) {},
		)
		assert.NoError(t, err)
	})

	t.Run("subnets", func(t *testing.T) {
		t.Parallel()
		const (
			subnetID = "test-subnet"
			srcPort  = "8080"
			dstPort  = "80"
		)

		// Create subnet
		charlieID := createEntity(t)
		daveID := createEntity(t)
		rt := map[string]string{aliceID.String(): aliceIP, bobID.String(): bobIP}
		assert.NoError(t, alice.CreateSubnet(context.Background(), subnetID, rt))

		// Destroy subnet
		assert.NoError(t, alice.DestroySubnet(subnetID))

		// Add/remove peers
		assert.NoError(t, alice.AddSubnetPeer(subnetID, charlieID.String(), charlieIP))
		assert.NoError(t, alice.RemoveSubnetPeer(subnetID, charlieID.String(), charlieIP))
		assert.NoError(t, alice.AcceptSubnetPeer(subnetID, daveID.String(), daveIP))

		// Port mapping
		assert.NoError(t, alice.MapPort(subnetID, "tcp", aliceIP, srcPort, bobIP, dstPort))
		assert.NoError(t, alice.UnmapPort(subnetID, "tcp", aliceIP, srcPort, "", ""))

		// DNS records
		dnsRecords := map[string]string{"alice.local": aliceIP, "bob.local": bobIP}
		assert.NoError(t, alice.AddSubnetDNSRecords(subnetID, dnsRecords))
		assert.NoError(t, alice.RemoveSubnetDNSRecord(subnetID, "alice.local"))
	})
}
