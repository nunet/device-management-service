package resources

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"gitlab.com/nunet/device-management-service/types"
)

// linuxSystemSpecs implements the SystemSpecs interface for Linux systems
type linuxSystemSpecs struct {
	store *store
}

// newSystemSpecs returns a new instance of linuxSystemSpecs
func newSystemSpecs(store *store) *linuxSystemSpecs {
	return &linuxSystemSpecs{
		store: store,
	}
}

var _ types.SystemSpecs = (*linuxSystemSpecs)(nil)

// getRAM returns the types.RAM information for the system
func getRAM() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %s", err)
	}

	return types.RAM{
		Size: v.Total,
	}, nil
}

// getDisk returns the types.Disk for the system
func getDisk() (types.Disk, error) {
	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return types.Disk{}, fmt.Errorf("failed to get partitions: %w", err)
	}

	var totalStorage uint64
	for p := range partitions {
		usage, err := disk.UsageWithContext(context.Background(), partitions[p].Mountpoint)
		if err != nil {
			return types.Disk{}, fmt.Errorf("failed to get disk usage: %w", err)
		}
		totalStorage += usage.Total
	}

	return types.Disk{
		Size: totalStorage,
	}, nil
}

// GetCPU returns the CPU information for the system
func getCPU() (types.CPU, error) {
	cores, err := cpu.Info()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %s", err)
	}

	var totalCompute float64
	for i := 0; i < len(cores); i++ {
		totalCompute += cores[i].Mhz
	}

	return types.CPU{
		Compute:    totalCompute,
		Cores:      float32(len(cores)),
		ClockSpeed: cores[0].Mhz * 1000000,
	}, nil
}

// TODO: move the following functions to the `gpu` sub-package
// https://gitlab.com/nunet/device-management-service/-/issues/546
// assignIndexToGPUs assigns an index to each GPU in the list starting from 0
func assignIndexToGPUs(gpus []types.GPU) []types.GPU {
	for i := range gpus {
		gpus[i].Index = i
	}
	return gpus
}

// getAMDGPUInfo returns the GPU information for AMD GPUs
func getAMDGPUInfo(metadata []gpuMetadata) ([]types.GPU, error) {
	cmd := exec.Command("rocm-smi", "--showid", "--showproductname", "--showmeminfo", "vram")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("AMD ROCm not installed, initialized, or configured (reboot recommended for newly installed AMD GPU Drivers): %s", err)
	}

	outputStr := string(output)
	// fmt.Println("rocm-smi vram output:\n", outputStr) // Print the output for debugging

	gpuNameRegex := regexp.MustCompile(`GPU\[\d+\]\s+: Card Series:\s+([^\n]+)`)
	totalRegex := regexp.MustCompile(`GPU\[\d+\]\s+: VRAM Total Memory \(B\):\s+(\d+)`)
	usedRegex := regexp.MustCompile(`GPU\[\d+\]\s+: VRAM Total Used Memory \(B\):\s+(\d+)`)

	gpuNameMatches := gpuNameRegex.FindAllStringSubmatch(outputStr, -1)
	totalMatches := totalRegex.FindAllStringSubmatch(outputStr, -1)
	usedMatches := usedRegex.FindAllStringSubmatch(outputStr, -1)

	if len(gpuNameMatches) == 0 || len(totalMatches) == 0 || len(usedMatches) == 0 {
		return nil, fmt.Errorf("failed to find AMD GPU information or vram information in the output")
	}

	if len(gpuNameMatches) != len(totalMatches) || len(totalMatches) != len(usedMatches) {
		return nil, fmt.Errorf("inconsistent AMD GPU information detected")
	}

	gpuInfos := make([]types.GPU, 0)
	for i := range gpuNameMatches {
		gpuName := gpuNameMatches[i][1]
		totalMemoryBytes, err := strconv.ParseInt(totalMatches[i][1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse total amdgpu vram: %s", err)
		}

		usedMemoryBytes, err := strconv.ParseInt(usedMatches[i][1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse used amdgpu vram: %s", err)
		}

		totalMemoryMiB := totalMemoryBytes / 1024 / 1024
		usedMemoryMiB := usedMemoryBytes / 1024 / 1024
		freeMemoryMiB := totalMemoryMiB - usedMemoryMiB

		gpuInfo := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Model:      gpuName,
			TotalVRAM:  uint64(totalMemoryMiB),
			UsedVRAM:   uint64(usedMemoryMiB),
			FreeVRAM:   uint64(freeMemoryMiB),
			Vendor:     types.GPUVendorAMDATI,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

// getNVIDIAGPUInfo returns the GPU information for NVIDIA GPUs
func getNVIDIAGPUInfo(metadata []gpuMetadata) ([]types.GPU, error) {
	// Initialize NVML
	ret := nvml.Init()
	if !errors.Is(ret, nvml.SUCCESS) {
		return nil, fmt.Errorf("NVIDIA Management Library not installed, initialized or configured (reboot recommended for newly installed NVIDIA GPU drivers): %s", nvml.ErrorString(ret))
	}

	defer func() {
		_ = nvml.Shutdown()
	}()

	// Get the number of GPU devices
	deviceCount, ret := nvml.DeviceGetCount()
	if !errors.Is(ret, nvml.SUCCESS) {
		return nil, fmt.Errorf("failed to get device count: %s", nvml.ErrorString(ret))
	}

	if deviceCount != len(metadata) {
		return nil, fmt.Errorf("failed to find NVIDIA GPU information for all GPUs")
	}

	var gpus []types.GPU
	// Iterate over each device
	for i := 0; i < deviceCount; i++ {
		// Get the device handle
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if !errors.Is(ret, nvml.SUCCESS) {
			return nil, fmt.Errorf("failed to get device handle for device %d: %s", i, nvml.ErrorString(ret))
		}

		// Get the device name
		name, ret := device.GetName()
		if !errors.Is(ret, nvml.SUCCESS) {
			return nil, fmt.Errorf("failed to get name for device %d: %s", i, nvml.ErrorString(ret))
		}

		// Get the memory info
		memory, ret := device.GetMemoryInfo()
		if !errors.Is(ret, nvml.SUCCESS) {
			return nil, fmt.Errorf("failed to get nvidiagpu vram info for device %d: %s", i, nvml.ErrorString(ret))
		}

		gpu := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Name:       name,
			Model:      name,
			TotalVRAM:  memory.Total / 1024 / 1024,
			UsedVRAM:   memory.Used / 1024 / 1024,
			FreeVRAM:   memory.Free / 1024 / 1024,
			Vendor:     types.GPUVendorNvidia,
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// getIntelGPUInfo returns the GPU information for Intel GPUs
func getIntelGPUInfo(metadata []gpuMetadata) ([]types.GPU, error) {
	// Determine the number of discrete Intel GPUs
	cmd := exec.Command("xpu-smi", "health", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("xpu-smi not installed, initialized, or configured: %s", err)
	}

	outputStr := string(output)
	// fmt.Println("xpu-smi health -l output:\n", outputStr) // Print the output for debugging

	// Use regex to find all instances of Device ID
	deviceIDRegex := regexp.MustCompile(`(?i)\| Device ID\s+\|\s+(\d+)\s+\|`)
	deviceIDMatches := deviceIDRegex.FindAllStringSubmatch(outputStr, -1)
	// fmt.Printf("Found device ID matches: %v\n", deviceIDMatches) // Print matched device IDs for debugging

	if len(deviceIDMatches) == 0 {
		return nil, fmt.Errorf("failed to find any Intel GPUs")
	}

	if len(deviceIDMatches) != len(metadata) {
		return nil, fmt.Errorf("failed to find Intel GPU information for all GPUs")
	}

	gpuInfos := make([]types.GPU, 0)
	for i, match := range deviceIDMatches {
		deviceID := match[1]

		// Get GPU details using xpu-smi discovery
		cmd = exec.Command("xpu-smi", "discovery", "-d", deviceID)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get discovery info for Intel GPU %s: %s", deviceID, err)
		}

		outputStr = string(output)
		// fmt.Printf("xpu-smi discovery -d %s output:\n%s", deviceID, outputStr) // Print the output for debugging

		// Use regex to find GPU name and total memory
		nameRegex := regexp.MustCompile(`(?i)Device Name:\s+([^\n|]+)`)
		totalMemRegex := regexp.MustCompile(`(?i)Memory Physical Size:\s+([^\s]+)\s+MiB`)

		nameMatch := nameRegex.FindStringSubmatch(outputStr)
		totalMemMatch := totalMemRegex.FindStringSubmatch(outputStr)
		if nameMatch == nil || totalMemMatch == nil {
			return nil, fmt.Errorf("failed to parse discovery info for Intel GPU %s", deviceID)
		}

		gpuName := strings.TrimSpace(nameMatch[1])
		totalMemoryMiB, err := strconv.ParseFloat(totalMemMatch[1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse total memory for Intel GPU %s: %s", deviceID, err)
		}

		// Get used memory using xpu-smi stats
		cmd = exec.Command("xpu-smi", "stats", "-d", deviceID)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats for Intel GPU %s: %s", deviceID, err)
		}

		outputStr = string(output)
		// fmt.Printf("xpu-smi stats -d %s output:\n%s", deviceID, outputStr) // Print the output for debugging

		// Use regex to find used memory
		usedMemRegex := regexp.MustCompile(`(?i)GPU Memory Used \(MiB\)\s+\|\s+(\d+)\s+\|`)
		usedMemMatch := usedMemRegex.FindStringSubmatch(outputStr)
		if usedMemMatch == nil {
			return nil, fmt.Errorf("failed to parse used memory for Intel GPU %s", deviceID)
		}

		usedMemoryMiB, err := strconv.ParseFloat(usedMemMatch[1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse used memory for Intel GPU %s: %s", deviceID, err)
		}

		freeMemoryMiB := totalMemoryMiB - usedMemoryMiB

		gpuInfo := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Model:      gpuName,
			TotalVRAM:  uint64(totalMemoryMiB),
			UsedVRAM:   uint64(usedMemoryMiB),
			FreeVRAM:   uint64(freeMemoryMiB),
			Vendor:     types.GPUVendorIntel,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

// getGPUs returns the GPUs based on the specified vendors. If no vendors are provided, it returns the information of all the GPUs
func getGPUs(metadata map[types.GPUVendor][]gpuMetadata, vendors ...types.GPUVendor) ([]types.GPU, error) {
	var gpus []types.GPU

	// Helper function to fetch and append GPU info
	fetchAndAppendGPUs := func(fetchFunc func(metadata []gpuMetadata) ([]types.GPU, error), vendor types.GPUVendor) {
		vendorMetadata, ok := metadata[vendor]
		if !ok {
			zlog.Sugar().Infof("No %s GPUs found", vendor)
			return
		}

		gpuList, err := fetchFunc(vendorMetadata)
		if err != nil {
			zlog.Sugar().Warnf("Failed to retrieve %s GPU information: %v", vendor, err)
			return
		}
		gpus = append(gpus, gpuList...)
	}

	if len(vendors) == 0 {
		// No specific vendor requested, fetch all types of GPUs
		fetchAndAppendGPUs(getIntelGPUInfo, types.GPUVendorIntel)
		fetchAndAppendGPUs(getNVIDIAGPUInfo, types.GPUVendorNvidia)
		fetchAndAppendGPUs(getAMDGPUInfo, types.GPUVendorAMDATI)
	} else {
		// Fetch GPUs for the specified vendor only
		for _, vendor := range vendors {
			switch vendor {
			case types.GPUVendorIntel:
				fetchAndAppendGPUs(getIntelGPUInfo, vendor)
			case types.GPUVendorNvidia:
				fetchAndAppendGPUs(getNVIDIAGPUInfo, vendor)
			case types.GPUVendorAMDATI:
				fetchAndAppendGPUs(getAMDGPUInfo, vendor)
			default:
				return nil, fmt.Errorf("unsupported GPU vendor: %v", vendor)
			}
		}
	}

	// Assign index to GPUs and return
	// Note: The index is internal to dms and is not the same as the device index
	return assignIndexToGPUs(gpus), nil
}

// fetchGPUMetadata fetches the GPU metadata for the system using `ghw.GPU()`
// TODO: Use one single library to fetch GPU information or improve the match criteria
// https://gitlab.com/nunet/device-management-service/-/issues/548
// TODO: write tests by mocking the gpu snapshot
// https://gitlab.com/nunet/device-management-service/-/issues/534
func (l *linuxSystemSpecs) fetchGPUMetadata() (map[types.GPUVendor][]gpuMetadata, error) {
	metadata := make(map[types.GPUVendor][]gpuMetadata)
	l.store.withGpuMetadataLock(func() {
		if l.store.gpuMetadata != nil {
			metadata = l.store.gpuMetadata
			return
		}
	})
	if len(metadata) > 0 {
		return metadata, nil
	}

	gpuInfo, err := ghw.GPU()
	if err != nil {
		return nil, err
	}

	for _, card := range gpuInfo.GraphicsCards {
		if card.DeviceInfo == nil {
			continue
		}
		pciAddress := card.Address
		vendor := types.ParseGPUVendor(card.DeviceInfo.Vendor.Name)
		metadata[vendor] = append(metadata[vendor], gpuMetadata{PCIAddress: pciAddress})
	}
	l.store.withGpuMetadataLock(func() {
		l.store.gpuMetadata = metadata
	})
	return metadata, nil
}

func (l *linuxSystemSpecs) GetMachineResources() (types.MachineResources, error) {
	var (
		ok               bool
		machineResources types.MachineResources
	)
	l.store.withMachineResourcesRLock(func() {
		if l.store.machineResources != nil {
			machineResources = *l.store.machineResources
			ok = true
		}
	})
	if ok {
		return machineResources, nil
	}

	metadata, err := l.fetchGPUMetadata()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to fetch GPU metadata: %s", err)
	}

	cpuDetails, err := getCPU()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get CPU: %s", err)
	}

	ram, err := getRAM()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get RAM: %s", err)
	}

	gpus, err := getGPUs(metadata)
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get GPUs: %s", err)
	}

	diskDetails, err := getDisk()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get DISK: %s", err)
	}

	machineResources = types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuDetails,
			RAM:  ram,
			Disk: diskDetails,
			GPUs: gpus,
		},
	}
	l.store.withMachineResourcesLock(func() {
		l.store.machineResources = &machineResources
	})
	// TODO: do we wanna store it in the db?
	return machineResources, nil
}
