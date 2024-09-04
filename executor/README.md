# executor

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#description)
2. [Structure and Organisation](#structure-and-organisation)
3. [Class Diagram](#class-diagram)
4. [Functionality](#functionality)
5. [Data Types](#data-types)
6. [Testing](#testing)
7. [Proposed Functionality/Requirements](#proposed-functionality--requirements)
8. [References](#references)

## Specification

### Description
The executor package is responsible for executing the jobs received by the device management service (DMS). It provides an unified interface to run various executors such as docker, firecracker etc

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/README.md): Current file which is aimed towards developers who wish to use and modify the executor functionality. 

* [init](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/init.go): This file initializes a logger instance for the executor package.

* [types](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/types.go): This file contains the interfaces that other packages in the DMS call to utilise functionality offered by the executor package.

* [docker](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker): This folder contains the implementation of docker executor.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/firecracker): This folder contains the implementation of firecracker executor.

### Class Diagram

#### Source

[executor class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/executor/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/executor"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```


### Functionality

The main functionality offered by the `executor` package is defined via the `Executor` interface. 

```
type Executor interface {
	// Start initiates an execution for the given ExecutionRequest.
	// It returns an error if the execution already exists and is in a started or terminal state.
	// Implementations may also return other errors based on resource limitations or internal faults.
	Start(ctx context.Context, request *types.ExecutionRequest) error

	// Run initiates and waits for the completion of an execution for the given ExecutionRequest.
	// It returns a ExecutionResult and an error if any part of the operation fails.
	// Specifically, it will return an error if the execution already exists and is in a started or terminal state.
	Run(ctx context.Context, request *types.ExecutionRequest) (*types.ExecutionResult, error)

	// Wait monitors the completion of an execution identified by its executionID.
	// It returns two channels:
	// 1. A channel that emits the execution result once the task is complete.
	// 2. An error channel that relays any issues encountered, such as when the
	//    execution is non-existent or has already concluded.
	Wait(ctx context.Context, executionID string) (<-chan *types.ExecutionResult, <-chan error)

	// Cancel attempts to cancel an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Cancel(ctx context.Context, executionID string) error

	// GetLogStream provides a stream of output for an ongoing or completed execution identified by its executionID.
	// The 'Tail' flag indicates whether to exclude hstorical data or not.
	// The 'follow' flag indicates whether the stream should continue to send data as it is produced.
	// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
	// Specifically, it will return an error if the execution does not exist.
	GetLogStream(ctx context.Context, executionID string) (io.ReadCloser, error)
}
```

Its methods are explained below:

#### Start

* signature: `Start(ctx context.Context, request executor.ExecutionRequest) -> error` <br/>
* input #1: `Go context` <br/>
* input #2: `executor.ExecutionRequest` <br/>
* output: `error` 

`Start` function takes a Go `context` object and a `executor.ExecutionRequest` type as input. It returns an error if the execution already exists and is in a started or terminal state. Implementations may also return other errors based on resource limitations or internal faults.

#### Run

* signature: `Run(ctx context.Context, request executor.ExecutionRequest) -> (executor.ExecutionResult, error)` <br/>
* input #1: `Go context` <br/>
* input #2: `executor.ExecutionRequest` <br/>
* output (success): `executor.ExecutionResult` <br/>
* output (error): `error`

`Run` initiates and waits for the completion of an execution for the given Execution Request. It returns a `executor.ExecutionResult` and an error if any part of the operation fails. Specifically, it will return an error if the execution already exists and is in a started or terminal state.

#### Wait

* signature: `Wait(ctx context.Context, executionID string) -> (<-chan executor.ExecutionResult, <-chan error)` <br/>
* input #1: `Go context` <br/>
* input #2: `executor.ExecutionRequest.ExecutionID` <br/>
* output #1: Channel that returns `executor.ExecutionResult` <br/>
* output #2: Channel that returns `error`

`Wait` monitors the completion of an execution identified by its `executionID`. It returns two channels:
1. A channel that emits the execution result once the task is complete;
2. An error channel that relays any issues encountered, such as when the execution is non-existent or has already concluded.

#### Cancel

* signature: `Cancel(ctx context.Context, executionID string) -> error` <br/>
* input #1: `Go context` <br/>
* input #2: `executor.ExecutionRequest.ExecutionID` <br/>
* output: `error`

`Cancel` attempts to terminate an ongoing execution identified by its `executionID`. It returns an error if the execution does not exist or is already in a terminal state.

#### GetLogStream

* signature: `GetLogStream(ctx context.Context, request executor.LogStreamRequest, executionID string) -> (io.ReadCloser, error)` <br/>
* input #1: `Go context` <br/>
* input #2: `executor.LogStreamRequest` <br/>
* input #3: `executor.ExecutionRequest.ExecutionID` <br/>
* output #1: `io.ReadCloser` <br/>
* output #2: `error`

`GetLogStream` provides a stream of output for an ongoing or completed execution identified by its `executionID`. There are two flags that can be used to modify the functionality:
* The `Tail` flag indicates whether to exclude historical data or not.
* The `follow` flag indicates whether the stream should continue to send data as it is produced.

It returns an `io.ReadCloser` object to read the output stream and an error if the operation fails. Specifically, it will return an error if the execution does not exist.

### Data Types

- `types.ExecutionRequest`: This is the input that `executor` receives to initiate a job execution. 

- `types.ExecutionResult`: This contains the result of the job execution. 

- `executor.LogStreamRequest`: This contains input parameters sent to the `executor` to get job execution logs.

```
// LogStreamRequest is the request object for streaming logs from an execution
type LogStreamRequest struct {
	// ID of the job
	JobID string

	// ID of the execution
	ExecutionID string

	// Tail the logs
	Tail bool

	// Follow the logs
	Follow bool
}
```

- `types.SpecConfig`: This allows arbitrary configuration/parameters as needed during implementation of specific executor. 

- `executor.ExecutionResources`: This contains resources to be used for execution.

```
type ExecutionResources struct {
	// CPU units
	CPU float64 `json:"cpu,omitempty"`

	// Memory in bytes
	Memory uint64 `json:"memory,omitempty"`

	// Disk in bytes
	Disk uint64 `json:"disk,omitempty"`

	// GPU configurations
	GPUs []GPU `json:"gpus,omitempty"`
}

type GPU struct {
	// Self-reported index of the device in the system
	Index uint64

	// Model name of the GPU e.g. Tesla T4
	Name string

	// Maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor

	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
}
```

- `storage.StorageVolume`: This contains parameters of storage volume used during execution. 

### Testing

Unit tests are defined in subpackages which implement the interface defined in this package.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `executor` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [executor package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A31&first_page_size=20)


### References
