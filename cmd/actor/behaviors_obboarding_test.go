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

type mockOnboardingBehaviorClient struct {
	client.DmsClient
	offboardFn      func(ctx context.Context, req node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error)
	onboardStatusFn func(ctx context.Context, opts ...client.Option) (node.OnboardStatusResponse, error)
}

func (m *mockOnboardingBehaviorClient) Offboard(ctx context.Context, req node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error) {
	return m.offboardFn(ctx, req, opts...)
}

func (m *mockOnboardingBehaviorClient) OnboardStatus(ctx context.Context, opts ...client.Option) (node.OnboardStatusResponse, error) {
	return m.onboardStatusFn(ctx, opts...)
}

func TestOffboardBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockOnboardingBehaviorClient{
				offboardFn: func(_ context.Context, _ node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.OffboardResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.OffboardBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOnboardStatusBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockOnboardingBehaviorClient{
				onboardStatusFn: func(_ context.Context, opts ...client.Option) (node.OnboardStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.OnboardStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.OnboardStatusBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
