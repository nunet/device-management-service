# resources

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure coding guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

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

`resources` deals with resource management for the machine. This includes calculation of available resources for new jobs or bid requests.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/README.md): Current file which is aimed towards developers who wish to use and modify the DMS functionality.

* [init](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/init.go): Contains the initialization of the package.

* [resource_manager](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/resource_manager.go): Contains the resource manager which is responsible for managing the resources of dms.

* [system_specs_linux](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/system_specs_linux.go): Contains the implementation of the `SystemSpecs` interface for linux.

* [system_specs_amd64_darwin](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/system_specs_amd64_darwin.go): Contains the implementation of the `SystemSpecs` interface for amd64 darwin.

* [system_specs_arm64_darwin](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/system_specs_arm64_darwin.go): Contains the implementation of the `SystemSpecs` interface for arm64 darwin.

* [usage_monitor](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/usage_monitor.go): Contains the implementation of the `UsageMonitor` interface.

All files with `*_test.go` contains unit tests for the corresponding functionality.

### 3. Class Diagram

The class diagram for the `resources` package is shown below.

#### Source file

[resources Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/dms/resources"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### Manager Interface

The interface methods are explained below.

##### `UpdateFreeResources`

- signature: `UpdateFreeResources(context.Context) (types.FreeResources, error)`
- input: `Context`
- output: `types.FreeResources`
- output (error): Error message

`UpdateFreeResources` calculates and returns the free resources of the machine. It also updates this value in the local database.  

##### `GetOnboardedResources`

- signature: `GetOnboardedResources(context.Context) (types.OnboardedResources, error)`
- input: `Context`
- output: `types.OnboardedResources`
- output (error): Error message

`GetOnboardedResources` fetches the onboarded resources of the machine from the database.

##### `GetRequiredResources`

- signature: `GetRequiredResources(context.Context) (types.Resource, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

`GetRequiredResources` calculates and returns the resources required by all the jobs that are scheduled to be run on the machine.

##### `UpdateOnboardedResources`

- signature: `UpdateOnboardedResources(context.Context, types.OnboardedResources) error`
- input: `Context`, `types.OnboardedResources`
- output: None
- output (error): Error message

`UpdateOnboardedResources` updates the onboarded resources of the machine in the database.

##### `SystemSpecs`

- signature: `SystemSpecs() types.SystemSpecs`
- input: None
- output: `types.SystemSpecs` instance
- output (error): None

`SystemSpecs` returns the `types.SystemSpecs` instance.

##### `UsageMonitor`

- signature: `UsageMonitor() types.UsageMonitor`
- input: None
- output: `types.UsageMonitor` instance
- output (error): None

`UsageMonitor` returns the `types.UsageMonitor` instance.

#### SystemSpecs Interface

This interface defines the methods to get the system specifications of the machine. These methods are explained below.

##### `GetSpecInfo`

- signature: `GetSpecInfo() (types.SpecInfo, error)`
- input: None
- output: `types.SpecInfo`
- output (error): Error message

`GetSpecInfo` returns the detailed specifications of the machine

##### `GetGPUVendors`

- signature: `GetGPUVendors() ([]types.GPUVendor, error)`
- input: None
- output: `[]types.GPUVendor`
- output (error): Error message

`GetGPUVendors` returns the vendors for GPU installed on the machine

##### `GetGPUs`

- signature: `GetGPUs(vendor ...types.GPUVendor) ([]types.GPU, error)`
- input: `[]types.GPUVendor`
- output: `[]types.GPU`
- output (error): Error message

`GetGPUs` returns the GPU data of the machine for the specified vendor(s). If no vendor is provided as input, it returns the information of all the GPUs.

##### `GetTotalMemory`

- signature: `GetTotalMemory() (uint64, error)`
- input: None
- output: `uint64`
- output (error): Error message

`GetTotalMemory` returns the total memory of the machine in MB.

##### `GetTotalStorage`

- signature: `GetTotalStorage() (uint64, error)`
- input: None
- output: `uint64`
- output (error): Error message

`GetTotalStorage` returns the total storage of the machine in MB.

##### `GetCPUInfo`

- signature: `GetCPUInfo() (types.CPUInfo, error)`
- input: None
- output: `types.CPUInfo`
- output (error): Error message

`GetCPUInfo` returns the CPU information of the machine.

##### `GetProvisionedResources`

- signature: `GetProvisionedResources() (types.Resource, error)`
- input: None
- output: `types.Resource`
- output (error): Error message

`GetProvisionedResources` returns the total resources of the machine.

### UsageMonitor Interface

This interface defines methods to monitor the system usage. The methods are explained below.

#### `GetUsage`

- signature: `GetUsage(context.Context) (types.Resource, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

`GetUsage` returns the resources currently used by the machine.

### 5. Data Types

- `types.Resources`: resources defined for the machine.

- `types.AvailableResources`: resources onboarded to Nunet.

- `types.FreeResources`: resources currently available for new jobs.

- `types.RequiredResources`: resources required by the jobs running on the machine.

- `types.GPUVendor`: GPU vendors available on the machine.

- `types.GPU`: GPU details.

- `types.GPUList`: A slice of `GPU`.

- `types.CPUInfo`: CPU information of the machine.

- `types.SpecInfo`: detailed specifications of the machine.

- `types.CPU`: CPU details.

- `types.RAM`: RAM details.

- `types.Disk`: Disk details.

- `types.NetworkInfo`: Network details.

- `types.ExecutionResource`: resources resources required to execute a task

- `dms.resources.negativeValueError`: It is used to return a custom error when result of resource operation is negative.

```go
type negativeValueError struct {
	resource string
	r1       any
	r2       any
}
```

### 6. Testing

Refer to `*_test.go` files for unit tests of different functionalities.

### 7. Proposed Functionality / Requirements

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)

### 8. References









