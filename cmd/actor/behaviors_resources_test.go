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

type mockResourcesBehaviorClient struct {
	client.DmsClient
	resourcesAllocatedFn func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
	resourcesFreeFn      func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
	resourcesOnboardedFn func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
}

func (m *mockResourcesBehaviorClient) ResourcesAllocated(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.resourcesAllocatedFn(ctx, opts...)
}

func (m *mockResourcesBehaviorClient) ResourcesFree(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.resourcesFreeFn(ctx, opts...)
}

func (m *mockResourcesBehaviorClient) ResourcesOnboarded(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.resourcesOnboardedFn(ctx, opts...)
}

func TestResourcesAllocatedBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockResourcesBehaviorClient{
				resourcesAllocatedFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.ResourcesAllocatedBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestResourcesFreeBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockResourcesBehaviorClient{
				resourcesFreeFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.ResourcesFreeBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestResourcesOnboardedBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockResourcesBehaviorClient{
				resourcesOnboardedFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.ResourcesOnboardedBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
