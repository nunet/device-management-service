# background_tasks

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

The `background_tasks` package is an internal package responsible for managing background jobs within DMS.
It contains a scheduler that registers tasks and run them according to the schedule defined by the task definition.

`proposed` Other packages that have their own background tasks register through this package:

1. Registration 
    1. The task itself, the arguments it needs
    2. priority 
    3. event (time period or other event to trigger task)
2. Start , Stop, Resume
3. Algorithm that accounts for the event and priority of the task (not yet clear) 
4. Monitor resource usage of tasks (not yet clear)

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [init](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks/init.go): This file initializes OpenTelemetry-based Zap logger.

* [scheduler](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks/scheduler.go): This file This file defines a background task scheduler that manages task execution based on triggers, priority, and retry policies.

* [task](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks/task.go): This file contains background task structs and their properties.

* [trigger](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks/trigger.go): This file defines various trigger types (PeriodicTrigger, EventTrigger, OneTimeTrigger) for background tasks, allowing execution based on time intervals, cron expressions, or external events

Files with `*_test.go` naming convention contain unit tests of the functionality in corresponding file.

### 3. Class Diagram

#### Source

[background_tasks class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/internal/background_tasks/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/internal/background_tasks"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### NewScheduler

* signature: `NewScheduler(maxRunningTasks int) *Scheduler` <br/>

* input: `maximum no of running tasks` <br/>

* output: `internal.background_tasks.Scheduler`

`NewScheduler` function creates a new scheduler which takes `maxRunningTasks` argument to limit the maximum number of tasks to run at a time.

#### Scheduler methods

`Scheduler` struct is the orchestrator that manages and runs the tasks. If the `Scheduler` task queue is full, remaining tasks that are triggered will wait until there is a slot available in the scheduler.

It has the following methods:

##### AddTask

* signature: `AddTask(task *Task) *Task` <br/>

* input: `internal.background_tasks.Task` <br/>

* output: `internal.background_tasks.Task`

`AddTask` registers a task to be run when triggered.

##### RemoveTask

* signature: `RemoveTask(taskID int)` <br/>

* input: `identifier of the Task` <br/>

* output: None

`RemoveTask` removes a task from the scheduler. Tasks with only OneTimeTrigger will be removed automatically once run.

##### Start

* signature: `Start()` <br/>

* input: None <br/>

* output: None

`Start` starts the scheduler to monitor tasks.

##### Stop

* signature: `Stop()` <br/>

* input: None <br/>

* output: None


`Stop` stops the scheduler.

##### runTask

* signature: `runTask(taskID int)` <br/>

* input: `identifier of the Task` <br/>

* output: None

`runTask` executes a task and manages its lifecycle and retry policy.

##### runTasks

* signature: `runTasks()` <br/>

* input: None <br/>

* output: None

`runTasks` checks and runs tasks based on their triggers and priority.

##### runningTasksCount

* signature: `runningTasksCount() int` <br/>

* input: None <br/>

* output: `number of running tasks`

`runningTasksCount` returns the count of running tasks.


#### Trigger Interface

```
type Trigger interface {
	IsReady() bool // Returns true if the trigger condition is met.
	Reset()        // Resets the trigger state.
}
```

Its methods are explained below:

##### IsReady

* signature: `IsReady() bool` <br/>

* input: None <br/>

* output: `bool`

`IsReady` should return true if the task should be run.

##### Reset

* signature: `Reset()` <br/>

* input: None <br/>

* output: None

`Reset` resets the trigger until the next event happens.

There are different implementations for the `Trigger` interface.

* `PeriodicTrigger`: Defines a trigger based on a duration interval or a cron expression.

* `EventTrigger`: Defines a trigger that is set by a trigger channel.

* `OneTimeTrigger`: A trigger that is only triggered once after a set delay.


### 5. Data Types

- `internal.background_tasks.Scheduler`

```
// Scheduler orchestrates the execution of tasks based on their triggers and priority.
type Scheduler struct {
	tasks           map[int]internal.background_tasks.Task // Map of tasks by their ID.
	runningTasks    map[int]bool  // Map to keep track of running tasks.
	ticker          *time.Ticker  // Ticker for periodic checks of task triggers.
	stopChan        chan struct{} // Channel to signal stopping the scheduler.
	maxRunningTasks int           // Maximum number of tasks that can run concurrently.
	lastTaskID      int           // Counter for assigning unique IDs to tasks.
	mu              sync.Mutex    // Mutex to protect access to task maps.
}
```

- `internal.background_tasks.RetryPolicy`

```
// RetryPolicy defines the policy for retrying tasks on failure.
type RetryPolicy struct {
	MaxRetries int           // Maximum number of retries.
	Delay      time.Duration // Delay between retries.
}
```

- `internal.background_tasks.Execution`

```
// Execution records the execution details of a task.
type Execution struct {
	StartedAt time.Time   // Start time of the execution.
	EndedAt   time.Time   // End time of the execution.
	Status    string      // Status of the execution (e.g., "SUCCESS", "FAILED").
	Error     string      // Error message if the execution failed.
	Event     interface{} // Event associated with the execution.
	Results   interface{} // Results of the execution.
}
```

- `internal.background_tasks.Task`

Task is a struct that defines a job. It includes the task's ID, Name, the function that is going to be run, the arguments for the function, the triggers that trigger the task to run, retry policy, etc.

```
// Task represents a schedulable task.
type Task struct {
	ID            int                          // Unique identifier for the task.
	Name          string                       // Name of the task.
	Description   string                       // Description of the task.
	Triggers      []internal.background_tasks.Trigger                    // List of triggers for the task.
	Function      func(args interface{}) error // Function to execute as the task.
	Args          []interface{}                // Arguments for the task function.
	RetryPolicy   internal.background_tasks.RetryPolicy                  // Retry policy for the task.
	Enabled       bool                         // Flag indicating if the task is enabled.
	Priority      int                          // Priority of the task for scheduling.
	ExecutionHist []internal.background_tasks.Execution                  // History of task executions.
}
```

- `internal.background_tasks.PeriodicTrigger`

```
// PeriodicTrigger triggers at regular intervals or based on a cron expression.
type PeriodicTrigger struct {
	Interval      time.Duration // Interval for periodic triggering.
	CronExpr      string        // Cron expression for triggering.
	lastTriggered time.Time     // Last time the trigger was activated.
}
```

- `internal.background_tasks.EventTrigger`

```
// EventTrigger triggers based on an external event signaled through a channel
type EventTrigger struct {
	Trigger chan bool // Channel to signal an event.
}
```

- `internal.background_tasks.OneTimeTrigger`

```
// OneTimeTrigger triggers once after a specified delay.
type OneTimeTrigger struct {
	Delay        time.Duration // The delay after which to trigger.
	registeredAt time.Time     // Time when the trigger was set.
}
```

### 6. Testing

Unit tests for each functionality are defined in files with `*_test.go` naming convention.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `internal` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [internal package implementation]() `TBD`

### 8. References






