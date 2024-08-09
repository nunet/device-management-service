package models

type GPUVendor string

const (
	GPUVendorNvidia GPUVendor = "NVIDIA"
	GPUVendorAMDATI GPUVendor = "AMD/ATI"
	GPUVendorIntel  GPUVendor = "Intel"
	Unknown         GPUVendor = "Unknown"
	None            GPUVendor = "None"
)

// Moved from models/machine.go for code clarity
// Using a single GPU struct instead of multiple ones for simplicity
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
	Index uint64 `json:"index,omitempty" description:"GPU index"`
	// Maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor `json:"vendor,omitempty" description:"GPU vendor"`
	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string `json:"pci_address,omitempty" description:"PCI address of the GPU"`
}

type CPU struct {
	Arch  string `json:"arch,omitempty" description:"CPU architecture"`
	Cores uint64 `json:"cores,omitempty" description:"Number of CPU cores"`
	Freq  uint64 `json:"freq,omitempty" description:"CPU frequency in MHz"`
}

// Memory represents the memory configuration
type Memory struct {
	Size  uint64 `json:"size,omitempty" description:"Memory size in MB"`
	Speed uint64 `json:"speed,omitempty" description:"Memory speed in MHz"`
}

// Disk represents the disk configuration
type Disk struct {
	Type string `json:"type,omitempty" description:"Disk type"`
	Size uint64 `json:"size,omitempty" description:"Disk size in MB"`
}

// ExecutionResources represents the resources required to execute a task
type ExecutionResources struct {
	// CPU configuration
	CPU CPU `json:"cpu,omitempty" description:"CPU configuration"`
	// Memory configuration
	Memory Memory `json:"memory,omitempty" description:"Memory configuration"`
	// Disk configuration
	Disk Disk `json:"disk,omitempty" description:"Disk configuration"`
	// GPU configuration
	GPUs []GPU `json:"gpus,omitempty" description:"GPU configuration"`
}
