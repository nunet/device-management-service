package libp2p

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/models"
)

func TestCreateTestNetwork(t *testing.T) {
	numNodes := 5
	nodes := createTestNetwork(t, numNodes, false)

	// verify the number of nodes
	assert.Equal(t, 5, len(nodes))

	time.Sleep(1 * time.Second)

	// verify if all nodes have more than 2 connections
	for _, node := range nodes {
		assert.Greater(t, len(node.Host.Network().Peers()), 2)
	}
}

func createTestNetwork(t *testing.T, n int, withSwarmKey bool) []*Libp2p {
	var peers []*Libp2p

	// initiating and configuring a single bootstrap node
	bootstrapConfig := setupPeerConfig(t, 45600, []multiaddr.Multiaddr{}, withSwarmKey)
	bootstrapNode, err := New(bootstrapConfig, afero.NewMemMapFs())
	assert.NoError(t, err)

	err = bootstrapNode.Init(context.TODO())
	assert.NoError(t, err)

	err = bootstrapNode.Start(context.TODO())
	assert.NoError(t, err)

	bootstrapMultiAddr, err := bootstrapNode.GetMultiaddr()
	assert.NoError(t, err)

	peers = append(peers, bootstrapNode)

	var bootstrapNodeBasePath string
	var bootstrapSwarmKey pnet.PSK
	if withSwarmKey {
		// base path where nunet things are stored (we'll use to copy the swarm key
		// to other peers)
		bootstrapNodeBasePath, err = getBasePath(bootstrapNode.fs)
		assert.NoError(t, err)

		// checking if swarm key was generated
		bootstrapSwarmKey, err = getSwarmKey(bootstrapNode.fs)
		assert.NoError(t, err)
		assert.NotNil(t, bootstrapSwarmKey)
	}

	// create the remaining hosts
	for i := 1; i < n; i++ {
		config := setupPeerConfig(t, 45600+i, bootstrapMultiAddr, withSwarmKey)
		p, err := New(config, afero.NewMemMapFs())
		assert.NoError(t, err)

		if withSwarmKey {
			peerBasePath, err := getBasePath(p.fs)
			assert.NoError(t, err)

			// copy swarm key from bootstrap FS to each peer's filesystem
			err = func(srcFs afero.Fs, dstFs afero.Fs, srcPath string, dstPath string) error {
				// Open the source file
				srcFile, err := srcFs.Open(srcPath)
				if err != nil {
					return err
				}
				defer srcFile.Close()

				// Create the destination file
				dstFile, err := dstFs.Create(dstPath)
				if err != nil {
					return err
				}
				defer dstFile.Close()

				// Copy the contents
				_, err = io.Copy(dstFile, srcFile)
				return err
			}(
				bootstrapNode.fs,
				p.fs,
				filepath.Join(bootstrapNodeBasePath, "swarm.key"),
				filepath.Join(peerBasePath, "swarm.key"),
			)
			assert.NoError(t, err)

			// check if swarm key was indeed copied
			key, err := getSwarmKey(p.fs)
			assert.NoError(t, err)
			assert.Equal(t, bootstrapSwarmKey, key)
		}

		err = p.Init(context.TODO())
		assert.NoError(t, err)

		err = p.Start(context.TODO())
		assert.NoError(t, err)

		peers = append(peers, p)
	}

	// close peers after tests are done
	t.Cleanup(func() {
		t.Log("Closing peers")
		for _, p := range peers {
			err := p.Host.Close()
			assert.NoError(t, err)
		}
	})

	return peers
}

func setupPeerConfig(t *testing.T, libp2pPort int, bootstrapPeers []multiaddr.Multiaddr,
	withSwarmKey bool) *models.Libp2pConfig {
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	return &models.Libp2pConfig{
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-randevouz",
		Server:                  false,
		Scheduler:               background_tasks.NewScheduler(10),
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", libp2pPort)},
		PeerCountDiscoveryLimit: 40,
		PrivateNetwork: models.PrivateNetworkConfig{
			WithSwarmKey: withSwarmKey,
		},
	}
}
