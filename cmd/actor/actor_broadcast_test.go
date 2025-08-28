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
	"encoding/json"
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

type mockActorBroadcastClient struct {
	client.DmsClient
	broadcastMessageRawFn func(ctx context.Context, req actor.Envelope) ([]actor.Envelope, error)
}

func (m *mockActorBroadcastClient) BroadcastMessageRaw(ctx context.Context, req actor.Envelope) ([]actor.Envelope, error) {
	if m.broadcastMessageRawFn == nil {
		panic("BroadcastMessageRaw called but not set on mock")
	}
	return m.broadcastMessageRawFn(ctx, req)
}

func TestActorBroadcastCmd_InvalidArgs(t *testing.T) {
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
			name:      "invalid json argument",
			args:      []string{"not-json"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Assuming setupTest can handle a nil client if no client methods are called by the command.
			// If setupTest requires a non-nil client, use &mockActorBroadcastClient{}.
			dmsCli := setupTest(t, nil)

			_, _, err := utils.ExecuteCommand(newActorBroadcastCmd(dmsCli), tt.args...)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActorBroadcastCmd_ClientError(t *testing.T) {
	t.Parallel()
	t.Run("client creation error", func(t *testing.T) {
		t.Parallel()

		msg, _ := json.Marshal(actor.Envelope{})

		dmsCli := utils.NewTestCli(cli.WithClientFn(
			func(_ *config.Config, _ actor.SecurityContext) (client.DmsClient, error) {
				return nil, errors.New("fail client")
			},
		))

		_, _, err := utils.ExecuteCommand(newActorBroadcastCmd(dmsCli), string(msg))
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not create client")
	})

	t.Run("broadcast error", func(t *testing.T) {
		t.Parallel()

		msg, _ := json.Marshal(actor.Envelope{})
		dmsCli := setupTest(t, &mockActorBroadcastClient{
			broadcastMessageRawFn: func(_ context.Context, _ actor.Envelope) ([]actor.Envelope, error) {
				return nil, errors.New("fail broadcast")
			},
		})

		_, _, err := utils.ExecuteCommand(newActorBroadcastCmd(dmsCli), string(msg))
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not broadcast message")
	})
}

func TestActorBroadcastCmd_Success(t *testing.T) {
	t.Parallel()

	dmsCli := setupTest(t, &mockActorBroadcastClient{
		broadcastMessageRawFn: func(_ context.Context, _ actor.Envelope) ([]actor.Envelope, error) {
			return []actor.Envelope{
				{
					Behavior: "behavior1",
				},
				{
					Behavior: "behavior2",
				},
			}, nil
		},
	})

	msg, _ := json.Marshal(actor.Envelope{})

	out, _, err := utils.ExecuteCommand(newActorBroadcastCmd(dmsCli), string(msg))
	require.NoError(t, err)
	assert.Contains(t, out, "behavior1")
	assert.Contains(t, out, "behavior2")
}
