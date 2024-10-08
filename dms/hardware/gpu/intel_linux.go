//go:build linux && amd64

package gpu

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"gitlab.com/nunet/device-management-service/types"
)

// runXpuSmiCommand runs the xpu-smi command with the provided arguments and returns the output as a string.
func runXpuSmiCommand(args ...string) (string, error) {
	cmd := exec.Command("xpu-smi", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("xpu-smi command failed: %s", err)
	}
	return string(output), nil
}

// getIntelGPUDeviceIDs extracts the device IDs of Intel GPUs from the xpu-smi output.
func getIntelGPUDeviceIDs(output string) ([]string, error) {
	deviceIDRegex := regexp.MustCompile(`(?i)\| Device ID\s+\|\s+(\d+)\s+\|`)
	deviceIDMatches := deviceIDRegex.FindAllStringSubmatch(output, -1)

	if len(deviceIDMatches) == 0 {
		return nil, fmt.Errorf("failed to find any Intel GPUs")
	}

	deviceIDs := make([]string, len(deviceIDMatches))
	for i, match := range deviceIDMatches {
		deviceIDs[i] = match[1]
	}

	return deviceIDs, nil
}

// getIntelGPUDiscoveryInfo retrieves the GPU name and total memory for a specific Intel GPU.
func getIntelGPUDiscoveryInfo(deviceID string) (string, float64, error) {
	output, err := runXpuSmiCommand("discovery", "-d", deviceID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get discovery info for Intel GPU %s: %s", deviceID, err)
	}

	// Extract the GPU name and total memory
	nameRegex := regexp.MustCompile(`(?i)Device Name:\s+([^\n|]+)`)
	totalMemRegex := regexp.MustCompile(`(?i)Memory Physical Size:\s+([^\s]+)\s+MiB`)

	nameMatch := nameRegex.FindStringSubmatch(output)
	totalMemMatch := totalMemRegex.FindStringSubmatch(output)

	if nameMatch == nil || totalMemMatch == nil {
		return "", 0, fmt.Errorf("failed to parse discovery info for Intel GPU %s", deviceID)
	}

	gpuName := strings.TrimSpace(nameMatch[1])
	totalMemoryMiB, err := strconv.ParseFloat(totalMemMatch[1], 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse total memory for Intel GPU %s: %s", deviceID, err)
	}

	return gpuName, types.ConvertMibToBytes(totalMemoryMiB), nil
}

// getIntelGPUUsedMemory retrieves the used memory for a specific Intel GPU.
func getIntelGPUUsedMemory(deviceID string) (float64, error) {
	output, err := runXpuSmiCommand("stats", "-d", deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get stats for Intel GPU %s: %s", deviceID, err)
	}

	// Extract the used memory
	usedMemRegex := regexp.MustCompile(`(?i)GPU Memory Used \(MiB\)\s+\|\s+(\d+)\s+\|`)
	usedMemMatch := usedMemRegex.FindStringSubmatch(output)

	if usedMemMatch == nil {
		return 0, fmt.Errorf("failed to parse used memory for Intel GPU %s", deviceID)
	}

	usedMemory, err := strconv.ParseFloat(usedMemMatch[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse used memory for Intel GPU %s: %s", deviceID, err)
	}

	return types.ConvertMibToBytes(usedMemory), nil
}

// getIntelGPUs returns the GPU information for Intel GPUs.
func getIntelGPUs(metadata []types.GPUMetadata) ([]types.GPU, error) {
	// Get the list of Intel GPU devices
	output, err := runXpuSmiCommand("health", "-l")
	if err != nil {
		return nil, fmt.Errorf("xpu-smi not installed, initialized, or configured: %s", err)
	}

	// Get Intel GPU device IDs
	deviceIDs, err := getIntelGPUDeviceIDs(output)
	if err != nil {
		return nil, err
	}

	if len(deviceIDs) != len(metadata) {
		return nil, fmt.Errorf("failed to find Intel GPU information for all GPUs")
	}

	gpuInfos := make([]types.GPU, 0, len(deviceIDs))
	for i, deviceID := range deviceIDs {
		// Get GPU discovery info
		gpuName, totalMemoryMiB, err := getIntelGPUDiscoveryInfo(deviceID)
		if err != nil {
			return nil, err
		}

		// Populate GPU info
		gpuInfo := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Model:      gpuName,
			VRAM:       totalMemoryMiB,
			Vendor:     types.GPUVendorIntel,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

// getIntelGPUUsage returns the GPU usage for Intel GPUs.
func getIntelGPUUsage(metadata []types.GPUMetadata) ([]types.GPU, error) {
	// Get the list of Intel GPU devices
	output, err := runXpuSmiCommand("health", "-l")
	if err != nil {
		return nil, fmt.Errorf("xpu-smi not installed, initialized, or configured: %s", err)
	}

	// Get Intel GPU device IDs
	deviceIDs, err := getIntelGPUDeviceIDs(output)
	if err != nil {
		return nil, err
	}

	if len(deviceIDs) != len(metadata) {
		return nil, fmt.Errorf("failed to find Intel GPU information for all GPUs")
	}

	gpuInfos := make([]types.GPU, 0, len(deviceIDs))
	for i, deviceID := range deviceIDs {
		gpuName, _, err := getIntelGPUDiscoveryInfo(deviceID)
		if err != nil {
			return nil, err
		}

		usedMemory, err := getIntelGPUUsedMemory(deviceID)
		if err != nil {
			return nil, err
		}

		// Populate GPU info
		gpuInfo := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Model:      gpuName,
			VRAM:       usedMemory,
			Vendor:     types.GPUVendorIntel,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}
