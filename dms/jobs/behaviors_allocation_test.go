// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/types"
)

// null.Executor does not produce meaningful tests for execHealthCheck, therefore
// they were skipped

func createTestEnvelope(t *testing.T, payload []byte) actor.Envelope {
	t.Helper()
	return actor.Envelope{
		From: actor.Handle{
			Address: actor.Address{
				HostID: "test-sender",
			},
		},
		Message: payload,
		Options: actor.EnvelopeOptions{
			ReplyTo: "test-sender",
		},
		Discard: func() {},
	}
}

func TestAllocation_handleAllocationStart(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		req := behaviors.AllocationStartRequest{
			SubnetIP:    "192.168.1.100",
			GatewayIP:   "192.168.1.1",
			PortMapping: map[int]int{8080: 9090},
		}

		reqBytes, err := json.Marshal(req)
		require.NoError(t, err)

		envelope := createTestEnvelope(t, reqBytes)
		alloc.handleAllocationStart(envelope)

		// checks allocation state
		require.Equal(t, AllocationRunning, alloc.status)
		require.Equal(t, req.SubnetIP, alloc.NetState.SubnetIP)
		require.Equal(t, req.GatewayIP, alloc.NetState.GatewayIP)
		require.Equal(t, req.PortMapping, alloc.GetPortMapping())

		noopActor, ok := alloc.Actor.(*actor.NoopActor)
		require.True(t, ok)
		require.NotNil(t, noopActor)

		sent := noopActor.GetSentMessages()
		require.Len(t, sent, 1)

		var resp behaviors.AllocationStartResponse
		err = json.Unmarshal(sent[0].Message, &resp)
		require.NoError(t, err)

		require.True(t, resp.OK)
		require.Empty(t, resp.Error)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		envelope := createTestEnvelope(t, []byte("invalid json"))
		alloc.handleAllocationStart(envelope)

		// check that allocation state hasn't changed
		require.Equal(t, AllocationPending, alloc.status)

		// check no reply was sent due to unmarshal error
		noopActor, ok := alloc.Actor.(*actor.NoopActor)
		require.True(t, ok)
		sent := noopActor.GetSentMessages()
		require.Len(t, sent, 0)
	})

	t.Run("already running allocation", func(t *testing.T) {
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		err = alloc.Run(context.Background(), "", "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.Status().Status)

		req := behaviors.AllocationStartRequest{
			SubnetIP:    "192.168.1.100",
			GatewayIP:   "192.168.1.1",
			PortMapping: map[int]int{8080: 9090},
		}

		reqBytes, err := json.Marshal(req)
		require.NoError(t, err)

		envelope := createTestEnvelope(t, reqBytes)
		alloc.handleAllocationStart(envelope)

		// check actor behavior - should NOT get error, just early return
		noopActor, ok := alloc.Actor.(*actor.NoopActor)
		require.True(t, ok)
		sent := noopActor.GetSentMessages()
		require.Len(t, sent, 1)

		var resp behaviors.AllocationStartResponse
		err = json.Unmarshal(sent[0].Message, &resp)
		require.NoError(t, err)

		require.True(t, resp.OK)
		require.Empty(t, resp.Error)
	})
}

func TestAllocation_handleAllocationRestart(t *testing.T) {
	t.Run("successful restart running allocation", func(t *testing.T) {
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		alloc.lock.Lock()
		alloc.NetState.SubnetIP = "192.168.1.1"
		alloc.lock.Unlock()

		err = alloc.Run(context.Background(), "192.168.1.100", "192.168.1.1", map[int]int{8080: 9090})
		require.NoError(t, err)

		envelope := createTestEnvelope(t, []byte{})
		alloc.handleAllocationRestart(envelope)

		noopActor, ok := alloc.Actor.(*actor.NoopActor)
		require.True(t, ok)
		require.NotNil(t, noopActor)

		sent := noopActor.GetSentMessages()
		require.Len(t, sent, 1)

		var resp behaviors.AllocationRestartResponse
		err = json.Unmarshal(sent[0].Message, &resp)
		require.NoError(t, err)

		require.True(t, resp.OK)
		require.Empty(t, resp.Error)
	})

	t.Run("should not error restart pending allocation", func(t *testing.T) {
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.NotNil(t, alloc)
		require.Equal(t, AllocationPending, alloc.status)

		alloc.lock.Lock()
		alloc.NetState.SubnetIP = "192.168.1.1" // cannot restart if empty
		alloc.lock.Unlock()

		envelope := createTestEnvelope(t, []byte{})
		alloc.handleAllocationRestart(envelope)

		noopActor, ok := alloc.Actor.(*actor.NoopActor)
		require.True(t, ok)
		sent := noopActor.GetSentMessages()
		require.Len(t, sent, 1)

		var resp behaviors.AllocationRestartResponse
		err = json.Unmarshal(sent[0].Message, &resp)
		require.NoError(t, err)

		require.True(t, resp.OK)
		require.Empty(t, resp.Error)
	})
}

func TestAllocation_handleRegisterHealthcheck(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name: "successful registration",
			payload: func() []byte {
				req := behaviors.RegisterHealthcheckRequest{
					EnsembleID: "test-ensemble",
					HealthCheck: types.HealthCheckManifest{
						Type:     "command",
						Exec:     []string{"echo", "healthy"},
						Interval: 30,
						Response: types.HealthCheckResponse{
							Type:  "contains",
							Value: "healthy",
						},
					},
				}
				reqBytes, _ := json.Marshal(req)
				return reqBytes
			}(),
			wantErr: false,
		},
		{
			name:    "should error if invalid JSON",
			payload: []byte("invalid json"),
			wantErr: true,
		},
		{
			name: "should error if unsupported healthcheck type",
			payload: func() []byte {
				req := behaviors.RegisterHealthcheckRequest{
					EnsembleID: "test-ensemble",
					HealthCheck: types.HealthCheckManifest{
						Type:     "unsupported",
						Interval: 30,
					},
				}
				reqBytes, _ := json.Marshal(req)
				return reqBytes
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alloc, err := createTestAllocation(t)
			require.NoError(t, err)
			require.NotNil(t, alloc)

			envelope := createTestEnvelope(t, tt.payload)
			alloc.handleRegisterHealthcheck(envelope)

			noopActor, ok := alloc.Actor.(*actor.NoopActor)
			require.True(t, ok)
			require.NotNil(t, noopActor)

			sent := noopActor.GetSentMessages()
			require.Len(t, sent, 1)

			var resp behaviors.RegisterHealthcheckResponse
			err = json.Unmarshal(sent[0].Message, &resp)
			require.NoError(t, err)

			if tt.wantErr {
				require.NotEmpty(t, resp.Error)
				require.False(t, resp.OK)
			} else {
				require.Empty(t, resp.Error)
				require.True(t, resp.OK)
			}
		})
	}
}

func TestAllocation_handleHealthcheck(t *testing.T) {
	tests := []struct {
		name            string
		healthcheckFunc func() error
		wantErr         bool
	}{
		{
			name:            "should not error if no healthcheck registered",
			healthcheckFunc: nil,
			wantErr:         false,
		},
		{
			name: "healthcheck succeeds",
			healthcheckFunc: func() error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "should error if healthcheck fails",
			healthcheckFunc: func() error {
				return errors.New("health check failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alloc, err := createTestAllocation(t)
			require.NoError(t, err)
			require.NotNil(t, alloc)

			if tt.healthcheckFunc != nil {
				alloc.SetHealthCheck(tt.healthcheckFunc)
			}

			envelope := createTestEnvelope(t, []byte{})
			alloc.handleHealthcheck(envelope)

			noopActor, ok := alloc.Actor.(*actor.NoopActor)
			require.True(t, ok)
			require.NotNil(t, noopActor)

			sent := noopActor.GetSentMessages()
			require.Len(t, sent, 1)

			var resp HealthCheckResponse
			err = json.Unmarshal(sent[0].Message, &resp)
			require.NoError(t, err)

			if tt.wantErr {
				require.NotEmpty(t, resp.Error)
			} else {
				require.Empty(t, resp.Error)
			}
		})
	}
}
