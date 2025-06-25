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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
)

type mockActorMsgClient struct {
	client.DmsClient
	newActorMessageFn func(ctx context.Context, behavior string, payload any, opts client.MessageOptions) (actor.Envelope, error)
}

func (m *mockActorMsgClient) NewActorMessage(ctx context.Context, behavior string, payload any, opts client.MessageOptions) (actor.Envelope, error) {
	if m.newActorMessageFn == nil {
		panic("NewActorMessage called but not set on mock")
	}
	return m.newActorMessageFn(ctx, behavior, payload, opts)
}

func TestActorMsgCmd_InvalidArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name:      "no args",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "one arg only",
			args:      []string{"/test/behavior"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Assuming setupTest can handle a nil client if no client methods are called by the command.
			// If setupTest requires a non-nil client, use &mockActorMsgClient{}.
			dmsCli := setupTest(t, nil)

			_, _, err := utils.ExecuteCommand(newActorMsgCmd(dmsCli), tt.args...)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActorMsgCmd_ClientError(t *testing.T) {
	t.Parallel()

	t.Run("client creation error", func(t *testing.T) {
		t.Parallel()

		dmsCli := utils.NewTestCli(cli.WithClientFn(
			func(_ *config.Config, _ actor.SecurityContext) (client.DmsClient, error) {
				return nil, errors.New("fail client")
			},
		))

		_, _, err := utils.NewCapabilityContext(dmsCli, utils.DefaultUserContextName)
		require.NoError(t, err)

		_, _, err = utils.ExecuteCommand(newActorMsgCmd(dmsCli), "test", "payload")
		assert.ErrorContains(t, err, "could not create client")
	})

	t.Run("new message error", func(t *testing.T) {
		t.Parallel()

		dmsCli := setupTest(t, &mockActorMsgClient{
			newActorMessageFn: func(context.Context, string, any, client.MessageOptions) (actor.Envelope, error) {
				return actor.Envelope{}, errors.New("fail new message")
			},
		})

		_, _, err := utils.ExecuteCommand(newActorMsgCmd(dmsCli), "test", "payload")
		assert.ErrorContains(t, err, "could not create message")
	})
}

func TestActorMsgCmd_Success(t *testing.T) {
	t.Parallel()

	t.Run("standard message", func(t *testing.T) {
		t.Parallel()

		dmsCli := setupTest(t, &mockActorMsgClient{
			newActorMessageFn: func(context.Context, string, any, client.MessageOptions) (actor.Envelope, error) {
				return actor.Envelope{
					Behavior: "/test/behavior",
				}, nil
			},
		})

		out, _, err := utils.ExecuteCommand(newActorMsgCmd(dmsCli), "/test/behavior", "payload")
		require.NoError(t, err)
		assert.Contains(t, out, "/test/behavior")
	})
}
