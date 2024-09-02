package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DEBUG
func (rs RESTServer) PingPeer(c *gin.Context) {
	reqCtx := c.Request.Context()
	id := c.Query("peerID")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "peerID not provided"})
		return
	}
	if id == rs.config.P2P.Host.ID().String() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "peerID can not be self peerID"})
		return
	}

	// decode only for validation
	target, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid string ID: could not decode string ID to peer ID"})
		return
	}

	res, err := rs.config.P2P.Ping(reqCtx, target.String(), time.Second*5)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("could not ping peer %s: %v", id, err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ping peer %s, success=%t, RTT=%d", id, res.Success, res.RTT)})
}
