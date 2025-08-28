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

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockPeersBehaviorClient struct {
	client.DmsClient
	peersSelfFn        func(ctx context.Context, opts ...client.Option) (node.PeerAddrInfoResponse, error)
	peersListFn        func(ctx context.Context, opts ...client.Option) (node.PeersListResponse, error)
	peersListFromDHTFn func(ctx context.Context, opts ...client.Option) (node.PeerDHTResponse, error)
	peerPingFn         func(ctx context.Context, req node.PingRequest, opts ...client.Option) (node.PingResponse, error)
	peerConnectFn      func(ctx context.Context, req node.PeerConnectRequest, opts ...client.Option) (node.PeerConnectResponse, error)
	peerScoreFn        func(ctx context.Context, opts ...client.Option) (node.PeerScoreResponse, error)
}

func (m *mockPeersBehaviorClient) PeersSelf(ctx context.Context, opts ...client.Option) (node.PeerAddrInfoResponse, error) {
	return m.peersSelfFn(ctx, opts...)
}

func (m *mockPeersBehaviorClient) PeersList(ctx context.Context, opts ...client.Option) (node.PeersListResponse, error) {
	return m.peersListFn(ctx, opts...)
}

func (m *mockPeersBehaviorClient) PeersListFromDHT(ctx context.Context, opts ...client.Option) (node.PeerDHTResponse, error) {
	return m.peersListFromDHTFn(ctx, opts...)
}

func (m *mockPeersBehaviorClient) PeerPing(ctx context.Context, req node.PingRequest, opts ...client.Option) (node.PingResponse, error) {
	return m.peerPingFn(ctx, req, opts...)
}

func (m *mockPeersBehaviorClient) PeerConnect(ctx context.Context, req node.PeerConnectRequest, opts ...client.Option) (node.PeerConnectResponse, error) {
	return m.peerConnectFn(ctx, req, opts...)
}

func (m *mockPeersBehaviorClient) PeerScore(ctx context.Context, opts ...client.Option) (node.PeerScoreResponse, error) {
	return m.peerScoreFn(ctx, opts...)
}

func TestPeersSelfBehavior(t *testing.T) {
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
		{
			name:    "valid did",
			args:    []string{"--context", "user", "--dest", "did:key:z6MkwQpRz8b7vJY4N1k2"},
			opts:    client.NewMessageOptions(client.WithDestination("did:key:z6MkwQpRz8b7vJY4N1k2")),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peersSelfFn: func(_ context.Context, opts ...client.Option) (node.PeerAddrInfoResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.PeerAddrInfoResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeerAddrInfoBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPeersListBehavior(t *testing.T) {
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
		{
			name:    "valid did",
			args:    []string{"--context", "user", "--dest", "did:key:z6MkwQpRz8b7vJY4N1k2"},
			opts:    client.NewMessageOptions(client.WithDestination("did:key:z6MkwQpRz8b7vJY4N1k2")),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peersListFn: func(_ context.Context, opts ...client.Option) (node.PeersListResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.PeersListResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeersListBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPeersListFromDHTBehavior(t *testing.T) {
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
		{
			name:    "valid did",
			args:    []string{"--context", "user", "--dest", "did:key:z6MkwQpRz8b7vJY4N1k2"},
			opts:    client.NewMessageOptions(client.WithDestination("did:key:z6MkwQpRz8b7vJY4N1k2")),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peersListFromDHTFn: func(_ context.Context, opts ...client.Option) (node.PeerDHTResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.PeerDHTResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeerDHTBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPeerPingBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.PingRequest
		wantErr     bool
	}{
		{
			name:        "no args",
			args:        []string{},
			opts:        client.NewMessageOptions(),
			expectedReq: node.PingRequest{},
			wantErr:     true,
		},
		{
			name: "valid host",
			args: []string{"--host", "localhost"},
			opts: client.NewMessageOptions(),
			expectedReq: node.PingRequest{
				Host: "localhost",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peerPingFn: func(_ context.Context, req node.PingRequest, opts ...client.Option) (node.PingResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.PingResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeerPingBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPeersConnectBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.PeerConnectRequest
		wantErr     bool
	}{
		{
			name:        "no args",
			args:        []string{},
			opts:        client.NewMessageOptions(),
			expectedReq: node.PeerConnectRequest{},
			wantErr:     true,
		},
		{
			name: "valid host",
			args: []string{"--address", "localhost"},
			opts: client.NewMessageOptions(),
			expectedReq: node.PeerConnectRequest{
				Address: "localhost",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peerConnectFn: func(_ context.Context, req node.PeerConnectRequest, opts ...client.Option) (node.PeerConnectResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.PeerConnectResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeerConnectBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPeerScoreBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPeersBehaviorClient{
				peerScoreFn: func(_ context.Context, opts ...client.Option) (node.PeerScoreResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.PeerScoreResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PeerScoreBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
