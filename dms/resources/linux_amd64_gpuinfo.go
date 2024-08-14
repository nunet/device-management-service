//go:build linux && amd64

package resources

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/jaypipes/ghw"
	"gitlab.com/nunet/device-management-service/types"
)

/*
GetGPUInfo Function Explanation:

1. Check for GPU Presence:
   - The GetGPUInfo function includes a check to see if any GPUs are detected.
   - If no GPUs are found, the function returns a specific error message ("no GPUs detected").

2. Handling GPU Vendors:
   - The function iterates over detected GPU vendors and attempts to gather information for each vendor.
   - For NVIDIA GPUs, the function calls GetNVIDIAGPUInfo to fetch GPU details using NVML.
   - For AMD GPUs, the function checks if rocm-smi is installed using exec.LookPath.
     - If rocm-smi is not installed, the function logs a message and skips AMD GPU information.
     - If rocm-smi is installed, the function calls GetAMDGPUInfo to fetch GPU details.
   - For Intel GPUs, the function checks if xpu-smi is installed using exec.LookPath.
     - If xpu-smi is not installed, the function logs a message and skips Intel GPU information.
     - If xpu-smi is installed, the function calls GetIntelGPUInfo to fetch GPU details.
   - The function logs a message if an unknown GPU vendor is detected.

3. Fetching GPU PCI Address and Index:
   - The function fetches PCI address and index details for the detected GPUs using FetchGPUPCIAddressandIndex.
   - This information is stored in the gpuDetails map.

4. Updating GPU Information:
   - For each detected GPU, the function updates the GPU struct with model information, VRAM, free VRAM, used VRAM, vendor, PCI address, and index.
   - The PCI address and index are fetched from the gpuDetails map.

5. Returning GPU Information:
   - If no GPU information is gathered (i.e., len(gpuInfo) == 0), the function returns an error message ("no GPUs detected").
   - Otherwise, the function returns the gathered GPU information.

6. Comprehensive Handling:
   - The function ensures that it can handle cases where no GPUs are present, as well as cases where any combination of NVIDIA, AMD, and Intel GPUs are present or absent.
   - It gracefully skips over GPUs that cannot be processed due to missing tools, logging appropriate messages.
*/

func GetGPUInfo() ([]types.GPU, error) {
	var gpuInfo []types.GPU
	vendors, err := DetectGPUVendors()
	if err != nil {
		return nil, fmt.Errorf("unable to detect GPU Vendor: %v", err)
	}

	if len(vendors) == 0 {
		return nil, fmt.Errorf("no GPUs detected")
	}

	// Fetch GPU details for PCI address and index
	gpuDetails, err := FetchGPUPCIAddressandIndex()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch GPU details: %v", err)
	}

	fmt.Println("Fetched GPU details for PCI addresses and indices:")
	for idx, detail := range gpuDetails {
		fmt.Printf("Index: %d, PCIAddress: %s, Vendor: %s\n", idx, detail.PCIAddress, detail.Vendor)
	}

	foundNVIDIA, foundAMD, foundIntel := false, false, false
	for _, vendor := range vendors {
		switch vendor {
		case types.GPUVendorNvidia:
			if !foundNVIDIA {
				info, err := GetNVIDIAGPUInfo()
				if err != nil {
					fmt.Println("Skipping NVIDIA GPU info:", err)
					continue
				}
				for _, gpu := range info {
					fmt.Printf("NVIDIA GPU Model: %s\n", gpu.Model)
					// Use the correct PCI address and index from gpuDetails
					for idx, detail := range gpuDetails {
						if detail.Vendor == types.GPUVendorNvidia {
							gpu.PCIAddress = detail.PCIAddress
							gpu.Index = detail.Index
							// Mark as assigned by updating the map
							detail.Vendor = types.Unknown
							gpuDetails[idx] = detail
							break
						}
					}
					gpuInfo = append(gpuInfo, gpu)
				}
				foundNVIDIA = true
			}
		case types.GPUVendorAMDATI:
			if !foundAMD {
				if _, err := exec.LookPath("rocm-smi"); err != nil {
					fmt.Println("Skipping AMD GPU info: rocm-smi not installed")
					continue
				}
				info, err := GetAMDGPUInfo()
				if err != nil {
					fmt.Println("Skipping AMD GPU info:", err)
					continue
				}
				for _, gpu := range info {
					fmt.Printf("AMD GPU Model: %s\n", gpu.Model)
					// Use the correct PCI address and index from gpuDetails
					for idx, detail := range gpuDetails {
						if detail.Vendor == types.GPUVendorAMDATI {
							gpu.PCIAddress = detail.PCIAddress
							gpu.Index = detail.Index
							// Mark as assigned by updating the map
							detail.Vendor = types.Unknown
							gpuDetails[idx] = detail
							break
						}
					}
					gpuInfo = append(gpuInfo, gpu)
				}
				foundAMD = true
			}
		case types.GPUVendorIntel:
			if !foundIntel {
				if _, err := exec.LookPath("xpu-smi"); err != nil {
					fmt.Println("Skipping Intel GPU info: xpu-smi not installed")
					continue
				}
				info, err := GetIntelGPUInfo()
				if err != nil {
					fmt.Println("Skipping Intel GPU info:", err)
					continue
				}
				for _, gpu := range info {
					fmt.Printf("Intel GPU Model: %s\n", gpu.Model)
					// Use the correct PCI address and index from gpuDetails
					for idx, detail := range gpuDetails {
						if detail.Vendor == types.GPUVendorIntel {
							gpu.PCIAddress = detail.PCIAddress
							gpu.Index = detail.Index
							// Mark as assigned by updating the map
							detail.Vendor = types.Unknown
							gpuDetails[idx] = detail
							break
						}
					}
					gpuInfo = append(gpuInfo, gpu)
				}
				foundIntel = true
			}
		case types.Unknown:
			fmt.Println("Unknown GPU(s) detected")
		}
	}

	if len(gpuInfo) == 0 {
		return nil, fmt.Errorf("no GPUs detected")
	}

	return gpuInfo, nil
}

type GPUList []types.GPU

// Determine the GPU vendor with the highest free VRAM: NVIDIA, AMD, or Intel.
// Useful for selecting the best GPU if multiple vendors are available,
// especially in multi-GPU systems or mining rigs.
func (gpus GPUList) GetGPUWithHighestFreeVRAM() (types.GPU, error) {
	if len(gpus) == 0 {
		// Return a GPU with Vendor set to None if no GPUs are detected - Useful for launching CPU-only containers
		return types.GPU{Vendor: types.None}, nil
	}

	var maxFreeVRAMGpu types.GPU
	maxFreeVRAM := uint64(0)
	for _, gpu := range gpus {
		if gpu.FreeVram > maxFreeVRAM {
			maxFreeVRAM = gpu.FreeVram
			maxFreeVRAMGpu = gpu
		}
	}

	return maxFreeVRAMGpu, nil
}

func FetchGPUPCIAddressandIndex() (map[uint64]types.GPU, error) {
	gpuInfo, err := ghw.GPU()
	if err != nil {
		return nil, err
	}

	gpuDetails := make(map[uint64]types.GPU)
	for index, card := range gpuInfo.GraphicsCards {
		pciAddress := card.Address
		vendor := identifyVendor(card.DeviceInfo.Vendor.Name)
		gpuDetails[uint64(index)] = types.GPU{
			PCIAddress: pciAddress,
			Index:      uint64(index),
			Vendor:     vendor,
		}
	}

	return gpuDetails, nil
}

func identifyVendor(vendor string) types.GPUVendor {
	switch {
	case strings.Contains(vendor, "NVIDIA"):
		return types.GPUVendorNvidia
	case strings.Contains(vendor, "AMD"):
		return types.GPUVendorAMDATI
	case strings.Contains(vendor, "Intel"):
		return types.GPUVendorIntel
	default:
		return types.Unknown
	}
}

func GetNVIDIAGPUInfo() ([]types.GPU, error) {
	// Initialize NVML
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("NVIDIA Management Library not installed, initialized or configured (reboot recommended for newly installed NVIDIA GPU drivers): %s", nvml.ErrorString(ret))
	}
	defer nvml.Shutdown()

	// Get the number of GPU devices
	deviceCount, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device count: %s", nvml.ErrorString(ret))
	}

	var gpuInfos []types.GPU

	// Iterate over each device
	for i := uint32(0); i < uint32(deviceCount); i++ {
		// Get the device handle
		device, ret := nvml.DeviceGetHandleByIndex(int(i))
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device handle for device %d: %s", i, nvml.ErrorString(ret))
		}

		// Get the device name
		name, ret := nvml.DeviceGetName(device)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get name for device %d: %s", i, nvml.ErrorString(ret))
		}

		// Get the memory info
		memory, ret := nvml.DeviceGetMemoryInfo(device)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get nvidiagpu vram info for device %d: %s", i, nvml.ErrorString(ret))
		}

		gpuInfo := types.GPU{
			Model:    name,
			VRAM:     memory.Total / 1024 / 1024,
			UsedVram: memory.Used / 1024 / 1024,
			FreeVram: memory.Free / 1024 / 1024,
			Vendor:   types.GPUVendorNvidia,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

func GetAMDGPUInfo() ([]types.GPU, error) {
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

	var gpuInfos []types.GPU
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
			Model:    gpuName,
			VRAM:     uint64(totalMemoryMiB),
			UsedVram: uint64(usedMemoryMiB),
			FreeVram: uint64(freeMemoryMiB),
			Vendor:   types.GPUVendorAMDATI,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

func GetIntelGPUInfo() ([]types.GPU, error) {
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

	var gpuInfos []types.GPU
	for _, match := range deviceIDMatches {
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
			Model:    gpuName,
			VRAM:     uint64(totalMemoryMiB),
			UsedVram: uint64(usedMemoryMiB),
			FreeVram: uint64(freeMemoryMiB),
			Vendor:   types.GPUVendorIntel,
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nil
}

func DetectGPUVendors() ([]types.GPUVendor, error) {
	var vendors []types.GPUVendor
	gpu, err := ghw.GPU()
	if err != nil {
		return nil, err
	}

	for _, card := range gpu.GraphicsCards {
		deviceInfo := card.DeviceInfo
		if deviceInfo != nil {
			class := deviceInfo.Class
			if class != nil {
				className := strings.ToLower(class.Name)
				if strings.Contains(className, "display controller") ||
					strings.Contains(className, "vga compatible controller") ||
					strings.Contains(className, "3d controller") ||
					strings.Contains(className, "2d controller") {
					vendor := card.DeviceInfo.Vendor
					if vendor != nil {
						if strings.Contains(strings.ToLower(vendor.Name), "nvidia") {
							vendors = append(vendors, types.GPUVendorNvidia)
						}
						if strings.Contains(strings.ToLower(vendor.Name), "amd") {
							vendors = append(vendors, types.GPUVendorAMDATI)
						}
						if strings.Contains(strings.ToLower(vendor.Name), "intel") {
							vendors = append(vendors, types.GPUVendorIntel)
						}
					}
				}
			}
		}
	}

	if len(vendors) == 0 {
		return []types.GPUVendor{types.Unknown}, nil
	}

	return vendors, nil
}
