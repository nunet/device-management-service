package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ListPeersHandler  godoc
//
//	@Summary		Return list of peers currently connected to
//	@Description	Gets a list of peers the libp2p node can see within the network and return a list of peers
//	@Tags			p2p
//	@Produce		json
//	@Failure		500	{object}	object	"host Node hasn't yet been initialized"
//	@Failure		500	{object}	object	"peers haven't yet been fetched"
//	@Failure		500	{object}	object	"no peers yet"
//	@Success		200	{object}	object	"list of peers"
//	@Router			/peers [get]
func ListPeersHandler(c *gin.Context, p2p libp2p.Libp2p) {
	peers, err := p2p.ListPeers()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, peers)
}

// ListDHTPeersHandler  godoc
//
//	@Summary		Return list of peers which have sent a dht update
//	@Description	Gets a list of peers the libp2p node has received a dht update from
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"List of peers"
//	@Failure		404	{object}	object	"No peers found"
//	@Failure		500	{object}	object	"Host Node hasn't yet been initialized"
//
//	@Router			/peers/dht [get]
func ListDHTPeersHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()
	peers, err := p2p.ListDHTPeers(reqCtx)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(peers) == 0 {
		c.JSON(404, gin.H{"message": "no peers found"})
		return
	}
	c.JSON(200, peers)
}

// ListKadDHTPeersHandler  godoc
//
//	@Summary		Return list of peers which have sent a dht update
//	@Description	Gets a list of peers the libp2p node has received a dht update from
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"List of DHT peers"
//	@Failure		400	{object}	object	"No peers found"
//	@Failure		500	{object}	object	"Host Node hasn't yet been initialized"
//	@Router			/peers/kad-dht [get]
func ListKadDHTPeersHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()

	peers, err := p2p.ListKadDHTPeers(c, reqCtx)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(peers) == 0 {
		c.JSON(404, gin.H{"message": "no peers found"})
		return
	}
	c.JSON(200, peers)
}

// SelfPeerInfoHandler  godoc
//
//	@Summary		Return self peer info
//	@Description	Gets self peer info of libp2p node
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"Self Peer Info"
//	@Failure		500	{object}	object	"host node hasn't yet been initialized"
//	@Router			/peers/self [get]
func SelfPeerInfoHandler(c *gin.Context, p2p libp2p.Libp2p) {
	self, err := p2p.SelfPeerInfo()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, self)
}

// ListChatHandler  godoc
//
//	@Summary		List incoming chat requests to current peer.
//	@Description	Get a list of chat requests from peers. List could be empty if no requests are pending.
//	@Tags			chat
//	@Produce		json
//	@Success		200	{object}	[]libp2p.OpenStream	"List of chat requests"
//	@Failure		404	{object}	object				"no incoming message stream"
//	@Router			/peers/chat [get]
func ListChatHandler(c *gin.Context, p2p libp2p.Libp2p) {
	chats, err := p2p.IncomingChatRequests()
	if err != nil {
		c.AbortWithStatusJSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, chats)
}

// ClearChatHandler  godoc
//
//	@Summary		Clear chat requests received to current peer.
//	@Description	Clear chat request streams from peers.
//	@Tags			chat
//	@Produce		json
//	@Success		200	{object}	object	"successfully cleared inbound chat requests"
//	@Failure		500	{object}	object	"no inbound chat streams"
//	@Router			/peers/chat/clear [get]
func ClearChatHandler(c *gin.Context, p2p libp2p.Libp2p) {
	err := p2p.ClearIncomingChatRequests()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "successfully cleared inbound chat requests"})
}

// StartChatHandler  godoc
//
//	@Summary		Start chat with a peer given the `peerID`.
//	@Description	Starts a chat session with a peer which is passed in `peerID`. The given peer must already be in the DHT. This endpoint is ultimately websocket upgraded.
//	@Tags			chat
//	@Produce		json
//	@Param			peerID	query		string	true	"Peer ID in DHT"
//	@Success		200		{object}	object	"WebSocket connection upgraded successfully"
//	@Failure		400		{object}	object	"invalid peer ID"
//	@Failure		500		{object}	object	"peerID can not be self peerID"
//	@Failure		500		{object}	object	"could not create stream with peer"
//	@Router			/peers/chat/start [get]
func StartChatHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()
	id := c.Query("peerID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peer ID: empty peer ID"})
		return
	}
	p, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peer ID: could not decode string ID to peer ID"})
		return
	}

	stream, err := p2p.CreateChatStream(reqCtx, p)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	p2p.StartChat(c.Writer, c.Request, stream, id)
	// should it return a response?
}

// JoinChatHandler  godoc
//
//	@Summary		Join chat with a peer given it's ID.
//	@Description	Join a chat session started by a peer. The given peer must already be in the DHT and ID can be derived from the list of incoming chat requests. This endpoint is ultimately websocket upgraded.
//	@Tags			chat
//	@Produce		json
//	@Param			streamID	query		int		0	"Chat ID from the chat requests"
//	@Success		200			{object}	object	"WebSocket connection upgraded successfully"
//	@Failure		400			{object}	object	"stream id not provided"
//	@Failure		400			{object}	object	"invalid type for streamID"
//	@Failure		500			{object}	object	"unknown streamID"
//	@Failure		500			{object}	object	"Failed to set websocket upgrade: {error}"
//	@Router			/peers/chat/join [get]
func JoinChatHandler(c *gin.Context, p2p libp2p.Libp2p) {
	streamID := c.Query("streamID")
	if streamID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "stream ID not provided"})
		return
	}
	stream, err := strconv.Atoi(streamID)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid type for streamID"})
		return
	}
	err = p2p.JoinChat(c.Writer, c.Request, stream)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	// Successfully upgraded to WebSocket
	response := gin.H{"message": "WebSocket connection upgraded successfully"}
	c.JSON(http.StatusOK, response)
}

// DumpDHTHandler  godoc
//
//	@Summary		Return a dump of the dht
//	@Description	Returns entire DHT content
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"List of DHT peers"
//	@Failure		500	{object}	object	"Host Node hasn't yet been initialized"
//	@Failure		404	{object}	object	"no content in DHT"
//	@Router			/peers/dht/dump [get]
func DumpDHTHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()
	dht, err := p2p.DumpDHT(reqCtx)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(dht) == 0 {
		c.JSON(200, gin.H{"message": "empty DHT"})
		return
	}
	c.JSON(200, dht)

}

// DefaultDepReqPeerHandler  godoc
//
//	@Summary		Manage default deployment request receiver peer
//	@Description	Set peer as the default receipient of deployment requests by setting the peerID parameter on GET request.
//	@Description	Show peer set as default deployment request receiver by sending a GET request without any parameters.
//	@Description	Remove default deployment request receiver by sending a GET request with peerID parameter set to '0'.
//	@Tags			p2p
//	@Produce		json
//	@Param			peerID	query		string	false	"Peer ID in DHT"
//	@Success		200		{object}	object	"Successfully set default deployment request receiver"
//	@Success		200		{object}	object	"no default peer set"
//	@Failure		500		{object}	object	"peerID can not be self peerID"
//	@Failure		500		{object}	object	"could not decode string ID to peerID"
//	@Failure		500		{object}	object	"default peer is <peerID>"
//	@Failure		500		{object}	object	"Peer not online"
//	@Failure		500		{object}	object	"Peer not in DHT yet"
//	@Router			/peers/depreq [get]
func DefaultDepReqPeerHandler(c *gin.Context, p2p libp2p.Libp2p) {
	peerID := c.Query("peerID")
	reqCtx := c.Request.Context()

	target, err := p2p.DefaultDepReqPeer(reqCtx, peerID)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("successfully set %s as target", target)})
}

// ClearFileTransferRequestsHandler  godoc
//
//	@Summary		Clear file transfer requests.
//	@Description	Clear file transfer request streams from peers.
//	@Tags			file
//	@Produce		json
//	@Success		200	{object}	object	"Successfully Cleard Inbound File Transfer Requests."
//	@Failure		500	{object}	object	"no inbound file transfer stream"
//	@Router			/peers/file/clear [get]
func ClearFileTransferRequestsHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()
	span := trace.SpanFromContext(reqCtx)
	span.SetAttributes(attribute.String("URL", "/peers/file/clear"))

	err := p2p.ClearIncomingFileRequests()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "successfully cleared inbound file transfer requests"})
}

// ListFileTransferRequestsHandler  godoc
//
//	@Summary		List file transfer requests to current peer.
//	@Description	Get a list of file transfer requests from peers. List could be empty if no requests are pending.
//	@Tags			file
//	@Produce		json
//	@Success		200	{object}	object	"List of file transfer requests"
//	@Failure		404	{object}	object	"no incoming file transfer stream"
//	@Router			/peers/file [get]
func ListFileTransferRequestsHandler(c *gin.Context, p2p libp2p.Libp2p) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("URL", "/peers/file"))

	req, err := p2p.IncomingFileTransferRequests()
	if err != nil {
		c.AbortWithStatusJSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, req)
}

// SendFileTransferHandler  godoc
//
//	@Summary		Send a file to a peer
//	@Description	Initiate file transfer to a peer. filePath and peerID are required arguments. filePath is the absolute path to the file to be sent. peerID is the peer ID of the peer to send the file to. peerID must be in the DHT. This endpoint is upgraded to websocket.
//	@Tags			file
//	@Produce		json
//	@Param			peerID		query	string	true	"Peer ID in DHT"
//	@Param			filePath	query	string	true	"File path to send to peer"
//	@Success		200
//	@Failure		400	{object}	object	"invalid peer string ID"
//	@Failure		400	{object}	object	"peerID can not be self peerID"
//	@Failure		400	{object}	object	"filePath not provided"
//	@Failure		500	{object}	object	"failed to set websocket upgrade"
//	@Failure		500	{object}	object	"could not send file to peer"
//	@Router			/peers/file/send [get]
func SendFileTransferHandler(c *gin.Context, p2p libp2p.Libp2p) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("URL", "/peers/file/send"))

	id := c.Query("peerID")
	p, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid peer string ID: could not decode string ID to peer ID"})
		return
	}
	if p == p2p.Host.ID() {
		c.AbortWithStatusJSON(400, gin.H{"error": "peer ID cannot be self peer ID"})
		return
	}

	path := c.Query("filePath")
	if len(path) == 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": "filePath not provided"})
		return
	}

	err = p2p.InitiateTransferFile(c, c.Writer, c.Request, p, path)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, nil)
}

// AcceptFileTransferHandler  godoc
//
//	@Summary		Accept an incoming file transfer.
//	@Description	Accept an incoming file transfer. Incoming file transfer `id` is a required parameter. This id can be derived from the list of incoming file transfer requests. This endpoint is upgraded to websocket. Only after accepting the file transfer, the file will be saved to the local file system.
//	@Produce		json
//	@Tags			file
//	@Success		200	{object}	object	"File Transfer Complete."
//	@Param			id	query		int		0	"File request ID from the file requests"
//	@Failure		400	{object}	object	"Invalid ID: {id}"
//	@Failure		400	{object}	object	"Unknown ID: {id}"
//	@Failure		500	{object}	object	"Could not accept file transfer - {error}"
//	@Failure		500	{object}	object	"Failed to set websocket upgrade: {error}"
//	@Router			/peers/file/accept [get]
func AcceptFileTransferHandler(c *gin.Context, p2p libp2p.Libp2p) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("URL", "/peers/file/accept"))

	id := c.Query("id")
	stream, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": fmt.Errorf("invalid ID: %w", err)})
		return
	}
	if stream != 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": fmt.Sprintf("unknown ID: %d", stream)})
		return
	}

	msg, err := p2p.AcceptPeerFileTransfer(c, c.Writer, c.Request, stream)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": msg})
}
