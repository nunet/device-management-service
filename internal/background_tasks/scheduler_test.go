// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package backgroundtasks

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	t.Parallel()

	s := NewScheduler(2, 100*time.Millisecond)
	assert.NotNil(t, s, "Expected non-nil Scheduler")
	assert.Equal(t, 2, cap(s.taskSem), "Expected taskSem cap 2")
}

func TestAddAndGetTask(t *testing.T) {
	t.Parallel()

	s := NewScheduler(1, 10*time.Millisecond)
	task := &Task{
		Name:     "test-task",
		Function: func(_ interface{}) error { return nil },
		Triggers: []Trigger{&OneTimeTrigger{After: 10 * time.Millisecond}},
	}
	s.AddTask(task)
	got, found := s.GetTask(task.ID)
	assert.True(t, found, "Task not found after AddTask")
	assert.Equal(t, "test-task", got.Name, "Expected task name")
}

func TestRemoveTask(t *testing.T) {
	t.Parallel()

	s := NewScheduler(1, 100*time.Millisecond)
	task := &Task{
		Name:     "to-remove",
		Function: func(_ interface{}) error { return nil },
		Triggers: []Trigger{&OneTimeTrigger{After: 10 * time.Millisecond}},
	}
	s.AddTask(task)
	s.RemoveTask(task.ID)
	_, found := s.GetTask(task.ID)
	assert.False(t, found, "Task should not be found after RemoveTask")
}

func TestSchedulerExecutesTask(t *testing.T) {
	t.Parallel()

	s := NewScheduler(1, 10*time.Millisecond)
	var called sync.WaitGroup
	called.Add(1)
	trigger := &OneTimeTrigger{After: 5 * time.Millisecond}
	task := &Task{
		Name: "exec-task",
		Function: func(_ interface{}) error {
			called.Done()
			return nil
		},
		Triggers: []Trigger{trigger},
	}
	s.AddTask(task)
	s.Start()
	waitCh := make(chan struct{})
	go func() {
		called.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// Success
	case <-time.After(200 * time.Millisecond):
		s.Stop()
		assert.FailNow(t, "Task function was not called by scheduler in time")
	}
	s.Stop()
}

func TestSchedulerPriority(t *testing.T) {
	t.Parallel()

	s := NewScheduler(2, 2*time.Millisecond)
	var executed []string
	var mu sync.Mutex

	highMedDelay := 10 * time.Millisecond
	lowDelay := 40 * time.Millisecond
	taskLow := &Task{
		Name:     "low-priority",
		Priority: 30,
		Triggers: []Trigger{&OneTimeTrigger{After: lowDelay}},
		Function: func(_ interface{}) error {
			mu.Lock()
			executed = append(executed, "low")
			mu.Unlock()
			return nil
		},
	}
	taskMed := &Task{
		Name:     "medium-priority",
		Priority: 20,
		Triggers: []Trigger{&OneTimeTrigger{After: highMedDelay}},
		Function: func(_ interface{}) error {
			mu.Lock()
			executed = append(executed, "med")
			mu.Unlock()
			return nil
		},
	}
	taskHigh := &Task{
		Name:     "high-priority",
		Priority: 10,
		Triggers: []Trigger{&OneTimeTrigger{After: highMedDelay}},
		Function: func(_ interface{}) error {
			mu.Lock()
			executed = append(executed, "high")
			mu.Unlock()
			return nil
		},
	}

	s.AddTask(taskLow)
	s.AddTask(taskMed)
	s.AddTask(taskHigh)
	s.Start()

	deadline := time.Now().Add(3000 * time.Millisecond)
	seen := map[string]bool{"high": false, "med": false, "low": false}
	var lastUnique string
	for {
		mu.Lock()
		for _, name := range executed {
			if !seen[name] {
				seen[name] = true
				lastUnique = name
			}
		}
		allSeen := seen["high"] && seen["med"] && seen["low"]
		mu.Unlock()
		if allSeen {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	s.Stop()

	require.True(t, seen["high"] && seen["med"] && seen["low"], "All three tasks should have executed at least once, got: %v", executed)
	assert.Equal(t, "low", lastUnique, "Low priority task should execute last among unique tasks")
}

func TestSchedulerMaxConcurrentTasks(t *testing.T) {
	maxConcurrent := 2
	s := NewScheduler(maxConcurrent, 5*time.Millisecond)
	var wg sync.WaitGroup
	totalTasks := 5
	startCh := make(chan struct{})
	concurrent := 0
	maxObserved := 0
	var mu sync.Mutex
	for i := 0; i < totalTasks; i++ {
		task := &Task{
			Name:     "concurrent-task",
			Triggers: []Trigger{&OneTimeTrigger{After: 1 * time.Millisecond}},
			Function: func(_ interface{}) error {
				mu.Lock()
				concurrent++
				if concurrent > maxObserved {
					maxObserved = concurrent
				}
				mu.Unlock()
				wg.Add(1)
				<-startCh // Block until released
				mu.Lock()
				concurrent--
				mu.Unlock()
				wg.Done()
				return nil
			},
		}
		s.AddTask(task)
	}
	s.Start()
	// Allow scheduler to start tasks
	time.Sleep(50 * time.Millisecond)
	close(startCh)
	wg.Wait()
	s.Stop()
	assert.LessOrEqual(t, maxObserved, maxConcurrent, "Observed too many concurrent tasks")
}

func TestSchedulerRetryLogic(t *testing.T) {
	t.Parallel()

	s := NewScheduler(1, 5*time.Millisecond)
	var callCount int
	failTimes := 2
	trigger := &OneTimeTrigger{After: 5 * time.Millisecond}
	task := &Task{
		Name: "retry-task",
		Function: func(_ interface{}) error {
			callCount++
			if callCount <= failTimes {
				return fmt.Errorf("fail %d", callCount)
			}
			return nil
		},
		Triggers: []Trigger{trigger},
		RetryPolicy: RetryPolicy{
			MaxRetries: failTimes,
			Delay:      5 * time.Millisecond,
		},
	}
	s.AddTask(task)
	s.Start()

	// Wait for expected number of executions or timeout
	done := make(chan struct{})
	go func() {
		for {
			s.mu.Lock()
			count := len(task.ExecutionHist)
			s.mu.Unlock()
			if count == failTimes+1 {
				close(done)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for task retries to complete")
	}
	s.Stop()

	assert.Equal(t, failTimes+1, callCount, "Expected number of attempts")
	assert.Len(t, task.ExecutionHist, failTimes+1, "Expected number of execution records")
	for i := 0; i < failTimes && i < len(task.ExecutionHist); i++ {
		assert.NotEmpty(t, task.ExecutionHist[i].Error, "Expected error in execution %d", i)
	}
	if len(task.ExecutionHist) == failTimes+1 {
		assert.Empty(t, task.ExecutionHist[failTimes].Error, "Expected last execution to succeed")
	}
}

func TestSchedulerStop(t *testing.T) {
	t.Parallel()

	s := NewScheduler(1, 10*time.Millisecond)
	var called bool
	task := &Task{
		Name: "stop-task",
		Function: func(_ interface{}) error {
			called = true
			return nil
		},
		Triggers: []Trigger{&OneTimeTrigger{After: 100 * time.Millisecond}},
	}
	s.AddTask(task)
	s.Start()
	time.Sleep(20 * time.Millisecond)
	s.Stop()
	time.Sleep(100 * time.Millisecond)
	if called {
		t.Error("Task should not have run after scheduler stopped")
	}
}
