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
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
)

func TestClient_Hello(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PublicHelloBehavior
	tests := []struct {
		name    string
		resp    node.HelloResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.HelloResponse{
				DID: did.DID{},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.Hello(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_BroadcastHello(t *testing.T) {
	expectedPath := client.ActorBroadcastEndpoint
	expectedBehavior := behaviors.BroadcastHelloBehavior
	tests := []struct {
		name    string
		resps   []node.HelloResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			[]node.HelloResponse{},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert HelloResponse slice to interface{} slice for the mock
			resps := make([]any, len(tt.resps))
			for i, r := range tt.resps {
				resps[i] = r
			}

			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.True(t, envelope.IsBroadcast())
				assert.Equal(t, envelope.Options.Topic, behaviors.BroadcastHelloTopic)
				return 200, resps
			})
			assert.NoError(t, err, "create client")

			result, err := c.BroadcastHello(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, len(result), len(tt.resps))
			for i, r := range result {
				assert.Equal(t, r, tt.resps[i])
			}
		})
	}
}

func TestClient_Status(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PublicStatusBehavior
	tests := []struct {
		name    string
		resp    node.PublicStatusResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PublicStatusResponse{
				Status:    "",
				Resources: types.Resources{},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.Status(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
