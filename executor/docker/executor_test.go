// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// ExecutorTestSuite is the test suite for the Docker executor.
type ExecutorTestSuite struct {
	suite.Suite
	executor *docker.Executor
}

// SetupTest sets up the test suite by initializing a new Docker executor.
func (s *ExecutorTestSuite) SetupTest() {
	randomSuffix, err := utils.RandomString(10)
	require.NoError(s.T(), err)
	e, err := docker.NewExecutor(context.Background(), "test_docker_executor"+randomSuffix)
	require.NoError(s.T(), err)
	s.executor = e
	s.T().Cleanup(func() {
		_ = s.executor.Cleanup(context.Background())
	})
}

// TestExecutorTestSuite runs the test suite for the Docker executor.
func TestExecutorTestSuite(t *testing.T) {
	ensureDockerSetup(t)
	suite.Run(t, new(ExecutorTestSuite))
}

// newJobRequest creates a new job request for testing.
func (s *ExecutorTestSuite) newJobRequest() *types.ExecutionRequest {
	engine := docker.NewDockerEngineBuilder(defaultImage).WithCmd(defaultCmd...).Build()
	return &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: "test_execution",
		EngineSpec:  engine,
		Resources: &types.Resources{
			CPU: types.CPU{ClockSpeed: 1024, Cores: 1},
			RAM: types.RAM{Size: 1024},
		},
	}
}

// Test StartJob tests the Start method of the Docker executor.
func (s *ExecutorTestSuite) TestStartJob() {
	request := s.newJobRequest()
	err := s.executor.Start(context.Background(), request)
	require.NoError(s.T(), err)
}

// Test RunJob tests the Run method of the Docker executor.
func (s *ExecutorTestSuite) TestRunJob() {
	request := s.newJobRequest()
	result, err := s.executor.Run(context.Background(), request)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)
	require.Equal(s.T(), types.ExecutionStatusCodeSuccess, result.ExitCode)
	require.NotNil(s.T(), result.STDOUT)
}

// Test WaitJob tests the Wait method of the Docker executor.
func (s *ExecutorTestSuite) TestWaitJob() {
	request := s.newJobRequest()
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
