// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

var DestroyTimeout = time.Second * 10

// executionHandler manages the lifecycle and execution of a Docker container for a specific job.
type executionHandler struct {
	// provided by the executor
	ID     string
	client *Client // Docker client for container management.

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
	result *types.ExecutionResult

	// TTY setting
	TTYEnabled bool // Indicates if TTY is enabled for the container.
}

// active checks if the execution handler's container is running.
func (h *executionHandler) active() bool {
	return h.running.Load()
}

// run starts the container and handles its execution lifecycle.
func (h *executionHandler) run(ctx context.Context) {
	endTrace := observability.StartTrace("docker_execution_handler_run_duration")
	defer endTrace()

	h.running.Store(true)
	defer func() {
		if err := h.destroy(DestroyTimeout); err != nil {
			log.Warnf("failed to destroy container: %v\n", err)
		}
		h.running.Store(false)
		close(h.waitCh)
	}()

	if err := h.client.StartContainer(ctx, h.containerID); err != nil {
		h.result = types.NewFailedExecutionResult(fmt.Errorf("failed to start container: %v", err))
		log.Errorw("docker_execution_handler_run_failure", "error", err)
		return
	}

	close(h.activeCh) // Indicate that the container has started.
	log.Infow("docker_execution_handler_run_success", "executionID", h.executionID)

	var containerError error
	var containerExitStatusCode int64

	// Wait for the container to finish or for an execution error.
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
			log.Errorw("docker_execution_handler_container_oom_killed", "executionID", h.executionID)
			return
		}
		if exitStatus.Error != nil {
			containerError = errors.New(exitStatus.Error.Message)
		}
	}

	// Follow container logs to capture stdout and stderr.
	stdoutPipe, stderrPipe, logsErr := h.client.FollowLogs(ctx, h.containerID)
	if logsErr != nil {
		followError := fmt.Errorf("failed to follow container logs: %w", logsErr)
		if containerError != nil {
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: fmt.Sprintf(
					"container error: '%s'. logs error: '%s'",
					containerError,
					followError,
				),
			}
		} else {
			h.result = &types.ExecutionResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: followError.Error(),
			}
		}
		log.Errorw("docker_execution_handler_follow_logs_failure", "error", logsErr)
		return
	}

	// Initialize the result with the exit status code.
	h.result = types.NewExecutionResult(int(containerExitStatusCode))

	// Capture the logs based on the TTY setting.
	if h.TTYEnabled {
		// TTY combines stdout and stderr, read from stdoutPipe only.
		h.result.STDOUT, _ = bufio.NewReader(stdoutPipe).ReadString('\x00') // EOF delimiter
	} else {
		// Read from stdout and stderr separately.
		h.result.STDOUT, _ = bufio.NewReader(stdoutPipe).ReadString('\x00') // EOF delimiter
		h.result.STDERR, _ = bufio.NewReader(stderrPipe).ReadString('\x00')
	}
	log.Infow("docker_execution_handler_run_logs_success", "executionID", h.executionID)
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
	endTrace := observability.StartTrace("docker_execution_handler_kill_duration")
	defer endTrace()

	timeout := int(DestroyTimeout)
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	err := h.client.StopContainer(ctx, h.containerID, stopOptions)
	if err != nil {
		log.Errorw("docker_execution_handler_kill_failure", "error", err, "executionID", h.executionID)
		return err
	}
	log.Infow("docker_execution_handler_kill_success", "executionID", h.executionID)
	return nil
}

// destroy cleans up the container and its associated resources.
func (h *executionHandler) destroy(timeout time.Duration) error {
	endTrace := observability.StartTrace("docker_execution_handler_destroy_duration")
	defer endTrace()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// stop the container
	if err := h.kill(ctx); err != nil {
		log.Errorw("docker_execution_handler_destroy_failure", "error", err, "executionID", h.executionID)
		return fmt.Errorf("failed to kill container (%s): %w", h.containerID, err)
	}

	if err := h.client.RemoveContainer(ctx, h.containerID); err != nil {
		log.Errorw("docker_execution_handler_destroy_failure", "error", err, "executionID", h.executionID)
		return err
	}

	// Remove related objects like networks or volumes created for this execution.
	err := h.client.RemoveObjectsWithLabel(
		ctx,
		labelExecutionID,
		labelExecutionValue(h.ID, h.jobID, h.executionID),
	)
	if err != nil {
		log.Errorw("docker_execution_handler_destroy_failure", "error", err, "executionID", h.executionID)
		return err
	}

	log.Infow("docker_execution_handler_destroy_success", "executionID", h.executionID)
	return nil
}

func (h *executionHandler) outputStream(
	ctx context.Context,
	request types.LogStreamRequest,
) (io.ReadCloser, error) {
	endTrace := observability.StartTrace("docker_execution_handler_output_stream_duration")
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
		log.Errorw("docker_execution_handler_output_stream_failure", "error", err, "executionID", h.executionID)
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
