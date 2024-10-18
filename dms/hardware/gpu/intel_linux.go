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

// getIntelGPUPCIAddress extracts the PCI bus ID from the xpu-smi output.
func getIntelGPUPCIAddress(deviceID string) (string, error) {
	output, err := runXpuSmiCommand("discovery", "-d", deviceID)
	if err != nil {
		return "", fmt.Errorf("failed to get PCI address for Intel GPU %s: %s", deviceID, err)
	}

	// Extract the PCI bus address
	pciRegex := regexp.MustCompile(`(?i)PCI\s+BDF\s+Address:\s+([^\n|]+)`)
	pciMatch := pciRegex.FindStringSubmatch(output)

	if pciMatch == nil {
		return "", fmt.Errorf("failed to parse PCI bus address for Intel GPU %s", deviceID)
	}

	pciAddress := strings.TrimSpace(pciMatch[1])
	return pciAddress, nil
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
func getIntelGPUs() ([]types.GPU, error) {
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

	gpuInfos := make([]types.GPU, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		// Get GPU discovery info
		gpuName, totalMemoryMiB, err := getIntelGPUDiscoveryInfo(deviceID)
		if err != nil {
			return nil, err
		}

		// Get PCI address
		pciAddress, err := getIntelGPUPCIAddress(deviceID)
		if err != nil {
			return nil, fmt.Errorf("get PCI address for Intel GPU %s: %s", deviceID, err)
		}

		// Populate GPU info
		gpuInfo := types.GPU{
			Model:      gpuName,
			VRAM:       totalMemoryMiB,
			Vendor:     types.GPUVendorIntel,
			PCIAddress: pciAddress,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

// getIntelGPUUsage returns the GPU usage for Intel GPUs.
func getIntelGPUUsage() ([]types.GPU, error) {
	// Reuse xpu-smi output and Intel GPU device IDs to avoid multiple executions
	output, err := runXpuSmiCommand("health", "-l")
	if err != nil {
		return nil, fmt.Errorf("xpu-smi not installed, initialized, or configured: %s", err)
	}

	// Get Intel GPU device IDs
	deviceIDs, err := getIntelGPUDeviceIDs(output)
	if err != nil {
		return nil, err
	}

	gpuInfos := make([]types.GPU, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		gpuName, _, err := getIntelGPUDiscoveryInfo(deviceID)
		if err != nil {
			return nil, err
		}

		usedMemory, err := getIntelGPUUsedMemory(deviceID)
		if err != nil {
			return nil, err
		}

		pciAddress, err := getIntelGPUPCIAddress(deviceID)
		if err != nil {
			return nil, fmt.Errorf("get PCI address for Intel GPU %s: %s", deviceID, err)
		}

		// Populate GPU info
		gpuInfo := types.GPU{
			PCIAddress: pciAddress,
			Model:      gpuName,
			VRAM:       usedMemory,
			Vendor:     types.GPUVendorIntel,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}
