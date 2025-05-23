// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package network

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"

	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewNetwork(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		config *types.NetworkConfig
		expErr string
	}{
		"no config given": {
			expErr: "network configuration is nil",
		},
		"invalid network": {
			config: &types.NetworkConfig{Type: "invalid-type"},
			expErr: "unsupported network type: invalid-type",
		},
		"nats network": {
			config: &types.NetworkConfig{Type: types.NATSNetwork},
			expErr: "not implemented",
		},
		"libp2p network": {
			config: &types.NetworkConfig{
				Type: types.Libp2pNetwork,
				Libp2pConfig: types.Libp2pConfig{
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
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			network, err := NewNetwork(tt.config, afero.NewMemMapFs())
			if tt.expErr != "" {
				assert.Nil(t, network)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, network)
			}
		})
	}
}

func TestLibp2pNetwork(t *testing.T) {
	config := &types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              &crypto.Secp256k1PrivateKey{},
			BootstrapPeers:          []multiaddr.Multiaddr{},
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               backgroundtasks.NewScheduler(1, time.Second),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/localhost/tcp/10219"},
			PeerCountDiscoveryLimit: 40,
		},
	}

	network, err := NewNetwork(config, afero.NewMemMapFs())

	assert.NoError(t, err)
	assert.NotNil(t, network)
	// access libp2p specific methods
	libp2pNet, ok := network.(*libp2p.Libp2p)
	assert.True(t, ok)
	assert.NotNil(t, libp2pNet)
}
