// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockCapBehaviorClient struct {
	client.DmsClient
	capListFn   func(ctx context.Context, req node.CapListRequest, opts ...client.Option) (node.CapListResponse, error)
	capAnchorFn func(ctx context.Context, req node.CapAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error)
}

func (m *mockCapBehaviorClient) CapList(ctx context.Context, req node.CapListRequest, opts ...client.Option) (node.CapListResponse, error) {
	return m.capListFn(ctx, req, opts...)
}

func (m *mockCapBehaviorClient) CapAnchor(ctx context.Context, req node.CapAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
	return m.capAnchorFn(ctx, req, opts...)
}

func TestCapListBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		opts    client.MessageOptions
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockCapBehaviorClient{
				capListFn: func(_ context.Context, _ node.CapListRequest, opts ...client.Option) (node.CapListResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.CapListResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.CapListBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCapAnchorBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.CapAnchorRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "multiple anchors",
			args: []string{
				"--root",
				"--require",
				"--provide",
				"--revoke",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no data",
			args: []string{
				"--root",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid root",
			args: []string{
				"--root",
				"did:example:123",
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.CapAnchorRequest{
				Root: []did.DID{{
					URI: "did:example:123",
				}},
				Require: ucan.TokenList{Tokens: []*ucan.Token{}},
				Provide: ucan.TokenList{Tokens: []*ucan.Token{}},
				Revoke:  ucan.TokenList{Tokens: []*ucan.Token{}},
			},
			wantErr: false,
		},
		{
			name: "invalid root",
			args: []string{
				"--root",
				"invalid",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid require",
			args: []string{
				"--require",
				`{"dms":{"act":"require"}}`,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.CapAnchorRequest{
				Root: []did.DID{},
				Require: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "require",
						},
					}},
				},
				Provide: ucan.TokenList{Tokens: []*ucan.Token{}},
				Revoke:  ucan.TokenList{Tokens: []*ucan.Token{}},
			},
			wantErr: false,
		},
		{
			name: "invalid require",
			args: []string{
				"--require",
				"invalid",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid provide",
			args: []string{
				"--provide",
				`{"dms":{"act":"provide"}}`,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.CapAnchorRequest{
				Root:    []did.DID{},
				Require: ucan.TokenList{Tokens: []*ucan.Token{}},
				Provide: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "provide",
						},
					}},
				},
				Revoke: ucan.TokenList{Tokens: []*ucan.Token{}},
			},
			wantErr: false,
		},
		{
			name: "invalid provide",
			args: []string{
				"--provide",
				"invalid",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid revoke",
			args: []string{
				"--revoke",
				`{"dms":{"act":"revoke"}}`,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.CapAnchorRequest{
				Root:    []did.DID{},
				Require: ucan.TokenList{Tokens: []*ucan.Token{}},
				Provide: ucan.TokenList{Tokens: []*ucan.Token{}},
				Revoke: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "revoke",
						},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid revoke",
			args: []string{
				"--revoke",
				"invalid",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockCapBehaviorClient{
				capAnchorFn: func(_ context.Context, req node.CapAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.CapAnchorResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.CapAnchorBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
