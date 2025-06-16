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

type mockAllocationsBehaviorClient struct {
	client.DmsClient
	allocationsListFn func(ctx context.Context, opts ...client.Option) (node.AllocationsListResponse, error)
}

func (m *mockAllocationsBehaviorClient) AllocationsList(ctx context.Context, opts ...client.Option) (node.AllocationsListResponse, error) {
	return m.allocationsListFn(ctx, opts...)
}

func TestAllocationsListBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockAllocationsBehaviorClient{
				allocationsListFn: func(_ context.Context, opts ...client.Option) (node.AllocationsListResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.AllocationsListResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.AllocationsListBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
