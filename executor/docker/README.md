# docker

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
This sub-package contains functionality including drivers and api for the Docker executor.

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/README.md): Current file which is aimed towards developers who wish to use and modify the docker functionality. 

* [client](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/client.go): This file provides a high level wrapper around the docker library.

* [executor](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/executor.go): This is the main implementation of the executor interface for docker. It is the entry point of the sub-package. It is intended to be used as a singleton.

* [handler](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/handler.go): This file contains a handler implementation to manage the lifecycle of a single job.

* [init](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/init.go): This file is responsible for initialization of the package. Currently it only initializes a logger to be used through out the sub-package.

* [types](https://gitlab.com/nunet/device-management-service/-/tree/main/executor/docker/types.go): This file contains Models that are specifically related to the docker executor. Mainly it contains the engine spec model that describes a docker job.

Files with `*_test.go` suffix contain unit tests for the functionality in corresponding file.

### Class Diagram

#### Source

[docker class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/executor/docker/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/executor/docker"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

Below methods have been implemented in this package:

#### NewExecutor

* signature: `NewExecutor(ctx context.Context, id string) -> (executor.docker.Executor, error)` <br/>

* input #1: `Go context` <br/>

* input #2: identifier of the executor <br/>

* output (sucess): Executor instance of type `executor.docker.Executor` <br/>

* output (error): error

`NewExecutor` function initializes a new Executor instance with a Docker client. It returns an error if Docker client initialization fails.

It is expecte that `NewExecutor` would be called prior to calling any other executor functions. The Executor instance returned would then be used to call other functions like `Start`, `Stop` etc.

#### Start

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/executor#start) 

`Start` function begins the execution of a request by starting a Docker container. It creates the container based on the configuration parameters provided in the execution request. It returns an error message if
* container is already started
* container execution is finished
* there is failure is creation of a new container


#### Wait

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/executor#wait)

`Wait` initiates a wait for the completion of a specific execution using its `executionID`. The function returns two channels: one for the result and another for any potential error. 

If the `executionID` is not found, an error is immediately sent to the error channel.

Otherwise, an internal goroutine is spawned to handle the asynchronous waiting. The entity calling should use the two returned channels to wait for the result of the execution or an error. If there is a cancellation request (context is done) before completion, an error is relayed to the error channel. When the execution is finished, both the channels are closed.

#### Cancel

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/executor#cancel)

`Cancel` tries to terminate an ongoing execution identified by its `executionID`. It returns an error if the execution does not exist.


#### GetLogStream

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/executor#getlogstream)

`GetLogStream` provides a stream of output logs for a specific execution. Parameters `tail` and `follow` specified in `executor.LogStreamRequest` provided as input control whether to include past logs and whether to keep the stream open for new logs, respectively.

It returns an error if the execution is not found.

#### Run

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/executor#run)

`Run` initiates and waits for the completion of an execution in one call. This method serves as a higher-level convenience function that internally calls `Start` and `Wait` methods. It returns the result of the execution as `executor.ExecutionResult` type. 

It returns an error in case of:
* failure in starting the container
* failure in waiting 
* context is cancelled

#### ConfigureHostConfig

* signature: configureHostConfig(vendor types.GPUVendor, params *types.ExecutionRequest, mounts []mount.Mount) container.HostConfig <br/>

* input #1: GPU vendor (types.GPUVendor) <br/>

* input #2: Execution request parameters (types.ExecutionRequest) <br/>

* input #3: List of container mounts ([]mount.Mount) <br/>

* output: Host configuration for the Docker container (container.HostConfig) <br/>

The `configureHostConfig` function sets up the host configuration for the container based on the GPU vendor and resources requested by the execution. It supports configurations for different types of GPUs and CPUs.

The function performs the following steps:

1. **NVIDIA GPUs**:
    - Configures the `DeviceRequests` to include all GPUs specified in the execution request.
    - Sets the memory and CPU resources according to the request parameters.

2. **AMD GPUs**:
    - Binds the necessary device paths (`/dev/kfd` and `/dev/dri`) to the container.
    - Adds the `video` group to the container.
    - Sets the memory and CPU resources according to the request parameters.

3. **Intel GPUs**:
    - Binds the `/dev/dri` directory to the container, exposing all Intel GPUs.
    - Sets the memory and CPU resources according to the request parameters.

4. **Default (CPU-only)**:
    - Configures the container with memory and CPU resources only, without any GPU-specific settings.

The function ensures that the appropriate resources and device paths are allocated to the container based on the available and requested GPU resources.

### Cleanup

* signature: `Cleanup(ctx context.Context) -> error` <br/>

* input: `Go context` <br/>

* output (sucess): None <br/>

* output (error): error

`Cleanup` removes all Docker resources associated with the executor. This includes removing containers including networks and volumes with the executor's label. It returns an error it if unable to remove the containers.

### Data Types

- `executor.docker.Executor`: This is the instance of the executor created by `NewExecutor` function. It contains the Docker client and other resources required to execute requests. 

```
import (
	"sync"

	"github.com/docker/docker/client"
)

// Executor manages the lifecycle of Docker containers for execution requests
type Executor struct {
	// ID is the identifier of the executor instance
	ID string

	// handlers maps execution IDs to their handlers
	handlers SyncMap[string, executionHandler]

	// client embeds Docker client for container management
	client Client
}

// Client wraps the Docker client to provide high-level operations on Docker containers and networks
type Client struct {
	// client embeds the Docker client
	client *client.Client
}

// A SyncMap is a concurrency-safe sync.Map that uses strongly-typed
// method signatures to ensure the types of its stored data are known.
type SyncMap[K comparable, V any] struct {
	sync.Map
}

```

- `executor.docker.executionHandler`: This contains necessary information to manage the execution of a docker container. 

```
// executionHandler manages the lifecycle and execution of a Docker container for a specific job.
type executionHandler struct {
	// provided by the executor
	ID string

	client Client // Docker client for container management.

	// meta data about the task
	jobID       string
	executionID string
	containerID string
	// Directory to store execution results
	resultsDir  string 

	// synchronization
	activeCh chan bool    // Blocks until the container starts running.
	waitCh   chan bool    // BLocks until execution completes or fails.
	running  *atomic.Bool // Indicates if the container is currently running.

	// result of the execution
	result executor.ExecutionResult
}
```

Refer to package readme for other data types.

### Testing

Unit tests for each functionality are defined in files with `*_test.go` naming convention.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `executor` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [executor package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A31&first_page_size=20)


### References

