// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthCheck(t *testing.T) {
	// Setup test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			_, err := w.Write([]byte("OK"))
			assert.NoError(t, err)
		case "/failure":
			_, err := w.Write([]byte("ERROR"))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	// Mock command executor function
	mockCommandExecutor := func(mf HealthCheckManifest) error {
		if len(mf.Exec) > 0 && mf.Exec[0] == "success" {
			return nil
		}
		return errors.New("command execution failed")
	}

	tests := []struct {
		name        string
		manifest    HealthCheckManifest
		expectError bool
		runCheck    bool
	}{
		{
			name: "valid http healthcheck - success",
			manifest: HealthCheckManifest{
				Type:     HealthCheckTypeHTTP,
				Endpoint: server.URL + "/success",
				Response: HealthCheckResponse{
					Value: "OK",
				},
				Interval: time.Second,
			},
			expectError: false,
			runCheck:    true,
		},
		{
			name: "valid http healthcheck - failure due to response mismatch",
			manifest: HealthCheckManifest{
				Type:     HealthCheckTypeHTTP,
				Endpoint: server.URL + "/success",
				Response: HealthCheckResponse{
					Value: "NOT_OK",
				},
				Interval: time.Second,
			},
			expectError: true,
			runCheck:    true,
		},
		{
			name: "valid http healthcheck - failure due to endpoint",
			manifest: HealthCheckManifest{
				Type:     HealthCheckTypeHTTP,
				Endpoint: server.URL + "/failure",
				Response: HealthCheckResponse{
					Value: "OK",
				},
				Interval: time.Second,
			},
			expectError: true,
			runCheck:    true,
		},
		{
			name: "valid command healthcheck - success",
			manifest: HealthCheckManifest{
				Type: HealthCheckTypeCommand,
				Exec: []string{"success"},
				Response: HealthCheckResponse{
					Value: "OK",
				},
				Interval: time.Second,
			},
			expectError: false,
			runCheck:    true,
		},
		{
			name: "valid command healthcheck - failure",
			manifest: HealthCheckManifest{
				Type: HealthCheckTypeCommand,
				Exec: []string{"failure"},
				Response: HealthCheckResponse{
					Value: "OK",
				},
				Interval: time.Second,
			},
			expectError: true,
			runCheck:    true,
		},
		{
			name: "invalid healthcheck type",
			manifest: HealthCheckManifest{
				Type:     "unknown",
				Interval: time.Second,
			},
			expectError: true,
			runCheck:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthcheck, err := NewHealthCheck(tt.manifest, mockCommandExecutor)

			if tt.expectError && err == nil && !tt.runCheck {
				t.Errorf("expected error but got nil")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.runCheck && healthcheck != nil {
				err = healthcheck()
				if tt.expectError && err == nil {
					t.Errorf("expected error from healthcheck execution but got nil")
				}

				if !tt.expectError && err != nil {
					t.Errorf("unexpected error from healthcheck execution: %v", err)
				}
			}
		})
	}
}
