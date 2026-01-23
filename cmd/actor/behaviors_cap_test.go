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
	"gitlab.com/nunet/device-management-service/lib/ucan"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockCapBehaviorClient struct {
	client.DmsClient
	capListFn            func(ctx context.Context, req node.CapListRequest, opts ...client.Option) (node.CapListResponse, error)
	capProvideAnchorFn   func(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error)
	capRequireAnchorFn   func(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error)
	capRevokeAnchorFn    func(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error)
	capRevokeBroadcastFn func(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) ([]node.CapAnchorResponse, error)
}

func (m *mockCapBehaviorClient) CapList(ctx context.Context, req node.CapListRequest, opts ...client.Option) (node.CapListResponse, error) {
	return m.capListFn(ctx, req, opts...)
}

func (m *mockCapBehaviorClient) ProvideCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
	return m.capProvideAnchorFn(ctx, req, opts...)
}

func (m *mockCapBehaviorClient) RequireCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
	return m.capRequireAnchorFn(ctx, req, opts...)
}

func (m *mockCapBehaviorClient) RevokeCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
	return m.capRevokeAnchorFn(ctx, req, opts...)
}

func (m *mockCapBehaviorClient) BroadcastCapRevoke(ctx context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) ([]node.CapAnchorResponse, error) {
	return m.capRevokeBroadcastFn(ctx, req, opts...)
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
		anchor      string
		opts        client.MessageOptions
		expectedReq node.CapTokenAnchorRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			anchor:  "provide",
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "multiple tokens - no error but expect only first one",
			args: []string{
				"--token",
				`{"dms":{"act":"testone"}}`,
				`{"dms":{"act":"testtwo"}}`,
				`{"dms":{"act":"testthree"}}`,
			},
			anchor: "require",
			opts:   client.NewMessageOptions(),
			expectedReq: node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{
						{
							DMS: &ucan.DMSToken{
								Action: "testone",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no data",
			args: []string{
				"--token",
			},
			anchor:  "revoke",
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid require",
			args: []string{
				"--token",
				`{"dms":{"act":"delegate"}}`,
			},
			anchor: "require",
			opts:   client.NewMessageOptions(),
			expectedReq: node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "delegate",
						},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid provide",
			args: []string{
				"--token",
				"invalid",
			},
			anchor:  "provide",
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid require",
			args: []string{
				"--token",
				`{"dms":{"act":"delegate"}}`,
			},
			anchor: "require",
			opts:   client.NewMessageOptions(),
			expectedReq: node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "delegate",
						},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid revoke",
			args: []string{
				"--token",
				`{"dms":{"act":"revoke"}}`,
			},
			anchor: "revoke",
			opts:   client.NewMessageOptions(),
			expectedReq: node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{{
						DMS: &ucan.DMSToken{
							Action: "revoke",
						},
					}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var err error
			switch tt.anchor {
			case "require":
				dmsCli := setupTest(t, &mockCapBehaviorClient{
					capRequireAnchorFn: func(_ context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
						checkMessageOptions(t, tt.opts, opts...)
						assert.Equal(t, tt.expectedReq, req)
						return node.CapAnchorResponse{}, nil
					},
				})

				actorCmd := newActorCmdGroup(dmsCli)
				_, _, err = utils.ExecuteCommand(actorCmd, append([]string{behaviors.RequireCapAnchorBehavior}, tt.args...)...)
			case "provide":
				dmsCli := setupTest(t, &mockCapBehaviorClient{
					capProvideAnchorFn: func(_ context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
						checkMessageOptions(t, tt.opts, opts...)
						assert.Equal(t, tt.expectedReq, req)
						return node.CapAnchorResponse{}, nil
					},
				})

				actorCmd := newActorCmdGroup(dmsCli)
				_, _, err = utils.ExecuteCommand(actorCmd, append([]string{behaviors.ProvideCapAnchorBehavior}, tt.args...)...)
			case "revoke":
				dmsCli := setupTest(t, &mockCapBehaviorClient{
					capRevokeAnchorFn: func(_ context.Context, req node.CapTokenAnchorRequest, opts ...client.Option) (node.CapAnchorResponse, error) {
						checkMessageOptions(t, tt.opts, opts...)
						assert.Equal(t, tt.expectedReq, req)
						return node.CapAnchorResponse{}, nil
					},
				})

				actorCmd := newActorCmdGroup(dmsCli)
				_, _, err = utils.ExecuteCommand(actorCmd, append([]string{behaviors.RevokeCapAnchorBehavior}, tt.args...)...)
			}

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
