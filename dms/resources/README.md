# resources

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

`resources` deals with resource management for the machine. This includes calculation of available resources for new jobs or bid requests.

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/README.md): Current file which is aimed towards developers who wish to use and modify the DMS functionality.

* [init](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/init.go): Contains the initialization of the package.

* [resource_manager](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/resource_manager.go): Contains the resource manager which is responsible for managing the resources of dms.

* [system_specs_linux](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/system_specs_linux.go): Contains the implementation of the `SystemSpecs` interface for linux.

* [system_specs_amd64_darwin](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/system_specs_amd64_darwin.go): Contains the implementation of the `SystemSpecs` interface for amd64 darwin.

* [system_specs_arm64_darwin](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/system_specs_arm64_darwin.go): Contains the implementation of the `SystemSpecs` interface for arm64 darwin.

* [usage_monitor](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/usage_monitor.go): Contains the implementation of the `UsageMonitor` interface.

* [store](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/store.go): Contains the implementation of the `store` for the resource manager.

All files with `*_test.go` contains unit tests for the corresponding functionality.

### Class Diagram

The class diagram for the `resources` package is shown below.

#### Source file

[resources Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/dms/resources"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

#### Manager Interface

The interface methods are explained below.

##### `AllocateResources`

- signature: `AllocateResources(context.Context, ResourceAllocation) error`
- input: `Context`
- output (error): Error message

`AllocateResources` allocates the resources to the job.

##### `DeallocateResources`

- signature: `DeallocateResources(context.Context, string) error`
- input: `Context`
- output (error): Error message

`DeallocateResources` deallocates the resources from the job.

##### `GetTotalAllocation`

- signature: `GetTotalAllocation() (Resources, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

`GetTotalAllocation` returns the total resources allocated to the jobs.

##### `GetFreeResources`

- signature: `GetFreeResources() (FreeResources, error)`
- input: None
- output: `FreeResources`
- output (error): Error message

`GetFreeResources` returns the available resources in the allocation pool.

##### `GetOnboardedResources`

- signature: `GetOnboardedResources(context.Context) (OnboardedResources, error)`
- input: `Context`
- output: `OnboardedResources`
- output (error): Error message

`GetOnboardedResources` returns the resources onboarded to dms.

##### `UpdateOnboardedResources`

- signature: `UpdateOnboardedResources(context.Context, OnboardedResources) error`
- input: `Context`
- input: `OnboardedResources`
- output (error): Error message

`UpdateOnboardedResources` updates the resources onboarded to dms.

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

##### `GetMachineResources`

- signature: `GetMachineResources(context.Context) (MachineResources, error)`
- input: `Context`
- output: `MachineResources`
- output (error): Error message

`GetMachineResources` returns the resources available on the machine.

### UsageMonitor Interface

This interface defines methods to monitor the system usage. The methods are explained below.

#### `GetUsage`

- signature: `GetUsage(context.Context) (types.Resource, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

`GetUsage` returns the resources currently used by the machine.

### Data Types

- `types.Resources`: resources defined for the machine.

- `types.AvailableResources`: resources onboarded to Nunet.

- `types.FreeResources`: resources currently available for new jobs.

- `types.ResourceAllocation`: resources allocated to a job.

- `types.MachineResources`: resources available on the machine.

- `types.GPUVendor`: GPU vendors available on the machine.

- `types.GPU`: GPU details.

- `types.GPUs`: A slice of `GPU`.

- `types.CPU`: CPU details.

- `types.RAM`: RAM details.

- `types.Disk`: Disk details.

- `types.NetworkInfo`: Network details.
```

### Testing

Refer to `*_test.go` files for unit tests of different functionalities.

### Proposed Functionality / Requirements

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)

### References









