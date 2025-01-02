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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	persistLogDurationTest = time.Second * 1
	removeContainerTimeout = time.Second * 5
	testDirLogs            = "/tmp/nunet/tests/"
)

// ExecutorTestSuite is the test suite for the Docker executor.
type ExecutorTestSuite struct {
	suite.Suite
	executor *docker.Executor

	fs afero.Afero
}

// SetupTest sets up the test suite by initializing a new Docker executor.
func (s *ExecutorTestSuite) SetupTest() {
	// Set observability to no-op mode for this test
	observability.SetNoOpMode(true)
	s.fs = afero.Afero{Fs: afero.NewOsFs()}
	e, err := docker.NewExecutor(context.Background(), s.fs, "test_docker_executor")
	s.NoError(err)
	s.executor = e
	s.T().Cleanup(func() {
		_ = s.executor.Cleanup(context.Background())
		// wait for logs being cleaned up from disk
		<-time.After(persistLogDurationTest + (time.Second * 1))
	})
}

// TestExecutorTestSuite runs the test suite for the Docker executor.
func TestExecutorTestSuite(t *testing.T) {
	ensureDockerSetup(t)
	suite.Run(t, new(ExecutorTestSuite))
}

// newExecutionRequest creates a new execution request for testing.
func (s *ExecutorTestSuite) newExecutionRequest(cmd []string) *types.ExecutionRequest {
	engine := docker.NewDockerEngineBuilder(defaultImage).WithCmd(cmd...).Build()
	execID := fmt.Sprintf("test_execution-%s", uuid.New())
	return &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: execID,
		EngineSpec:  engine,
		Resources: &types.Resources{
			CPU: types.CPU{ClockSpeed: 1024, Cores: 1},
			RAM: types.RAM{Size: 1024},
		},
		ResultsDir:          filepath.Join(testDirLogs, execID),
		PersistLogsDuration: persistLogDurationTest,
		GatewayIP:           "10.0.0.1",
	}
}

// TestStartJob tests the Start method of the Docker executor.
func (s *ExecutorTestSuite) TestStartJob() {
	request := s.newExecutionRequest(transientCmd)
	err := s.executor.Start(context.Background(), request)
	s.NoError(err)
}

func (s *ExecutorTestSuite) TestRemoveContainer() {
	request := s.newExecutionRequest(transientCmd)
	err := s.executor.Start(context.Background(), request)
	s.NoError(err)

	ctx := context.Background()
	_, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	cont, err := s.executor.FindRunningContainer(ctx, request.JobID, request.ExecutionID)
	s.NoError(err)
	s.NotEmpty(cont)

	err = s.executor.Remove(request.ExecutionID, removeContainerTimeout)
	s.NoError(err)

	cont, err = s.executor.FindRunningContainer(ctx, request.JobID, request.ExecutionID)
	s.Error(err)
	s.Contains(err.Error(), "unable to find container")
	s.Empty(cont)
}

// TestSavedLogs starts a job and checks if logs are being persisted to disk.
// Log files are updated while the container is running
func (s *ExecutorTestSuite) TestSavedLogs() {
	wordBeforeSleep := "one"
	wordAfterSleep := "two"

	// Create command that prints first word, sleeps, then prints second word
	request := s.newExecutionRequest([]string{
		"sh", "-c",
		fmt.Sprintf("echo '%s'; sleep 5; echo '%s'", wordBeforeSleep, wordAfterSleep),
	})

	ctx := context.Background()

	// Start the execution
	err := s.executor.Start(ctx, request)
	s.NoError(err)

	// Wait for container to actually be running
	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)
	time.Sleep(time.Second * 2)

	// Check intermediate state - should only have first word
	stdoutPath := filepath.Join(request.ResultsDir, "stdout.log")
	content, err := s.fs.ReadFile(stdoutPath)
	s.NoError(err)
	s.Contains(string(content), wordBeforeSleep, "First word should be written to log file while container is still running")
	s.NotContains(string(content), wordAfterSleep, "Second word should not be written yet")

	// Wait for completion
	resultCh, errCh := s.executor.Wait(ctx, request.ExecutionID)

	select {
	case result := <-resultCh:
		s.Equal(0, result.ExitCode)

		s.Contains(result.STDOUT, wordBeforeSleep, "First word should be in final result")
		s.Contains(result.STDOUT, wordAfterSleep, "Second word should be in final result")
	case err := <-errCh:
		s.NoError(err)
	}
}

// TestRunJob tests the Run method of the Docker executor.
func (s *ExecutorTestSuite) TestRunJob() {
	request := s.newExecutionRequest(transientCmd)
	result, err := s.executor.Run(context.Background(), request)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(types.ExecutionStatusCodeSuccess, result.ExitCode)
	s.NotEmpty(result.STDOUT)
}

// Test RunJobWithInitScripts tests the Run method of the Docker executor with ProvisionScripts.
//
// CI/CD: if the runner is within a container/VM, `/tmp/nunet/` need to be mounted to it since
// Docker socket expects the path to available on the host
func (s *ExecutorTestSuite) TestRunJobWithInitScripts() {
	request := s.newExecutionRequest(transientCmd)
	request.ProvisionScripts = map[string][]byte{
		"script1": []byte("#!/usr/bin/env python\nprint(\"hello_init\")"),
		"script2": []byte("#!/bin/sh\necho bye_init"),
	}

	result, err := s.executor.Run(context.Background(), request)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(types.ExecutionStatusCodeSuccess, result.ExitCode)
	s.Contains(result.STDOUT, "hello_init")
	s.Contains(result.STDOUT, "bye_init")
}

// Test WaitJob tests the Wait method of the Docker executor.
func (s *ExecutorTestSuite) TestWaitJob() {
	request := s.newExecutionRequest(transientCmd)
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
	request := s.newExecutionRequest(persistentCmd)
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

	// Stop the container and check status
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
	s.Equal(types.ExecutionStatusFailed, status)

	// Create and start a transient container
	request = s.newExecutionRequest(transientCmd)
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
