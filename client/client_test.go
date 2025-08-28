// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client_test

import (
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/client"
)

// MockTransport implements http.RoundTripper for testing
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestNewClient(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping client test on windows")
	}
	// Get a security context for testing
	sctx := makeSecurityContext(t)

	tests := []struct {
		name    string
		cfg     client.Config
		wantErr bool
	}{
		{
			name: "valid TCP config",
			cfg: client.Config{
				Host:           "localhost:8080",
				Protocol:       client.ConnectionTCP,
				APIPrefix:      "/api",
				Version:        "1",
				ConnectTimeout: 5 * time.Second,
				RequestTimeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid TCP config with defaults",
			cfg: client.Config{
				Host:     "localhost:8080",
				Protocol: client.ConnectionTCP,
			},
			wantErr: false,
		},
		{
			name: "Invalid ConnectionType",
			cfg: client.Config{
				Host:     "localhost:8080",
				Protocol: "invalid",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := client.NewClient(tt.cfg, sctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}

func TestNewClientWithTransport(t *testing.T) {
	// Get a security context for testing
	sctx := makeSecurityContext(t)

	tests := []struct {
		name      string
		cfg       client.Config
		transport http.RoundTripper
		wantErr   bool
	}{
		{
			name: "custom transport",
			cfg: client.Config{
				Host:     "localhost:8080",
				Protocol: client.ConnectionTCP,
			},
			transport: &MockTransport{
				RoundTripFunc: func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{}")),
					}, nil
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := client.NewClientWithTransport(tt.cfg, tt.transport, sctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}
