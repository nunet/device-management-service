// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
	StartedAt time.Time // Start time of the execution.
	EndedAt   time.Time // End time of the execution.
	Error     string    // Error message if the execution failed.
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
