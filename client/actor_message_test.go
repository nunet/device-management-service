// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
)

func TestClient_NewActorMessage(t *testing.T) {
	expectedPath := client.ActorHandleEndpoint

	tests := []struct {
		name     string
		behavior string
		payload  any
		msgOpts  client.MessageOptions
		wantErr  bool
	}{
		{
			name:     "basic message",
			behavior: "test.behavior",
			payload:  map[string]string{"key": "value"},
			msgOpts:  client.MessageOptions{},
			wantErr:  false,
		},
		{
			name:     "message with topic",
			behavior: "test.topic",
			msgOpts: client.MessageOptions{
				Topic: "test/topic",
			},
			wantErr: false,
		},
		{
			name:     "message with expiry",
			behavior: "test.expiry",
			msgOpts: client.MessageOptions{
				Expiry: time.Now().Add(time.Minute),
			},
			wantErr: false,
		},
		{
			name:     "message with timeout",
			behavior: "test.timeout",
			msgOpts: client.MessageOptions{
				Timeout: time.Minute,
			},
			wantErr: false,
		},
		{
			name:     "reply to overrides invocation reply to",
			behavior: "test.replyto",
			msgOpts: client.MessageOptions{
				ReplyTo:      "/test/reply/path",
				IsInvocation: true,
			},
			wantErr: false,
		},
		{
			name:     "invocation message",
			behavior: "test.invoke",
			msgOpts: client.MessageOptions{
				IsInvocation: true,
			},
			wantErr: false,
		},
		{
			name:     "message with reply to",
			behavior: "test.invoke",
			msgOpts: client.MessageOptions{
				ReplyTo: "/test/reply/path",
			},
			wantErr: false,
		},
		{
			name:     "broadcast message",
			behavior: "test.broadcast",
			msgOpts: client.MessageOptions{
				Topic: "/test/topic",
			},
			wantErr: false,
		},
		{
			name:     "message with destination",
			behavior: "test.dest",
			msgOpts: client.MessageOptions{
				Destination: makeSecurityContext(t).DID().String(),
			},
			wantErr: false,
		},
		{
			name:     "message with invalid destination",
			behavior: "test.dest",
			msgOpts: client.MessageOptions{
				Destination: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mocked DMS handle endpoint
			c, sctx, err := makeMockBehaviorClient(t, expectedPath, func(_ *testing.T, _ *actor.Envelope) (int, any) {
				// This function shouldn't be called in this test
				return 200, nil
			})
			require.NoError(t, err, "create client")

			dmsHandle, err := c.GetDMSHandle(context.Background())
			assert.NoError(t, err)

			msg, err := c.NewActorMessage(context.Background(), tt.behavior, tt.payload, tt.msgOpts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.False(t, msg.From.Empty(), "message should have a sender")
			assert.Equal(t, sctx.ID(), msg.From.ID)
			assert.Equal(t, sctx.DID(), msg.From.DID)
			assert.Equal(t, dmsHandle.Address.HostID, msg.From.Address.HostID, "the hostID must be dms hostID")
			assert.Equal(t, tt.behavior, msg.Behavior)

			if tt.payload != nil {
				data, err := json.Marshal(tt.payload)
				assert.NoError(t, err, "marshal payload")
				assert.Equal(t, data, msg.Message)
			}

			if !msg.Expired() {
				assert.NoError(t, sctx.Verify(msg), "message should be signed by the client")
			}

			if tt.msgOpts.Topic != "" {
				assert.True(t, msg.IsBroadcast(), "message should be broadcast")
				assert.Equal(t, tt.msgOpts.Topic, msg.Options.Topic)
				if tt.msgOpts.ReplyTo == "" {
					assert.True(t, strings.HasPrefix(msg.Options.ReplyTo, "/public"), "Broadcast message should have a public reply-to")
				}
				assert.True(t, msg.To.Empty())
			} else {
				assert.False(t, msg.To.Empty(), "message should have a recipient")
				if tt.msgOpts.Destination != "" {
					dest, err := actor.HandleFromDID(tt.msgOpts.Destination)
					assert.NoError(t, err, "create destination handle")
					assert.Equal(t, dest, msg.To, "message should have the specified destination")
				} else {
					assert.Equal(t, dmsHandle, msg.To, "destination should be the DMS handle")
				}
			}

			if tt.msgOpts.IsInvocation && tt.msgOpts.ReplyTo == "" {
				assert.True(t, strings.HasPrefix(msg.Options.ReplyTo, "/private"), "invocation message should have a private reply-to")
			}

			if tt.msgOpts.ReplyTo != "" {
				assert.Equal(t, tt.msgOpts.ReplyTo, msg.Options.ReplyTo, "message should have the specified reply-to")
			}

			if !tt.msgOpts.Expiry.IsZero() {
				assert.Equal(t, tt.msgOpts.Expiry.UnixNano(), msg.Expiry().UnixNano(), "message should have the specified expiry")
			}

			if tt.msgOpts.Timeout > 0 {
				assert.Equal(t, time.Now().Add(tt.msgOpts.Timeout).Round(time.Second).UnixNano(), msg.Expiry().Round(time.Second).UnixNano(), "message should have the specified timeout")
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	expectedPath := client.ActorSendMessageEndpoint
	tests := []struct {
		name     string
		behavior string
		payload  any
		options  []client.Option
		wantErr  bool
	}{
		{
			name:     "basic send",
			behavior: "test.send",
			payload:  map[string]string{"key": "value"},
			wantErr:  false,
		},
		{
			name:     "timeout",
			behavior: "test.send.timeout",
			payload:  "test payload",
			options: []client.Option{
				client.WithTimeout(time.Nanosecond),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mocked endpoint
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, tt.behavior, envelope.Behavior)
				return 200, envelope
			})
			require.NoError(t, err, "create client")

			// Call the function
			_, err = c.SendMessage(context.Background(), tt.behavior, tt.payload, tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClient_InvokeBehavior(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	tests := []struct {
		name     string
		behavior string
		payload  any
		options  []client.Option
		wantErr  bool
	}{
		{
			name:     "basic invoke",
			behavior: "test.invoke",
			payload:  map[string]string{"key": "value"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mocked endpoint
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, tt.behavior, envelope.Behavior)
				assert.Contains(t, envelope.Options.ReplyTo, "/private/user/")
				return 200, envelope
			})
			require.NoError(t, err, "create client")

			// Call the function
			_, err = c.InvokeBehavior(context.Background(), tt.behavior, tt.payload, tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClient_BroadcastMessage(t *testing.T) {
	expectedPath := client.ActorBroadcastEndpoint
	tests := []struct {
		name     string
		behavior string
		topic    string
		payload  interface{}
		options  []client.Option
		wantErr  bool
	}{
		{
			name:     "basic broadcast",
			behavior: "test.broadcast",
			topic:    "test/topic",
			payload:  map[string]string{"key": "value"},
			wantErr:  false,
		},
		{
			name:     "missing topic",
			behavior: "test.broadcast",
			topic:    "",
			payload:  "test payload",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mocked endpoint
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, tt.behavior, envelope.Behavior)
				assert.Contains(t, envelope.Options.ReplyTo, "/public")
				assert.Equal(t, tt.topic, envelope.Options.Topic)
				assert.True(t, envelope.IsBroadcast())

				// Return two mock responses
				return 200, []any{"response1", "response2"}
			})
			require.NoError(t, err, "create client")

			// Call the function
			responses, err := c.BroadcastMessage(context.Background(), tt.behavior, tt.topic, tt.payload, tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, responses, 2)
		})
	}
}
