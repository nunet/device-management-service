// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

// Constants for actor test topics and protocols
const (
	TestTopicName   = "test-topic"
	TestInbox1Name  = "test-inbox"
	TestInbox2Name  = "test-inbox-2"
	ProtocolVersion = "0.0.1"
)

// setupTestNetwork creates a new test network using the substrate
func setupTestNetwork(t *testing.T, substrate *network.Substrate) network.Network {
	t.Helper()
	_, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		panic(err)
	}
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		panic(err)
	}

	return substrate.MakeNetwork(peerID)
}

// Helper function to create protocol string from inbox name
func createProtocolString(inboxName string) string {
	return fmt.Sprintf("actor/%s/messages/%s", inboxName, ProtocolVersion)
}

func TestActorHandle(t *testing.T) {
	t.Parallel()

	t.Run("must fail if P2P is nil", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		server := NewServer(&ServerConfig{
			P2P:  nil,
			Port: 8080,
			Addr: "localhost",
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req, _ := http.NewRequest("GET", "/api/v1/actor/handle", nil)
		c.Request = req

		server.ActorHandle(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ErrHostNotInitialized, response["error"])
	})

	t.Run("must be able to get actor handle", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req, _ := http.NewRequest("GET", "/api/v1/actor/handle", nil)
		c.Request = req

		server.ActorHandle(c)
		assert.Equal(t, http.StatusOK, w.Code)

		var handle actor.Handle
		err := json.Unmarshal(w.Body.Bytes(), &handle)
		assert.NoError(t, err)
		assert.NotEmpty(t, handle.ID)
		assert.NotEmpty(t, handle.DID)
		assert.Equal(t, testNet.GetHostID().String(), handle.Address.HostID)
		assert.Equal(t, "root", handle.Address.InboxAddress)
	})
}

func TestActorSendMessage(t *testing.T) {
	t.Parallel()

	t.Run("must fail if P2P is nil", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)
		server := NewServer(&ServerConfig{
			P2P:  nil,
			Port: 8080,
			Addr: "localhost",
		})
		envelope := actor.Envelope{}
		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/send", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorSendMessage(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ErrHostNotInitialized, response["error"])
	})

	t.Run("must be able to send message", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)

		msgReceived := false
		protocol := fmt.Sprintf("actor/%s/messages/0.0.1", TestInbox1Name)
		err := testNet.HandleMessage(protocol, func(_ []byte, _ peer.ID) {
			msgReceived = true
		})
		assert.NoError(t, err)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})
		envelope := actor.Envelope{
			To: actor.Handle{
				Address: actor.Address{
					HostID:       testNet.GetHostID().String(),
					InboxAddress: TestInbox1Name,
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(time.Minute).UnixNano()),
			},
		}
		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/send", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		server.ActorSendMessage(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "message sent", response["message"])
		assert.True(t, msgReceived, "Message handler should have been called")
	})

	t.Run("must fail if invalid JSON is sent", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})
		invalidJSON := []byte(`{"invalid":"json"}`)
		req, _ := http.NewRequest("POST", "/api/v1/actor/send", bytes.NewBuffer(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		server.ActorSendMessage(c)
		assert.True(t, w.Code == http.StatusInternalServerError)
	})
}

func TestActorInvoke(t *testing.T) {
	t.Parallel()

	t.Run("must fail if P2P is nil", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		server := NewServer(&ServerConfig{
			P2P:  nil,
			Port: 8080,
			Addr: "localhost",
		})
		envelope := actor.Envelope{}
		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/invoke", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorInvoke(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ErrHostNotInitialized, response["error"])
	})

	t.Run("must be able to invoke actor", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet.GetHostID().String(),
					InboxAddress: "test-inbox",
				},
			},
			To: actor.Handle{
				Address: actor.Address{
					HostID:       testNet.GetHostID().String(),
					InboxAddress: "target-inbox",
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(time.Minute).UnixNano()),
			},
			Message: []byte("test message"),
		}

		protocol := fmt.Sprintf("actor/%s/messages/0.0.1", envelope.From.Address.InboxAddress)

		targetProtocol := fmt.Sprintf("actor/%s/messages/0.0.1", envelope.To.Address.InboxAddress)
		err := testNet.HandleMessage(targetProtocol, func(_ []byte, _ peer.ID) {
			// Create and send a response back
			responseEnvelope := actor.Envelope{
				From:    envelope.To,
				To:      envelope.From,
				Message: []byte("response data"),
			}

			// Send the response back
			responseData, _ := json.Marshal(responseEnvelope)
			_ = testNet.SendMessage(context.Background(), envelope.From.Address.HostID,
				types.MessageEnvelope{
					Type: types.MessageType(protocol),
					Data: responseData,
				},
				time.Now().Add(time.Minute))
		})
		assert.NoError(t, err)

		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/invoke", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		server.ActorInvoke(c)
		assert.Equal(t, http.StatusOK, w.Code)

		var actualResponse actor.Envelope
		err = json.Unmarshal(w.Body.Bytes(), &actualResponse)
		assert.NoError(t, err)
		assert.Equal(t, []byte("response data"), actualResponse.Message)
	})

	t.Run("must fail if invalid json is sent", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})
		invalidJSON := []byte(`{"invalid":}`)

		req, _ := http.NewRequest("POST", "/api/v1/actor/invoke", bytes.NewBuffer(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorInvoke(c)
		assert.True(t, w.Code == http.StatusBadRequest)
	})
}

func TestActorBroadcast(t *testing.T) {
	t.Parallel()

	t.Run("must fail if P2P is nil", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		server := NewServer(&ServerConfig{
			P2P:  nil,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       "test-host-id",
					InboxAddress: TestInbox1Name,
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(time.Minute).UnixNano()),
				Topic:  TestTopicName,
			},
			Message: []byte("broadcast message"),
		}
		envelopeJSON, _ := json.Marshal(envelope)

		req, _ := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorBroadcast(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ErrHostNotInitialized, response["error"])
	})

	t.Run("must fail if invalid JSON is sent", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})

		invalidJSON := []byte(`{"invalid-json"}`)

		req, _ := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorBroadcast(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("must fail if it fails to unmarshal message", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet1 := setupTestNetwork(t, substrate)
		testNet2 := setupTestNetwork(t, substrate)

		server := NewServer(&ServerConfig{
			P2P:  testNet1,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet1.GetHostID().String(),
					InboxAddress: "test-inbox",
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(5 * time.Second).UnixNano()),
				Topic:  "test-topic-unmarshal-error",
			},
			Message: []byte("broadcast message"),
		}

		var wg sync.WaitGroup
		wg.Add(1)

		_, err := testNet2.Subscribe(context.Background(), "test-topic-unmarshal-error", func(_ []byte) {
			defer wg.Done()

			// Send an invalid JSON response that will cause an unmarshal error
			protocol := fmt.Sprintf("actor/%s/messages/0.0.1", envelope.From.Address.InboxAddress)
			invalidJSON := []byte(`{"invalid":}`) // This will cause an unmarshal error
			_ = testNet2.SendMessage(context.Background(), envelope.From.Address.HostID,
				types.MessageEnvelope{
					Type: types.MessageType(protocol),
					Data: invalidJSON,
				},
				time.Now().Add(5*time.Second))
		}, nil)
		assert.NoError(t, err)

		envelopeJSON, err := json.Marshal(envelope)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(envelopeJSON))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		server.ActorBroadcast(c)

		wg.Wait()
		assert.Equal(t, http.StatusOK, w.Code)

		// The response should be an empty array since the unmarshal error prevented adding to messages
		var responses []actor.Envelope
		err = json.Unmarshal(w.Body.Bytes(), &responses)
		assert.NoError(t, err)
		assert.Empty(t, responses)
	})

	t.Run("must fail if the message is not a broadcast", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet := setupTestNetwork(t, substrate)
		server := NewServer(&ServerConfig{
			P2P:  testNet,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					InboxAddress: "test-inbox",
				},
			},
			To: actor.Handle{
				Address: actor.Address{
					HostID: "some-host",
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(time.Minute).UnixNano()),
				Topic:  "test-topic",
			},
		}

		envelopeJSON, _ := json.Marshal(envelope)

		req, _ := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		server.ActorBroadcast(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["error"])
	})

	t.Run("must be able to broadcast message", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet1 := setupTestNetwork(t, substrate)
		testNet2 := setupTestNetwork(t, substrate)

		server := NewServer(&ServerConfig{
			P2P:  testNet1,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet1.GetHostID().String(),
					InboxAddress: TestInbox1Name,
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(5 * time.Second).UnixNano()),
				Topic:  TestTopicName,
			},
			Message: []byte("broadcast message"),
		}
		responseEnvelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet2.GetHostID().String(),
					InboxAddress: TestInbox2Name,
				},
			},
			To:      envelope.From,
			Message: []byte("broadcast response"),
		}

		testnet1Protocol := createProtocolString(envelope.From.Address.InboxAddress)
		responseChannel := make(chan actor.Envelope, 1)
		err := testNet1.HandleMessage(testnet1Protocol, func(data []byte, _ peer.ID) {
			var respEnvelope actor.Envelope
			if err := json.Unmarshal(data, &respEnvelope); err != nil {
				return
			}
			responseChannel <- respEnvelope
		})
		assert.NoError(t, err)

		_, err = testNet2.Subscribe(context.Background(), TestTopicName, func(data []byte) {
			// When testNet2 receives a broadcast message, it sends a response back to testNet1
			var broadcastEnvelope actor.Envelope
			if err := json.Unmarshal(data, &broadcastEnvelope); err != nil {
				return
			}

			responseData, _ := json.Marshal(responseEnvelope)
			err := testNet2.SendMessage(context.Background(), broadcastEnvelope.From.Address.HostID,
				types.MessageEnvelope{
					Type: types.MessageType(testnet1Protocol),
					Data: responseData,
				},
				time.Now().Add(time.Minute))
			assert.NoError(t, err)
		}, nil)
		assert.NoError(t, err)

		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		server.ActorBroadcast(c)
		assert.Equal(t, http.StatusOK, w.Code)

		var responses []actor.Envelope
		err = json.Unmarshal(w.Body.Bytes(), &responses)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(responses))
		assert.Equal(t, responseEnvelope.From.Address.HostID, responses[0].From.Address.HostID)
		assert.Equal(t, responseEnvelope.From.Address.InboxAddress, responses[0].From.Address.InboxAddress)
		assert.Equal(t, string(responseEnvelope.Message), string(responses[0].Message))
	})

	t.Run("must ignore messages received after expiry", func(t *testing.T) {
		t.Parallel()
		gin.SetMode(gin.TestMode)
		substrate := network.NewSubstrate()
		testNet1 := setupTestNetwork(t, substrate)
		testNet2 := setupTestNetwork(t, substrate)

		server := NewServer(&ServerConfig{
			P2P:  testNet1,
			Port: 8080,
			Addr: "localhost",
		})

		envelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet1.GetHostID().String(),
					InboxAddress: TestInbox1Name,
				},
			},
			Options: actor.EnvelopeOptions{
				Expire: uint64(time.Now().Add(5 * time.Second).UnixNano()),
				Topic:  TestTopicName,
			},
			Message: []byte("broadcast message"),
		}
		responseEnvelope := actor.Envelope{
			From: actor.Handle{
				Address: actor.Address{
					HostID:       testNet2.GetHostID().String(),
					InboxAddress: TestInbox2Name,
				},
			},
			To:      envelope.From,
			Message: []byte("broadcast response"),
		}

		testnet1Protocol := createProtocolString(envelope.From.Address.InboxAddress)
		_, err := testNet2.Subscribe(context.Background(), TestTopicName, func(data []byte) {
			// When testNet2 receives a broadcast message, it sends a response back to testNet1
			var broadcastEnvelope actor.Envelope
			if err := json.Unmarshal(data, &broadcastEnvelope); err != nil {
				return
			}

			// Sleep for msg.Expire + 1 second
			time.Sleep(time.Duration(broadcastEnvelope.Options.Expire+uint64(time.Now().UnixNano())) + time.Second)

			responseData, _ := json.Marshal(responseEnvelope)
			err := testNet2.SendMessage(context.Background(), broadcastEnvelope.From.Address.HostID,
				types.MessageEnvelope{
					Type: types.MessageType(testnet1Protocol),
					Data: responseData,
				},
				time.Now().Add(time.Minute))
			assert.NoError(t, err)
		}, nil)
		assert.NoError(t, err)

		envelopeJSON, _ := json.Marshal(envelope)
		req, _ := http.NewRequest("POST", "/api/v1/actor/broadcast", bytes.NewBuffer(envelopeJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		server.ActorBroadcast(c)
		assert.Equal(t, http.StatusOK, w.Code)

		var responses []actor.Envelope
		err = json.Unmarshal(w.Body.Bytes(), &responses)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(responses))
	})
}
