# firecracker

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure coding guidelines](https://gitlab.com/nunet/documentation/-/wikis/secure-coding-guidelines)

## Table of Contents

1. [Description](#1-description)
2. [Structure and organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description
This sub-package contains functionality including drivers and api for the Firecracker executor.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/README.md): Current file which is aimed towards developers who wish to use and modify the Firecracker functionality. 

* [client](https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/client.go): This file provides a high level wrapper around the [Firecracker](github.com/firecracker-microvm/firecracker-go-sdk) library.

* [executor]https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/(executor.go): This is the main implementation of the executor interface for Firecracker. It is the entry point of the sub-package. It is intended to be used as a singleton.

* [handler](https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/handler.go): This file contains a handler implementation to manage the lifecycle of a single job.

* [init](https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/init.go): This file is responsible for initialization of the package. Currently it only initializes a logger to be used through out the sub-package.

* [types](https://gitlab.com/nunet/device-management-service/-/tree/develop/executor/firecracker/types.go): This file contains Models that are specifically related to the Firecracker executor. Mainly it contains the engine spec model that describes a Firecracker job.

Files with `*_test.go` suffix contain unit tests for the functionality in corresponding file.

### 3. Class Diagram

#### Source

[firecracker class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/executor/firecracker/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/executor/firecracker"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

Below methods have been implemented in this package:

#### NewExecutor

* signature: `NewExecutor(_ context.Context, id string) -> (dms.executor.firecracker.Executor, error)` <br/>

* input #1: `Go context` <br/>

* input #2: identifier of the executor <br/>

* output (sucess): Executor instance of type `executor.firecracker.Executor` <br/>

* output (error): error

`NewExecutor` function initializes a new Executor instance for Firecracker VMs. 

It is expected that `NewExecutor` would be called prior to calling any other executor functions. The Executor instance returned would then be used to call other functions like `Start`, `Stop` etc.

#### IsInstalled

For function signature refer to the package [readme](../README.md#isinstalled)

`IsInstalled` checks if the Firecracker is installed on the host. It returns `true` if Firecracker is installed and accessible, `false` otherwise. 

#### Start

For function signature refer to the package [readme](../README.md#start) 

`Start` function begins the execution of a request by starting a Firecracker VM. It creates the VM based on the configuration parameters provided in the execution request. It returns an error message if
* execution is already started
* execution is already finished
* there is failure is creation of a new VM

#### Wait

For function signature refer to the package [readme](../README.md#wait)

`Wait` initiates a wait for the completion of a specific execution using its `executionID`. The function returns two channels: one for the result and another for any potential error. 

If the `executionID` is not found, an error is immediately sent to the error channel.

Otherwise, an internal goroutine is spawned to handle the asynchronous waiting. The entity calling should use the two returned channels to wait for the result of the execution or an error. If there is a cancellation request (context is done) before completion, an error is relayed to the error channel. When the execution is finished, both the channels are closed.

#### Cancel

For function signature refer to the package [readme](../README.md#cancel)

`Cancel` tries to terminate an ongoing execution identified by its `executionID`. It returns an error if the execution does not exist.

#### Run

For function signature refer to the package [readme](../README.md#run)

`Run` initiates and waits for the completion of an execution in one call. This method serves as a higher-level convenience function that internally calls `Start` and `Wait` methods. It returns the result of the execution as `executor.ExecutionResult` type. 

It returns an error in case of:
* failure in starting the VM
* failure in waiting 
* context is cancelled

### Cleanup

* signature: `Cleanup(ctx context.Context) -> error` <br/>

* input: `Go context` <br/>

* output (sucess): None <br/>

* output (error): error

`Cleanup` removes all firecracker resources associated with the executor. This includes stopping and removing all running VMs and deleting their socket paths. It returns an error it it is unable to remove the containers.

### 5. Data Types

`executor.firecracker.Executor`: This is the instance of the executor created by `NewExecutor` function. It contains the firecracker client and other resources required to execute requests.

```
// Executor manages the lifecycle of Firecracker VMs for execution requests
type Executor struct {
	// ID is the identifier of the executor instance
	ID string

	// handlers maps execution IDs to their handlers
	handlers SyncMap[string, executor.firecracker.executionHandler]

	// client is Firecracker client for VM management
	client Client
}

// Client wraps the Firecracker SDK to provide high-level operations on Firecracker VMs.
type Client struct{}

// A SyncMap is a concurrency-safe sync.Map that uses strongly-typed
// method signatures to ensure the types of its stored data are known.
type SyncMap[K comparable, V any] struct {
	sync.Map
}
```

`executor.firecracker.executionHandler`: This contains necessary information to manage the execution of a firecracker VM. 

```
// executionHandler is a struct that holds the necessary information to 
// manage the execution of a firecracker VM.
type executionHandler struct {
	// provided by the executor
	ID string

	// Firecracker client for container management
	client Client 

	// meta data about the task
	jobID       string
	executionID string
	machine     *firecracker.Machine
	// Directory to store execution results
	resultsDir  string 

	// synchronization
	// Blocks until the container starts running
	activeCh chan bool    
	
	// BLocks until execution completes or fails
	waitCh   chan bool   
	
	// Indicates if the container is currently running.
	running  *atomic.Bool 

	// result of the execution
	result dms.executor.ExecutionResult
}
```

Refer to package [readme](../README.md#4-data-types) for other data types.

### 6. Testing

Unit tests for each functionality are defined in files with `*_test.go` naming convention.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `executor` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [executor package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A31&first_page_size=20)


### 8. References
