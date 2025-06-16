package actor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockPublicBehaviorClient struct {
	client.DmsClient
	helloFn              func(ctx context.Context, opts ...client.Option) (node.HelloResponse, error)
	broadcastHelloFn     func(ctx context.Context, opts ...client.Option) ([]node.HelloResponse, error)
	statusFn             func(ctx context.Context, opts ...client.Option) (node.PublicStatusResponse, error)
	discoveryFn          func(ctx context.Context, opts ...client.Option) (node.DiscoveryStatusResponse, error)
	discoveryBroadcastFn func(ctx context.Context, opts ...client.Option) ([]node.DiscoveryStatusResponse, error)
}

func (m *mockPublicBehaviorClient) Hello(ctx context.Context, opts ...client.Option) (node.HelloResponse, error) {
	return m.helloFn(ctx, opts...)
}

func (m *mockPublicBehaviorClient) BroadcastHello(ctx context.Context, opts ...client.Option) ([]node.HelloResponse, error) {
	return m.broadcastHelloFn(ctx, opts...)
}

func (m *mockPublicBehaviorClient) Status(ctx context.Context, opts ...client.Option) (node.PublicStatusResponse, error) {
	return m.statusFn(ctx, opts...)
}

func (m *mockPublicBehaviorClient) Discovery(ctx context.Context, opts ...client.Option) (node.DiscoveryStatusResponse, error) {
	return m.discoveryFn(ctx, opts...)
}

func (m *mockPublicBehaviorClient) DiscoveryBroadcast(ctx context.Context, opts ...client.Option) ([]node.DiscoveryStatusResponse, error) {
	return m.discoveryBroadcastFn(ctx, opts...)
}

func TestPublicHelloBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPublicBehaviorClient{
				helloFn: func(_ context.Context, opts ...client.Option) (node.HelloResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.HelloResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PublicHelloBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBroadcastHelloBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPublicBehaviorClient{
				broadcastHelloFn: func(_ context.Context, opts ...client.Option) ([]node.HelloResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return []node.HelloResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.BroadcastHelloBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPublicStatusBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPublicBehaviorClient{
				statusFn: func(_ context.Context, opts ...client.Option) (node.PublicStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.PublicStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.PublicStatusBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStatusDiscoveryBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPublicBehaviorClient{
				discoveryFn: func(_ context.Context, opts ...client.Option) (node.DiscoveryStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.DiscoveryStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.StatusDiscoveryBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBroadcastStatusDiscoveryBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockPublicBehaviorClient{
				discoveryBroadcastFn: func(_ context.Context, opts ...client.Option) ([]node.DiscoveryStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return []node.DiscoveryStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.BroadcastStatusDiscoveryBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
