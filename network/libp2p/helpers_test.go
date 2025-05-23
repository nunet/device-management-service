// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/network/utils"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: handle concurrency (maybe use TestSuite) or simply don't use t.Parallel from callers
func newNetwork(
	t *testing.T, numHosts int,
	withDiscovery bool,
) []*Libp2p {
	t.Helper()

	hosts := make([]*Libp2p, 0, numHosts)

	require.True(t, numHosts > 0)
	require.True(t, numHosts <= 10, "do not explode your cpu")

	ports, err := utils.GetMultipleAvailablePorts(numHosts)
	require.NoError(t, err)
	require.Len(t, ports, numHosts)

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

	for i := 0; i < bootstrapPeersNum; i++ {
		bootstrapCfg := setupPeerConfig(t, ports[i], bootstrapPeers)
		host := newPeer(t, bootstrapCfg)
		require.NotNil(t, host)

		addrs, err := host.GetMultiaddr()
		require.NoError(t, err)

		bootstrapPeers = append(bootstrapPeers, addrs...)
		hosts = append(hosts, host)
	}

	for _, port := range ports[bootstrapPeersNum:] {
		peerCfg := setupPeerConfig(t, port, bootstrapPeers)
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
					// TODO: go routine?
					peerInfo := otherHost.Host.Peerstore().PeerInfo(
						otherHost.Host.ID())
					require.NoError(t, err)

					err = host.Host.Connect(host.ctx, peerInfo)
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

func setupPeerConfig(t *testing.T, libp2pPort int, bootstrapPeers []multiaddr.Multiaddr) *types.Libp2pConfig {
	t.Helper()
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	return &types.Libp2pConfig{
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-randevouz",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10, time.Second),
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           []string{fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1/", libp2pPort)},
		PeerCountDiscoveryLimit: 40,
	}
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
