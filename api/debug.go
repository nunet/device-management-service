package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DEBUG
func (p *P2pHandler) PingPeerHandler(c *gin.Context) {
	reqCtx := c.Request.Context()
	id := c.Query("peerID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "peerID not provided"})
		return
	}
	if id == p.p2p.Host.ID().String() {
		c.AbortWithStatusJSON(400, gin.H{"error": "peerID can not be self peerID"})
		return
	}

	// decode only for validation
	target, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid string ID: could not decode string ID to peer ID"})
		return
	}

	// res, err := p.p2p.Ping(reqCtx, target.String(), time.Second*5)
	res, _ := p.p2p.Ping(reqCtx, target)
	result := <-res
	if result.Error != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": fmt.Sprintf("could not ping peer %s: %v", id, err)})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("ping peer %s, success=%t, RTT=%d", id, result.Error == nil, result.RTT)})
}
