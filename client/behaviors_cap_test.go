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
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestClient_CapList(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.CapListBehavior
	tests := []struct {
		name    string
		req     node.CapListRequest
		resp    node.CapListResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.CapListRequest{
				Context: "dms",
			},
			node.CapListResponse{
				OK:      true,
				Error:   "",
				Roots:   []did.DID{},
				Require: ucan.TokenList{},
				Provide: ucan.TokenList{},
				Revoke:  ucan.TokenList{},
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.CapList(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_CapAnchor(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.CapAnchorBehavior
	tests := []struct {
		name    string
		req     node.CapAnchorRequest
		resp    node.CapAnchorResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.CapAnchorRequest{
				Root:    []did.DID{},
				Require: ucan.TokenList{},
				Provide: ucan.TokenList{},
				Revoke:  ucan.TokenList{},
			},
			node.CapAnchorResponse{
				OK:    true,
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior, "unexpected behavior")
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.CapAnchor(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
