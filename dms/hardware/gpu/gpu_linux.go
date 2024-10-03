//go:build linux && amd64

package gpu

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"gitlab.com/nunet/device-management-service/types"
)

// getAMDGPUInfo returns the GPU information for AMD GPUs
func getAMDGPUInfo(metadata []types.GPUMetadata) ([]types.GPU, error) {
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
func getNVIDIAGPUInfo(metadata []types.GPUMetadata) ([]types.GPU, error) {
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
func getIntelGPUInfo(metadata []types.GPUMetadata) ([]types.GPU, error) {
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

// GetGPUs returns the GPUs based on the specified vendors. If no vendors are provided, it returns the information of all the GPUs
func GetGPUs(vendors ...types.GPUVendor) ([]types.GPU, error) {
	var gpus []types.GPU

	gpuMetadata, err := fetchGPUMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GPU metadata: %v", err)
	}

	// Helper function to fetch and append GPU info
	fetchAndAppendGPUs := func(fetchFunc func(metadata []types.GPUMetadata) ([]types.GPU, error), vendor types.GPUVendor) {
		vendorMetadata, ok := gpuMetadata[vendor]
		if !ok {
			// TODO: log a warning here
			return
		}

		gpuList, err := fetchFunc(vendorMetadata)
		if err != nil {
			// TODO: log a warning here
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

// fetchGPUMetadata returns the GPU metadata for all GPUs
// TODO: Use one single library to fetch GPU information or improve the match criteria
// https://gitlab.com/nunet/device-management-service/-/issues/548
// TODO: write tests by mocking the gpu snapshot
// https://gitlab.com/nunet/device-management-service/-/issues/534
func fetchGPUMetadata() (map[types.GPUVendor][]types.GPUMetadata, error) {
	metadata := make(map[types.GPUVendor][]types.GPUMetadata)

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
		metadata[vendor] = append(metadata[vendor], types.GPUMetadata{PCIAddress: pciAddress})
	}

	return metadata, nil
}
