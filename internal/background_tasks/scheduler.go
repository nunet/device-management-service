// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package backgroundtasks

import (
	"slices"
	"sort"
	"sync"
	"time"
)

// Scheduler orchestrates the execution of tasks based on their triggers and priority.
type Scheduler struct {
	tasks        []*Task        // List of tasks.
	pollInterval time.Duration  // Ticker for periodic checks of task triggers.
	stopChan     chan struct{}  // Channel to signal stopping the scheduler.
	taskSem      chan struct{}  // Semaphore to limit the number of running tasks.
	nextTaskID   int            // Counter for assigning unique IDs to tasks.
	mu           sync.Mutex     // Mutex to protect access to task maps.
	taskWg       sync.WaitGroup // Wait group to wait for all tasks to finish.
}

// NewScheduler creates a new Scheduler with a specified limit on running tasks.
func NewScheduler(maxRunningTasks int, pollInterval time.Duration) *Scheduler {
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second
	}
	return &Scheduler{
		tasks:        make([]*Task, 0),
		taskSem:      make(chan struct{}, maxRunningTasks),
		stopChan:     make(chan struct{}),
		pollInterval: pollInterval,
		nextTaskID:   0,
	}
}

// AddTask adds a new task to the scheduler and initializes its state.
func (s *Scheduler) AddTask(task *Task) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, trigger := range task.Triggers {
		trigger.Reset(time.Now().UTC())
	}
	task.Enabled = true
	task.ID = s.nextTaskID
	s.nextTaskID++
	s.tasks = append(s.tasks, task)

	return task
}

func (s *Scheduler) GetTask(taskID int) (*Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return nil, false
}

// RemoveTask removes a task from the scheduler.
func (s *Scheduler) RemoveTask(taskID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use slices.DeleteFunc to safely remove the task
	s.tasks = slices.DeleteFunc(s.tasks, func(task *Task) bool {
		return task.ID == taskID
	})
}

// Start begins the scheduler's task execution loop.
func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.pollInterval)
	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			case now := <-ticker.C:
				s.checkAndDispatchTasks(now.UTC())
			}
		}
	}()
}

func (s *Scheduler) checkAndDispatchTasks(currentTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasksToCheck := make([]*Task, len(s.tasks))
	copy(tasksToCheck, s.tasks)

	sort.SliceStable(tasksToCheck, func(i, j int) bool {
		if tasksToCheck[i].Priority != tasksToCheck[j].Priority {
			return tasksToCheck[i].Priority < tasksToCheck[j].Priority
		}
		return tasksToCheck[i].ID < tasksToCheck[j].ID
	})

	for _, task := range tasksToCheck {
		if !task.Enabled {
			continue
		}
		for _, trigger := range task.Triggers {
			if trigger.IsReady(currentTime) {
				s.dispatchTask(task, trigger)
			}
		}
	}
}

func (s *Scheduler) dispatchTask(task *Task, trigger Trigger) {
	s.taskWg.Add(1)
	select {
	case s.taskSem <- struct{}{}:
		trigger.MarkTriggered(time.Now().UTC())
		go s.executeTask(task)
	case <-s.stopChan:
		s.taskWg.Done()
	}
}

func (s *Scheduler) executeTask(task *Task) {
	defer func() {
		<-s.taskSem
		s.taskWg.Done()
	}()

	for retries := 0; retries <= task.RetryPolicy.MaxRetries; retries++ {
		execution := Execution{
			StartedAt: time.Now().UTC(),
		}

		err := task.Function(task.Args)

		execution.EndedAt = time.Now().UTC()
		if err != nil {
			execution.Error = err.Error()
			s.mu.Lock()
			task.ExecutionHist = append(task.ExecutionHist, execution)
			s.mu.Unlock()
			if retries < task.RetryPolicy.MaxRetries {
				select {
				case <-s.stopChan:
					return
				case <-time.After(task.RetryPolicy.Delay):
				}
			}
		} else {
			s.mu.Lock()
			task.ExecutionHist = append(task.ExecutionHist, execution)
			s.mu.Unlock()
			return
		}
	}
}

// Stop signals the scheduler to stop running tasks.
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.taskWg.Wait()
}
