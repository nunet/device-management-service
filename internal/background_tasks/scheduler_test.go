// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
