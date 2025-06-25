// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package null

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewExecutor(t *testing.T) {
	ctx := context.Background()
	executor, err := NewExecutor(ctx, "test-id")

	assert.NoError(t, err, "NewExecutor() should not return error")
	assert.NotNil(t, executor, "NewExecutor() should not return nil executor")
}

func TestGetID(t *testing.T) {
	executor := &Executor{}
	id := executor.GetID()

	assert.Empty(t, id, "GetID() should return empty string")
}

func TestExec(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	exitCode, stdout, stderr, err := executor.Exec(ctx, "test-command", []string{"arg1", "arg2"})

	assert.NoError(t, err, "Exec() should not return error")
	assert.Equal(t, 0, exitCode, "Exec() exitCode should be 0")
	assert.Empty(t, stdout, "Exec() stdout should be empty string")
	assert.Empty(t, stderr, "Exec() stderr should be empty string")
}

func TestStart(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()
	req := &types.ExecutionRequest{}

	err := executor.Start(ctx, req)
	assert.NoError(t, err, "Start() should not return error")
}

func TestRun(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()
	req := &types.ExecutionRequest{}

	result, err := executor.Run(ctx, req)
	assert.NoError(t, err, "Run() should not return error")
	assert.Nil(t, result, "Run() result should be nil")
}

func TestWait(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	resultCh, errCh := executor.Wait(ctx, "test-execution-id")

	// Channels should be closed immediately
	select {
	case _, ok := <-resultCh:
		assert.False(t, ok, "resultCh should be closed")
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "resultCh should be closed immediately")
	}

	select {
	case _, ok := <-errCh:
		assert.False(t, ok, "errCh should be closed")
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "errCh should be closed immediately")
	}
}

func TestCancel(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	err := executor.Cancel(ctx, "test-execution-id")
	assert.NoError(t, err, "Cancel() should not return error")
}

func TestRemove(t *testing.T) {
	executor := &Executor{}

	err := executor.Remove("test-execution-id", time.Minute)
	assert.NoError(t, err, "Remove() should not return error")
}

func TestList(t *testing.T) {
	executor := &Executor{}

	items := executor.List()
	assert.NotNil(t, items, "List() should not return nil")
	assert.Len(t, items, 0, "List() should return 0 items")
}

func TestCleanup(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	err := executor.Cleanup(ctx)
	assert.NoError(t, err, "Cleanup() should not return error")
}

func TestGetStatus(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	status, err := executor.GetStatus(ctx, "test-execution-id")
	assert.NoError(t, err, "GetStatus() should not return error")
	assert.Empty(t, status, "GetStatus() status should be empty string")
}

func TestPause(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	err := executor.Pause(ctx, "test-execution-id")
	assert.NoError(t, err, "Pause() should not return error")
}

func TestResume(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()

	err := executor.Resume(ctx, "test-execution-id")
	assert.NoError(t, err, "Resume() should not return error")
}

func TestWaitForStatus(t *testing.T) {
	executor := &Executor{}
	ctx := context.Background()
	timeout := time.Minute

	err := executor.WaitForStatus(ctx, "test-execution-id", "running", &timeout)
	assert.NoError(t, err, "WaitForStatus() should not return error")
}
