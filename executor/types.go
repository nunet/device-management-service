// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package executor

import (
	"context"
	"io"
	"time"

	"gitlab.com/nunet/device-management-service/types"
)

// Executor serves as an execution manager for running jobs on a specific backend, such as a Docker daemon.
// It provides a comprehensive set of methods to initiate, monitor, terminate, and retrieve output streams for executions.
type Executor interface {
	// GetID returns the unique identifier for the executor.
	GetID() string
	// Start initiates an execution for the given ExecutionRequest.
	// It returns an error if the execution already exists and is in a started or terminal state.
	// Implementations may also return other errors based on resource limitations or internal faults.
	Start(ctx context.Context, request *types.ExecutionRequest) error

	// Run initiates and waits for the completion of an execution for the given ExecutionRequest.
	// It returns a ExecutionResult and an error if any part of the operation fails.
	// Specifically, it will return an error if the execution already exists and is in a started or terminal state.
	Run(ctx context.Context, request *types.ExecutionRequest) (*types.ExecutionResult, error)

	// Pause attempts to pause an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Pause(ctx context.Context, executionID string) error

	// Resume attempts to resume a paused execution identified by its executionID.
	// Returns an error if the execution does not exist, is not paused, or is already in a terminal state.
	Resume(ctx context.Context, executionID string) error

	// Wait monitors the completion of an execution identified by its executionID.
	// It returns two channels:
	// 1. A channel that emits the execution result once the task is complete.
	// 2. An error channel that relays any issues encountered, such as when the
	//    execution is non-existent or has already concluded.
	Wait(ctx context.Context, executionID string) (<-chan *types.ExecutionResult, <-chan error)

	// Cancel attempts to cancel an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Cancel(ctx context.Context, executionID string) error

	// GetLogStream provides a stream of output for an ongoing or completed execution identified by its executionID.
	// The 'Tail' flag indicates whether to exclude hstorical data or not.
	// The 'follow' flag indicates whether the stream should continue to send data as it is produced.
	// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
	// Specifically, it will return an error if the execution does not exist.
	GetLogStream(ctx context.Context, request types.LogStreamRequest) (io.ReadCloser, error)

	// List returns a slice of ExecutionListItem containing information about current executions.
	// This includes the execution ID and whether it's currently running.
	List() []types.ExecutionListItem

	// GetStatus returns the status of the execution identified by its executionID.
	// It returns the status of the execution and an error if the execution is not found or status is unknown.
	GetStatus(ctx context.Context, executionID string) (types.ExecutionStatus, error)

	// WaitForStatus waits for the execution to reach a specific status.
	// It returns an error if the execution is not found or the status is unknown.
	WaitForStatus(ctx context.Context, executionID string, status types.ExecutionStatus, timeout *time.Duration) error

	// Exec executes a command in a container and returns the exit code, output, and an error if the operation fails.
	Exec(ctx context.Context, containerID string, command []string) (int, string, error)
}
