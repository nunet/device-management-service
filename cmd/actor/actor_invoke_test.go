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

type mockActorInvokeClient struct {
	client.DmsClient
	invokeBehaviorRawFn func(ctx context.Context, req actor.Envelope) (actor.Envelope, error)
}

func (m *mockActorInvokeClient) InvokeBehaviorRaw(ctx context.Context, req actor.Envelope) (actor.Envelope, error) {
	if m.invokeBehaviorRawFn == nil {
		panic("InvokeBehaviorRaw called but not set on mock")
	}
	return m.invokeBehaviorRawFn(ctx, req)
}

func TestActorInvokeCmd_InvalidArgs(t *testing.T) {
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

	// Assuming setupTest can handle a nil client if no client methods are called by the command.
	// If setupTest requires a non-nil client, use &mockActorInvokeClient{}.
	dmsCli := setupTest(t, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := utils.ExecuteCommand(newActorInvokeCmd(dmsCli), tt.args...)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActorInvokeCmd_MissingReplyTo(t *testing.T) {
	t.Parallel()

	// Create a message with no replyTo field
	msg, _ := json.Marshal(actor.Envelope{
		Behavior: "test",
	})

	// Assuming setupTest can handle a nil client if no client methods are called by the command.
	// If setupTest requires a non-nil client, use &mockActorInvokeClient{}.
	dmsCli := setupTest(t, nil)

	_, _, err := utils.ExecuteCommand(newActorInvokeCmd(dmsCli), string(msg))
	assert.ErrorContains(t, err, "missing replyTo field in message")
}

func TestActorInvokeCmd_ClientError(t *testing.T) {
	t.Parallel()

	// Create a valid message with replyTo field
	msg, _ := json.Marshal(actor.Envelope{
		Behavior: "test",
		Options: actor.EnvelopeOptions{
			ReplyTo: "test-reply",
		},
	})

	t.Run("client creation error", func(t *testing.T) {
		t.Parallel()

		dmsCli := utils.NewTestCli(cli.WithClientFn(
			func(_ *config.Config, _ actor.SecurityContext) (client.DmsClient, error) {
				return nil, errors.New("fail client")
			},
		))

		_, _, err := utils.ExecuteCommand(newActorInvokeCmd(dmsCli), string(msg))
		assert.ErrorContains(t, err, "could not create client")
	})

	t.Run("invoke error", func(t *testing.T) {
		t.Parallel()

		dmsCli := setupTest(t, &mockActorInvokeClient{
			invokeBehaviorRawFn: func(_ context.Context, _ actor.Envelope) (actor.Envelope, error) {
				return actor.Envelope{}, errors.New("fail invoke")
			},
		})

		_, _, err := utils.ExecuteCommand(newActorInvokeCmd(dmsCli), string(msg))
		assert.ErrorContains(t, err, "could not invoke behaviour")
	})
}

func TestActorInvokeCmd_Success(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		msg, _ := json.Marshal(actor.Envelope{
			Behavior: "test",
			Options: actor.EnvelopeOptions{
				ReplyTo: "test-reply",
			},
		})

		dmsCli := setupTest(t, &mockActorInvokeClient{
			invokeBehaviorRawFn: func(_ context.Context, _ actor.Envelope) (actor.Envelope, error) {
				return actor.Envelope{
					Message: json.RawMessage(`{"result":"success"}`),
				}, nil
			},
		})

		out, _, err := utils.ExecuteCommand(newActorInvokeCmd(dmsCli), string(msg))
		require.NoError(t, err)
		assert.Contains(t, out, "success")
	})
}
