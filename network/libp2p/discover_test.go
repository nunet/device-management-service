package libp2p

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoverDialPeersFullFlow tests the complete peer discovery flow by:
// 1. Creating a network where all peers are initially connected
// 2. Disconnecting peers from each other (but keeping bootstrap connections)
// 3. Having all peers advertise themselves for rendezvous discovery
// 4. Using discoverDialPeers to rediscover and reconnect peers
// This simulates a realistic scenario where peers need to find each other through discovery
// rather than being pre-connected.
//
// Note: it has the chance of becoming flake, if that is the case, remove it
func TestDiscoverDialPeersFullFlow(t *testing.T) {
	hosts := newNetwork(t, 3, true)
	require.Len(t, hosts, 3)

	// hosts[0] is the bootstrap peer (as per newNetwork implementation)
	bootstrap := hosts[0]
	alice := hosts[1]
	bob := hosts[2]

	// Disconnect alice and bob from each other (but keep bootstrap connections)
	// This simulates peers that need to discover each other
	aliceToBobConns := alice.Host.Network().ConnsToPeer(bob.Host.ID())
	for _, conn := range aliceToBobConns {
		err := conn.Close()
		require.NoError(t, err)
	}

	bobToAliceConns := bob.Host.Network().ConnsToPeer(alice.Host.ID())
	for _, conn := range bobToAliceConns {
		err := conn.Close()
		require.NoError(t, err)
	}

	// Verify alice and bob are disconnected from each other
	require.Eventually(t, func() bool {
		return !alice.PeerConnected(bob.Host.ID()) && !bob.PeerConnected(alice.Host.ID())
	}, 2*time.Second, 100*time.Millisecond, "alice and bob should be disconnected")

	// Verify they're still connected to bootstrap
	assert.True(t, alice.PeerConnected(bootstrap.Host.ID()))
	assert.True(t, bob.PeerConnected(bootstrap.Host.ID()))

	// Wait for DHT to stabilize after disconnections
	time.Sleep(2 * time.Second)

	// All peers advertise themselves for rendezvous discovery
	err := bootstrap.advertiseForRendezvousDiscovery(context.Background())
	require.NoError(t, err)
	err = alice.advertiseForRendezvousDiscovery(context.Background())
	require.NoError(t, err)
	err = bob.advertiseForRendezvousDiscovery(context.Background())
	require.NoError(t, err)

	// Wait for advertisements to propagate through the DHT
	time.Sleep(2 * time.Second)

	// Now alice discovers and dials peers
	err = alice.discoverDialPeers(context.Background())
	require.NoError(t, err)

	// Give time for connections to establish
	require.Eventually(t, func() bool {
		return alice.PeerConnected(bob.Host.ID())
	}, 2*time.Second, 100*time.Millisecond, "alice should have reconnected to bob through discovery")
}
