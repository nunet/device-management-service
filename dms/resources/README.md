# resources

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
6. [References](#6-references)


## Specification

### 1. Description

`resources` deals with resource management for the machine. This includes calculation of available resources for new jobs or bid requests.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/README.md): Current file which is aimed towards developers who wish to use and modify the DMS functionality.

* [init](init.go): Contains the initialization of the package.
* [resource_manager](resource_manager.go): Contains the resource manager which is responsible for managing the resources of dms.
* [system_specs_linux](system_specs_linux.go): Contains the implementation of the `SystemSpecs` interface for linux.
* [system_specs_amd64_darwin](system_specs_amd64_darwin.go): Contains the implementation of the `SystemSpecs` interface for amd64 darwin.
* [system_specs_arm64_darwin](system_specs_arm64_darwin.go): Contains the implementation of the `SystemSpecs` interface for arm64 darwin.
* [usage_monitor](usage_monitor.go): Contains the implementation of the `UsageMonitor` interface.

All files with `*_test.go` contains unit tests for the corresponding functionality.

### 3. Functionality

### ResourceManager

#### `UpdateFreeResources`

- signature: `UpdateFreeResources(context.Context) (types.FreeResources, error)`
- input: `Context`
- output: `types.FreeResources`
- output (error): Error message

#### `GetOnboardedResources`

- signature: `GetOnboardedResources(context.Context) (types.OnboardedResources, error)`
- input: `Context`
- output: `types.OnboardedResources`
- output (error): Error message

#### `GetRequiredResources`

- signature: `GetRequiredResources(context.Context) (types.Resource, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

#### `UpdateOnboardedResources`

- signature: `UpdateOnboardedResources(context.Context, types.OnboardedResources) error`
- input: `Context`, `types.OnboardedResources`
- output: None
- output (error): Error message

#### `SystemSpecs`

- signature: `SystemSpecs() types.SystemSpecs`
- input: None
- output: `types.SystemSpecs` instance
- output (error): None

#### `UsageMonitor`

- signature: `UsageMonitor() types.UsageMonitor`
- input: None
- output: `types.UsageMonitor` instance
- output (error): None

### System Specs

#### `GetSpecInfo`

- signature: `GetSpecInfo() (types.SpecInfo, error)`
- input: None
- output: `types.SpecInfo`
- output (error): Error message

#### `GetGPUVendors`

- signature: `GetGPUVendors() ([]types.GPUVendor, error)`
- input: None
- output: `[]types.GPUVendor`
- output (error): Error message

#### `GetGPUs`

- signature: `GetGPUs(vendor ...types.GPUVendor) ([]types.GPU, error)`
- input: `[]types.GPUVendor`
- output: `[]types.GPU`
- output (error): Error message

#### `GetTotalMemory`

- signature: `GetTotalMemory() (uint64, error)`
- input: None
- output: `uint64`
- output (error): Error message

#### `GetTotalStorage`

- signature: `GetTotalStorage() (uint64, error)`
- input: None
- output: `uint64`
- output (error): Error message

#### `GetCPUInfo`

- signature: `GetCPUInfo() (types.CPUInfo, error)`
- input: None
- output: `types.CPUInfo`
- output (error): Error message

#### `GetProvisionedResources`

- signature: `GetProvisionedResources() (types.Resource, error)`
- input: None
- output: `types.Resource`
- output (error): Error message

### System Specs

#### `GetUsage`

- signature: `GetUsage(context.Context) (types.Resource, error)`
- input: `Context`
- output: `types.Resource`
- output (error): Error message

### 4. Data Types

- `types.Resources`: resources defined for the machine.

```go
type Resources struct {
    CPU      float64
    NumCores uint64
    GPU      []GPU `gorm:"foreignKey:ResourceID"`
    RAM      uint64
    Disk     uint64
}
```

- `types.AvailableResources`: resources onboarded to Nunet.

```go
type AvailableResources struct {
    models.BaseDBModel
    Resources
}
```

- `types.FreeResources`: resources currently available for new jobs.

```go
type FreeResources struct {
    models.BaseDBModel
    Resources
}
```

- `types.RequiredResources`: resources required by the jobs running on the machine.

```go
type RequiredResources struct {
    models.BaseDBModel
    Resources
}
```

- `types.GPUVendor`: GPU vendors available on the machine.

```go
type GPUVendor string

const (
	GPUVendorNvidia  GPUVendor = "NVIDIA"
	GPUVendorAMDATI  GPUVendor = "AMD/ATI"
	GPUVendorIntel   GPUVendor = "Intel"
	GPUVendorUnknown GPUVendor = "Unknown"
	None             GPUVendor = "None"
)
```

- `types.GPU`: GPU details.

```go
type GPU struct {
	// Index is the self-reported index of the device in the system
	Index int
	// Name is the model name of the GPU e.g. Tesla T4
	Name string
	// Vendor is the maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor
	// PCIAddress is the PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
	// Model of the GPU, e.g. A100
	Model string `json:"model" description:"GPU model, ex A100"`
	// TotalVRAM is the total amount of VRAM on the device
	TotalVRAM uint64
	// UsedVRAM is the amount of VRAM currently in use
	UsedVRAM uint64
	// FreeVRAM is the amount of VRAM currently free
	FreeVRAM uint64

	// Gorm fields
	ResourceID uint `gorm:"foreignKey:ID"`
}
```

- `types.GPUList`: A slice of `GPU`.

```go
type GPUList []GPU
```

- `types.CPUInfo`: CPU information of the machine.

```go
type CPUInfo struct {
    NumCores   uint64
    MHzPerCore float64
    Compute    float64
}
```

- `types.SpecInfo`: detailed specifications of the machine.

```go
type SpecInfo struct {
	CPUs    []CPU
	GPUs    []GPU
	RAMs    []RAM
	Disks   []Disk
	Network NetworkInfo
}
```

- `types.CPU`: CPU details.

```go
type CPU struct {
	// Model represents the CPU model, e.g., "Intel Core i7-9700K", "AMD Ryzen 9 5900X"
	Model string

	// Vendor represents the CPU manufacturer, e.g., "Intel", "AMD"
	Vendor string

	// ClockSpeedHz represents the CPU clock speed in Hz
	ClockSpeedHz uint64

	// Cores represents the number of physical CPU cores
	Cores int

	// Threads represents the number of logical CPU threads (including hyperthreading)
	Threads int

	// Architecture represents the CPU architecture, e.g., "x86", "x86_64", "arm64"
	Architecture string

	// Cache size in bytes
	CacheSize uint64
}
```

- `types.RAM`: RAM details.

```go
type RAM struct {
	// Size in bytes
	Size uint64

	// Clock speed in Hz
	ClockSpeedHz uint64

	// Type represents the RAM type, e.g., "DDR4", "DDR5", "LPDDR4"
	Type string
}
```

- `types.Disk`: Disk details.

```go
type Disk struct {
	// Model represents the disk model, e.g., "Samsung 970 EVO Plus", "Western Digital Blue SN550"
	Model string

	// Vendor represents the disk manufacturer, e.g., "Samsung", "Western Digital"
	Vendor string

	// Size in bytes
	Size uint64

	// Type represents the disk type, e.g., "SSD", "HDD", "NVMe"
	Type string

	// Interface represents the disk interface, e.g., "SATA", "PCIe", "M.2"
	Interface string

	// Read speed in bytes per second
	ReadSpeed uint64
	// Write speed in bytes per second
	WriteSpeed uint64
}
```

- `types.NetworkInfo`: Network details.

```go
	// Bandwidth in bits per second (b/s)
	Bandwidth uint64

	// NetworkType represents the network type, e.g., "Ethernet", "Wi-Fi", "Cellular"
	NetworkType string
}
```

- `types.ExecutionResource`: resources resources required to execute a task

```go
type ExecutionResources struct {
	// CPU configuration
	CPU CPU `json:"cpu,omitempty" description:"CPU configuration"`
	// Memory configuration
	Memory RAM `json:"memory,omitempty" description:"Memory configuration"`
	// Disk configuration
	Disk Disk `json:"disk,omitempty" description:"Disk configuration"`
	// GPU configuration
	GPUs []GPU `json:"gpus,omitempty" description:"GPU configuration"`
}
```

- `negativeValueError`: It is used to return a custom error when result of resource operation is negative.

```go
type negativeValueError struct {
	resource string
	r1       any
	r2       any
}
```

### 5. Testing

Refer to `*_test.go` files for unit tests of different functionalities.

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)

### 6. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)









