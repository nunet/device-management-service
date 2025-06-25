// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"gitlab.com/nunet/device-management-service/types"
)

// StreamHandler is a function type that processes data from a stream.
type StreamHandler func(stream network.Stream)

// HandlerRegistry manages the registration of stream handlers for different protocols.
type HandlerRegistry struct {
	host          host.Host
	handlers      map[protocol.ID]StreamHandler
	bytesHandlers map[protocol.ID]func(data []byte, peerId peer.ID)
	mu            sync.RWMutex
}

// NewHandlerRegistry creates a new handler registry instance.
func NewHandlerRegistry(host host.Host) *HandlerRegistry {
	return &HandlerRegistry{
		host:          host,
		handlers:      make(map[protocol.ID]StreamHandler),
		bytesHandlers: make(map[protocol.ID]func(data []byte, peerId peer.ID)),
	}
}

// RegisterHandlerWithStreamCallback registers a stream handler for a specific protocol.
func (r *HandlerRegistry) RegisterHandlerWithStreamCallback(messageType types.MessageType, handler StreamHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	protoID := protocol.ID(messageType)
	_, ok := r.handlers[protoID]
	if ok {
		return ErrStreamRegistered
	}

	r.handlers[protoID] = handler
	r.host.SetStreamHandler(protoID, network.StreamHandler(handler))
	return nil
}

// RegisterHandlerWithBytesCallback registers a stream handler for a specific protocol and sends the bytes back to callback.
func (r *HandlerRegistry) RegisterHandlerWithBytesCallback(
	messageType types.MessageType,
	s StreamHandler, handler func(data []byte, peerId peer.ID),
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	protoID := protocol.ID(messageType)
	_, ok := r.bytesHandlers[protoID]
	if ok {
		return errors.New("stream with this protocol is already registered")
	}

	r.bytesHandlers[protoID] = handler
	r.host.SetStreamHandler(protoID, network.StreamHandler(s))
	return nil
}

// SendMessageToLocalHandler given the message type it sends data to the local handler found.
func (r *HandlerRegistry) SendMessageToLocalHandler(messageType types.MessageType, data []byte, peerID peer.ID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protoID := protocol.ID(messageType)
	h, ok := r.bytesHandlers[protoID]
	if !ok {
		return
	}

	// we need this goroutine to avoid blocking the caller goroutine
	go h(data, peerID)
}

// UnregisterHandler unregisters a stream handler for a specific protocol.
func (r *HandlerRegistry) UnregisterHandler(messageType types.MessageType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	protoID := protocol.ID(messageType)
	delete(r.handlers, protoID)
	delete(r.bytesHandlers, protoID)
	r.host.RemoveStreamHandler(protoID)
}
