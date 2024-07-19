package models

type GPUVendor string

const (
	GPUVendorNvidia GPUVendor = "NVIDIA"
	GPUVendorAMDATI GPUVendor = "AMD/ATI"
	GPUVendorIntel  GPUVendor = "Intel"
)

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
	// Model is GPU model as determined by a vendor, ex A100
	Model  string `json:"model" description:"GPU model, ex A100"`
	VRAM   int    `json:"vram" description:"GPU VRAM size in MB"`	
}

type ExecutionResources struct {
	// CPU units
	CPU float64 `json:"cpu,omitempty"`
	// Memory in bytes
	Memory uint64 `json:"memory,omitempty"`
	// Disk in bytes
	Disk uint64 `json:"disk,omitempty"`
	// GPUs GPU configurations
	GPUs []GPU `json:"gpus,omitempty"`
}

