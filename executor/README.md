# executor

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
3. [Functionality](#3-functionality)
4. [Data Types](#4-data-types)
5. [Testing](#5-testing)
6. [Proposed Functionality/Requirements](#6-proposed-functionality--requirements)
7. [References](#7-references)

## Specification

### 1. Description
The executor package is responsible for executing the jobs received by the device management service (DMS). It provides an unified interface to run various executors such as docker, firecracker etc

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](README.md): Current file which is aimed towards developers who wish to use and modify the executor functionality. 

* [init](init.go): This file initializes a logger instance for the executor package.

* [types](types.go): This file contains the interfaces that other packages in the DMS call to utilise functionality offered by the executor package.

* [docker](docker): This folder contains the implementation of docker executor.

### 3. Functionality

The main functionality offered by the `executor` package is defined via the `Executor` interface. 

```
type Executor interface {
	// IsInstalled checks if the executor is installed and available for use.
	IsInstalled(ctx context.Context) bool

	// Start initiates an execution for the given ExecutionRequest.
	// It returns an error if the execution already exists and is in a started or terminal state.
	// Implementations may also return other errors based on resource limitations or internal faults.
	Start(ctx context.Context, request *models.ExecutionRequest) error

	// Run initiates and waits for the completion of an execution for the given ExecutionRequest.
	// It returns a ExecutionResult and an error if any part of the operation fails.
	// Specifically, it will return an error if the execution already exists and is in a started or terminal state.
	Run(ctx context.Context, request *models.ExecutionRequest) (*models.ExecutionResult, error)

	// Wait monitors the completion of an execution identified by its executionID.
	// It returns two channels:
	// 1. A channel that emits the execution result once the task is complete.
	// 2. An error channel that relays any issues encountered, such as when the
	//    execution is non-existent or has already concluded.
	Wait(ctx context.Context, executionID string) (<-chan *models.ExecutionResult, <-chan error)

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

#### IsInstalled

* signature: `IsInstalled(ctx context.Context) -> bool` <br/>
* input: `Go context` <br/>
* output: `bool` 

`IsInstalled` checks if the executor is installed and available for use. It takes the Go `context` object as input and returns a boolean indicating if the executor is installed or not.

#### Start

* signature: `Start(ctx context.Context, request dms.executor.ExecutionRequest) -> error` <br/>
* input #1: `Go context` <br/>
* input #2: `dms.executor.ExecutionRequest` <br/>
* output: `error` 

`Start` function takes a Go `context` object and a `dms.executor.ExecutionRequest` type as input. It returns an error if the execution already exists and is in a started or terminal state. Implementations may also return other errors based on resource limitations or internal faults.

#### Run

* signature: `Run(ctx context.Context, request dms.executor.ExecutionRequest) -> (dms.executor.ExecutionResult, error)` <br/>
* input #1: `Go context` <br/>
* input #2: `dms.executor.ExecutionRequest` <br/>
* output (success): `dms.executor.ExecutionResult` <br/>
* output (error): `error`

`Run` initiates and waits for the completion of an execution for the given Execution Request. It returns a `dms.executor.ExecutionResult` and an error if any part of the operation fails. Specifically, it will return an error if the execution already exists and is in a started or terminal state.

#### Wait

* signature: `Wait(ctx context.Context, executionID string) -> (<-chan dms.executor.ExecutionResult, <-chan error)` <br/>
* input #1: `Go context` <br/>
* input #2: `dms.executor.ExecutionRequest.ExecutionID` <br/>
* output #1: Channel that returns `dms.executor.ExecutionResult` <br/>
* output #2: Channel that returns `error`

`Wait` monitors the completion of an execution identified by its `executionID`. It returns two channels:
1. A channel that emits the execution result once the task is complete;
2. An error channel that relays any issues encountered, such as when the execution is non-existent or has already concluded.

#### Cancel

* signature: `Cancel(ctx context.Context, executionID string) -> error` <br/>
* input #1: `Go context` <br/>
* input #2: `dms.executor.ExecutionRequest.ExecutionID` <br/>
* output: `error`

`Cancel` attempts to terminate an ongoing execution identified by its `executionID`. It returns an error if the execution does not exist or is already in a terminal state.

#### GetLogStream

* signature: `GetLogStream(ctx context.Context, request dms.executor.LogStreamRequest, executionID string) -> (io.ReadCloser, error)` <br/>
* input #1: `Go context` <br/>
* input #2: `dms.executor.LogStreamRequest` <br/>
* input #3: `dms.executor.ExecutionRequest.ExecutionID` <br/>
* output #1: `io.ReadCloser` <br/>
* output #2: `error`

`GetLogStream` provides a stream of output for an ongoing or completed execution identified by its `executionID`. There are two flags that can be used to modify the functionality:
* The `Tail` flag indicates whether to exclude historical data or not.
* The `follow` flag indicates whether the stream should continue to send data as it is produced.

It returns an `io.ReadCloser` object to read the output stream and an error if the operation fails. Specifically, it will return an error if the execution does not exist.

### 4. Data Types

- `models.ExecutionRequest`: This is the input that `executor` receives to initiate a job execution. 

- `models.ExecutionResult`: This contains the result of the job execution. 

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

- `models.SpecConfig`: This allows arbitrary configuration/parameters as needed during implementation of specific executor. 

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

### 5. Testing

Unit tests are defined in subpackages which implement the interface defined in this package.

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `executor` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [executor package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A31&first_page_size=20)


### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)