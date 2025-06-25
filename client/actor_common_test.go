package client_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
)

func TestClient_GetDMSHandle(t *testing.T) {
	expectedPath := client.ActorHandleEndpoint
	sctx := makeSecurityContext(t)
	handle, err := actor.HandleFromDID(sctx.DID().String())
	require.NoError(t, err, "make handle from DID")

	tests := []struct {
		name       string
		resp       *actor.Handle
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful handle retrieval",
			resp:       &handle,
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "error response",
			resp:       nil,
			statusCode: 500,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := makeMockClientConfig()
			c, err := client.NewClientWithTransport(
				cfg,
				func(t *testing.T) handlerFunc {
					return func(req *http.Request) (*http.Response, error) {
						assert.Equal(t, expectedPath, req.URL.Path, "unexpected request path")
						body, err := json.Marshal(tt.resp)
						assert.NoError(t, err, "marshal handle")
						return &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					}
				}(t),
				sctx,
			)
			require.NoError(t, err, "create client")

			// Call the function
			handle, err := c.GetDMSHandle(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, handle.ID)
			assert.NotEmpty(t, handle.DID)
		})
	}
}
