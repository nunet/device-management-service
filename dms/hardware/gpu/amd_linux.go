//go:build linux && amd64

package gpu

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"gitlab.com/nunet/device-management-service/types"
)

// runROCmSmiCommand executes the rocm-smi command and returns the output as a string.
func runROCmSmiCommand() (string, error) {
	cmd := exec.Command("rocm-smi", "--showid", "--showproductname", "--showmeminfo", "vram")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AMD ROCm not installed, initialized, or configured (reboot recommended for newly installed AMD GPU Drivers): %s", err)
	}
	return string(output), nil
}

// parseRegex extracts all matches from the given regex pattern and returns the matches.
func parseRegex(pattern, output string) [][]string {
	regex := regexp.MustCompile(pattern)
	return regex.FindAllStringSubmatch(output, -1)
}

// getAMDGPUTotalVRAM extracts the total VRAM from the command output and converts it to MiB.
func getAMDGPUTotalVRAM(output string) ([]float64, error) {
	totalMatches := parseRegex(`GPU\[\d+\]\s+: VRAM Total Memory \(B\):\s+(\d+)`, output)
	if len(totalMatches) == 0 {
		return nil, fmt.Errorf("find total VRAM in the output")
	}

	totalVRAMs := make([]float64, len(totalMatches))
	for i, match := range totalMatches {
		memoryBytes, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse total VRAM for GPU %d: %s", i, err)
		}
		totalVRAMs[i] = memoryBytes
	}
	return totalVRAMs, nil
}

// getAMDGPUUsedVRAM extracts the used VRAM from the command output and converts it to MiB.
func getAMDGPUUsedVRAM(output string) ([]float64, error) {
	usedMatches := parseRegex(`GPU\[\d+\]\s+: VRAM Total Used Memory \(B\):\s+(\d+)`, output)
	if len(usedMatches) == 0 {
		return nil, fmt.Errorf("find used VRAM in the output")
	}

	usedVRAMs := make([]float64, len(usedMatches))
	for i, match := range usedMatches {
		memoryBytes, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse used VRAM for GPU %d: %s", i, err)
		}
		usedVRAMs[i] = memoryBytes
	}
	return usedVRAMs, nil
}

// getAMDGPUName extracts the GPU name from the command output.
func getAMDGPUName(output string) ([]string, error) {
	nameMatches := parseRegex(`GPU\[\d+\]\s+: Card Series:\s+([^\n]+)`, output)
	if len(nameMatches) == 0 {
		return nil, fmt.Errorf("find GPU names in the output")
	}

	names := make([]string, len(nameMatches))
	for i, match := range nameMatches {
		names[i] = match[1]
	}
	return names, nil
}

// getAMDGPUs returns the GPU information for AMD GPUs.
func getAMDGPUs(metadata []types.GPUMetadata) ([]types.GPU, error) {
	output, err := runROCmSmiCommand()
	if err != nil {
		return nil, err
	}

	gpuNameMatches, err := getAMDGPUName(output)
	if err != nil {
		return nil, err
	}

	totalVRAMs, err := getAMDGPUTotalVRAM(output)
	if err != nil {
		return nil, err
	}

	gpuInfos := make([]types.GPU, 0, len(gpuNameMatches))
	for i := range gpuNameMatches {
		gpuInfo := types.GPU{
			PCIAddress: metadata[i].PCIAddress,
			Model:      gpuNameMatches[i],
			VRAM:       totalVRAMs[i],
			Vendor:     types.GPUVendorAMDATI,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

// getAMDGPUUsage returns the GPU usage for AMD GPUs.
func getAMDGPUUsage(_ []types.GPUMetadata) ([]types.GPU, error) {
	output, err := runROCmSmiCommand()
	if err != nil {
		return nil, err
	}

	gpuNameMatches, err := getAMDGPUName(output)
	if err != nil {
		return nil, err
	}

	usedVRAMs, err := getAMDGPUUsedVRAM(output)
	if err != nil {
		return nil, err
	}

	gpus := make([]types.GPU, 0, len(usedVRAMs))
	for i, usedVRAM := range usedVRAMs {
		gpuInfo := types.GPU{
			Model: gpuNameMatches[i],
			VRAM:  usedVRAM,
		}
		gpus = append(gpus, gpuInfo)
	}

	return gpus, nil
}
