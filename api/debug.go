package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

// DEBUG
func ManualDHTUpdateHandler(c *gin.Context, p2p libp2p.Libp2p) {
	go p2p.UpdateKadDHT()
	p2p.GetDHTUpdates(c)
	c.JSON(200, gin.H{"message": "DHT update initiated"})
}

// DEBUG
func CleanupPeerHandler(c *gin.Context) {
	id := c.Query("peerID")
	p, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid string ID: could not decode string ID to peer ID"})
		return
	}

	err = libp2p.CleanupPeer(p)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": "unable to cleanup peer"})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("successfully cleaned up peer: %s", id)})
}

// DEBUG
func PingPeerHandler(c *gin.Context, p2p libp2p.Libp2p) {
	reqCtx := c.Request.Context()
	id := c.Query("peerID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "peerID not provided"})
		return
	}
	if id == p2p.Host.ID().String() {
		c.AbortWithStatusJSON(400, gin.H{"error": "peerID can not be self peerID"})
		return
	}
	target, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid string ID: could not decode string ID to peer ID"})
		return
	}

	status, result := libp2p.PingPeer(reqCtx, target)
	if result.Error != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": fmt.Sprintf("could not ping peer %s", id), "peer_in_dht": status, "RTT": result.RTT})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("ping successful with peer %s", id), "peer_in_dht": status, "RTT": result.RTT})
}

// DEBUG ONLY
func OldPingPeerHandler(c *gin.Context, p2p libp2p.Libp2p) {
	id := c.Query("peerID")
	if id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "peer ID not provided"})
		return
	}
	if id == p2p.Host.ID().String() {
		c.AbortWithStatusJSON(400, gin.H{"error": "peer ID cannot be self peer ID"})
		return
	}
	target, err := peer.Decode(id)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid string ID: could not decode string ID to peer ID"})
		return
	}
	status, result := libp2p.OldPingPeer(c, target)
	if result.Error != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": fmt.Errorf("could not ping peer %s: %w", id, result.Error), "peer_in_dht": status, "RTT": result.RTT})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("ping successful with peer %s", id), "peer_in_dht": status, "RTT": result.RTT})
}

// DEBUG
func DumpKademliaDHTHandler(c *gin.Context) {
	reqCtx := c.Request.Context()
	dht, err := libp2p.DumpKademliaDHT(reqCtx)
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
