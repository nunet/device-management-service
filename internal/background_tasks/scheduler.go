package backgroundtasks

import (
	"sort"
	"sync"
	"time"
)

// Scheduler orchestrates the execution of tasks based on their triggers and priority.
type Scheduler struct {
	tasks           map[int]*Task // Map of tasks by their ID.
	runningTasks    map[int]bool  // Map to keep track of running tasks.
	ticker          *time.Ticker  // Ticker for periodic checks of task triggers.
	stopChan        chan struct{} // Channel to signal stopping the scheduler.
	maxRunningTasks int           // Maximum number of tasks that can run concurrently.
	lastTaskID      int           // Counter for assigning unique IDs to tasks.
	mu              sync.Mutex    // Mutex to protect access to task maps.
}

// NewScheduler creates a new Scheduler with a specified limit on running tasks.
func NewScheduler(maxRunningTasks int) *Scheduler {
	return &Scheduler{
		tasks:           make(map[int]*Task),
		runningTasks:    make(map[int]bool),
		ticker:          time.NewTicker(1 * time.Second),
		stopChan:        make(chan struct{}),
		maxRunningTasks: maxRunningTasks,
		lastTaskID:      0,
	}
}

// AddTask adds a new task to the scheduler and initializes its state.
func (s *Scheduler) AddTask(task *Task) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = s.lastTaskID
	task.Enabled = true

	for _, trigger := range task.Triggers {
		trigger.Reset()
	}

	s.tasks[task.ID] = task
	s.lastTaskID++

	return task
}

// RemoveTask removes a task from the scheduler.
func (s *Scheduler) RemoveTask(taskID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, taskID)
}

// Start begins the scheduler's task execution loop.
func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			case <-s.ticker.C:
				s.runTasks()
			}
		}
	}()
}

// runningTasksCount returns the count of running tasks.
func (s *Scheduler) runningTasksCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, isRunning := range s.runningTasks {
		if isRunning {
			count++
		}
	}
	return count
}

// runTasks checks and runs tasks based on their triggers and priority.
func (s *Scheduler) runTasks() {
	// Sort tasks by priority.
	sortedTasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		sortedTasks = append(sortedTasks, task)
	}
	sort.Slice(sortedTasks, func(i, j int) bool {
		return sortedTasks[i].Priority > sortedTasks[j].Priority
	})

	for _, task := range sortedTasks {
		if !task.Enabled || s.runningTasks[task.ID] {
			continue
		}

		if len(task.Triggers) == 0 {
			s.RemoveTask(task.ID)
			continue
		}

		for _, trigger := range task.Triggers {
			if trigger.IsReady() && s.runningTasksCount() < s.maxRunningTasks {
				s.runningTasks[task.ID] = true
				go s.runTask(task.ID)
				trigger.Reset()
				break
			}
		}
	}
}

// Stop signals the scheduler to stop running tasks.
func (s *Scheduler) Stop() {
	close(s.stopChan)
}

// runTask executes a task and manages its lifecycle and retry policy.
func (s *Scheduler) runTask(taskID int) {
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.runningTasks[taskID] = false
	}()

	task := s.tasks[taskID]
	execution := Execution{StartedAt: time.Now()}

	defer func() {
		s.mu.Lock()
		task.ExecutionHist = append(task.ExecutionHist, execution)
		s.tasks[taskID] = task
		s.mu.Unlock()
	}()

	for i := 0; i < task.RetryPolicy.MaxRetries+1; i++ {
		err := runTaskWithRetry(task.Function, task.Args, task.RetryPolicy.Delay)
		if err == nil {
			execution.Status = "SUCCESS"
			execution.EndedAt = time.Now()
			return
		}
		execution.Error = err.Error()
	}

	execution.Status = "FAILED"
	execution.EndedAt = time.Now()
}

// runTaskWithRetry attempts to execute a task with a retry policy.
func runTaskWithRetry(
	fn func(args interface{}) error,
	args []interface{},
	delay time.Duration,
) error {
	err := fn(args)
	if err != nil {
		time.Sleep(delay)
		return err
	}
	return nil
}
