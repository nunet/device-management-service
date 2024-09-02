package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

// setupTestP2P creates a new P2PHandler with a mock Libp2p instance
func setupTestP2P() (*RESTServer, error) {
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
		Scheduler:       backgroundtasks.NewScheduler(5),
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

	return &RESTServer{config: &RESTServerConfig{P2P: p2p}}, nil
}

// XXX: non-deterministic test - depends on network conditions by waiting for discovery
func TestListPeers(t *testing.T) {
	ctx := context.Background()
	router := setupTestRouter()

	p2pRC := &RESTServer{config: &RESTServerConfig{P2P: nil}}
	var err error
	router.GET("/peers", p2pRC.ListPeers)

	// host not initialised
	w := performRequest(router, "GET", "/peers")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2pRC.config.P2P = np2p.config.P2P

	// test no peers - host not started
	w = performRequest(router, "GET", "/peers")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "no peers yet")

	// start host - discover peers
	err = p2pRC.config.P2P.Start(ctx)
	assert.NoError(t, err)

	// wait for discovery
	time.Sleep(5 * time.Second)
	w = performRequest(router, "GET", "/peers")

	assert.Equal(t, 200, w.Code)

	// check fomat is in {ID: ..., Addr: ...}
	data := []peer.AddrInfo{}
	err = json.Unmarshal(w.Body.Bytes(), &data)
	assert.NoError(t, err)
}

func TestKnownPeers(t *testing.T) {
	router := setupTestRouter()

	p2pRC := &RESTServer{config: &RESTServerConfig{P2P: nil}}
	var err error
	router.GET("/peers/dht", p2pRC.KnownPeers)

	// host not initialised
	w := performRequest(router, "GET", "/peers/dht")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2pRC.config.P2P = np2p.config.P2P

	// test peers - host not started but should at least have it's own id
	w = performRequest(router, "GET", "/peers/dht")
	assert.Equal(t, 200, w.Code)

	// check fomat and self id
	data := []peer.AddrInfo{}
	err = json.Unmarshal(w.Body.Bytes(), &data)
	assert.NoError(t, err)
	assert.Equal(t, p2pRC.config.P2P.Host.ID().String(), data[0].ID.String())
}

func TestSelfPeerInfo(t *testing.T) {
	p2pRC := &RESTServer{config: &RESTServerConfig{P2P: nil}}
	var err error
	router := setupTestRouter()
	router.GET("/peers/self", p2pRC.SelfPeerInfo)

	// host not initialised
	w := performRequest(router, "GET", "/peers/self")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2pRC.config.P2P = np2p.config.P2P

	w = performRequest(router, "GET", "/peers/self")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), p2pRC.config.P2P.Host.ID().String())
}

func TestDumpDHT(t *testing.T) {
	ctx := context.Background()
	p2p := &RESTServer{config: &RESTServerConfig{P2P: nil}}
	var err error
	router := setupTestRouter()
	router.GET("/peers/dht/dump", p2p.DumpDHT)

	// host not initialised
	w := performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "host node hasn't yet been initialized")

	np2p, err := setupTestP2P()
	assert.NoError(t, err)
	p2p.config.P2P = np2p.config.P2P

	// test no peers - host not started
	w = performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "empty DHT")

	// start host - discover peers
	err = p2p.config.P2P.Start(ctx)
	assert.NoError(t, err)
	w = performRequest(router, "GET", "/peers/dht/dump")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "LastSuccessfulOutboundQueryAt")
}
