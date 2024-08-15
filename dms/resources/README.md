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
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. `proposed` Description

`resources` deals with resource management for the machine. This includes calculation of available resources for new jobs or bid requests.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/README.md): Current file which is aimed towards developers who wish to use and modify the DMS functionality.

* [calc_resources](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/calc_resources.go): This contains methods to calculate and update free resources.

* [darwin_amd64_resources](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/darwin_amd64_resources.go): This contains methods to calculate machine resources.

* [darwin_arm64_gpu](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/darwin_arm64_gpu.go): This contains placeholder method to detect GPU on machine. 

* [darwin_arm64_resources](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/darwin_arm64_resources.go): This contains methods to calculate machine resources.

* [handler](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/handler.go): This contains methods related to resources management on a machine.

* [init](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/init.go): This initializes a logger instance.

* [linux_amd64_gpuinfo](linux_amd64_gpuinfo.go): This contains methods to collect GPU info.

* [linux_amd64_resources](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/linux_amd64_resources.go): This contains methods to calculate machine resources.

* [res_operations](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/res_operations.go): This contains methods to perform various operations (addition, subtraction etc) on machine resources.

All files with `*_test.go` contains unit tests for the corresponding functionality.

### 3. Class Diagram

#### Source

[resources class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/resources/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/dms/resources"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### `GetFreeResource`

- signature: `GetFreeResource(ctx context.Context) (*models.FreeResources, error)`

- input: `Context object`

- output: `models.FreeResources`

- output (error): Error message

`GetFreeResource` calculates the FreeResources based on the AvailableResources and all processes started by DMS, and updates the FreeResources database table accordingly.


#### `updateDBFreeResources`

- signature: `updateDBFreeResources(freeRes models.FreeResources) error`

- input: `models.FreeResources`

- output: None

- output (error): Error message

`updateDBFreeResources` updates the database using the FreeResources provided.

#### `getServiceResourcesRequirements`

- signature: `getServiceResourcesRequirements(gormDB *gorm.DB) (map[int]models.ServiceResourceRequirements, error) `

- input: `gorm DB object` 

- output: map of `models.ServiceResourceRequirements`

- output (error): Error message

`getServiceResourcesRequirements` returns a map `models.ServiceResourceRequirements` from the database provided.

#### `GetFreeResources`

- signature: `func GetFreeResources() (models.FreeResources, error)`

- input: None

- output: `models.FreeResources`

- output (error): Error message

`GetFreeResources` retrieves the currently available free resources of a machine.


#### `GetAvailableResources`

- signature: `GetAvailableResources(gormDB *gorm.DB) (models.AvailableResources, error)`

- input: `gorm DB object`

- output: `models.AvailableResources`

- output (error): Error message

`GetAvailableResources` retrieves and returns the available resources onboarded to Nunet.

#### `GetGPUInfo`

- signature: `GetGPUInfo() ([]models.GPU, error)`

- input: None

- output: `[]models.GPU`

- output (error): Error message

`GetGPUInfo` detects the presence of GPUs, fetches detailed information for NVIDIA, AMD, and Intel GPUs, and returns this information in a list of `models.GPU`. It includes checks for GPU vendors, handles cases where vendor-specific tools (like `rocm-smi` for AMD or `xpu-smi` for Intel) are not installed, and retrieves the PCI address and index for each detected GPU.

#### `GetGPUWithHighestFreeVRAM`

- signature: `GetGPUWithHighestFreeVRAM() (models.GPU, error)`

- input: None

- output: `models.GPU`

- output (error): Error message

`GetGPUWithHighestFreeVRAM` determines the GPU vendor with the highest free VRAM from the available GPUs and returns the corresponding `models.GPU` structure.

#### `FetchGPUPCIAddressandIndex`

- signature: `FetchGPUPCIAddressandIndex() (map[uint64]models.GPU, error)`

- input: None

- output: `map[uint64]models.GPU`

- output (error): Error message

`FetchGPUPCIAddressandIndex` fetches the PCI address and index details for detected GPUs and returns a map of these details.

#### `GetNVIDIAGPUInfo`

- signature: `GetNVIDIAGPUInfo() ([]models.GPU, error)`

- input: None

- output: `[]models.GPU`

- output (error): Error message

`GetNVIDIAGPUInfo` fetches detailed information about NVIDIA GPUs using NVML and returns this information in a list of `models.GPU`.

#### `GetAMDGPUInfo`

- signature: `GetAMDGPUInfo() ([]models.GPU, error)`

- input: None

- output: `[]models.GPU`

- output (error): Error message

`GetAMDGPUInfo` fetches detailed information about AMD GPUs using `rocm-smi` and returns this information in a list of `models.GPU`.  CLI based information retrieval will be stopped in future and made programmatic like NVML.

#### `GetIntelGPUInfo`

- signature: `GetIntelGPUInfo() ([]models.GPU, error)`

- input: None

- output: `[]models.GPU`

- output (error): Error message

`GetIntelGPUInfo` fetches detailed information about Intel GPUs using `xpu-smi` and returns this information in a list of `models.GPU`. CLI based information retrieval will be stopped in future and made programmatic like NVML.

#### `DetectGPUVendors`

- signature: `DetectGPUVendors() ([]models.GPUVendor, error)`

- input: None

- output: `[]models.GPUVendor`

- output (error): Error message

`DetectGPUVendors` detects the GPU vendors present in the system and returns a list of detected `models.GPUVendor`.

**Note: the functionality of DMS is being currently developed. The above methods are expected to be modified. See the [proposed](#6-proposed-functionality--requirements) section for the suggested design.**


### 4. Data Types

- `models.AvailableResources`: resources onboarded to Nunet.

- `models.FreeResources`: resources currently available for new jobs.

- `models.Provisioned`: total capacity of the machine

- `models.GPU`: contains GPU related parameters.

```
// Currently functional version (check below for future version with todos)
type GPU struct {
	// added from the proposed specifications
	// Model name of the GPU e.g. Tesla T4 A100
	Model string `json:"model" description:"GPU model, ex Tesla T4 A100"`
	// Total VRAM in MB
	VRAM uint64 `json:"vram" description:"GPU VRAM size in MB"`
	// Used VRAM in MB
	UsedVram uint64 `json:"used_vram" description:"Used GPU VRAM size in MB"`
	// Free VRAM in MB
	FreeVram uint64 `json:"free_vram" description:"Free GPU VRAM size in MB"`
	// Self-reported index of the device in the system
	Index uint64
	// Maker of the GPU, e.g. NVIDIA, AMD, Intel
	Vendor GPUVendor
	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
}
```

- `negativeValueError`: It is used to return a custom error when result of resource operation is negative.

```
type negativeValueError struct {
	fieldName string
	r1        int
	r2        int
}
``` 


**Note: the functionality of DMS is being currently developed. See the [proposed](#6-proposed-functionality--requirements) section for the suggested data types.**


### 6. Testing

`proposed` Refer to `*_test.go` files for unit tests of different functionalities.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


#### Data types

##### `proposed` Resources data models

Below are data models proposed for handling machine resoures. Relevant issue can be found [here](https://gitlab.com/nunet/device-management-service/-/issues/259#note_1896703735)

```
type Resources struct {
	CPUs    []CPU
	GPUs    []GPU
	RAMs    []RAM
	Disks   []Disk
	Network NetworkInfo
}

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

type GPU struct {
	// Model represents the GPU model, e.g., "NVIDIA GeForce RTX 3080", "AMD Radeon RX 6800 XT"
	Model string

	// Vendor represents the GPU manufacturer, e.g., "NVIDIA", "AMD"
	Vendor string

	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string

	// Memory size in bytes - currently functional as VRAM
	MemorySize uint64

	// MemoryType represents the type of GPU memory, e.g., "GDDR6", "HBM2"
	// TODO: may be removed as it may be too specific for our case
	MemoryType string

	// TODO: ClockSpeedHz in Hz
	ClockSpeedHz uint64

	// TODO: ComputeUnits represents the number of compute units (e.g., CUDA cores for NVIDIA, Stream Processors for AMD)
	ComputeUnits int

	// TODO
	CUDASupport    bool //Specific only to NVIDIA GPUs and depends on the results of the `nunet capacity --cuda-tensor` command
	HIPSupport	   bool //Specific to AMD GPUs but not restricted to other vendors. Depends on the results of the `nunet capacity --rocm-hip` command
	XPUSupport	   bool //Specific only to Intel GPUs. Depends on the results of the `nunet capacity --intel-xpu` command
	OpenCLSupport  bool //Device independent and not restricted to GPUs alone.
	DirectXSupport bool //Specific only to Windows Operating System.
}

type RAM struct {
	// Size in bytes
	Size uint64

	// Clock speed in Hz
	ClockSpeedHz uint64

	// Type represents the RAM type, e.g., "DDR4", "DDR5", "LPDDR4"
	Type string
}

type Disk struct {
	// Model represents the disk model, e.g., "Samsung 970 EVO Plus", "Western Digital Blue SN550"
	// TODO: may be removed as Disk models will be usually irrelevant, right?
	Model string

	// Vendor represents the disk manufacturer, e.g., "Samsung", "Western Digital"
	// TODO: may be removed as Disk vendors will be usually irrelevant, right?
	Vendor string

	// Size in bytes
	Size uint64

	// Type represents the disk type, e.g., "SSD", "HDD", "NVMe"
	Type string

	// Interface represents the disk interface, e.g., "SATA", "PCIe", "M.2"
	Interface string

	// Read speed in bytes per second
	// TODO: may be removed as it may be too specific for our case
	ReadSpeed uint64
	// Write speed in bytes per second
	// TODO: may be removed as it may be too specific for our case
	WriteSpeed uint64
}

type NetworkInfo struct {
	// Bandwidth in bits per second (b/s)
	Bandwidth uint64

	// NetworkType represents the network type, e.g., "Ethernet", "Wi-Fi", "Cellular"
	NetworkType string
}

```

```
type FreeResources struct {
    Resources
}

```

```
type OnboardedResources struct {
    Resources
}
```




### 8. References









