package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

type P2pHandler struct {
	p2p *libp2p.Libp2p
}

// ListPeersHandler  godoc
//
//		@Summary		Return list of peers currently connected to
//		@Description	Gets a list of peers the libp2p node can see within the network and return a list of peers
//		@Tags			p2p
//		@Produce		json
//		@Failure		500	{object}	object	"no peers yet"
//	    @Failure		500	{object}	object	"host node hasn't yet been initialized"
//		@Success		200	{object}	object	"list of peers"
//		@Router			/peers [get]
func (p *P2pHandler) ListPeersHandler(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(500, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	peers := p.p2p.VisiblePeers()
	if len(peers) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no peers yet"})
		return
	}
	c.JSON(200, peers)
}

// KnownPeersHandler  godoc
//
//		@Summary		Return list of peers which have sent a dht update
//		@Description	Gets a list of peers the libp2p node has received a dht update from
//		@Tags			p2p
//		@Produce		jsonfunc (m *MockHandler) ListChatHandler(c *gin.Context) {
//		chats, err := mockListChat()
//		if err != nil {
//			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
//		}
//		c.JSON(200, chats)
//	}
//
//	@Success		200	{object}	object	"List of peers"
//	@Failure		404	{object}	object	"No peers found"
//	@Failure		500	{object}	object	"Host Node hasn't yet been initialized"
//
//	@Router			/peers/dht [get]
func (p *P2pHandler) KnownPeersHandler(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(500, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	peers, err := p.p2p.KnownPeers()
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
func (p *P2pHandler) SelfPeerInfoHandler(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(500, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	self := p.p2p.Stat()
	c.JSON(200, self)
}

// DumpDHTHandler  godoc
//
//	@Summary		Return a dump of the dht
//	@Description	Returns entire DHT content
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"List of DHT peers"
//	@Failure		500	{object}	object	"host node hasn't yet been initialized"
//	@Failure		404	{object}	object	"no content in DHT"
//	@Router			/peers/dht/dump [get]
func (p *P2pHandler) DumpDHTHandler(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(500, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	dht, err := p.p2p.DumpDHTRoutingTable()
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
