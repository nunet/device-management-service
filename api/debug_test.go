package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingPeer(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*P2PHandler)
		expectedStatus int
		expectedBody   string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := setupTestP2P()
			assert.NoError(t, err)
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			router := setupTestRouter()
			router.GET("/peers/ping", handler.PingPeer)

			w := performRequest(router, "GET", "/peers/ping")

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}
