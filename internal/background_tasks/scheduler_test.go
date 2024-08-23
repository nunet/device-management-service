package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSchedulerAddAndRemoveTask(t *testing.T) {
	scheduler := NewScheduler(2)

	task := &Task{
		Name:        "Test Task",
		Description: "A task for testing",
		Function: func(_ interface{}) error {
			return nil
		},
		Triggers: []Trigger{&OneTimeTrigger{Delay: 1 * time.Second}},
	}

	addedTask := scheduler.AddTask(task)
	assert.Equal(t, 0, addedTask.ID, "Task ID should be set correctly")

	scheduler.RemoveTask(0)
	assert.Equal(t, 0, len(scheduler.tasks), "Task should be removed from scheduler")
}

func TestSchedulerTaskExecution(t *testing.T) {
	scheduler := NewScheduler(1)

	triggered := make(chan bool, 1)
	task := &Task{
		Name:        "Test Task",
		Description: "A task for testing",
		Function: func(_ interface{}) error {
			triggered <- true
			return nil
		},
		Triggers: []Trigger{&OneTimeTrigger{Delay: 1 * time.Millisecond}},
	}

	scheduler.AddTask(task)
	scheduler.Start()
	defer scheduler.Stop()

	select {
	case <-triggered:
		// Test passed
	case <-time.After(2 * time.Second):
		t.Error("Task was not executed within the expected time")
	}
}
