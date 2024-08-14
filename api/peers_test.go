package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

// setupTestP2P creates a new P2PHandler with a mock Libp2p instance
func setupTestP2P() (*P2PHandler, error) {
	ctx := context.Background()
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, fmt.Errorf("unable to generate key")
	}
	bootstrapPeers := make([]multiaddr.Multiaddr, len(config.GetConfig().P2P.BootstrapPeers))
	for i, addr := range config.GetConfig().P2P.BootstrapPeers {
		bootstrapPeers[i], _ = multiaddr.NewMultiaddr(addr)
	}

	cfg := &types.Libp2pConfig{
		PrivateKey:      priv,
		BootstrapPeers:  bootstrapPeers,
		Rendezvous:      "nunet-test",
		Server:          true,
		Scheduler:       background_tasks.NewScheduler(5),
		CustomNamespace: "/nunet-dht-1/",
		DHTPrefix:       "/nunet",
		ListenAddress: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic",
		},
		PeerCountDiscoveryLimit: 5,
		PrivateNetwork: types.PrivateNetworkConfig{
			WithSwarmKey: false,
		},
	}

	p2p, err := libp2p.New(cfg, afero.NewMemMapFs())
	if err != nil {
		return nil, fmt.Errorf("unable to create libp2p host")
	}

	if err = p2p.Init(ctx); err != nil {
		return nil, fmt.Errorf("unable to initialise libp2p host")
	}

	return &P2PHandler{p2p: p2p}, nil
}

func TestListPeers(t *testing.T) {
	ctx := context.Background()
	router := setupTestRouter()

	p2p := &P2PHandler{p2p: nil}
	var err error
	router.GET("/peers", p2p.ListPeers)

	// host not initialised
	w := performRequest(router, "GET", "/peers")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2p.p2p = np2p.p2p

	// test no peers - host not started
	w = performRequest(router, "GET", "/peers")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "no peers yet")

	// start host - discover peers
	err = p2p.p2p.Start(ctx)
	assert.NoError(t, err)
	w = performRequest(router, "GET", "/peers")
	assert.Equal(t, 200, w.Code)

	// check fomat is in {ID: ..., Addr: ...}
	data := []peer.AddrInfo{}
	err = json.Unmarshal(w.Body.Bytes(), &data)
	assert.NoError(t, err)
}

func TestKnownPeers(t *testing.T) {
	router := setupTestRouter()

	p2p := &P2PHandler{p2p: nil}
	var err error
	router.GET("/peers/dht", p2p.KnownPeers)

	// host not initialised
	w := performRequest(router, "GET", "/peers/dht")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2p.p2p = np2p.p2p

	// test peers - host not started but should at least have it's own id
	w = performRequest(router, "GET", "/peers/dht")
	assert.Equal(t, 200, w.Code)

	// check fomat and self id
	data := []peer.AddrInfo{}
	err = json.Unmarshal(w.Body.Bytes(), &data)
	assert.NoError(t, err)
	assert.Equal(t, p2p.p2p.Host.ID().String(), data[0].ID.String())
}

func TestSelfPeerInfo(t *testing.T) {
	p2p := &P2PHandler{p2p: nil}
	var err error
	router := setupTestRouter()
	router.GET("/peers/self", p2p.SelfPeerInfo)

	// host not initialised
	w := performRequest(router, "GET", "/peers/self")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2p.p2p = np2p.p2p

	w = performRequest(router, "GET", "/peers/self")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), p2p.p2p.Host.ID().String())

}

func TestDumpDHT(t *testing.T) {
	ctx := context.Background()
	p2p := &P2PHandler{p2p: nil}
	var err error
	router := setupTestRouter()
	router.GET("/peers/dht/dump", p2p.DumpDHT)

	// host not initialised
	w := performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2p.p2p = np2p.p2p

	// test no peers - host not started
	w = performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "empty DHT")

	// start host - discover peers
	err = p2p.p2p.Start(ctx)
	assert.NoError(t, err)
	w = performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "LastSuccessfulOutboundQueryAt")

}
