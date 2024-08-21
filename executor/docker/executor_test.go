package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/types"
)

// ExecutorTestSuite is the test suite for the Docker executor.
type ExecutorTestSuite struct {
	suite.Suite
	executor *docker.Executor
}

// SetupTest sets up the test suite by initializing a new Docker executor.
func (s *ExecutorTestSuite) SetupTest() {
	e, err := docker.NewExecutor(context.Background(), "test_docker_executor")
	require.NoError(s.T(), err)
	s.executor = e
	s.T().Cleanup(func() {
		_ = s.executor.Cleanup(context.Background())
	})
}

// TestExecutorTestSuite runs the test suite for the Docker executor.
func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

// newJobRequest creates a new job request for testing.
func (s *ExecutorTestSuite) newJobRequest() *types.ExecutionRequest {
	engine := docker.NewDockerEngineBuilder(defaultImage).WithCmd(defaultCmd...).Build()
	return &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: "test_execution",
		EngineSpec:  engine,
		Resources: &types.ExecutionResources{
			CPU:    types.CPU{ClockSpeedHz: 1024, Cores: 1},
			Memory: types.RAM{Size: 1024},
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
