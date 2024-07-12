package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/models"
)

func (m *MockHandler) ListPeersHandler(c *gin.Context) {
	peers, err := mockPeerAddrInfos()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": "could not get peer list"})
	}
	c.JSON(200, peers)
}

func (m *MockHandler) ListDHTPeersHandler(c *gin.Context) {
	if mockHostID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "host node has not yet been initialized"})
		return
	}
	peers := mockDHTPeers()
	c.JSON(200, peers)
}

func (m *MockHandler) SelfPeerInfoHandler(c *gin.Context) {
	if mockHostID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "host node has not yet been initialized"})
		return
	}
	self := models.NetworkStats{
		ID:         mockHostID,
		ListenAddr: mockAddr,
	}
	c.JSON(200, self)
}

func (m *MockHandler) DumpDHTHandler(c *gin.Context) {
	if mockHostID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "host node has not yet been initialized"})
		return
	}
	peers := []models.PeerData{
		{
			PeerID:      "foobarfoobarfoobar",
			IsAvailable: false,
		},
		{
			PeerID:      "foobazfoobazfoobaz",
			IsAvailable: true,
		},
		{
			PeerID:      "bazbazbazbazbazbaz",
			IsAvailable: false,
		},
	}
	c.JSON(200, peers)
}

func TestListPeersHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestListDHTPeersHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers/dht", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSelfPeerInfoHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers/self", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func mockPeerAddrInfos() ([]peer.AddrInfo, error) {
	var addrInfos []peer.AddrInfo
	peerData := []struct {
		ID   string
		Addr string
	}{
		{"12D3KooWEgUjXjxGnZL7DwExVnEz5pcL5U3jxKpB3o6XJgXrMuXz", "/ip4/127.0.0.1/tcp/13001"},
		{"12D3KooWLrudbCjki3qfQpY8ghy7MbpHLWGvQYqXBL8Xs3ss2yLH", "/ip4/127.0.0.1/tcp/13002"},
	}

	for _, pd := range peerData {
		pid, err := peer.Decode(pd.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peer ID: %w", err)
		}

		maddr, err := multiaddr.NewMultiaddr(pd.Addr)
		if err != nil {
			return nil, fmt.Errorf("failed to create multiaddr: %w", err)
		}

		addrInfos = append(addrInfos, peer.AddrInfo{
			ID:    pid,
			Addrs: []multiaddr.Multiaddr{maddr},
		})
	}
	return addrInfos, nil
}

func mockDHTPeers() []peer.ID {
	if dhtPeers == 0 {
		return []peer.ID{}
	}
	return []peer.ID{"Qm0xbarbarbar", "Qm1xbazbazbaz", "Qm2xfoobarfoobar", "Qm3xfoobazfoobaz", "Qm4xfoofoofoo"}
}

func mockKadDHTPeers() []string {
	if kadDHTPeers == 0 {
		return []string{}
	}
	return []string{"Qm0xfoobar", "Qm1xfoobarbarbar", "Qm2xbazbazfoo", "Qm3xfoobarbarfoo"}
}
