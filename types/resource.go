package types

import (
	"fmt"
	"strings"
)

type GPUVendor string

const (
	GPUVendorNvidia  GPUVendor = "NVIDIA"
	GPUVendorAMDATI  GPUVendor = "AMD/ATI"
	GPUVendorIntel   GPUVendor = "Intel"
	GPUVendorUnknown GPUVendor = "Unknown"
	None             GPUVendor = "None"
)

func ParseGPUVendor(vendor string) GPUVendor {
	switch {
	case strings.Contains(vendor, "NVIDIA"):
		return GPUVendorNvidia
	case strings.Contains(vendor, "AMD") || strings.Contains(vendor, "ATI"):
		return GPUVendorAMDATI
	case strings.Contains(vendor, "Intel"):
		return GPUVendorIntel
	default:
		return GPUVendorUnknown
	}
}

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
	// Team, is this the right way to do this? What is the best practice we're following?
	ResourceID uint `gorm:"foreignKey:ID"`
}

func (g *GPU) Equal(gpu *GPU) bool {
	if g.Model == gpu.Model &&
		g.TotalVRAM == gpu.TotalVRAM &&
		g.UsedVRAM == gpu.UsedVRAM &&
		g.FreeVRAM == gpu.FreeVRAM &&
		g.Index == gpu.Index &&
		g.Vendor == gpu.Vendor &&
		g.PCIAddress == gpu.PCIAddress {
		return true
	}
	return false
}

type GPUList []GPU

// GetGPUWithHighestFreeVRAM Determine the GPU vendor with the highest free VRAM: NVIDIA, AMD, or Intel.
// Useful for selecting the best GPU if multiple vendors are available,
// especially in multi-GPU systems or mining rigs.
func (gpus GPUList) GetGPUWithHighestFreeVRAM() (GPU, error) {
	if len(gpus) == 0 {
		// Return a GPU with Vendor set to None if no GPUs are detected - Useful for launching CPU-only containers
		return GPU{Vendor: None}, nil
	}

	var maxFreeVRAMGpu GPU
	maxFreeVRAM := uint64(0)
	for _, gpu := range gpus {
		if gpu.FreeVRAM > maxFreeVRAM {
			maxFreeVRAM = gpu.FreeVRAM
			maxFreeVRAMGpu = gpu
		}
	}

	return maxFreeVRAMGpu, nil
}

// negativeValueError is a type struct used to return a custom error for negative values in resources subtraction
type negativeValueError struct {
	resource string
	r1       any
	r2       any
}

// Error returns the error message
func (e *negativeValueError) Error() string {
	return fmt.Sprintf("Error: %s subtraction results in negative values. (%d - %d)", e.resource, e.r1, e.r2)
}

// ResourceOps defines the operations on resources
// TODO: Check how to handle GPU resources
type ResourceOps interface {
	// Add returns the sum of the resources
	Add(r Resources) Resources
	// Subtract returns the difference of the resources
	Subtract(r Resources) (Resources, error)
}

// Resources represents the resources of the machine
type Resources struct {
	CPU      float64
	NumCores int
	GPU      []GPU `gorm:"foreignKey:ResourceID"`
	RAM      uint64
	Disk     uint64
}

// Add returns the sum of the resources
func (r Resources) Add(r2 Resources) Resources {
	// TODO: GPU addition
	return Resources{
		CPU:  r.CPU + r2.CPU,
		RAM:  r.RAM + r2.RAM,
		Disk: r.Disk + r2.Disk,
	}
}

// Subtract returns the difference of the resources
func (r Resources) Subtract(r2 Resources) (Resources, error) {
	// Check if the subtraction results in negative values

	// Team, why do we need to return a negative value error?
	// Can't we just return the negative value indicating that the machine is overused?
	// We can then handle the scenario in the calling function by checking if the result is negative if required.
	if r.CPU < r2.CPU {
		return Resources{}, &negativeValueError{resource: "CPU", r1: r.CPU, r2: r2.CPU}
	}

	// TODO: GPU subtraction

	if r.RAM < r2.RAM {
		return Resources{}, &negativeValueError{resource: "RAM", r1: r.RAM, r2: r2.RAM}
	}

	if r.Disk < r2.Disk {
		return Resources{}, &negativeValueError{resource: "Disk", r1: r.Disk, r2: r2.Disk}
	}

	return Resources{
		CPU:  r.CPU - r2.CPU,
		RAM:  r.RAM - r2.RAM,
		Disk: r.Disk - r2.Disk,
	}, nil
}

var _ ResourceOps = (*Resources)(nil)

// FreeResources represents the free resources of the machine
type FreeResources struct {
	BaseDBModel
	Resources
}

// OnboardedResources represents the onboarded resources of the machine
type OnboardedResources struct {
	BaseDBModel
	Resources
}

// RequiredResources represents the resources required by the jobs running on the machine
// TODO: this is a replacement for ServiceResourceRequirements. Check with the team on this.
type RequiredResources struct {
	BaseDBModel
	JobID int
	Resources
}

// CPUInfo represents the CPU information of the machine
type CPUInfo struct {
	NumCores   uint64
	MHzPerCore float64
	Compute    float64
}

// SpecInfo represents the machine specifications
// TODO: Finalise the fields required in this struct
// https://gitlab.com/nunet/device-management-service/-/issues/533
type SpecInfo struct {
	CPUs    []CPU
	GPUs    []GPU
	RAMs    []RAM
	Disks   []Disk
	Network NetworkInfo
}

// CPU represents the CPU information
type CPU struct {
	// Model represents the CPU model, e.g., "Intel Core i7-9700K", "AMD Ryzen 9 5900X"
	Model string

	// Vendor represents the CPU manufacturer, e.g., "Intel", "AMD"
	Vendor string

	// ClockSpeedHz represents the CPU clock speed in Hz
	ClockSpeedHz int64

	// Cores represents the number of physical CPU cores
	Cores uint32

	// Threads represents the number of logical CPU threads (including hyperthreading)
	Threads int

	// Architecture represents the CPU architecture, e.g., "x86", "x86_64", "arm64"
	Architecture string

	// Cache size in bytes
	CacheSize uint64
}

// RAM represents the RAM information
type RAM struct {
	// Size in bytes
	Size int64

	// Clock speed in Hz
	ClockSpeedHz uint64

	// Type represents the RAM type, e.g., "DDR4", "DDR5", "LPDDR4"
	Type string
}

// Disk represents the disk information
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

// NetworkInfo represents the network information
type NetworkInfo struct {
	// Bandwidth in bits per second (b/s)
	Bandwidth uint64

	// NetworkType represents the network type, e.g., "Ethernet", "Wi-Fi", "Cellular"
	NetworkType string
}

// ExecutionResources represents the resources required to execute a task
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
