// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

var DestroyTimeout = time.Second * 10

// executionHandler manages the lifecycle and execution of a Docker container for a specific job.
type executionHandler struct {
	// provided by the executor
	ID     string
	client *Client // Docker client for container management.
	fs     afero.Afero

	// meta data about the task
	jobID       string
	executionID string
	containerID string
	resultsDir  string // Directory to store execution results.

	// synchronization
	activeCh chan bool    // Blocks until the container starts running.
	waitCh   chan bool    // Blocks until execution completes or fails.
	running  *atomic.Bool // Indicates if the container is currently running.

	// result of the execution
	result              *types.ExecutionResult
	persistLogsDuration time.Duration

	// log files
	stdoutFile afero.File
	stderrFile afero.File

	// TTY setting
	TTYEnabled bool // Indicates if TTY is enabled for the container.
}

// active checks if the execution handler's container is running.
func (h *executionHandler) active() bool {
	return h.running.Load()
}

// run starts the container and handles its execution lifecycle.
func (h *executionHandler) run(ctx context.Context) {
	endTrace := observability.StartTrace(ctx, "docker_execution_handler_run_duration")
	defer endTrace()

	h.running.Store(true)
	defer close(h.waitCh)

	if err := h.prepareLogFiles(); err != nil {
		err = fmt.Errorf("failed to create execution log files: %v", err)
		log.Errorw("docker_execution_handler_run_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		h.result = types.NewFailedExecutionResult(err)
		return
	}

	// monitor kill events to see if the container was externally killed
	eventsCh, errEventsCh := h.client.client.Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("container", h.containerID),
			filters.Arg("event", "kill"),
		),
	})

	if err := h.client.StartContainer(ctx, h.containerID); err != nil {
		h.result = types.NewFailedExecutionResult(fmt.Errorf("failed to start container: %v", err))
		log.Errorw("docker_execution_handler_run_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}

	close(h.activeCh)
	log.Infow("docker_execution_handler_run_success",
		"labels", string(observability.LabelDeployment),
		"executionID", h.executionID)

	var containerError error
	var containerExitStatusCode int64

	// Start streaming logs before waiting for container exit
	stdoutPipe, stderrPipe, logsErr := h.client.FollowLogs(ctx, h.containerID)
	if logsErr != nil {
		followError := fmt.Errorf("failed to follow container logs: %w", logsErr)
		log.Errorw("docker_execution_handler_follow_logs_failure",
			"error", logsErr)
		h.result = &types.ExecutionResult{
			ExitCode: int(containerExitStatusCode),
			ErrorMsg: followError.Error(),
		}
		return
	}

	// Create channels to signal when log copying is done
	stdoutDone := make(chan bool)
	stderrDone := make(chan bool)

	// Start copying stdout in background
	go func() {
		defer close(stdoutDone)
		defer func() {
			if h.stdoutFile != nil {
				err := h.stdoutFile.Close()
				if err != nil {
					log.Warnf("Error closing stdout file: %v", err)
				}
			}
		}()

		if _, err := io.Copy(h.stdoutFile, stdoutPipe); err != nil {
			log.Warnf("Error copying stdout: %v", err)
		}
	}()

	// Start copying stderr in background
	go func() {
		defer close(stderrDone)
		defer func() {
			if h.stderrFile != nil {
				err := h.stderrFile.Close()
				if err != nil {
					log.Warnf("Error closing stderr file: %v", err)
				}
			}
		}()

		if _, err := io.Copy(h.stderrFile, stderrPipe); err != nil {
			log.Warnf("Error copying stderr: %v", err)
		}
	}()

	// Wait for container exit status while logs are being copied
	statusCh, errCh := h.client.WaitContainer(ctx, h.containerID)
	select {
	case status := <-ctx.Done():
		h.result = types.NewFailedExecutionResult(fmt.Errorf("execution cancelled: %v", status))
		log.Errorw("docker_execution_handler_run_failure_cancelled", "executionID", h.executionID)
		return
	case err := <-errCh:
		log.Errorf("error while waiting for container: %v\n", err)
		h.result = types.NewFailedExecutionResult(
			fmt.Errorf("failed to wait for container: %v", err),
		)
		return
	case exitStatus := <-statusCh:
		containerExitStatusCode = exitStatus.StatusCode
		containerJSON, err := h.client.InspectContainer(ctx, h.containerID)
		if err != nil {
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: err.Error(),
			}
			log.Errorw("docker_execution_handler_inspect_container_failure", "error", err)
			return
		}

		if containerJSON.State.OOMKilled {
			containerError = errors.New("container was killed due to OOM")
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: containerError.Error(),
			}
			log.Errorw("docker_execution_handler_container_oom_killed",
				"labels", string(observability.LabelAllocation),
				"executionID", h.executionID)
			return
		}

		if exitStatus.Error != nil {
			containerError = errors.New(exitStatus.Error.Message)
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: containerError.Error(),
			}
			log.Errorw("docker_execution_handler_container_error", "executionID", h.executionID)
			return
		}

		if containerExitStatusCode != 0 {
			containerError = errors.New("container either killed or application returned non-zero exit code")
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: containerError.Error(),
			}
			log.Errorw("docker_execution_handler_non_zero_exit_code", "executionID", h.executionID)
			return
		}

		// Here we're trying to identify containers terminated by external means but
		// were returned, gracefully, with a 0 exit code.
		if containerExitStatusCode == 0 {
			// Check if there were any kill events
			select {
			case err := <-errEventsCh:
				if err != nil {
					log.Warnw("docker_execution_handler_events_error", "error", err)
				}
			case <-eventsCh:
				h.result = &types.ExecutionResult{
					ExitCode: int(containerExitStatusCode),
					ErrorMsg: "container was killed by external signal",
					Killed:   true,
				}
				log.Infow("docker_execution_handler_container_killed", "executionID", h.executionID)
				return
			default:
			}
		}
	}

	// Initialize the result with the exit status code
	h.result = types.NewExecutionResult(int(containerExitStatusCode))

	// Wait for log copying to complete
	<-stdoutDone
	<-stderrDone

	// Read the complete logs for the result
	if stdout, err := os.ReadFile(filepath.Join(h.resultsDir, "stdout.log")); err == nil {
		h.result.STDOUT = string(stdout)
	} else {
		log.Errorf("failed to read stdout logs file and retrieve logs: %v", err)
	}

	if stderr, err := os.ReadFile(filepath.Join(h.resultsDir, "stderr.log")); err == nil {
		h.result.STDERR = string(stderr)
	} else {
		log.Errorf("failed to read stderr logs file and retrieve logs: %v", err)
	}

	log.Infow("docker_execution_handler_run_logs_success", "executionID", h.executionID)

	h.running.Store(false)
}

// pause pauses the main process of the container without terminating it.
func (h *executionHandler) pause(ctx context.Context) error {
	return h.client.PauseContainer(ctx, h.containerID)
}

// resume resumes the process execution within the container
func (h *executionHandler) resume(ctx context.Context) error {
	return h.client.ResumeContainer(ctx, h.containerID)
}

// kill sends a stop signal to the container.
func (h *executionHandler) kill(ctx context.Context) error {
	endTrace := observability.StartTrace(ctx, "docker_execution_handler_kill_duration")
	defer endTrace()

	timeout := int(DestroyTimeout)
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	err := h.client.StopContainer(ctx, h.containerID, stopOptions)
	if err != nil {
		log.Errorw("docker_execution_handler_kill_failure",
			"labels", string(observability.LabelAllocation),
			"error", err,
			"executionID", h.executionID)
		return err
	}
	log.Infow("docker_execution_handler_kill_success",
		"labels", string(observability.LabelAllocation),
		"executionID", h.executionID)
	return nil
}

// destroy cleans up the container and its associated resources.
func (h *executionHandler) destroy(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	endTrace := observability.StartTrace(ctx, "docker_execution_handler_destroy_duration")
	defer endTrace()

	// stop the container
	if err := h.kill(ctx); err != nil {
		log.Errorw("docker_execution_handler_destroy_failure",
			"labels", string(observability.LabelDeployment),
			"error", err,
			"executionID", h.executionID)
		return fmt.Errorf("failed to kill container (%s): %w", h.containerID, err)
	}

	if err := h.client.RemoveContainer(ctx, h.containerID); err != nil {
		log.Errorw("docker_execution_handler_destroy_failure",
			"labels", string(observability.LabelDeployment),
			"error", err,
			"executionID", h.executionID)
		return err
	}

	// remove init scripts
	err := os.RemoveAll(initScriptsBaseDir + h.executionID)
	if err != nil {
		return err
	}

	// Remove related objects like networks or volumes created for this execution.
	err = h.client.RemoveObjectsWithLabel(
		ctx,
		labelExecutionID,
		labelExecutionValue(h.ID, h.jobID, h.executionID),
	)
	if err != nil {
		log.Errorw("docker_execution_handler_destroy_failure",
			"labels", string(observability.LabelDeployment),
			"error", err,
			"executionID", h.executionID)
		return err
	}

	h.handleDeletionOfLogFiles(h.persistLogsDuration)

	log.Infow("docker_execution_handler_destroy_success",
		"labels", string(observability.LabelDeployment),
		"executionID", h.executionID)
	return nil
}

func (h *executionHandler) outputStream(
	ctx context.Context,
	request types.LogStreamRequest,
) (io.ReadCloser, error) {
	endTrace := observability.StartTrace(ctx, "docker_execution_handler_output_stream_duration")
	defer endTrace()

	since := "1" // Default to the start of UNIX time to get all logs.
	if request.Tail {
		since = strconv.FormatInt(time.Now().Unix(), 10)
	}
	select {
	case <-ctx.Done():
		log.Errorw("docker_execution_handler_output_stream_canceled", "executionID", h.executionID)
		return nil, ctx.Err()
	case <-h.activeCh: // Ensure the container is active before attempting to stream logs.
	}

	// Gets the underlying reader, and provides data since the value of the `since` timestamp.
	reader, err := h.client.GetOutputStream(ctx, h.containerID, since, request.Follow)
	if err != nil {
		log.Errorw("docker_execution_handler_output_stream_failure",
			"error", err,
			"executionID", h.executionID)
		return nil, err
	}

	log.Infow("docker_execution_handler_output_stream_success", "executionID", h.executionID)
	return reader, nil
}

// status returns the result of the execution.
func (h *executionHandler) status(ctx context.Context) (types.ExecutionStatus, error) {
	if h.result != nil {
		if h.result.ExitCode == types.ExecutionStatusCodeSuccess {
			return types.ExecutionStatusSuccess, nil
		}
		return types.ExecutionStatusFailed, nil
	}
	info, err := h.client.InspectContainer(ctx, h.containerID)
	if err != nil {
		return types.ExecutionStatusFailed, fmt.Errorf("failed to get container status: %v", err)
	}
	switch info.State.Status {
	case "created":
		return types.ExecutionStatusPending, nil
	case "running":
		return types.ExecutionStatusRunning, nil
	case "paused":
		return types.ExecutionStatusPaused, nil
	case "exited":
		if info.State.ExitCode == 0 {
			return types.ExecutionStatusSuccess, nil
		}
		return types.ExecutionStatusFailed, nil
	case "dead":
		return types.ExecutionStatusFailed, nil
	default:
		return types.ExecutionStatusFailed, fmt.Errorf("unknown container status: %s", info.State.Status)
	}
}

// prepareLogFiles creates/opens the log files in the results directory
func (h *executionHandler) prepareLogFiles() error {
	log.Debug("preparing log files")
	if err := os.MkdirAll(h.resultsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	var err error
	h.stdoutFile, err = h.fs.OpenFile(filepath.Join(h.resultsDir, "stdout.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open stdout file: %w", err)
	}
	log.Debugf("stdout saved to: %s", filepath.Join(h.resultsDir, "stdout.log"))

	if !h.TTYEnabled {
		h.stderrFile, err = h.fs.OpenFile(filepath.Join(h.resultsDir, "stderr.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			h.stdoutFile.Close()
			return fmt.Errorf("failed to open stderr file: %w", err)
		}
		log.Debugf("stderr saved to: %s", filepath.Join(h.resultsDir, "stderr.log"))
	}

	return nil
}

// handleDeletionOfLogFiles closes the log files
func (h *executionHandler) handleDeletionOfLogFiles(after time.Duration) {
	go func() {
		log.Debugf("deleting log files after %s", after)
		<-time.After(after)
		log.Debug("closing log files")
		err := h.fs.RemoveAll(h.resultsDir)
		if err != nil {
			log.Errorf("failed to remove log files: %v", err)
		}
	}()
}
