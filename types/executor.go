// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"context"
	"io"
	"reflect"
	"time"
)

// Executor serves as an interface for running jobs on a specific backend, such as a Docker daemon, firecracker, etc.
// It provides a comprehensive set of methods to initiate, monitor, terminate, and retrieve output streams for executions.
type Executor interface {
	// GetID returns the unique identifier for the executor.
	GetID() string
	// Start initiates an execution for the given ExecutionRequest.
	// It returns an error if the execution already exists and is in a started or terminal state.
	// Implementations may also return other errors based on resource limitations or internal faults.
	Start(ctx context.Context, request *ExecutionRequest) error

	// Run initiates and waits for the completion of an execution for the given ExecutionRequest.
	// It returns a ExecutionResult and an error if any part of the operation fails.
	// Specifically, it will return an error if the execution already exists and is in a started or terminal state.
	Run(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error)

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
	Wait(ctx context.Context, executionID string) (<-chan *ExecutionResult, <-chan error)

	// Cancel attempts to cancel an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Cancel(ctx context.Context, executionID string) error

	// Remove removes an execution identified by its executionID.
	// Returns an error if the execution does not exist
	Remove(executionID string, timeout time.Duration) error

	// Cleanup removes all resources associated with the executor.
	// This includes stopping and removing all running containers or VMs and deleting their resources.
	Cleanup(ctx context.Context) error

	// GetLogStream provides a stream of output for an ongoing or completed execution identified by its executionID.
	// The 'Tail' flag indicates whether to exclude hstorical data or not.
	// The 'follow' flag indicates whether the stream should continue to send data as it is produced.
	// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
	// Specifically, it will return an error if the execution does not exist.
	GetLogStream(ctx context.Context, request LogStreamRequest) (io.ReadCloser, error)

	// List returns a slice of ExecutionListItem containing information about current executions.
	// This includes the execution ID and whether it's currently running.
	List() []ExecutionListItem

	// GetStatus returns the status of the execution identified by its executionID.
	// It returns the status of the execution and an error if the execution is not found or status is unknown.
	GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error)

	// WaitForStatus waits for the execution to reach a specific status.
	// It returns an error if the execution is not found or the status is unknown.
	WaitForStatus(ctx context.Context, executionID string, status ExecutionStatus, timeout *time.Duration) error

	// Exec executes a command in a container and returns the exit code, output, and an error if the operation fails.
	Exec(ctx context.Context, containerID string, command []string) (int, string, string, error)
}

// ExecutorType is the type of the executor
type ExecutorType string

const (
	ExecutorTypeDocker      ExecutorType = "docker"
	ExecutorTypeFirecracker ExecutorType = "firecracker"
	ExecutorTypeWasm        ExecutorType = "wasm"

	ExecutionStatusCodeSuccess = 0
)

// implementing Comparable interface
var _ Comparable[ExecutorType] = (*ExecutorType)(nil)

// Compare compares two ExecutorType objects
func (e ExecutorType) Compare(other ExecutorType) (Comparison, error) {
	return LiteralComparator(string(e), string(other)), nil
}

// String returns the string representation of the ExecutorType
func (e ExecutorType) String() string {
	return string(e)
}

// Executors is a list of Executor objects
type Executors []ExecutorType

// implementing Comparable and Calculable interface
var (
	_ Comparable[Executors] = (*Executors)(nil)
	_ Calculable[Executors] = (*Executors)(nil)
)

// Add adds the Executor object to another Executor object
func (e *Executors) Add(other Executors) error {
	// append to Executors slice
	*e = append(*e, other...)
	return nil
}

// Subtract subtracts the Executor object from another Executor object
func (e *Executors) Subtract(other Executors) error {
	if len(other) == 0 {
		return nil
	}

	toRemove := make(map[ExecutorType]struct{})
	for _, ex := range other {
		toRemove[ex] = struct{}{}
	}

	result := (*e)[:0]
	for _, ex := range *e {
		if _, found := toRemove[ex]; !found {
			result = append(result, ex)
		}
	}

	*e = result[:len(result):len(result)]
	return nil
}

// Contains checks if an Executor object is in the list of Executors
func (e *Executors) Contains(executorType ExecutorType) bool {
	executors := *e
	for _, ex := range executors {
		if ex == executorType {
			return true
		}
	}
	return false
}

// Compare compares two Executors objects
func (e *Executors) Compare(other Executors) (Comparison, error) {
	if reflect.DeepEqual(*e, other) {
		return Equal, nil
	}

	// comparator for Executors types:
	// left represent machine capabilities;
	// right represent required capabilities;
	lSlice := make([]interface{}, 0, len(*e))
	rSlice := make([]interface{}, 0, len(other))

	for _, ex := range *e {
		lSlice = append(lSlice, ex)
	}
	for _, ex := range other {
		rSlice = append(rSlice, ex)
	}

	if !IsSameShallowType(lSlice, rSlice) {
		return None, nil
	}

	switch {
	case reflect.DeepEqual(lSlice, rSlice):
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		return Equal, nil

	case IsStrictlyContained(lSlice, rSlice):
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'
		return Better, nil

	case IsStrictlyContained(rSlice, lSlice):
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		return Worse, nil
	default:
		return None, nil
	}
}
