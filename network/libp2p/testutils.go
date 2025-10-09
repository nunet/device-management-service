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

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

func createPeer(t *testing.T, port, quicPort int, bootstrapPeers []multiaddr.Multiaddr, factory ...NetInterfaceFactory) *Libp2p { //nolint
	peerConfig := setupPeerConfig(t, port, quicPort, bootstrapPeers)

	if len(factory) > 0 && factory[0] != nil {
		peerConfig.NetIfaceFactory = func(name string) (interface{}, error) {
			return factory[0](name)
		}
	}

	peer1, err := New(peerConfig, afero.NewMemMapFs())

	require.NoError(t, err)

	// No need to set NetIfaceFactory after construction

	require.NoError(t, peer1.Init(&config.Config{}))

	// Add test cleanup to ensure proper shutdown
	t.Cleanup(func() {
		if peer1 != nil {
			err := peer1.Stop()
			if err != nil {
				t.Logf("Error stopping peer: %v", err)
			}
			time.Sleep(100 * time.Millisecond) // Give time for resources to be released
		}
	})

	return peer1
}

func setupPeerConfig(t *testing.T, libp2pPort, quicPort int, bootstrapPeers []multiaddr.Multiaddr) *types.Libp2pConfig {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	assert.NoError(t, err)

	// Use TCP addresses only - the QUIC address will be added automatically in host.go
	// Using only the TCP transport avoids QUIC collisions
	listenAddresses := []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", libp2pPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", quicPort),
	}

	return &types.Libp2pConfig{
		Env:                     "test",
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-randevouz",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10, time.Second),
		DHTPrefix:               "/nunet",
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           listenAddresses,
		PeerCountDiscoveryLimit: 40,
	}
}
