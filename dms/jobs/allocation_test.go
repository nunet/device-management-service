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
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
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
				eventhandler.New(context.Background(), 1, 1, time.Second, time.Second, func(_ eventhandler.Event) error { return nil }),
				nil, // contractStore - nil for tests
				"",
				func(_ string, _ jobtypes.AllocationStatus) {},
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
		err = alloc.Run(context.Background(), "10.1.0.20", "10.1.0.1", map[int]int{31000: 80})
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.NotEmpty(t, alloc.resultsDir)
		require.Equal(t, "10.1.0.20", alloc.NetState.SubnetIP)
		require.Equal(t, "10.1.0.1", alloc.NetState.GatewayIP)
		require.Equal(t, map[int]int{31000: 80}, alloc.NetState.PortMapping)
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
			status := alloc.Status()
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

	t.Run("restart running allocation", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)

		// state.subnetIP is not updated by any method and assigned at constructor
		// that's why changing internal state here
		alloc.NetState.SubnetIP = "192.168.1.10"

		// success - restart running allocation
		err = alloc.Start()
		require.NoError(t, err)
		require.True(t, alloc.actorRunning)

		err = alloc.Run(context.Background(), alloc.NetState.SubnetIP, "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.True(t, alloc.actorRunning)

		err = alloc.Restart(context.Background())
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.True(t, alloc.actorRunning)
	})

	t.Run("restart failed allocation", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)
		alloc.NetState.SubnetIP = "192.168.1.11"

		// success - restarting a simulated failed allocation
		alloc.setStatus(AllocationFailed, "alloc failed", false)

		err = alloc.Restart(context.Background())
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.True(t, alloc.actorRunning)
	})

	t.Run("restart without subnet IP", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)
		require.Equal(t, AllocationPending, alloc.status)
		alloc.NetState.SubnetIP = "192.168.1.12"

		err = alloc.Start()
		require.NoError(t, err)

		err = alloc.Run(context.Background(), alloc.NetState.SubnetIP, "", nil)
		require.NoError(t, err)
		require.Equal(t, AllocationRunning, alloc.status)
		require.True(t, alloc.actorRunning)

		err = alloc.Stop(context.Background())
		require.NoError(t, err)
		require.Equal(t, AllocationStopped, alloc.status)
		require.False(t, alloc.actorRunning)

		// error - restart allocation without subnet IP
		alloc.NetState.SubnetIP = ""

		err = alloc.Restart(context.Background())
		require.Error(t, err)
		require.Equal(t, AllocationRestarting, alloc.Status().Status) // stays on restarting
		require.False(t, alloc.actorRunning)                          // stays down
	})
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

func TestAllocation_PersistState(t *testing.T) {
	t.Parallel()

	t.Run("persist state of alloc with empty priv key", func(t *testing.T) {
		t.Parallel()

		// noop actor has no priv key
		alloc, err := createTestAllocation(t)
		require.NoError(t, err)

		state := alloc.PersistState()
		require.Empty(t, state.AllocationID)
		require.Empty(t, state.DeploymentID)
	})

	t.Run("persist state correct values", func(t *testing.T) {
		t.Parallel()

		alloc, err := createTestAllocation(t)
		require.NoError(t, err)

		substrate := network.NewSubstrate()
		alloc.Actor, _, _, _, _ = actor.NewMockActorForTest(t, actor.Handle{}, substrate)

		committedPorts := map[int]int{31000: 80}
		alloc.SetCommittedPorts(committedPorts, 2)
		alloc.NetState.SubnetIP = "10.0.0.10"
		alloc.NetState.GatewayIP = "10.0.0.1"
		alloc.NetState.PortMapping = map[int]int{31000: 80}
		alloc.ApplyPersistedNetworkMetadata(
			"subnet-1",
			map[string]string{"10.0.0.10": "peer-a"},
			map[string]string{"service.local": "10.0.0.10"},
			[]jobtypes.AllocationPortMapping{
				{
					SubnetID:   "subnet-1",
					Protocol:   "tcp",
					SourceIP:   "10.0.0.10",
					SourcePort: "31000",
					DestIP:     "10.0.0.11",
					DestPort:   "80",
				},
			},
		)

		state := alloc.PersistState()
		require.Equal(t, alloc.ID, state.AllocationID)
		require.Equal(t, alloc.DeploymentID, state.DeploymentID)
		require.Equal(t, 2, state.DynamicPortsNum)
		require.Equal(t, "subnet-1", state.SubnetID)
		require.Equal(t, "10.0.0.10", state.NetState.SubnetIP)
		require.Equal(t, 80, state.Ports[31000])
		require.Equal(t, "peer-a", state.RoutingTable["10.0.0.10"])
		require.Equal(t, "10.0.0.10", state.DNSRecords["service.local"])
		require.Len(t, state.PortMapping, 1)
		require.Equal(t, "31000", state.PortMapping[0].SourcePort)

		// Returned maps are copies and should not mutate allocation state.
		state.Ports[32000] = 8080
		state.RoutingTable["10.0.0.99"] = "peer-z"
		state.DNSRecords["other.local"] = "10.0.0.99"

		again := alloc.PersistState()
		_, hasPort := again.Ports[32000]
		_, hasRoute := again.RoutingTable["10.0.0.99"]
		_, hasDNS := again.DNSRecords["other.local"]
		require.False(t, hasPort)
		require.False(t, hasRoute)
		require.False(t, hasDNS)
	})
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

	alloc, err := NewAllocation(
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
		eventhandler.New(context.Background(), 1, 1, time.Second, time.Second, func(_ eventhandler.Event) error { return nil }),
		nil, // contractStore - nil for tests
		"",
		func(_ string, _ jobtypes.AllocationStatus) {},
	)
	if err != nil {
		return nil, err
	}

	// disable liveness reporting for tests
	alloc.liveness.enabled = false

	return alloc, err
}
