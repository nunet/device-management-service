package libp2p

import (
	"context"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubnetCreate(t *testing.T) {
	peer1 := createPeer(t, 4001)
	require.NotNil(t, peer1)

	err := peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, "subnet1", peer1.subnets["subnet1"].info.id)
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].ifaces))
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].info.rtable.All()))
}

func TestSubnetAddPeer(t *testing.T) {
	peer1 := createPeer(t, 4001)
	require.NotNil(t, peer1)

	err := peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	err = peer1.AddSubnetPeer("subnet1", peer1.Host.ID().String(), "10.0.0.1")
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, 1, len(peer1.subnets["subnet1"].ifaces))
	assert.Equal(t, 1, len(peer1.subnets["subnet1"].info.rtable.All()))

	peerID, ok := peer1.subnets["subnet1"].info.rtable.Get(peer1.Host.ID())
	require.True(t, ok)

	assert.Equal(t, "10.0.0.1", peerID)

	ip, ok := peer1.subnets["subnet1"].info.rtable.GetByIP("10.0.0.1")
	require.True(t, ok)

	assert.Equal(t, peer1.Host.ID(), ip)
}

func createPeer(t *testing.T, port int) *Libp2p {
	peerConfig := setupPeerConfig(t, port, []multiaddr.Multiaddr{})
	peer1, err := New(peerConfig, afero.NewMemMapFs())

	require.NoError(t, err)
	require.NoError(t, peer1.Init())

	return peer1
}
