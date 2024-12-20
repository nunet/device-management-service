package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/avast/retry-go"

	"gitlab.com/nunet/device-management-service/actor"

	"gitlab.com/nunet/device-management-service/lib/crypto"

	"github.com/stretchr/testify/require"
)

func doRequest(t *testing.T, method, url string, body []byte) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)

	return http.DefaultClient.Do(req)
}

func startServer(t *testing.T, port uint32, p2pHandler *MockNetwork) *Server {
	t.Helper()

	cfg := &ServerConfig{
		P2P:  p2pHandler,
		Port: port,
		Addr: "localhost",
	}
	server := NewServer(cfg)
	require.NotNil(t, server)
	server.SetupRoutes()

	go func() {
		t.Logf("Starting server on port %d", port)
		if err := server.Run(); err != nil {
			t.Errorf("Failed to start server: %v", err)
			return
		}
	}()

	// Ensure the server is up and running
	err := retry.Do(func() error {
		t.Logf("Checking server health on port %d", port)
		resp, err := doRequest(t, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server not up yet")
		}
		return nil
	},
		retry.Attempts(20),
		retry.Delay(5*time.Second),
	)
	require.NoError(t, err)

	return server
}

func getTestPublicKey(t *testing.T) crypto.PubKey {
	t.Helper()

	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	return pubk
}

func getMockEnvelope(t *testing.T, msg actor.Envelope) *bytes.Buffer {
	t.Helper()

	requestBody := new(bytes.Buffer)
	require.NoError(t, json.NewEncoder(requestBody).Encode(msg))

	return requestBody
}

func getMockBroadcastMessage(t *testing.T) *bytes.Buffer {
	t.Helper()

	return getMockEnvelope(t, actor.Envelope{
		Options: actor.EnvelopeOptions{
			Topic: "test",
		},
	})
}
