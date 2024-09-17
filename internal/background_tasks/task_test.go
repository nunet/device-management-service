package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskExecution(t *testing.T) {
	task := Task{
		Name:        "Test Task",
		Description: "A task for testing",
		Function: func(_ interface{}) error {
			// Simple test function that does nothing
			return nil
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 1,
			Delay:      1 * time.Second,
		},
	}

	err := task.Function(nil)
	assert.NoError(t, err, "Task function should execute without error")
}
