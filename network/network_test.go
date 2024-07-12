package network

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

func TestNewNetwork(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		config *models.NetworkConfig
		expErr string
	}{
		"no config given": {
			expErr: "network configuration is nil",
		},
		"invalid network": {
			config: &models.NetworkConfig{Type: "invalid-type"},
			expErr: "unsupported network type: invalid-type",
		},
		"nats network": {
			config: &models.NetworkConfig{Type: models.NATSNetwork},
			expErr: "not implemented",
		},
		"libp2p network": {
			config: &models.NetworkConfig{
				Type: models.Libp2pNetwork,
				Libp2pConfig: models.Libp2pConfig{
					PrivateKey:              &crypto.Secp256k1PrivateKey{},
					BootstrapPeers:          []multiaddr.Multiaddr{},
					Rendezvous:              "nunet-randevouz",
					Server:                  false,
					Scheduler:               background_tasks.NewScheduler(1),
					CustomNamespace:         "/nunet-dht-1/",
					ListenAddress:           []string{"/ip4/localhost/tcp/10209"},
					PeerCountDiscoveryLimit: 40,
					PrivateNetwork: models.PrivateNetworkConfig{
						WithSwarmKey: false,
					},
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
	config := &models.NetworkConfig{
		Type: models.Libp2pNetwork,
		Libp2pConfig: models.Libp2pConfig{
			PrivateKey:              &crypto.Secp256k1PrivateKey{},
			BootstrapPeers:          []multiaddr.Multiaddr{},
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               background_tasks.NewScheduler(1),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/localhost/tcp/10219"},
			PeerCountDiscoveryLimit: 40,
			PrivateNetwork: models.PrivateNetworkConfig{
				WithSwarmKey: false,
			},
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
