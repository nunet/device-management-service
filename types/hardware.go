// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"fmt"
	"slices"
	"strings"
)

// byte units
const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

// HardwareManager defines the interface for managing machine resources.
type HardwareManager interface {
	GetMachineResources() (MachineResources, error)
	GetUsage() (Resources, error)
	GetFreeResources() (Resources, error)
	CheckCapacity(resources Resources) (bool, error)
	Shutdown() error
}

// GPUManager defines the interface for managing GPU resources.
type GPUManager interface {
	GetGPUs() (GPUs, error)
	GetGPUUsage(uuid ...string) (GPUs, error)
	Shutdown() error
}

// GPUConnector vendor-specific adapter that interacts with actual GPU hardware by using vendor-specific libraries
type GPUConnector interface {
	GetGPUs() (GPUs, error)
	GetGPUUsage(uuid string) (uint64, error)
	Shutdown() error
}

type GPUVendor string

const (
	GPUVendorNvidia  GPUVendor = "NVIDIA"
	GPUVendorAMDATI  GPUVendor = "AMD/ATI"
	GPUVendorIntel   GPUVendor = "Intel"
	GPUVendorUnknown GPUVendor = "Unknown"
	GPUVendorNone    GPUVendor = "None"
)

// implementing Comparable interface
var _ Comparable[GPUVendor] = (*GPUVendor)(nil)

func (g GPUVendor) Compare(other GPUVendor) (Comparison, error) {
	if g == other {
		return Equal, nil
	}
	return None, nil
}

// ParseGPUVendor parses the GPU vendor string and returns the corresponding GPUVendor enum
func ParseGPUVendor(vendor string) GPUVendor {
	switch {
	case strings.Contains(strings.ToUpper(vendor), "NVIDIA"):
		return GPUVendorNvidia
	case strings.Contains(strings.ToUpper(vendor), "AMD") ||
		strings.Contains(strings.ToUpper(vendor), "ATI"):
		return GPUVendorAMDATI
	case strings.Contains(strings.ToUpper(vendor), "INTEL"):
		return GPUVendorIntel
	default:
		return GPUVendorUnknown
	}
}

// GPU represents the GPU information
type GPU struct {
	// Index is the self-reported index of the device in the system
	Index int `json:"index" description:"GPU index in the system"`
	// Vendor is the maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor `json:"vendor" description:"GPU vendor, e.g., NVidia, AMD, Intel"`
	// PCIAddress is the PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string `json:"pci_address" description:"PCI address of the device, in the format AAAA:BB:CC.C"`
	// Model represents the GPU model name, e.g., "Tesla T4", "A100"
	Model string `json:"model" description:"GPU model, e.g., Tesla T4, A100"`
	// VRAM is the total amount of VRAM on the device
	VRAM uint64 `json:"vram" description:"Total amount of VRAM on the device"`
	// UUID is the unique identifier of the device
	UUID string `json:"uuid" description:"Unique identifier of the device"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[GPU] = (*GPU)(nil)
	_ Calculable[GPU] = (*GPU)(nil)
)

// Compare compares the GPU with the other GPU
func (g *GPU) Compare(other GPU) (Comparison, error) {
	comparison := make(ComplexComparison)

	// compare the VRAM
	switch {
	case g.VRAM > other.VRAM:
		comparison["VRAM"] = Better
	case g.VRAM < other.VRAM:
		comparison["VRAM"] = Worse
	default:
		comparison["VRAM"] = Equal
	}

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and VRAM is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare types.of GPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;

	return comparison["VRAM"], nil
}

// Add adds the other GPU to the current GPU
func (g *GPU) Add(other GPU) error {
	g.VRAM += other.VRAM
	return nil
}

// Subtract subtracts the other GPU from the current GPU
func (g *GPU) Subtract(other GPU) error {
	if g.VRAM < other.VRAM {
		return fmt.Errorf("total VRAM: underflow, cannot subtract %v from %v", other.VRAM, g.VRAM)
	}

	g.VRAM -= other.VRAM
	return nil
}

// Equal checks if the two GPUs are equal
func (g *GPU) Equal(other GPU) bool {
	return g.Model == other.Model &&
		g.VRAM == other.VRAM &&
		g.Index == other.Index &&
		g.Vendor == other.Vendor &&
		g.PCIAddress == other.PCIAddress
}

// VRAMInGB returns the VRAM in gigabytes
func (g *GPU) VRAMInGB() uint64 {
	return ConvertBytesToGB(g.VRAM)
}

type GPUs []GPU

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[GPUs] = (*GPUs)(nil)
	_ Comparable[GPUs] = (*GPUs)(nil)
)

// Copy returns a safe copy of the GPUs
func (gpus GPUs) Copy() GPUs {
	gpusCopy := make(GPUs, len(gpus))
	copy(gpusCopy, gpus)
	return gpusCopy
}

func (gpus GPUs) Compare(other GPUs) (Comparison, error) {
	interimComparison1 := make([][]Comparison, 0)
	for _, otherGPU := range other {
		var interimComparison2 []Comparison
		for _, ownGPU := range gpus {
			c, err := ownGPU.Compare(otherGPU)
			if err != nil {
				return None, fmt.Errorf("error comparing GPU: %v", err)
			}
			interimComparison2 = append(interimComparison2, c)
		}
		// this matrix structure will hold the comparison results for each GPU on the right
		// with each GPU on the left in the order they are in the slices
		// first dimension represents left GPUs
		// second dimension represents right GPUs
		interimComparison1 = append(interimComparison1, interimComparison2)
	}
	// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right

	var finalComparison []Comparison
	for i := 0; i < len(interimComparison1); i++ {
		// we need to find the best match for each GPU on the right
		if len(interimComparison1[i]) < i {
			break
		}
		c := interimComparison1[i]
		bestMatch, index := returnBestMatch(c)
		finalComparison = append(finalComparison, bestMatch)
		interimComparison1 = removeIndex(interimComparison1, index)
	}

	if slices.Contains(finalComparison, Worse) {
		return Worse, nil
	}
	if SliceContainsOneValue(finalComparison, Equal) {
		return Equal, nil
	}
	return Better, nil
}

func (gpus GPUs) Add(other GPUs) error {
	// TODO: I think this logic needs to change
	// 1. if other gpu is in own gpus, add the total vram
	// 2. if other gpu is not in own gpus, append it to own gpus

	// assuming that the GPUs are ordered by index
	// which may not be the case
	otherGPUs := make(map[int]GPU)
	for _, otherGPU := range other {
		otherGPUs[otherGPU.Index] = otherGPU
	}

	for i, gpu := range gpus {
		if otherGPU, ok := otherGPUs[gpu.Index]; ok {
			if err := gpus[i].Add(otherGPU); err != nil {
				return fmt.Errorf("failed to add GPU %s: %w", gpu.Model, err)
			}
		}
	}

	return nil
}

func (gpus GPUs) Subtract(other GPUs) error {
	// assuming that the GPUs are ordered by index
	// which may not be the case
	otherGPUs := make(map[int]GPU)
	for _, otherGPU := range other {
		otherGPUs[otherGPU.Index] = otherGPU
	}

	for i, gpu := range gpus {
		if otherGPU, ok := otherGPUs[gpu.Index]; ok {
			if err := gpus[i].Subtract(otherGPU); err != nil {
				return fmt.Errorf("failed to subtract GPU %s: %w", gpu.Model, err)
			}
		}
	}

	return nil
}

func (gpus GPUs) Equal(other GPUs) bool {
	if len(gpus) != len(other) {
		return false
	}

	used := make([]bool, len(other))
	for _, gpu := range gpus {
		found := false
		for j, otherGPU := range other {
			if !used[j] && gpu.Equal(otherGPU) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// MaxFreeVRAMGPU returns the GPU with the maximum free VRAM from the list of GPUs
func (gpus GPUs) MaxFreeVRAMGPU() (GPU, error) {
	if len(gpus) == 0 {
		return GPU{}, fmt.Errorf("no GPUs found")
	}

	var maxFreeVRAMGPU GPU
	for _, gpu := range gpus {
		if gpu.VRAM > maxFreeVRAMGPU.VRAM {
			maxFreeVRAMGPU = gpu
		}
	}

	return maxFreeVRAMGPU, nil
}

// GetWithIndex returns the GPU with the specified index
func (gpus GPUs) GetWithIndex(index int) (GPU, error) {
	for _, gpu := range gpus {
		if gpu.Index == index {
			return gpu, nil
		}
	}
	return GPU{}, fmt.Errorf("GPU with index %d not found", index)
}

// CPU represents the CPU information
type CPU struct {
	// ClockSpeed represents the CPU clock speed in Hz
	ClockSpeed float64 `json:"clock_speed,omitempty" yaml:"clock_speed,omitempty" description:"CPU clock speed in Hz"`

	// Cores represents the number of physical CPU cores
	Cores float32 `json:"cores" yaml:"cores" description:"Number of physical CPU cores"`

	// TODO: capture the below fields if required
	// Model represents the CPU model, e.g., "Intel Core i7-9700K", "AMD Ryzen 9 5900X"
	Model string `json:"model,omitempty" yaml:"model,omitempty" description:"CPU model, e.g., Intel Core i7-9700K, AMD Ryzen 9 5900X"`

	// Vendor represents the CPU manufacturer, e.g., "Intel", "AMD"
	Vendor string `json:"vendor,omitempty" yaml:"vendor,omitempty" description:"CPU manufacturer, e.g., Intel, AMD"`

	// Threads represents the number of logical CPU threads (including hyperthreading)
	Threads int `json:"threads,omitempty" yaml:"threads,omitempty" description:"Number of logical CPU threads (including hyperthreading)"`

	// Architecture represents the CPU architecture, e.g., "x86", "x86_64", "arm64"
	Architecture string `json:"architecture,omitempty" yaml:"architecture,omitempty" description:"CPU architecture, e.g., x86, x86_64, arm64"`

	// Cache size in bytes
	CacheSize uint64 `json:"cache_size,omitempty" yaml:"cache_size,omitempty" description:"CPU cache size in bytes"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[CPU] = (*CPU)(nil)
	_ Comparable[CPU] = (*CPU)(nil)
)

// Compare compares the CPU with the other CPU
func (c *CPU) Compare(other CPU) (Comparison, error) {
	perfComparison := NumericComparator(
		float64(c.Cores)*c.ClockSpeed,
		float64(other.Cores)*other.ClockSpeed,
	)

	archComparison := LiteralComparator(c.Architecture, other.Architecture)
	if archComparison == Equal {
		return perfComparison, nil
	}

	return None, nil

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and frequency is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare types.of CPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;
}

// Add adds the other CPU to the current CPU
func (c *CPU) Add(other CPU) error {
	c.Cores = round(c.Cores+other.Cores, 2)
	return nil
}

// Subtract subtracts the other CPU from the current CPU
func (c *CPU) Subtract(other CPU) error {
	if c.Cores < other.Cores {
		return fmt.Errorf("core: underflow, cannot subtract %v from %v", other.Cores, c.Cores)
	}

	c.Cores = round(c.Cores-other.Cores, 2)
	return nil
}

// Compute returns the total compute power of the CPU in Hz
func (c *CPU) Compute() float64 {
	return float64(c.Cores) * c.ClockSpeed
}

// ComputeInGHz returns the total compute power of the CPU in GHz
func (c *CPU) ComputeInGHz() float64 {
	return ConvertHzToGHz(c.Compute())
}

// ClockSpeedInGHz returns the clock speed in GHz
func (c *CPU) ClockSpeedInGHz() float64 {
	return ConvertHzToGHz(c.ClockSpeed)
}

// RAM represents the RAM information
type RAM struct {
	// Size in bytes
	Size uint64 `json:"size" yaml:"size" description:"Size of the RAM in bytes"`

	// TODO: capture the below fields if required
	// Clock speed in Hz
	ClockSpeed uint64 `json:"clock_speed,omitempty" yaml:"clock_speed,omitempty" description:"Clock speed of the RAM in Hz"`

	// Type represents the RAM type, e.g., "DDR4", "DDR5", "LPDDR4"
	Type string `json:"type,omitempty" yaml:"type,omitempty" description:"RAM type, e.g., DDR4, DDR5, LPDDR4"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[RAM] = (*RAM)(nil)
	_ Comparable[RAM] = (*RAM)(nil)
)

// Compare compares the RAM with the other RAM
func (r *RAM) Compare(other RAM) (Comparison, error) {
	comparison := make(ComplexComparison)

	// compare the Size
	comparison["Size"] = NumericComparator(r.Size, other.Size)
	comparison["ClockSpeed"] = NumericComparator(r.ClockSpeed, other.ClockSpeed)

	return comparison["Size"], nil
}

// Add adds the other RAM to the current RAM
func (r *RAM) Add(other RAM) error {
	r.Size += other.Size
	return nil
}

// Subtract subtracts the other RAM from the current RAM
func (r *RAM) Subtract(other RAM) error {
	if r.Size < other.Size {
		return fmt.Errorf("size: underflow, cannot subtract %v from %v", other.Size, r.Size)
	}

	r.Size -= other.Size
	return nil
}

// SizeInGB returns the size in gigabytes
func (r *RAM) SizeInGB() uint64 {
	return ConvertBytesToGB(r.Size)
}

// Disk represents the disk information
type Disk struct {
	// Size in bytes
	Size uint64 `json:"size" yaml:"size" description:"Size of the disk in bytes"`

	// TODO: capture the below fields if required
	// Model represents the disk model, e.g., "Samsung 970 EVO Plus", "Western Digital Blue SN550"
	Model string `json:"model,omitempty" yaml:"model,omitempty" description:"Disk model, e.g., Samsung 970 EVO Plus, Western Digital Blue SN550"`

	// Vendor represents the disk manufacturer, e.g., "Samsung", "Western Digital"
	Vendor string `json:"vendor,omitempty" yaml:"vendor,omitempty" description:"Disk manufacturer, e.g., Samsung, Western Digital"`

	// Type represents the disk type, e.g., "SSD", "HDD", "NVMe"
	Type string `json:"type,omitempty" yaml:"type,omitempty" description:"Disk type, e.g., SSD, HDD, NVMe"`

	// Interface represents the disk interface, e.g., "SATA", "PCIe", "M.2"
	Interface string `json:"interface,omitempty" yaml:"interface,omitempty" description:"Disk interface, e.g., SATA, PCIe, M.2"`

	// Read speed in bytes per second
	ReadSpeed uint64 `json:"read_speed,omitempty" yaml:"read_speed,omitempty" description:"Read speed in bytes per second"`

	// Write speed in bytes per second
	WriteSpeed uint64 `json:"write_speed,omitempty" yaml:"write_speed,omitempty" description:"Write speed in bytes per second"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[Disk] = (*Disk)(nil)
	_ Comparable[Disk] = (*Disk)(nil)
)

// Compare compares the Disk with the other Disk
func (d *Disk) Compare(other Disk) (Comparison, error) {
	comparison := make(ComplexComparison)

	// compare the Size
	comparison["Size"] = NumericComparator(d.Size, other.Size)

	return comparison["Size"], nil
}

// Add adds the other Disk to the current Disk
func (d *Disk) Add(other Disk) error {
	d.Size += other.Size
	return nil
}

// Subtract subtracts the other Disk from the current Disk
func (d *Disk) Subtract(other Disk) error {
	if d.Size < other.Size {
		return fmt.Errorf("size: underflow, cannot subtract %v from %v", other.Size, d.Size)
	}

	d.Size -= other.Size
	return nil
}

// SizeInGB returns the size in gigabytes
func (d *Disk) SizeInGB() uint64 {
	return ConvertBytesToGB(d.Size)
}

// NetworkInfo represents the network information
// TODO: not yet used, but can be used to capture the network information
type NetworkInfo struct {
	// Bandwidth in bits per second (b/s)
	Bandwidth uint64

	// NetworkType represents the network type, e.g., "Ethernet", "Wi-Fi", "Cellular"
	NetworkType string
}

// TODO use units and convert the base only?

// ConvertBytesToGB converts bytes to gigabytes
// TODO should be GiB?
func ConvertBytesToGB(bytes uint64) uint64 {
	return bytes / 1e9
}

// ConvertGBToBytes converts gigabytes to bytes
// TODO should be GiB?
func ConvertGBToBytes(gb uint64) uint64 {
	return gb * 1e9
}

// ConvertMibToBytes converts mebibytes to bytes
// TODO should be MB?
func ConvertMibToBytes(mib uint64) uint64 {
	return mib * 1024 * 1024
}

// ConvertHzToGHz converts hertz to gigahertz
func ConvertHzToGHz(hz float64) float64 {
	return hz / 1e9
}
