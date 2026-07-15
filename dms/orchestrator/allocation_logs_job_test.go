// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestStartFetchAllocationLogs(t *testing.T) {
	t.Parallel()

	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				},
			},
		},
	}

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	payload := strings.Repeat("x", 1500)
	behavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, "test-ensemble")
	require.NoError(t, provider.actor.AddBehavior(behavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var req AllocationLogsRequest
		require.NoError(t, json.Unmarshal(msg.Message, &req))

		var streamPayload []byte
		switch req.Stream {
		case behaviors.LogStreamStdout:
			streamPayload = []byte(payload)
		case behaviors.LogStreamStderr:
			streamPayload = nil
		default:
			t.Fatalf("unexpected stream: %s", req.Stream)
		}

		if req.Offset > int64(len(streamPayload)) {
			t.Fatalf("offset beyond payload")
		}
		chunk := streamPayload[req.Offset:]
		if len(chunk) > req.MaxBytes {
			chunk = chunk[:req.MaxBytes]
		}
		nextOffset := req.Offset + int64(len(chunk))

		reply, err := actor.ReplyTo(msg, AllocationLogsResponse{
			Stream: req.Stream,
			ChunkedTransferResponse: behaviors.ChunkedTransferResponse{
				Offset:     req.Offset,
				NextOffset: nextOffset,
				TotalSize:  int64(len(streamPayload)),
				EOF:        nextOffset >= int64(len(streamPayload)),
				Data:       chunk,
			},
		})
		require.NoError(t, err)
		reply.To = msg.From
		reply.From = provider.handle
		require.NoError(t, provider.actor.Send(reply))
	}))

	ctx := context.Background()
	fs := afero.NewMemMapFs()
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator(), nil, nil)
	require.NoError(t, err)

	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() { deployDone <- o.Deploy(expiry) }()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Minute):
		t.Fatal("deployment timeout")
	}

	requesterDID := "did:key:test-requester"
	job, err := o.StartFetchAllocationLogs(requesterDID, "node1.alloc1", AllocationLogsFetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, AllocationLogsJobRunning, job.Status)
	assert.NotEmpty(t, job.LogsWrittenTo)
	assert.Equal(t, requesterDID, job.RequesterDID)
	assert.False(t, job.Follow)

	// Second start while running returns the same singleton job.
	again, err := o.StartFetchAllocationLogs(requesterDID, "node1.alloc1", AllocationLogsFetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, job.LogsWrittenTo, again.LogsWrittenTo)
	assert.Equal(t, AllocationLogsJobRunning, again.Status)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, ok := o.GetFetchAllocationLogsJob(requesterDID, "node1.alloc1")
		require.True(t, ok)
		if status.Status == AllocationLogsJobComplete {
			stdout, err := afero.ReadFile(fs, fmt.Sprintf("%s/stdout.log", job.LogsWrittenTo))
			require.NoError(t, err)
			assert.Equal(t, payload, string(stdout))
			assert.Equal(t, int64(len(payload)), status.BytesWritten)
			return
		}
		if status.Status == AllocationLogsJobFailed {
			t.Fatalf("job failed: %s", status.Error)
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("timed out waiting for async log fetch to complete")
}

func TestStartFetchAllocationLogsFollowAndStop(t *testing.T) {
	t.Parallel()

	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				},
			},
		},
	}

	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	var mu sync.Mutex
	payload := []byte("hello")

	behavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, "test-ensemble")
	require.NoError(t, provider.actor.AddBehavior(behavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var req AllocationLogsRequest
		require.NoError(t, json.Unmarshal(msg.Message, &req))

		mu.Lock()
		streamPayload := payload
		if req.Stream != behaviors.LogStreamStdout {
			streamPayload = nil
		}
		mu.Unlock()

		if req.Offset > int64(len(streamPayload)) {
			t.Fatalf("offset beyond payload")
		}
		chunk := streamPayload[req.Offset:]
		if len(chunk) > req.MaxBytes {
			chunk = chunk[:req.MaxBytes]
		}
		nextOffset := req.Offset + int64(len(chunk))

		reply, err := actor.ReplyTo(msg, AllocationLogsResponse{
			Stream: req.Stream,
			ChunkedTransferResponse: behaviors.ChunkedTransferResponse{
				Offset:     req.Offset,
				NextOffset: nextOffset,
				TotalSize:  int64(len(streamPayload)),
				EOF:        nextOffset >= int64(len(streamPayload)),
				Data:       chunk,
			},
		})
		require.NoError(t, err)
		reply.To = msg.From
		reply.From = provider.handle
		require.NoError(t, provider.actor.Send(reply))
	}))

	ctx := context.Background()
	fs := afero.NewMemMapFs()
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator(), nil, nil)
	require.NoError(t, err)

	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() { deployDone <- o.Deploy(expiry) }()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Minute):
		t.Fatal("deployment timeout")
	}

	requesterDID := "did:key:test-requester-follow"
	job, err := o.StartFetchAllocationLogs(requesterDID, "node1.alloc1", AllocationLogsFetchOpts{
		Follow:         true,
		FollowInterval: time.Second,
	})
	require.NoError(t, err)
	assert.True(t, job.Follow)
	assert.Equal(t, time.Second, job.FollowInterval)

	// Wait until first catch-up written, still running (follow).
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status, ok := o.GetFetchAllocationLogsJob(requesterDID, "node1.alloc1")
		require.True(t, ok)
		require.NotEqual(t, AllocationLogsJobFailed, status.Status, status.Error)
		if status.BytesWritten >= int64(len(payload)) && status.Status == AllocationLogsJobRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	payload = append(payload, []byte(" world")...)
	mu.Unlock()

	// Wait for follow poll to append new bytes.
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status, ok := o.GetFetchAllocationLogsJob(requesterDID, "node1.alloc1")
		require.True(t, ok)
		if status.BytesWritten >= int64(len("hello world")) {
			stdout, err := afero.ReadFile(fs, fmt.Sprintf("%s/stdout.log", job.LogsWrittenTo))
			require.NoError(t, err)
			assert.Equal(t, "hello world", string(stdout))
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	stopped, err := o.StopFetchAllocationLogs(requesterDID, "node1.alloc1")
	require.NoError(t, err)
	assert.Equal(t, AllocationLogsJobStopped, stopped.Status)
}
