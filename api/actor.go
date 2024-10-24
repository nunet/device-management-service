// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

// ActorHandle godoc
//
//	@Summary		Retrieve actor handle
//	@Description	Retrieve actor handle with ID, DID, and inbox address
//	@Tags			actor
//	@Produce		json
//	@Success		200	{object}	actor.Handle
//	@Failure		500	{object}	object	"host node hasn't yet been initialized"
//	@Failure		500	{object}	object	"handle id is invalid"
//	@Router			/actor/handle [get]
func (rs RESTServer) ActorHandle(c *gin.Context) {
	p2p := rs.config.P2P
	if p2p == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}

	// get handle here

	pubk := p2p.Host.Peerstore().PubKey(p2p.Host.ID())
	id, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "handle id is invalid"})
		return
	}

	did := did.FromPublicKey(pubk)

	handle := actor.Handle{
		ID:  id,
		DID: did,
		Address: actor.Address{
			HostID:       p2p.Host.ID().String(),
			InboxAddress: "root",
		},
	}

	c.JSON(http.StatusOK, handle)
}

// ActorSendMessage godoc
//
//		@Summary		Send message to actor
//		@Description	Send message to actor
//		@Tags			actor
//		@Accept			json
//		@Produce		json
//	 @Param			message	body	actor.Envelope	true	"Message to send"
//		@Success		200	{object}	object	"message sent"
//	 @Failure		400	{object}	object	"invalid request data"
//		@Failure		500	{object}	object	"host node hasn't yet been initialized"
//		@Failure		500	{object}	object	"failed to marshal message"
//		@Failure		500	{object}	object	"destination address can't be resolved"
//		@Failure		500	{object}	object	"failed to send message to destination"
//		@Router			/actor/send [post]
func (rs RESTServer) ActorSendMessage(c *gin.Context) {
	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}

	err := SendMessage(c.Request.Context(), p2p, msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message sent"})
}

// ActorInvoke godoc
//
//		@Summary		Invoke actor
//		@Description	Invoke actor with message
//		@Tags			actor
//		@Accept			json
//		@Produce		json
//	 @Param			message	body	actor.Envelope	true	"Message to send"
//		@Success		200	{object}	object	"response message"
//	 @Failure		400	{object}	object	"invalid request data"
//		@Failure		500	{object}	object	"host node hasn't yet been initialized"
//		@Failure		500	{object}	object	"failed to marshal message"
//		@Failure		500	{object}	object	"destination address can't be resolved"
//		@Failure		500	{object}	object	"failed to send message to destination"
//		@Router			/actor/invoke [post]
func (rs RESTServer) ActorInvoke(c *gin.Context) {
	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
		return
	}

	// Register a message handler for the responseCh
	protocol := fmt.Sprintf("actor/%s/messages/0.0.1", msg.From.Address.InboxAddress)
	responseCh := make(chan actor.Envelope, 1)
	err := p2p.HandleMessage(protocol, func(data []byte) {
		var envelope actor.Envelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			// TODO log this
			return
		}
		responseCh <- envelope
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unregister the message handler before returning
	defer p2p.UnregisterMessageHandler(protocol)

	err = SendMessage(c.Request.Context(), p2p, msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	select {
	case responseMsg := <-responseCh:
		c.JSON(http.StatusOK, responseMsg)
		return
	case <-time.After(time.Until(msg.Expiry())):
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
		return
	case <-c.Request.Context().Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
		return
	}
}

// ActorBroadcast godoc
//
//	     @Summary		Broadcast message to actors
//				@Description	Broadcast message to actors
//				@Tags			actor
//				@Accept			json
//				@Produce		json
//			  @Param			message	body	actor.Envelope	true	"Message to send"
//				@Success		200	{object}	object	"received responses"
//		   @Failure		400	{object}	object	"invalid request data"
//				@Failure		500	{object}	object	"host node hasn't yet been initialized"
//				@Failure		500	{object}	object	"failed to marshal message"
//				@Failure		500	{object}	object	"failed to publish message"
//				@Router			/actor/broadcast [post]
func (rs RESTServer) ActorBroadcast(c *gin.Context) {
	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "host node hasn't yet been initialized"})
	}

	if !msg.IsBroadcast() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is not a broadcast message"})
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal message"})
		return
	}

	// register message handler to collect responses
	protocol := fmt.Sprintf("actor/%s/messages/0.0.1", msg.From.Address.InboxAddress)
	var messages []actor.Envelope
	var mu sync.Mutex
	err = p2p.HandleMessage(protocol, func(data []byte) {
		var envelope actor.Envelope
		if err = json.Unmarshal(data, &envelope); err != nil {
			return
		}
		mu.Lock()
		messages = append(messages, envelope)
		mu.Unlock()
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unregister the message handler before returning
	defer p2p.UnregisterMessageHandler(protocol)

	// Publish the message
	if err := p2p.Publish(c.Request.Context(), msg.Options.Topic, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish message"})
		return
	}

	// Wait for either context done or timeout
	select {
	case <-time.After(time.Until(msg.Expiry())):
		// message expiry time reached
	case <-c.Request.Context().Done():
		// request context done
	}
	c.JSON(http.StatusOK, messages)
}

func SendMessage(ctx context.Context, net *libp2p.Libp2p, msg actor.Envelope) (err error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = net.SendMessageSync(
		ctx,
		msg.To.Address.HostID,
		types.MessageEnvelope{
			Type: types.MessageType(
				fmt.Sprintf("actor/%s/messages/0.0.1", msg.To.Address.InboxAddress),
			),
			Data: data,
		},
		msg.Expiry(),
	)
	if err != nil {
		return fmt.Errorf("failed to send message to %s: %w", msg.To.ID, err)
	}
	return nil
}
