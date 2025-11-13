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
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	baseSleep              = 2
	baseTimeout            = time.Second * 2
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
		JobID:       uuid.NewString(),
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
	request := s.newExecutionRequest(persistentCmd)
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
		fmt.Sprintf("echo '%s'; sleep %d; echo '%s'", wordBeforeSleep, baseSleep, wordAfterSleep),
	})

	ctx := context.Background()

	// Start the execution
	err := s.executor.Start(ctx, request)
	s.NoError(err)

	// Wait for container to actually be running
	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	// Check intermediate state - should only have first word
	stdoutPath := filepath.Join(request.ResultsDir, "stdout.log")
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		content, err := s.fs.ReadFile(stdoutPath)
		assert.NoError(c, err)
		assert.Contains(c, string(content), wordBeforeSleep, "First word should be written to log file while container is still running")
		assert.NotContains(c, string(content), wordAfterSleep, "Second word should not be written yet")
	}, baseTimeout+1, 100*time.Millisecond)

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

// TestRunJobWithKeys tests running a job with SSH keys.
func (s *ExecutorTestSuite) TestRunJobWithKeys() {
	ctx := context.Background()
	request := s.newExecutionRequest(persistentCmd)
	expectedKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"
	request.Keys = []types.AllocationKey{
		{
			Type: types.KeySSH,
			File: expectedKey,
		},
	}

	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	// check if the SSH key was copied correctly
	exitCode, stdout, stderr, err := s.executor.Exec(ctx, request.ExecutionID, []string{"cat", "/root/.ssh/authorized_keys"})
	s.NoError(err)
	s.Equal(0, exitCode)
	s.Contains(stdout, expectedKey)
	s.Empty(stderr)

	// verify the file permissions are correct (600)
	exitCode, stdout, _, err = s.executor.Exec(ctx, request.ExecutionID, []string{"stat", "-c", "%a", "/root/.ssh/authorized_keys"})
	s.NoError(err)
	s.Equal(0, exitCode)
	s.Equal("600", strings.TrimSpace(stdout))

	err = s.executor.Cancel(ctx, request.ExecutionID)
	s.NoError(err)
}

// TestRunJobWithPortBinding tests running a job with port binding.
func (s *ExecutorTestSuite) TestRunJobWithPortBinding() {
	// TODO: Improve this test to actually test the port binding,
	// Reason: netcat was not working on CI
	request := s.newExecutionRequest(persistentCmd)
	request.PortsToBind = []types.PortsToBind{
		{
			IP:           "127.0.0.1",
			HostPort:     8080,
			ExecutorPort: 3000, // use unprivileged port
		},
	}

	err := s.executor.Start(context.Background(), request)
	s.NoError(err)

	// Clean up
	err = s.executor.Cancel(context.Background(), request.ExecutionID)
	s.NoError(err)
}

// TestRunJobWithVolumes tests running a job with input/output volumes.
func (s *ExecutorTestSuite) TestRunJobWithVolumes() {
	// Create temporary directories for input and output
	inputDir := filepath.Join(testDirLogs, "input")
	outputDir := filepath.Join(testDirLogs, "output")

	err := s.fs.MkdirAll(inputDir, 0o755)
	s.NoError(err)
	err = s.fs.MkdirAll(outputDir, 0o755)
	s.NoError(err)

	// Create a test input file
	inputFile := filepath.Join(inputDir, "test.txt")
	expected := "test input"
	err = s.fs.WriteFile(inputFile, []byte(expected), 0o644)
	s.NoError(err)

	request := s.newExecutionRequest([]string{"sh", "-c", "cp /input/test.txt /output/result.txt"})
	request.Inputs = []*types.StorageVolumeExecutor{
		{
			Type:     "bind",
			Source:   inputDir,
			Target:   "/input",
			ReadOnly: true,
		},
	}
	request.Outputs = []*types.StorageVolumeExecutor{
		{
			Type:   "bind",
			Source: outputDir,
			Target: "/output",
		},
	}

	result, err := s.executor.Run(context.Background(), request)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(types.ExecutionStatusCodeSuccess, result.ExitCode)

	// Verify output file was created
	outputFile := filepath.Join(outputDir, "result.txt")
	content, err := s.fs.ReadFile(outputFile)
	s.NoError(err)
	s.Equal(expected, string(content))
}

// TestFindRunningContainerNotFound tests finding a container that doesn't exist.
func (s *ExecutorTestSuite) TestFindRunningContainerNotFound() {
	ctx := context.Background()
	containerID, err := s.executor.FindRunningContainer(ctx, "non-existent-job", "non-existent-execution")
	s.Error(err)
	s.Empty(containerID)

	_, err = s.executor.GetStatus(context.Background(), "non-existent")
	s.Error(err)

	err = s.executor.Remove("non-existent", removeContainerTimeout)
	s.Error(err)

	timeout := 100 * time.Millisecond
	err = s.executor.WaitForStatus(context.Background(), "non-existent", types.ExecutionStatusRunning, &timeout)
	s.Error(err)
}

// TestRunWithContextCancel tests running a job with context cancellation.
func (s *ExecutorTestSuite) TestRunWithContextCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	request := s.newExecutionRequest(persistentCmd)

	// Start the job
	err := s.executor.Start(ctx, request)
	s.NoError(err)

	// Cancel the context
	cancel()

	// Wait should return context error
	resultCh, errCh := s.executor.Wait(ctx, request.ExecutionID)
	select {
	case <-resultCh:
		s.Fail("Expected context cancellation error")
	case err := <-errCh:
		s.Error(err)
		s.Equal(context.Canceled, err)
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

// TestStartActiveHandler tests starting an execution that already exists.
func (s *ExecutorTestSuite) TestStartActiveHandler() {
	request := s.newExecutionRequest(persistentCmd)
	err := s.executor.Start(context.Background(), request)
	s.NoError(err)

	// Update JobID so that Job is not found, but execution is the same
	request.JobID = uuid.NewString()

	// Try to start the same execution again
	err = s.executor.Start(context.Background(), request)
	s.Error(err)
}

// TestPauseResume tests the Pause and Resume methods of the Docker executor.
func (s *ExecutorTestSuite) TestPauseResume() {
	ctx := context.Background()
	request := s.newExecutionRequest(persistentCmd)

	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	// Pause the container
	err = s.executor.Pause(ctx, request.ExecutionID)
	s.NoError(err)

	status, err := s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusPaused, status)

	// Resume the container
	err = s.executor.Resume(ctx, request.ExecutionID)
	s.NoError(err)

	status, err = s.executor.GetStatus(ctx, request.ExecutionID)
	s.NoError(err)
	s.Equal(types.ExecutionStatusRunning, status)

	// Test for non-existent executions
	err = s.executor.Pause(context.Background(), "non-existent")
	s.Error(err)

	err = s.executor.Resume(context.Background(), "non-existent")
	s.Error(err)
}

// TestCancelJob tests the Cancel method of the Docker executor.
func (s *ExecutorTestSuite) TestCancelJob() {
	ctx := context.Background()
	request := s.newExecutionRequest(persistentCmd)

	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	err = s.executor.Cancel(ctx, request.ExecutionID)
	s.NoError(err)

	// Wait for the execution to finish
	resultCh, errCh := s.executor.Wait(ctx, request.ExecutionID)
	select {
	case result := <-resultCh:
		s.NotNil(result)
	case err := <-errCh:
		s.NoError(err)
	}
}

// TestGetLogStream tests the GetLogStream method of the Docker executor.
func (s *ExecutorTestSuite) TestGetLogStream() {
	const msg = "test log"
	request := s.newExecutionRequest([]string{"sh", "-c", fmt.Sprintf("echo '%s'; sleep %d", msg, baseSleep-1)})

	ctx := context.Background()
	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	logRequest := types.LogStreamRequest{
		ExecutionID: request.ExecutionID,
		Tail:        false,
		Follow:      false,
	}

	stream, err := s.executor.GetLogStream(ctx, logRequest)
	s.NoError(err)
	s.NotNil(stream)
	defer stream.Close()

	readBytes, err := io.ReadAll(stream)
	s.NoError(err)
	s.Contains(string(readBytes), msg)

	// Test for non-existing request
	logRequest = types.LogStreamRequest{
		ExecutionID: "non-existent",
		Tail:        false,
		Follow:      false,
	}

	// Timeout after 5 seconds
	stream, err = s.executor.GetLogStream(context.Background(), logRequest)
	s.Error(err)
	s.Nil(stream)
}

// TestExec tests the Exec method of the Docker executor.
func (s *ExecutorTestSuite) TestExec() {
	ctx := context.Background()
	request := s.newExecutionRequest(persistentCmd)

	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	expected := "exec test"
	exitCode, stdout, stderr, err := s.executor.Exec(ctx, request.ExecutionID, []string{"echo", expected})
	s.NoError(err)
	s.Equal(0, exitCode)
	s.Contains(stdout, expected)
	s.Empty(stderr)
}

// TestStats tests the Stats method of the Docker executor.
func (s *ExecutorTestSuite) TestStats() {
	ctx := context.Background()
	request := s.newExecutionRequest(persistentCmd)

	err := s.executor.Start(ctx, request)
	s.NoError(err)

	err = s.executor.WaitForStatus(ctx, request.ExecutionID, types.ExecutionStatusRunning, nil)
	s.NoError(err)

	// Get stats for the running container
	stats, err := s.executor.Stats(ctx, request.ExecutionID)
	s.NoError(err)
	s.NotNil(stats)

	// Verify stats structure is populated
	s.Greater(stats.Timestamp, int64(0), "Timestamp should be set")
	s.GreaterOrEqual(stats.CPUUsage.TotalUsage, uint64(0), "CPU total usage should be non-negative")
	s.GreaterOrEqual(stats.CPUUsage.Percent, 0.0, "CPU percent should be non-negative")
	s.GreaterOrEqual(stats.Memory.Usage, uint64(0), "Memory usage should be non-negative")
	s.GreaterOrEqual(stats.Memory.Limit, uint64(0), "Memory limit should be non-negative")
	s.GreaterOrEqual(stats.Memory.Percent, 0.0, "Memory percent should be non-negative")

	// Test for non-existent execution
	stats, err = s.executor.Stats(ctx, "non-existent-execution")
	s.Error(err)
	s.Nil(stats)
	s.Contains(err.Error(), "execution (non-existent-execution) not found")

	// Clean up
	err = s.executor.Cancel(ctx, request.ExecutionID)
	s.NoError(err)
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

	resultCh, errCh = s.executor.Wait(context.Background(), "non-existent")

	select {
	case result := <-resultCh:
		s.Nil(result)
	case err := <-errCh:
		s.Error(err)
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
	// wait until it is killed
	resCh, errCh := s.executor.Wait(ctx, request.ExecutionID)
	err = s.executor.Cancel(ctx, request.ExecutionID)
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
