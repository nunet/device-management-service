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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.NoError(s.T(), err)
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

// newJobRequest creates a new job request for testing.
func (s *ExecutorTestSuite) newJobRequest(executionID string) *types.ExecutionRequest {
	engine := firecracker.NewFirecrackerEngineBuilder(rootDrivePath).
		WithKernelImage(kernelImagePath).
		Build()
	// This is here to make sure even long running tests will eventually finish.
	go func() {
		time.Sleep(3 * time.Second)
		_ = s.executor.Cancel(context.Background(), executionID)
	}()

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
	request := s.newJobRequest("start_job_test")
	err := s.executor.Start(context.Background(), request)
	require.NoError(s.T(), err)
}

// // Test RunJob tests the Run method of the Firecracker executor.
func (s *ExecutorTestSuite) TestRunJob() {
	request := s.newJobRequest("run_job_test")
	result, err := s.executor.Run(context.Background(), request)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)
	require.Equal(s.T(), types.ExecutionStatusCodeSuccess, result.ExitCode)
}

// Test WaitJob tests the Wait method of the Firecracker executor.
func (s *ExecutorTestSuite) TestWaitJob() {
	request := s.newJobRequest("wait_job_test")
	err := s.executor.Start(context.Background(), request)
	require.NoError(s.T(), err)

	resultCh, errCh := s.executor.Wait(context.Background(), request.ExecutionID)
	select {
	case result := <-resultCh:
		require.NotNil(s.T(), result)
		require.Equal(s.T(), types.ExecutionStatusCodeSuccess, result.ExitCode)
	case err := <-errCh:
		require.NoError(s.T(), err)
	}
}
