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

type mockActorSendClient struct {
	client.DmsClient
	sendMessageRawFn func(ctx context.Context, req actor.Envelope) (actor.Envelope, error)
}

func (m *mockActorSendClient) SendMessageRaw(ctx context.Context, req actor.Envelope) (actor.Envelope, error) {
	if m.sendMessageRawFn == nil {
		panic("SendMessageRaw called but not set on mock")
	}
	return m.sendMessageRawFn(ctx, req)
}

func TestActorSendCmd_InvalidArgs(t *testing.T) {
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
			// If setupTest requires a non-nil client, use &mockActorSendClient{}.
			dmsCli := setupTest(t, nil)

			_, _, err := utils.ExecuteCommand(newActorSendCmd(dmsCli), tt.args...)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActorSendCmd_ClientError(t *testing.T) {
	t.Parallel()

	t.Run("client creation error", func(t *testing.T) {
		t.Parallel()

		// Create a valid message
		msg, _ := json.Marshal(actor.Envelope{
			Behavior: "test",
		})

		dmsCli := utils.NewTestCli(cli.WithClientFn(
			func(_ *config.Config, _ actor.SecurityContext) (client.DmsClient, error) {
				return nil, errors.New("fail client")
			},
		))

		_, _, err := utils.ExecuteCommand(newActorSendCmd(dmsCli), string(msg))
		assert.ErrorContains(t, err, "could not create client")
	})

	t.Run("send error", func(t *testing.T) {
		t.Parallel()

		// Create a valid message
		msg, _ := json.Marshal(actor.Envelope{
			Behavior: "test",
		})

		dmsCli := setupTest(t, &mockActorSendClient{
			sendMessageRawFn: func(_ context.Context, _ actor.Envelope) (actor.Envelope, error) {
				return actor.Envelope{}, errors.New("fail send")
			},
		})

		_, _, err := utils.ExecuteCommand(newActorSendCmd(dmsCli), string(msg))
		assert.ErrorContains(t, err, "could not send message")
	})
}

func TestActorSendCmd_Success(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		// Create a valid message
		msg, _ := json.Marshal(actor.Envelope{
			Behavior: "test",
		})

		dmsCli := setupTest(t, &mockActorSendClient{
			sendMessageRawFn: func(_ context.Context, _ actor.Envelope) (actor.Envelope, error) {
				return actor.Envelope{
					Message: json.RawMessage(`{"result":"success"}`),
				}, nil
			},
		})

		out, _, err := utils.ExecuteCommand(newActorSendCmd(dmsCli), string(msg))
		require.NoError(t, err)
		assert.Contains(t, out, "success")
	})
}
