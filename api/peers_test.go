package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/network/libp2p"
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

func (m *MockHandler) ListKadDHTPeersHandler(c *gin.Context) {
	if mockHostID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "host node has not yet been initialized"})
		return
	}
	peers := mockKadDHTPeers()
	c.JSON(200, peers)
}

func (m *MockHandler) SelfPeerInfoHandler(c *gin.Context) {
	if mockHostID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "host node has not yet been initialized"})
		return
	}
	self := libp2p.SelfPeer{
		ID:    mockHostID,
		Addrs: mockMaddrs(),
	}
	c.JSON(200, self)
}

func mockListChat() ([]libp2p.OpenStream, error) {
	if mockInboundChats == 0 {
		return nil, fmt.Errorf("no incoming message stream")
	}
	return []libp2p.OpenStream{
		{
			ID:         0,
			StreamID:   "abc",
			FromPeer:   "foobar",
			TimeOpened: "192389203",
		},
		{
			ID:         1,
			StreamID:   "def",
			FromPeer:   "barfoo",
			TimeOpened: "942093409",
		},
		{
			ID:         2,
			StreamID:   "ghi",
			FromPeer:   "bazfoo",
			TimeOpened: "39840238",
		},
	}, nil
}

func (m *MockHandler) ListChatHandler(c *gin.Context) {
	chats, err := mockListChat()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
	}
	c.JSON(200, chats)
}

func (m *MockHandler) ClearChatHandler(c *gin.Context) {
	if mockInboundChats == 0 {
		c.AbortWithStatusJSON(500, gin.H{"error": "no inbound message streams"})
	}
	c.JSON(200, gin.H{"message": "Successfully Cleard Inbound Chat Requests."})
}

func (m *MockHandler) StartChatHandler(c *gin.Context) {
	id := c.Query("peerID")
	if !validateMockID(id) {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peerID query: string is not peerID"})
		return
	} else if id == mockHostID {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peerID query: peerID cannot be self peer ID"})
		return
	}
	fmt.Fprintf(m.buf, "started chat with %s\n", id)
}

func (m *MockHandler) JoinChatHandler(c *gin.Context) {
	id := c.Query("streamID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "stream ID not provided"})
		return
	}
	stream, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid type for streamID"})
		return
	}
	fmt.Fprintf(m.buf, "joined chat %d\n", stream)
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

func (m *MockHandler) DefaultDepReqPeerHandler(c *gin.Context) {
	var target string
	id := c.Query("peerID")
	if id == "0" {
		defaultPeer = ""
	} else if id == "" && defaultPeer == "" {
		target = ""
	} else if id == "" {
		target = defaultPeer
	} else if id == mockHostID {
		c.AbortWithStatusJSON(400, gin.H{"error": "target peer cannot be self peerID"})
		return
	} else if !validateMockID(id) {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peerID"})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("successfully set %s as target peer", target)})
}

func (m *MockHandler) ClearFileTransferRequestsHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "successfully cleared inbound file transfer requests"})
}

func (m *MockHandler) ListFileTransferRequestsHandler(c *gin.Context) {
	result := fmt.Sprintf("Time: %s\nFile Name: %s\nFile Size: %d bytes\n", "989348", "foobar.tar.gz", 80)
	c.JSON(200, result)
}

func (m *MockHandler) SendFileTransferHandler(c *gin.Context) {
	id := c.Query("peerID")
	if len(id) == 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": "peer ID not provided"})
		return
	}
	if id == mockHostID {
		c.AbortWithStatusJSON(400, gin.H{"error": "peer ID cannot be self peer ID"})
		return
	}
	if !validateMockID(id) {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peer string ID: could not decode string ID to peer ID"})
		return
	}
	path := c.Query("filePath")
	if len(path) == 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": "filepath not provided"})
		return
	}
	c.JSON(200, nil)
}

func (m *MockHandler) AcceptFileTransferHandler(c *gin.Context) {
	id := c.Query("streamID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "stream ID not provided"})
		return
	}

	stream, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": fmt.Sprintf("invalid stream ID: %s", id)})
		return
	}
	if stream != 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": fmt.Sprintf("unknown stream ID: %d", stream)})
		return
	}
	c.JSON(200, nil)
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

func TestListKadDHTPeersHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers/kad-dht", nil)
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

func TestListChatHandler(t *testing.T) {
	router := SetupMockRouter()
	tests := []struct {
		description  string
		chats        int
		expectedCode int
	}{
		{
			description:  "no chats",
			chats:        0,
			expectedCode: 500,
		},
		{
			description:  "available chats",
			chats:        5,
			expectedCode: 200,
		},
	}
	for _, tc := range tests {
		mockInboundChats = tc.chats

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/chat", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

func TestClearChatHandler(t *testing.T) {
	router := SetupMockRouter()
	tests := []struct {
		description  string
		chats        int
		expectedCode int
	}{
		{
			description:  "no chats",
			chats:        0,
			expectedCode: 500,
		},
		{
			description:  "available chats",
			chats:        5,
			expectedCode: 200,
		},
	}
	for _, tc := range tests {
		mockInboundChats = tc.chats

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/chat/clear", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

func TestStartChatHandlerWithQueries(t *testing.T) {
	router := SetupMockRouter()

	tests := []struct {
		description  string
		query        string
		expectedCode int
	}{
		{
			description:  "valid peer ID",
			query:        "?peerID=Qmxfoobarfoobar",
			expectedCode: 200,
		},
		{
			description:  "invalid peer ID",
			query:        "?peerID=invalidPeerID",
			expectedCode: 400,
		},
		{
			description:  "missing peer ID",
			query:        "",
			expectedCode: 400,
		},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/chat/start"+tc.query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

func TestJoinChatHandlerWithQueries(t *testing.T) {
	router := SetupMockRouter()

	tests := []struct {
		description  string
		query        string
		expectedCode int
	}{
		{
			description:  "valid stream ID",
			query:        "?streamID=123",
			expectedCode: 200,
		},
		{
			description:  "invalid stream ID",
			query:        "?streamID=abc",
			expectedCode: 400,
		},
		{
			description:  "missing stream ID",
			query:        "",
			expectedCode: 400,
		},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/chat/join"+tc.query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

// func TestDumpDHTHandler(t *testing.T) {
// 	router := SetupMockRouter()

// 	w := httptest.NewRecorder()
// 	req, _ := http.NewRequest("GET", "/api/v1/dht", nil)
// 	router.ServeHTTP(w, req)

// 	assert.Equal(t, 200, w.Code)
// }

func TestDefaultDepReqPeerHandlerWithQueries(t *testing.T) {
	router := SetupMockRouter()

	tests := []struct {
		description  string
		query        string
		expectedCode int
	}{
		{
			description:  "valid peer ID",
			query:        "?peerID=Qmxfoobarfoobarfoobar",
			expectedCode: 200,
		},
		{
			description:  "remove default peer ID",
			query:        "?peerID=0",
			expectedCode: 200,
		},
		{
			description:  "missing peer ID",
			query:        "",
			expectedCode: 200,
		},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/depreq"+tc.query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

func TestClearFileTransferRequestsHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers/file/clear", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestListFileTransferRequestsHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/peers/file", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSendFileTransferHandlerWithQueries(t *testing.T) {
	router := SetupMockRouter()

	tests := []struct {
		description  string
		query        string
		expectedCode int
	}{
		{
			description:  "valid peer ID and file path",
			query:        "?peerID=Qmxabcabcabdffd&filePath=/foo/bar",
			expectedCode: 200,
		},
		{
			description:  "missing peer ID",
			query:        "?filePath=/path/to/file",
			expectedCode: 400,
		},
		{
			description:  "missing file path",
			query:        "?peerID=somePeerID",
			expectedCode: 400,
		},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/file/send"+tc.query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
}

func TestAcceptFileTransferHandlerWithQueries(t *testing.T) {
	router := SetupMockRouter()

	tests := []struct {
		description  string
		query        string
		expectedCode int
	}{
		{
			description:  "valid stream ID",
			query:        "?streamID=0",
			expectedCode: 200,
		},
		{
			description:  "invalid stream ID int value",
			query:        "?streamID=23",
			expectedCode: 400,
		},
		{
			description:  "invalid type stream ID",
			query:        "?streamID=abc",
			expectedCode: 400,
		},
		{
			description:  "missing stream ID",
			query:        "",
			expectedCode: 400,
		},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/peers/file/accept"+tc.query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
	}
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

func mockMaddrs() []multiaddr.Multiaddr {
	var multiaddrs []multiaddr.Multiaddr
	maddrStrings := []string{
		"/ip4/127.0.0.1/tcp/8080",
		"/ip6/::1/udp/3000",
		"/dns4/example.com/tcp/443/https",
	}
	for _, maddrString := range maddrStrings {
		maddr, err := multiaddr.NewMultiaddr(maddrString)
		if err != nil {
			continue
		}
		multiaddrs = append(multiaddrs, maddr)
	}
	return multiaddrs
}
