// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

func TestWait(t *testing.T) {
	t.Parallel()

	t.Run("execution not found", func(t *testing.T) {
		t.Parallel()

		e := &Executor{}
		resultCh, errCh := e.Wait(context.Background(), "missing")

		select {
		case result, ok := <-resultCh:
			require.False(t, ok)
			require.Nil(t, result)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for result channel close")
		}

		select {
		case err, ok := <-errCh:
			require.True(t, ok)
			require.Error(t, err)
			require.Contains(t, err.Error(), "execution (missing) not found")
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for error channel value")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		t.Parallel()

		e := &Executor{}
		state := &executionState{
			running: &atomic.Bool{},
			doneCh:  make(chan struct{}),
		}
		e.executions.Put("exec-cancelled", state)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resultCh, errCh := e.Wait(ctx, "exec-cancelled")

		select {
		case result, ok := <-resultCh:
			require.False(t, ok)
			require.Nil(t, result)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for result channel close")
		}

		select {
		case err, ok := <-errCh:
			require.True(t, ok)
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for context error")
		}
	})

	t.Run("done with nil result", func(t *testing.T) {
		t.Parallel()

		e := &Executor{}
		state := &executionState{
			running: &atomic.Bool{},
			doneCh:  make(chan struct{}),
		}
		e.executions.Put("exec-nil", state)

		resultCh, errCh := e.Wait(context.Background(), "exec-nil")
		close(state.doneCh)

		select {
		case result, ok := <-resultCh:
			require.False(t, ok)
			require.Nil(t, result)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for result channel close")
		}

		select {
		case err, ok := <-errCh:
			require.True(t, ok)
			require.Error(t, err)
			require.Contains(t, err.Error(), "result is nil")
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for nil-result error")
		}
	})

	t.Run("done with result", func(t *testing.T) {
		t.Parallel()

		e := &Executor{}
		expected := &types.ExecutionResult{STDOUT: "out", STDERR: "err", ExitCode: 0}
		state := &executionState{
			running: &atomic.Bool{},
			doneCh:  make(chan struct{}),
			result:  expected,
		}
		e.executions.Put("exec-done", state)

		resultCh, errCh := e.Wait(context.Background(), "exec-done")
		close(state.doneCh)

		select {
		case result, ok := <-resultCh:
			require.True(t, ok)
			require.Equal(t, expected, result)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for execution result")
		}

		select {
		case err, ok := <-errCh:
			require.False(t, ok)
			require.Nil(t, err)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for error channel close")
		}
	})
}

func TestGetStatusFromCachedState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		result         *types.ExecutionResult
		expectedStatus types.ExecutionStatus
	}{
		{
			name:           "success",
			result:         &types.ExecutionResult{ExitCode: types.ExecutionStatusCodeSuccess},
			expectedStatus: types.ExecutionStatusSuccess,
		},
		{
			name:           "failed",
			result:         &types.ExecutionResult{ExitCode: 1},
			expectedStatus: types.ExecutionStatusFailed,
		},
		{
			name:           "pending without result",
			result:         nil,
			expectedStatus: types.ExecutionStatusPending,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &Executor{}
			running := &atomic.Bool{}
			running.Store(false)
			e.executions.Put("exec-status", &executionState{
				running: running,
				result:  tc.result,
				doneCh:  make(chan struct{}),
			})

			status, err := e.GetStatus(context.Background(), "exec-status")
			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestGetStatusExecutionNotFound(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	_, err := e.GetStatus(context.Background(), "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution (missing) not found")
}

func TestWaitForStatus(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	running := &atomic.Bool{}
	running.Store(false)
	e.executions.Put("exec-ready", &executionState{
		running: running,
		result:  &types.ExecutionResult{ExitCode: 0},
		doneCh:  make(chan struct{}),
	})

	err := e.WaitForStatus(context.Background(), "exec-ready", types.ExecutionStatusSuccess, nil)
	require.NoError(t, err)
}

func TestWaitForStatusTimeout(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	running := &atomic.Bool{}
	running.Store(false)
	e.executions.Put("exec-timeout", &executionState{
		running: running,
		doneCh:  make(chan struct{}),
	})

	timeout := 150 * time.Millisecond
	err := e.WaitForStatus(context.Background(), "exec-timeout", types.ExecutionStatusSuccess, &timeout)
	require.Error(t, err)
	require.Contains(t, err.Error(), "did not reach status")
}

func TestCancel(t *testing.T) {
	t.Parallel()

	e := &Executor{}

	err := e.Cancel(context.Background(), "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution not found")

	running := &atomic.Bool{}
	running.Store(false)
	e.executions.Put("exec-stopped", &executionState{
		running: running,
		doneCh:  make(chan struct{}),
	})

	require.NoError(t, e.Cancel(context.Background(), "exec-stopped"))
}

func TestGetLogStreamAndList(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	running := &atomic.Bool{}
	running.Store(true)

	state := &executionState{
		running: running,
		doneCh:  make(chan struct{}),
	}
	_, _ = state.stdout.WriteString("hello ")
	_, _ = state.stderr.WriteString("world")
	e.executions.Put("exec-logs", state)

	stream, err := e.GetLogStream(context.Background(), types.LogStreamRequest{ExecutionID: "exec-logs"})
	require.NoError(t, err)
	defer stream.Close()

	content, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(content))

	items := e.List()
	require.Len(t, items, 1)
	require.Equal(t, "exec-logs", items[0].ExecutionID)
	require.True(t, items[0].Running)
}

func TestGetNetInfoAndGetInfo(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	running := &atomic.Bool{}
	running.Store(false)

	expectedNet := types.ExecutorNetInfo{
		InterfaceName: "eth0",
		HostBridge:    "br-test",
		IPAddress:     "10.0.0.2",
		CIDR:          "10.0.0.2/24",
	}

	state := &executionState{
		image:   "alpine:latest",
		running: running,
		doneCh:  make(chan struct{}),
		result:  &types.ExecutionResult{ExitCode: 0},
		network: &networkSetup{netInfo: expectedNet},
	}
	e.executions.Put("exec-info", state)

	netInfo, err := e.GetNetInfo(context.Background(), "exec-info")
	require.NoError(t, err)
	require.Equal(t, expectedNet, *netInfo)

	info, err := e.GetInfo(context.Background(), "exec-info")
	require.NoError(t, err)
	require.Equal(t, "exec-info", info.ExecutionID)
	require.Equal(t, "exec-info", info.ContainerID)
	require.Equal(t, "alpine:latest", info.Image)
	require.Equal(t, types.ExecutorTypeContainerd, info.Runtime)
	require.Equal(t, types.ExecutionStatusSuccess, info.Status)
	require.Equal(t, expectedNet, info.Net)
}

func TestUnsupportedMethodsAndHelpers(t *testing.T) {
	t.Parallel()

	e := &Executor{}

	require.ErrorContains(t, e.Pause(context.Background(), "x"), "not supported")
	require.ErrorContains(t, e.Resume(context.Background(), "x"), "not supported")

	exitCode, stdout, stderr, err := e.Exec(context.Background(), "x", []string{"echo", "test"})
	require.ErrorContains(t, err, "not supported")
	require.Equal(t, 0, exitCode)
	require.Equal(t, "", stdout)
	require.Equal(t, "", stderr)

	stats, err := e.Stats(context.Background(), "x")
	require.ErrorContains(t, err, "not supported")
	require.Nil(t, stats)

	require.False(t, IsShimAvailable("definitely-not-a-runtime"))
}

func TestCheckContainerdSocketAccess(t *testing.T) {
	t.Parallel()

	t.Run("missing socket", func(t *testing.T) {
		t.Parallel()
		err := checkContainerdSocketAccess(filepath.Join(t.TempDir(), "missing.sock"))
		require.Error(t, err)
	})

	t.Run("socket owned by current user", func(t *testing.T) {
		t.Parallel()
		sock := filepath.Join(t.TempDir(), "containerd.sock")
		require.NoError(t, os.WriteFile(sock, nil, 0o600))

		err := checkContainerdSocketAccess(sock)
		require.NoError(t, err)
	})

	if os.Geteuid() != 0 {
		t.Run("root-owned socket requires root process", func(t *testing.T) {
			t.Parallel()
			sock := filepath.Join(t.TempDir(), "containerd.sock")
			require.NoError(t, os.WriteFile(sock, nil, 0o600))
			if err := os.Chown(sock, 0, 0); err != nil {
				t.Skipf("cannot chown socket to root: %v", err)
			}

			err := checkContainerdSocketAccess(sock)
			require.Error(t, err)
			require.Contains(t, err.Error(), "owned by root")
		})
	}
}

func TestExecutionStateSetGetResult(t *testing.T) {
	t.Parallel()

	state := &executionState{}
	require.Nil(t, state.getResult())

	expected := &types.ExecutionResult{ExitCode: 7}
	state.setResult(expected)
	require.Same(t, expected, state.getResult())
}

func TestCommandArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entrypoint []string
		cmd        []string
		expected   []string
	}{
		{
			name:       "entrypoint and cmd",
			entrypoint: []string{"/bin/sh", "-c"},
			cmd:        []string{"echo hi"},
			expected:   []string{"/bin/sh", "-c", "echo hi"},
		},
		{
			name:       "only cmd",
			entrypoint: nil,
			cmd:        []string{"sleep", "1"},
			expected:   []string{"sleep", "1"},
		},
		{
			name:       "empty",
			entrypoint: nil,
			cmd:        nil,
			expected:   []string{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, commandArgs(tc.entrypoint, tc.cmd))
		})
	}
}

func TestWithNamespace(t *testing.T) {
	t.Parallel()

	e := &Executor{cfg: config.Containerd{Namespace: "test-ns"}}

	ctx := e.withNamespace(context.Background())
	ns, ok := namespaces.Namespace(ctx)
	require.True(t, ok)
	require.Equal(t, "test-ns", ns)
}
