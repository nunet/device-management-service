package models

type Executor struct {
	ExecutorType ExecutorType `json:"executor_type"`
}

type ExecutorType string

const (
	ExecutorTypeDocker      = "docker"
	ExecutorTypeFirecracker = "firecracker"
	ExecutorTypeWasm        = "wasm"

	ExecutionStatusCodeSuccess = 0
)

// ExecutionRequest is the request object for executing a job
type ExecutionRequest struct {
	JobID       string                   // ID of the job to execute
	ExecutionID string                   // ID of the execution
	EngineSpec  *SpecConfig              // Engine spec for the execution
	Resources   *ExecutionResources      // Resources for the execution
	Inputs      []*StorageVolumeExecutor // Input volumes for the execution
	Outputs     []*StorageVolumeExecutor // Output volumes for the results
	ResultsDir  string                   // Directory to store the results
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
