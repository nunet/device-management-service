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

type mockLoggerBehaviorClient struct {
	client.DmsClient
	loggerConfigFn func(ctx context.Context, req node.LoggerConfigRequest, opts ...client.Option) (node.LoggerConfigResponse, error)
}

func (m *mockLoggerBehaviorClient) LoggerConfig(ctx context.Context, req node.LoggerConfigRequest, opts ...client.Option) (node.LoggerConfigResponse, error) {
	return m.loggerConfigFn(ctx, req, opts...)
}

func TestLoggerConfigBehavior(t *testing.T) {
	t.Parallel()
	trueVal := true

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.LoggerConfigRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no url, level and interval",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{
				"--url", "http://localhost:8080",
				"--level", "info",
				"--interval", "10",
				"--api-key", "test_api_key",
				"--apm-url", "http://localhost:8080",
				"--enable-elastic", "true",
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.LoggerConfigRequest{
				URL:            "http://localhost:8080",
				Level:          "info",
				Interval:       10,
				APIKey:         "test_api_key",
				APMURL:         "http://localhost:8080",
				ElasticEnabled: &trueVal,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockLoggerBehaviorClient{
				loggerConfigFn: func(_ context.Context, req node.LoggerConfigRequest, opts ...client.Option) (node.LoggerConfigResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.LoggerConfigResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.LoggerConfigBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
