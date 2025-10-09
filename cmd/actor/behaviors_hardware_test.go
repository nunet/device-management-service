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

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockHardwareBehaviorClient struct {
	client.DmsClient
	hardwareSpecFn  func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
	hardwareUsageFn func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
}

func (m *mockHardwareBehaviorClient) HardwareSpec(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.hardwareSpecFn(ctx, opts...)
}

func (m *mockHardwareBehaviorClient) HardwareUsage(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.hardwareUsageFn(ctx, opts...)
}

func TestHardwareSpecBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockHardwareBehaviorClient{
				hardwareSpecFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.HardwareSpecBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHardwareUsageBehavior(t *testing.T) {
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

			dmsCli := setupTest(t, &mockHardwareBehaviorClient{
				hardwareUsageFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.HardwareUsageBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
