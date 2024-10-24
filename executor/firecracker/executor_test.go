// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package firecracker_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

// ExecutorTestSuite is the test suite for the Firecracker executor.
type ExecutorTestSuite struct {
	suite.Suite
	executor *firecracker.Executor
}

// SetupTest sets up the test suite by initializing a new Firecracker executor.
func (s *ExecutorTestSuite) SetupTest() {
	e, err := firecracker.NewExecutor(context.Background(), "test_firecracker_executor")
	s.NoError(err)

	s.executor = e
	s.T().Cleanup(func() {
		_ = s.executor.Cleanup()
	})
}

// TestExecutorTestSuite runs the test suite for the Firecracker executor.
func TestExecutorTestSuite(t *testing.T) {
	ensureFirecrackerSetup(t)
	suite.Run(t, new(ExecutorTestSuite))
}

// newExecutionRequest creates a new job request for testing.
func (s *ExecutorTestSuite) newExecutionRequest(persistent bool) *types.ExecutionRequest {
	engine := firecracker.NewFirecrackerEngineBuilder(rootDrivePath).
		WithKernelImage(kernelImagePath).
		Build()
	executionID := fmt.Sprintf("test_execution-%s", uuid.New())
	// This is here to make sure even long running tests will eventually finish.
	if !persistent {
		go func() {
			// time.Sleep(time.Second)
			_ = s.executor.Cancel(context.Background(), executionID)
		}()
	}

	return &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: executionID,
		EngineSpec:  engine,
		Resources: &types.Resources{
			CPU: types.CPU{Cores: 1},
			RAM: types.RAM{Size: 1024},
		},
	}
}

// Test StartJob tests the Start method of the Firecracker executor.
func (s *ExecutorTestSuite) TestStartJob() {
	request := s.newExecutionRequest(false)
	err := s.executor.Start(context.Background(), request)
	s.NoError(err)
}

// // Test RunJob tests the Run method of the Firecracker executor.
func (s *ExecutorTestSuite) TestRunJob() {
	request := s.newExecutionRequest(false)
	result, err := s.executor.Run(context.Background(), request)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(types.ExecutionStatusCodeSuccess, result.ExitCode)
}

// Test WaitJob tests the Wait method of the Firecracker executor.
func (s *ExecutorTestSuite) TestWaitJob() {
	request := s.newExecutionRequest(false)
	err := s.executor.Start(context.Background(), request)
	s.NoError(err)

	resultCh, errCh := s.executor.Wait(context.Background(), request.ExecutionID)
	select {
	case result := <-resultCh:
		s.NotNil(result)
		s.Equal(types.ExecutionStatusCodeSuccess, result.ExitCode)
	case err := <-errCh:
		s.NoError(err)
	}
}

// Test GetStatus tests the GetStatus metod of the Docker executor.
func (s *ExecutorTestSuite) TestGetStatus() {
	ctx := context.Background()
	// Create and start a persistent container
	request := s.newExecutionRequest(true)
	err := s.executor.Start(ctx, request)
	s.NoError(err)

	// Check container is running or pending
	status, err := s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Contains([]types.ExecutionStatus{types.ExecutionStatusPending, types.ExecutionStatusRunning}, status)

	// Wait for the container execution status is running
	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	status, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusRunning, status)

	// Pause the container and check status
	err = s.executor.Pause(ctx, request.ExecutionID)
	s.NoError(err)
	status, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusPaused, status)

	// Resume the container and check status
	err = s.executor.Resume(ctx, request.ExecutionID)
	s.NoError(err)
	status, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusRunning, status)

	err = s.executor.Cancel(ctx, request.ExecutionID)
	// wait until it is killed
	resCh, errCh := s.executor.Wait(ctx, request.ExecutionID)
	select {
	case <-resCh:
	case <-errCh:
	}
	s.NoError(err)
	status, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusSuccess, status)

	// Create and start a transient VM
	request = s.newExecutionRequest(false)
	err = s.executor.Start(ctx, request)
	s.NoError(err)

	// Wait for the container to complete
	resultCh, errCh := s.executor.Wait(ctx, request.ExecutionID)
	select {
	case <-resultCh:
		status, err = s.executor.GetStatus(ctx, request.ExecutionID)
		s.NoError(err)
		s.Equal(types.ExecutionStatusSuccess, status)
	case err := <-errCh:
		s.NoError(err)
	}
}
