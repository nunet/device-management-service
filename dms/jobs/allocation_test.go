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
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewAllocation(t *testing.T) {
	t.Parallel()

	mockExecutor, _ := null.NewExecutor(context.Background(), "")

	mockNetwork, err := network.NewMemoryNetHost()
	require.NoError(t, err)

	mockActor := actor.NewNoopActor()

	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	tests := []struct {
		name     string
		executor types.Executor
		actor    actor.Actor
		network  network.Network
		wantErr  bool
	}{
		{
			name:     "nil executor",
			executor: nil,
			actor:    mockActor,
			network:  mockNetwork,
			wantErr:  true,
		},
		{
			name:     "nil network",
			executor: mockExecutor,
			actor:    mockActor,
			network:  nil,
			wantErr:  true,
		},
		{
			name:     "nil actor",
			executor: mockExecutor,
			actor:    nil,
			network:  mockNetwork,
			wantErr:  true,
		},
		{
			name:     "success",
			executor: mockExecutor,
			actor:    mockActor,
			network:  mockNetwork,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			alloc, err := NewAllocation(
				"test-allocation-id",
				jobtypes.AllocationTypeService,
				actor.Handle{},
				fs,
				"/tmp/workdir",
				tt.actor,
				createDetails([]types.VolumeConfig{}),
				tt.network,
				tt.executor,
				func() error { return nil },
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, alloc)
			} else {
				require.NoError(t, err)
				require.NotNil(t, alloc)
			}
		})
	}
}

func TestAllocation_Run(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)

		// success - run allocation
		// no volumes
		err = alloc.Run(context.Background(), "", "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.NotEmpty(t, alloc.resultsDir)
	})

	t.Run("should not error run already running allocation", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)

		// success - run allocation
		// no volumes
		err = alloc.Run(context.Background(), "", "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.NotEmpty(t, alloc.resultsDir)
		// success - run already running allocation
		// no volumes
		err = alloc.Run(context.Background(), "", "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.NotEmpty(t, alloc.resultsDir)
	})

	t.Run("success with glusterfs volume", func(t *testing.T) {
		t.Parallel()

		glusterfsVolume := types.VolumeConfig{
			Type:             "glusterfs",
			MountDestination: "/data",
			ReadOnly:         false,
			Name:             "test-volume",
			Servers:          []string{"gluster1.example.com", "gluster2.example.com"},
			ClientPrivateKey: "test-private-key",
			ClientPEM:        "test-client-pem",
		}

		alloc, err := createTestAllocation(t, glusterfsVolume)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)

		// success - run allocation with glusterfs volume
		err = alloc.Run(context.Background(), "", "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.NotEmpty(t, alloc.resultsDir)
	})
}

func TestAllocation_handleTransience(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     *types.ExecutionResult
		err        error
		wantStatus AllocationStatus
	}{
		{
			name: "successful execution",
			result: &types.ExecutionResult{
				ExitCode: 0,
				STDOUT:   "success output",
				STDERR:   "",
				Killed:   false,
			},
			err:        nil,
			wantStatus: AllocationCompleted,
		},
		{
			name: "execution with non-zero exit code",
			result: &types.ExecutionResult{
				ExitCode: 1,
				STDOUT:   "",
				STDERR:   "error output",
				Killed:   false,
			},
			err:        nil,
			wantStatus: AllocationFailed,
		},
		{
			name: "killed execution",
			result: &types.ExecutionResult{
				ExitCode: 0,
				STDOUT:   "partial output",
				STDERR:   "",
				Killed:   true,
			},
			err:        nil,
			wantStatus: AllocationFailed,
		},
		{
			name:       "execution with error",
			result:     nil,
			err:        errors.New("execution error"),
			wantStatus: AllocationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			alloc, err := createTestAllocation(t)
			require.NoError(t, err)

			alloc.handleTransience(tt.result, tt.err)
			status := alloc.Status(context.Background())
			require.Equal(t, tt.wantStatus, status.Status)
		})
	}
}

func TestAllocation_stopExecution(t *testing.T) {
	t.Parallel()

	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotEmpty(t, alloc)

	err = alloc.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)

	// neutral - stopping already stopped allocation
	err = alloc.stopExecution(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)

	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)

	// success - stop running allocation
	err = alloc.stopExecution(context.Background())
	require.NoError(t, err)

	// neutral - stop running allocation with nil executor
	alloc.lock.Lock()
	alloc.executor = nil
	alloc.lock.Unlock()
	err = alloc.stopExecution(context.Background())
	require.NoError(t, err)
}

func TestAllocation_Cleanup(t *testing.T) {
	// TODO: make a single test
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)

		err = alloc.Cleanup()
		require.NoError(t, err)

		executions := alloc.executor.List()
		require.Empty(t, executions)
	})
	t.Run("nil executor", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)

		alloc.lock.Lock()
		alloc.executor = nil
		alloc.lock.Unlock()

		err = alloc.Cleanup()
		require.NoError(t, err)
	})
}

func TestAllocation_Terminate(t *testing.T) {
	t.Parallel()

	alloc, err := createTestAllocation(t)
	require.NoError(t, err)

	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)

	// success - terminate running allocation
	err = alloc.Terminate(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationTerminated, alloc.status)

	executions := alloc.executor.List()
	require.Empty(t, executions)

	// neutral - terminate stopped allocation
	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)

	err = alloc.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)

	err = alloc.Terminate(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationTerminated, alloc.status)

	executions = alloc.executor.List()
	require.Empty(t, executions)
}

func TestAllocation_stopActor(t *testing.T) {
	t.Parallel()

	t.Run("should be able to stop started actor", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)

		err = alloc.Start()
		require.NoError(t, err)

		err = alloc.stopActor()
		require.NoError(t, err)
		require.False(t, alloc.actorRunning)
	})
	t.Run("should not error stopping already stopped actor", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.False(t, alloc.actorRunning)

		err = alloc.stopActor()
		require.NoError(t, err)
		require.False(t, alloc.actorRunning)
	})
}

func TestAllocation_Stop(t *testing.T) {
	t.Parallel()

	alloc, err := createTestAllocation(t)
	require.NoError(t, err)

	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)

	// success - stop a running allocation
	err = alloc.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)
	require.False(t, alloc.actorRunning)

	// neutral - stop an already stopped allocation (no error)
	err = alloc.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)
	require.False(t, alloc.actorRunning)
}

func TestAllocation_Start(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)

	// success
	err = alloc.Start()
	require.NoError(t, err)
	require.True(t, alloc.actorRunning)

	// neutral - start already started allocation
	err = alloc.Start()
	require.NoError(t, err)
	require.True(t, alloc.actorRunning)
}

func TestAllocation_Restart(t *testing.T) {
	t.Parallel()

	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.Equal(t, AllocationPending, alloc.status)

	// state.subnetIP is not updated by any method and assigned at constructor
	// that's why changing internal state here
	alloc.state.subnetIP = "192.168.1.12"

	// success - restart running allocation
	err = alloc.Start()
	require.NoError(t, err)
	require.True(t, alloc.actorRunning)

	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)
	require.True(t, alloc.actorRunning)

	err = alloc.Restart(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)
	require.True(t, alloc.actorRunning)

	// success - restarting a simulated failed allocation
	alloc.status = AllocationFailed

	err = alloc.Restart(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)
	require.True(t, alloc.actorRunning)

	// error - restart allocation without subnet IP
	alloc.state.subnetIP = ""
	err = alloc.Run(context.Background(), "", "", nil)
	require.NoError(t, err)
	require.Equal(t, AllocationRunning, alloc.status)
	require.True(t, alloc.actorRunning)

	err = alloc.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, AllocationStopped, alloc.status)
	require.False(t, alloc.actorRunning)

	err = alloc.Restart(context.Background())
	require.Error(t, err)
	require.Equal(t, AllocationStopped, alloc.status) // make sure it remains stopped
	require.False(t, alloc.actorRunning)
}

func TestAllocation_sendReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		msg   actor.Envelope
		reply any
	}{
		{
			name: "successful reply to regular message",
			msg: actor.Envelope{
				To: actor.Handle{
					Address: actor.Address{
						HostID: "test-host",
					},
				},
				From:     actor.Handle{Address: actor.Address{HostID: "sender"}},
				Behavior: "test-behavior",
				Nonce:    12345,
				Options: actor.EnvelopeOptions{
					ReplyTo: "test-behavior",
				},
			},
			reply: map[string]string{"status": "ok"},
		},
		{
			name: "successful reply to broadcast message",
			msg: actor.Envelope{
				To:       actor.Handle{}, // empty indicates broadcast
				From:     actor.Handle{Address: actor.Address{HostID: "broadcaster"}},
				Behavior: "test-broadcast",
				Nonce:    54321,
				Options: actor.EnvelopeOptions{
					Topic:   "test-topic",
					ReplyTo: "test-behavior",
				},
			},
			reply: map[string]string{"response": "received"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			alloc, err := createTestAllocation(t)
			require.NoError(t, err)

			noopActor := alloc.Actor.(*actor.NoopActor)
			noopActor.SetHandle(tt.msg.To)

			alloc.sendReply(tt.msg, tt.reply)

			sentMessages := noopActor.GetSentMessages()
			require.Len(t, sentMessages, 1)

			reply := sentMessages[0]
			require.Equal(t, tt.msg.From, reply.To)
			require.Equal(t, tt.msg.To, reply.From)
		})
	}
}

func createDetails(v []types.VolumeConfig) AllocationDetails {
	return AllocationDetails{
		Job: Job{
			Resources: types.Resources{},
			Execution: types.SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
			Volume: v,
		},
		NodeID:   "test-node-id",
		SourceID: "test-source-id",
	}
}

func createTestAllocation(t *testing.T, vol ...types.VolumeConfig) (*Allocation, error) {
	t.Helper()

	mockExecutor, _ := null.NewExecutor(context.Background(), "")
	mockNetwork, err := network.NewMemoryNetHost()
	require.NoError(t, err)
	mockActor := actor.NewNoopActor()

	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	return NewAllocation(
		"test-allocation-id",
		jobtypes.AllocationTypeService,
		actor.Handle{},
		fs,
		"/tmp/workdir",
		mockActor,
		createDetails(vol),
		mockNetwork,
		mockExecutor,
		func() error { return nil },
	)
}
