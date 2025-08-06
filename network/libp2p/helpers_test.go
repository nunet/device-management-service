// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: handle concurrency (maybe use TestSuite) or simply don't use t.Parallel from callers
func newNetwork(
	t *testing.T, numHosts int,
	withDiscovery bool,
	env string,
) []*Libp2p {
	t.Helper()

	hosts := make([]*Libp2p, 0, numHosts)

	require.True(t, numHosts > 0)
	require.True(t, numHosts <= 10, "do not explode your cpu")

	// Calculate bootstrap peers based on network size
	var bootstrapPeersNum int
	switch {
	case numHosts <= 3:
		bootstrapPeersNum = 1
	case numHosts <= 10:
		bootstrapPeersNum = 2
	default:
		bootstrapPeersNum = 3
	}

	// each peer may have multiple multiaddresses
	bootstrapPeers := make(
		[]multiaddr.Multiaddr, 0, bootstrapPeersNum*5)

	for range bootstrapPeersNum {
		bootstrapCfg := setupPeerConfig(t, 0, 0, bootstrapPeers)
		bootstrapCfg.Env = env
		host := newPeer(t, bootstrapCfg)
		require.NotNil(t, host)

		addrs, err := host.GetMultiaddr()
		require.NoError(t, err)

		bootstrapPeers = append(bootstrapPeers, addrs...)
		hosts = append(hosts, host)
	}

	for i := bootstrapPeersNum; i < numHosts; i++ {
		peerCfg := setupPeerConfig(t, 0, 0, bootstrapPeers)
		peerCfg.Env = env
		host := newPeer(t, peerCfg)
		require.NotNil(t, host)
		hosts = append(hosts, host)
	}

	require.Len(t, hosts, numHosts)

	if !withDiscovery {
		// Ensure peers are connected to each other explicitly
		for i, host := range hosts {
			for j, otherHost := range hosts {
				if i != j {
					peerInfo := otherHost.Host.Peerstore().PeerInfo(
						otherHost.Host.ID())

					err := host.Host.Connect(host.ctx, peerInfo)
					require.NoError(t, err)
				}
			}
		}

		// Without discovery, we require all hosts to be connected to all other hosts
		for _, host := range hosts {
			verifyConnections(t, host, numHosts-1)
		}
	}

	if withDiscovery {
		// For discovery mode, we only require each host to be connected to some peers
		minConnections := numHosts / 3
		if minConnections < 1 && numHosts > 1 {
			minConnections = 1
		}
		for _, host := range hosts {
			verifyConnections(t, host, minConnections)
		}
	}

	t.Cleanup(func() {
		for _, host := range hosts {
			err := host.Stop()
			assert.NoErrorf(t, err, "failed to stop peer %d", host.Host.ID())
		}
	})

	return hosts
}

func newPeer(t *testing.T, hostCfg *types.Libp2pConfig) *Libp2p {
	t.Helper()
	cfg := &config.Config{}

	peer, err := New(hostCfg, afero.NewMemMapFs())
	require.NoError(t, err)
	require.NotNil(t, peer)

	err = peer.Init(cfg)
	require.NoError(t, err)

	err = peer.Start()
	require.NoError(t, err)

	return peer
}

// verifyConnections checks that a host has the expected number of connections.
// The host must be connected to at least minConnections other hosts.
func verifyConnections(t *testing.T, host *Libp2p, minConnections int) {
	t.Helper()

	if minConnections < 1 {
		return
	}

	timeout := 5 * time.Second
	retryInterval := 100 * time.Millisecond

	require.Eventually(t, func() bool {
		connectedPeers := host.Host.Network().Peers()
		return len(connectedPeers) >= minConnections
	}, timeout, retryInterval,
		"A host has only %d connections but needs at least %d",
		len(host.Host.Network().Peers()), minConnections)
}
