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
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// log is the logger for the actor API package
var log = logging.Logger("actor-api")

const (
	ErrHostNotInitialized = "host node hasn't yet been initialized"
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
func (rs *Server) ActorHandle(c *gin.Context) {
	endSpan := observability.StartSpan(c, "actor_handle_retrieve_duration")
	defer endSpan()

	if rs.config.P2P == nil {
		log.Errorw("actor_handle_retrieve_failure", "error", ErrHostNotInitialized)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": ErrHostNotInitialized})
		return
	}

	// get handle here
	pubk := rs.config.P2P.GetPeerPubKey(rs.config.P2P.GetHostID())

	id, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		log.Errorw("actor_handle_retrieve_failure", "error", "handle id is invalid")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "handle id is invalid"})
		return
	}

	actorDID := did.FromPublicKey(pubk)
	handle := actor.Handle{
		ID:  id,
		DID: actorDID,
		Address: actor.Address{
			HostID:       rs.config.P2P.GetHostID().String(),
			InboxAddress: "root",
		},
	}

	log.Debugw("actor_handle_retrieve_success", "id", id, "DID", actorDID)
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
func (rs *Server) ActorSendMessage(c *gin.Context) {
	endSpan := observability.StartSpan(c, "actor_send_message_duration")
	defer endSpan()

	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		log.Errorw("actor_send_message_failure", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		log.Errorw("actor_send_message_failure", "error", ErrHostNotInitialized)
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrHostNotInitialized})
		return
	}

	err := sendMessage(c.Request.Context(), p2p, msg)
	if err != nil {
		log.Errorw("actor_send_message_failure", "error", err.Error(), "destination", msg.To.Address.HostID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Infow("actor_send_message_success", "destination", msg.To.Address.HostID)
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
func (rs *Server) ActorInvoke(c *gin.Context) {
	endSpan := observability.StartSpan(c, "actor_invoke_duration")
	defer endSpan()

	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		log.Errorw("actor_invoke_failure", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		log.Errorw("actor_invoke_failure", "error", ErrHostNotInitialized)
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrHostNotInitialized})
		return
	}

	// Register a message handler for the responseCh
	protocol := fmt.Sprintf("actor/%s/messages/0.0.1", msg.From.Address.InboxAddress)
	responseCh := make(chan actor.Envelope, 1)
	err := p2p.HandleMessage(protocol, func(data []byte, _ peer.ID) {
		var envelope actor.Envelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			log.Errorw("actor_invoke_response_failure", "error", err.Error())
			return
		}
		responseCh <- envelope
	})
	if err != nil {
		log.Errorw("actor_invoke_failure", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unregister the message handler before returning
	defer p2p.UnregisterMessageHandler(protocol)

	err = sendMessage(c.Request.Context(), p2p, msg)
	if err != nil {
		log.Errorw("actor_invoke_failure", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	select {
	case responseMsg := <-responseCh:
		log.Debugw("actor_invoke_success", "destination", msg.To.Address.HostID)
		c.JSON(http.StatusOK, responseMsg)
		return
	case <-time.After(time.Until(msg.Expiry())):
		log.Errorw("actor_invoke_failure", "error", "request timeout")
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
		return
	case <-c.Request.Context().Done():
		log.Errorw("actor_invoke_failure", "error", "request timeout")
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
func (rs *Server) ActorBroadcast(c *gin.Context) {
	endSpan := observability.StartSpan(c, "actor_broadcast_duration")
	defer endSpan()

	var msg actor.Envelope
	if err := c.ShouldBindJSON(&msg); err != nil {
		log.Errorw("actor_broadcast_failure", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2p := rs.config.P2P
	if p2p == nil {
		log.Errorw("actor_broadcast_failure", "error", ErrHostNotInitialized)
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrHostNotInitialized})
		return
	}

	if !msg.IsBroadcast() {
		log.Errorw("actor_broadcast_failure", "error", "message is not a broadcast message")
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is not a broadcast message"})
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Errorw("actor_broadcast_failure", "error", "failed to marshal message")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal message"})
		return
	}

	// register message handler to collect responses
	protocol := fmt.Sprintf("actor/%s/messages/0.0.1", msg.From.Address.InboxAddress)
	var messages []actor.Envelope
	var mu sync.Mutex
	err = p2p.HandleMessage(protocol, func(data []byte, _ peer.ID) {
		var envelope actor.Envelope
		if err = json.Unmarshal(data, &envelope); err != nil {
			log.Errorw("actor_broadcast_failure", "error", "failed to unmarshal response message")
			return
		}
		mu.Lock()
		messages = append(messages, envelope)
		mu.Unlock()
	})
	if err != nil {
		log.Errorw("actor_broadcast_failure", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unregister the message handler before returning
	defer p2p.UnregisterMessageHandler(protocol)

	// Publish the message
	if err := p2p.Publish(c.Request.Context(), msg.Options.Topic, data); err != nil {
		log.Errorw("actor_broadcast_failure", "error", "failed to publish message")
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
	log.Infow("actor_broadcast_success", "fromAddress", msg.From.Address.HostID, "responsesCount", len(messages))
	c.JSON(http.StatusOK, messages)
}

func sendMessage(ctx context.Context, net network.Network, msg actor.Envelope) (err error) {
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
