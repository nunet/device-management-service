package types

import (
	"fmt"
	"slices"
	"strings"
)

// HardwareManager defines the interface for managing machine resources.
type HardwareManager interface {
	GetMachineResources() (MachineResources, error)
	GetUsage() (Resources, error)
	GetFreeResources() (Resources, error)
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
	Index int
	// Vendor is the maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor
	// PCIAddress is the PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
	// Model represents the GPU model name, e.g., "Tesla T4", "A100"
	Model string `json:"model" description:"GPU model, e.g., Tesla T4, A100"`
	// VRAM is the total amount of VRAM on the device
	VRAM float64

	// Gorm fields
	// Team, is this the right way to do this? What is the best practice we're following?
	ResourceID uint `gorm:"foreignKey:ID"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[GPU] = (*GPU)(nil)
	_ Calculable[GPU] = (*GPU)(nil)
)

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

func (g *GPU) Add(other GPU) error {
	g.VRAM += other.VRAM
	return nil
}

func (g *GPU) Subtract(other GPU) error {
	if g.VRAM < other.VRAM {
		return fmt.Errorf("total VRAM: underflow, cannot subtract %v from %v", g.VRAM, other.VRAM)
	}

	g.VRAM -= other.VRAM
	return nil
}

func (g *GPU) Equal(other GPU) bool {
	return g.Model == other.Model &&
		g.VRAM == other.VRAM &&
		g.Index == other.Index &&
		g.Vendor == other.Vendor &&
		g.PCIAddress == other.PCIAddress
}

type GPUs []GPU

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[GPUs] = (*GPUs)(nil)
	_ Comparable[GPUs] = (*GPUs)(nil)
)

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

// CPU represents the CPU information
type CPU struct {
	// ClockSpeed represents the CPU clock speed in Hz
	ClockSpeed float64

	// Cores represents the number of physical CPU cores
	Cores float32

	// TODO: capture the below fields if required
	// Model represents the CPU model, e.g., "Intel Core i7-9700K", "AMD Ryzen 9 5900X"
	Model string

	// Vendor represents the CPU manufacturer, e.g., "Intel", "AMD"
	Vendor string

	// Threads represents the number of logical CPU threads (including hyperthreading)
	Threads int

	// Architecture represents the CPU architecture, e.g., "x86", "x86_64", "arm64"
	Architecture string

	// Cache size in bytes
	CacheSize uint64
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[CPU] = (*CPU)(nil)
	_ Comparable[CPU] = (*CPU)(nil)
)

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

func (c *CPU) Add(other CPU) error {
	c.Cores = round(c.Cores+other.Cores, 2)
	return nil
}

func (c *CPU) Subtract(other CPU) error {
	if c.Cores < other.Cores {
		return fmt.Errorf("core: underflow, cannot subtract %v from %v", c.Cores, other.Cores)
	}

	c.Cores = round(c.Cores-other.Cores, 2)
	return nil
}

func (c *CPU) Compute() float64 {
	return float64(c.Cores) * c.ClockSpeed
}

// RAM represents the RAM information
type RAM struct {
	// Size in bytes
	Size float64

	// TODO: capture the below fields if required
	// Clock speed in Hz
	ClockSpeed uint64

	// Type represents the RAM type, e.g., "DDR4", "DDR5", "LPDDR4"
	Type string
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[RAM] = (*RAM)(nil)
	_ Comparable[RAM] = (*RAM)(nil)
)

func (r *RAM) Compare(other RAM) (Comparison, error) {
	comparison := make(ComplexComparison)

	// compare the Size
	comparison["Size"] = NumericComparator(r.Size, other.Size)
	comparison["ClockSpeed"] = NumericComparator(r.ClockSpeed, other.ClockSpeed)

	return comparison["Size"], nil
}

func (r *RAM) Add(other RAM) error {
	r.Size += other.Size
	return nil
}

func (r *RAM) Subtract(other RAM) error {
	if r.Size < other.Size {
		return fmt.Errorf("size: underflow, cannot subtract %v from %v", r.Size, other.Size)
	}

	r.Size -= other.Size
	return nil
}

// Disk represents the disk information
type Disk struct {
	// Size in bytes
	Size float64

	// TODO: capture the below fields if required
	// Model represents the disk model, e.g., "Samsung 970 EVO Plus", "Western Digital Blue SN550"
	Model string

	// Vendor represents the disk manufacturer, e.g., "Samsung", "Western Digital"
	Vendor string

	// Type represents the disk type, e.g., "SSD", "HDD", "NVMe"
	Type string

	// Interface represents the disk interface, e.g., "SATA", "PCIe", "M.2"
	Interface string

	// Read speed in bytes per second
	ReadSpeed uint64

	// Write speed in bytes per second
	WriteSpeed uint64
}

// implementing Comparable and Calculable interfaces
var (
	_ Calculable[Disk] = (*Disk)(nil)
	_ Comparable[Disk] = (*Disk)(nil)
)

func (d *Disk) Compare(other Disk) (Comparison, error) {
	comparison := make(ComplexComparison)

	// compare the Size
	comparison["Size"] = NumericComparator(d.Size, other.Size)

	return comparison["Size"], nil
}

func (d *Disk) Add(other Disk) error {
	d.Size += other.Size
	return nil
}

func (d *Disk) Subtract(other Disk) error {
	if d.Size < other.Size {
		return fmt.Errorf("size: underflow, cannot subtract %v from %v", d.Size, other.Size)
	}

	d.Size -= other.Size
	return nil
}

// NetworkInfo represents the network information
// TODO: not yet used, but can be used to capture the network information
type NetworkInfo struct {
	// Bandwidth in bits per second (b/s)
	Bandwidth uint64

	// NetworkType represents the network type, e.g., "Ethernet", "Wi-Fi", "Cellular"
	NetworkType string
}

// GPUMetadata holds the metadata of the GPU
type GPUMetadata struct {
	PCIAddress string
}

// ConvertBytesToGB converts bytes to gigabytes
func ConvertBytesToGB(bytes float64) float64 {
	return float64(bytes) / 1e9
}

// ConvertGBToBytes converts gigabytes to bytes
func ConvertGBToBytes(gb float64) float64 {
	return gb * 1e9
}

// ConvertMiBToGB converts mebibytes to gigabytes
func ConvertMiBToGB(mib float64) float64 {
	return (mib * 1024 * 1024) / 1_000_000_000
}

// ConvertMibToBytes converts mebibytes to bytes
func ConvertMibToBytes(mib float64) float64 {
	return mib * 1024 * 1024
}
