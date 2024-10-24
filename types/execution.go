// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

// ExecutionStatus is the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusRunning ExecutionStatus = "running"
	ExecutionStatusPaused  ExecutionStatus = "paused"
	ExecutionStatusFailed  ExecutionStatus = "failed"
	ExecutionStatusSuccess ExecutionStatus = "success"
)

// ExecutionRequest is the request object for executing a job
type ExecutionRequest struct {
	JobID       string                   // ID of the job to execute
	ExecutionID string                   // ID of the execution
	EngineSpec  *SpecConfig              // Engine spec for the execution
	Resources   *Resources               // Resources for the execution
	Inputs      []*StorageVolumeExecutor // Input volumes for the execution
	Outputs     []*StorageVolumeExecutor // Output volumes for the results
	ResultsDir  string                   // Directory to store the results
}

// ExecutionListItem is the result of the current executions.
type ExecutionListItem struct {
	ExecutionID string // ID of the execution
	Running     bool
}

// ExecutionResult is the result of an execution
type ExecutionResult struct {
	STDOUT   string `json:"stdout"`    // STDOUT of the execution
	STDERR   string `json:"stderr"`    // STDERR of the execution
	ExitCode int    `json:"exit_code"` // Exit code of the execution
	ErrorMsg string `json:"error_msg"` // Error message if the execution failed
}

// NewExecutionResult creates a new ExecutionResult object
func NewExecutionResult(code int) *ExecutionResult {
	return &ExecutionResult{
		STDOUT:   "",
		STDERR:   "",
		ExitCode: code,
	}
}

// NewFailedExecutionResult creates a new ExecutionResult object for a failed execution
// It sets the error message from the provided error and sets the exit code to -1
func NewFailedExecutionResult(err error) *ExecutionResult {
	return &ExecutionResult{
		STDOUT:   "",
		STDERR:   "",
		ExitCode: -1,
		ErrorMsg: err.Error(),
	}
}

// LogStreamRequest is the request object for streaming logs from an execution
type LogStreamRequest struct {
	JobID       string // ID of the job
	ExecutionID string // ID of the execution
	Tail        bool   // Tail the logs
	Follow      bool   // Follow the logs
}
