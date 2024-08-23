package backgroundtasks

import (
	"time"
)

// RetryPolicy defines the policy for retrying tasks on failure.
type RetryPolicy struct {
	MaxRetries int           // Maximum number of retries.
	Delay      time.Duration // Delay between retries.
}

// Execution records the execution details of a task.
type Execution struct {
	StartedAt time.Time   // Start time of the execution.
	EndedAt   time.Time   // End time of the execution.
	Status    string      // Status of the execution (e.g., "SUCCESS", "FAILED").
	Error     string      // Error message if the execution failed.
	Event     interface{} // Event associated with the execution.
	Results   interface{} // Results of the execution.
}

// Task represents a schedulable task.
type Task struct {
	ID            int                          // Unique identifier for the task.
	Name          string                       // Name of the task.
	Description   string                       // Description of the task.
	Triggers      []Trigger                    // List of triggers for the task.
	Function      func(args interface{}) error // Function to execute as the task.
	Args          []interface{}                // Arguments for the task function.
	RetryPolicy   RetryPolicy                  // Retry policy for the task.
	Enabled       bool                         // Flag indicating if the task is enabled.
	Priority      int                          // Priority of the task for scheduling.
	ExecutionHist []Execution                  // History of task executions.
}
