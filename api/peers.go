package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

// P2PHandler is a controller for /peers endpoint functionalities
// TODO: Create a service type for these functionalities
// and embed inside the handler
type P2PHandler struct {
	p2p *libp2p.Libp2p
}

// ListPeers  godoc
//
//		@Summary		Return list of peers currently connected to
//		@Description	Gets a list of peers the libp2p node can see within the network and return a list of peers
//		@Tags			p2p
//		@Produce		json
//		@Failure		500	{object}	object	"no peers yet"
//	    @Failure		500	{object}	object	"host node hasn't yet been initialized"
//		@Success		200	{object}	object	"list of peers"
//		@Router			/peers [get]
func (p *P2PHandler) ListPeers(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	peers := p.p2p.VisiblePeers()
	if len(peers) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no peers yet"})
		return
	}
	c.JSON(http.StatusOK, peers)
}

// KnownPeers  godoc
//
//	@Summary	Return list of peers which have sent a dht update
//	@Description	Gets a list of peers the libp2p node has received a dht update from
//	@Tags		p2p
//	@Produce	json
//	@Success	200	{object}	object	"List of peers"
//	@Failure	404	{object}	object	"No peers found"
//	@Failure	500	{object}	object	"Host Node hasn't yet been initialized"
//	@Router		/peers/dht [get]
func (p *P2PHandler) KnownPeers(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	peers, err := p.p2p.KnownPeers()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(peers) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "no peers found"})
		return
	}
	c.JSON(http.StatusOK, peers)
}

// SelfPeerInfo  godoc
//
//	@Summary		Return self peer info
//	@Description	Gets self peer info of libp2p node
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"Self Peer Info"
//	@Failure		500	{object}	object	"host node hasn't yet been initialized"
//	@Router			/peers/self [get]
func (p *P2PHandler) SelfPeerInfo(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	self := p.p2p.Stat()
	c.JSON(http.StatusOK, self)
}

// DumpDHT  godoc
//
//	@Summary		Return a dump of the dht
//	@Description	Returns entire DHT content
//	@Tags			p2p
//	@Produce		json
//	@Success		200	{object}	object	"List of DHT peers"
//	@Failure		500	{object}	object	"host node hasn't yet been initialized"
//	@Failure		500	{object}	object	"no content in DHT"
//	@Router			/peers/dht/dump [get]
func (p *P2PHandler) DumpDHT(c *gin.Context) {
	if p.p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}
	dht, err := p.p2p.DumpDHTRoutingTable()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(dht) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "empty DHT"})
		return
	}
	c.JSON(http.StatusOK, dht)
}
