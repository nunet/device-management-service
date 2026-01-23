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
	"github.com/stretchr/testify/require"
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
	tests := []struct {
		name         string
		req          node.CapTokenAnchorRequest
		resp         any
		anchor       string
		opts         []client.Option
		expectedPath string
		wantErr      bool
	}{
		{
			"success",
			node.CapTokenAnchorRequest{
				Token: ucan.TokenList{},
			},
			node.CapAnchorResponse{
				OK:    true,
				Error: "",
			},
			"provide",
			nil,
			"/actor/invoke",
			false,
		},
		{
			"success",
			node.CapTokenAnchorRequest{
				Token: ucan.TokenList{},
			},
			node.CapAnchorResponse{
				OK:    true,
				Error: "",
			},
			"require",
			nil,
			"/actor/invoke",
			false,
		},
		{
			"success",
			node.CapTokenAnchorRequest{
				Token: ucan.TokenList{},
			},
			node.CapAnchorResponse{
				OK:    true,
				Error: "",
			},
			"revoke",
			nil,
			"/actor/invoke",
			false,
		},
		{
			"success",
			node.CapTokenAnchorRequest{
				Token: ucan.TokenList{},
			},
			[]node.CapAnchorResponse{
				{
					OK:    true,
					Error: "",
				},
			},
			"revoke/broadcast",
			nil,
			"/actor/broadcast",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *client.Client
			var err error

			switch tt.anchor {
			case "provide":
				c, _, err = makeMockBehaviorClient(t, tt.expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
					assert.Equal(t, envelope.Behavior, behaviors.ProvideCapAnchorBehavior, "unexpected behavior")
					return 200, tt.resp
				})
				require.NoError(t, err)

				result, err := c.ProvideCapAnchor(context.Background(), tt.req, tt.opts...)
				if tt.wantErr {
					assert.Error(t, err)
				}
				assert.NoError(t, err)
				assert.Equal(t, result, tt.resp)
			case "require":
				c, _, err = makeMockBehaviorClient(t, tt.expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
					assert.Equal(t, envelope.Behavior, behaviors.RequireCapAnchorBehavior, "unexpected behavior")
					return 200, tt.resp
				})
				require.NoError(t, err)

				result, err := c.RequireCapAnchor(context.Background(), tt.req, tt.opts...)
				if tt.wantErr {
					assert.Error(t, err)
				}
				assert.NoError(t, err)
				assert.Equal(t, result, tt.resp)
			case "revoke":
				c, _, err = makeMockBehaviorClient(t, tt.expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
					assert.Equal(t, envelope.Behavior, behaviors.RevokeCapAnchorBehavior, "unexpected behavior")
					return 200, tt.resp
				})
				require.NoError(t, err)

				assert.NoError(t, err, "create client")
				result, err := c.RevokeCapAnchor(context.Background(), tt.req, tt.opts...)
				if tt.wantErr {
					assert.Error(t, err)
				}
				assert.NoError(t, err)
				assert.Equal(t, result, tt.resp)
			case "revoke/broadcast":
				c, _, err = makeMockBehaviorClient(t, tt.expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
					assert.Equal(t, envelope.Behavior, behaviors.BroadcastRevokeCapBehavior, "unexpected behavior")
					resp := toAnySlice(tt.resp.([]node.CapAnchorResponse))
					return 200, resp
				})
				require.NoError(t, err)

				assert.NoError(t, err, "create client")
				result, err := c.BroadcastCapRevoke(context.Background(), tt.req, tt.opts...)
				if tt.wantErr {
					assert.Error(t, err)
				}
				assert.NoError(t, err)
				expectedResp := tt.resp.([]node.CapAnchorResponse)
				assert.Equal(t, expectedResp, result)
			}
		})
	}
}
