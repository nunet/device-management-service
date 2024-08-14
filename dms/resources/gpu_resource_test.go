package resources

import (
	"fmt"
	"testing"

	"gitlab.com/nunet/device-management-service/types"
)

func TestGetNVIDIAGPUInfo(t *testing.T) {
	gpuInfos, err := GetNVIDIAGPUInfo()
	if err != nil {
		t.Skip("Skipping test: NVIDIA GPU info not available:", err)
		return
	}

	fmt.Println("NVIDIA GPU Memory Info:")
	for _, gpuInfo := range gpuInfos {
		fmt.Printf("Model: %s, Vendor: %s, TotalMemory: %d MB, UsedMemory: %d MB, FreeMemory: %d MB\n",
			gpuInfo.Model, gpuInfo.Vendor, gpuInfo.VRAM, gpuInfo.UsedVram, gpuInfo.FreeVram)
	}
}

func TestGetAMDGPUInfo(t *testing.T) {

	gpuInfos, err := GetAMDGPUInfo()
	if err != nil {
		t.Skip("Skipping test: AMD GPU info not available:", err)
		return
	}

	fmt.Println("AMD GPU Memory Info:")
	for _, gpuInfo := range gpuInfos {
		fmt.Printf("Model: %s, Vendor: %s, TotalMemory: %d MB, UsedMemory: %d MB, FreeMemory: %d MB\n",
			gpuInfo.Model, gpuInfo.Vendor, gpuInfo.VRAM, gpuInfo.UsedVram, gpuInfo.FreeVram)
	}
}

func TestGetIntelGPUInfo(t *testing.T) {

	gpuInfos, err := GetIntelGPUInfo()
	if err != nil {
		t.Skip("Skipping test: Intel GPU info not available:", err)
		return
	}

	fmt.Println("Intel GPU Memory Info:")
	for _, gpuInfo := range gpuInfos {
		fmt.Printf("Model: %s, Vendor: %s, TotalMemory: %d MB, UsedMemory: %d MB, FreeMemory: %d MB\n",
			gpuInfo.Model, gpuInfo.Vendor, gpuInfo.VRAM, gpuInfo.UsedVram, gpuInfo.FreeVram)
	}
}

func TestGetGPUInfo(t *testing.T) {
	gpuInfo, err := GetGPUInfo()
	if err != nil {
		t.Fatalf("Error checking GPUs: %v", err)
	}

	if len(gpuInfo) == 0 {
		t.Fatal("No GPUs detected")
	}

	fmt.Println("Detected GPU Info:")
	for _, gpu := range gpuInfo {
		fmt.Printf("Model: %s, Total VRAM: %d MB, Free VRAM: %d MB, Used VRAM: %d MB, Vendor: %s, PCI Address: %s, Index: %d\n",
			gpu.Model, gpu.VRAM, gpu.FreeVram, gpu.UsedVram, gpu.Vendor, gpu.PCIAddress, gpu.Index)
	}
}

func TestDetectGPUVendors(t *testing.T) {
	vendors, err := DetectGPUVendors()
	if err != nil {
		t.Fatalf("Error detecting GPU vendors: %v", err)
	}

	if len(vendors) == 0 {
		t.Fatal("No GPU vendors detected")
	}

	// Print detected vendors for debugging purposes
	for _, vendor := range vendors {
		t.Logf("Detected GPU vendor: %v", vendor)
	}

	// Check if the detected vendors are among the expected values
	expectedVendors := map[types.GPUVendor]bool{
		types.GPUVendorNvidia: true,
		types.GPUVendorAMDATI: true,
		types.GPUVendorIntel:  true,
		types.Unknown:         true,
	}

	for _, vendor := range vendors {
		if !expectedVendors[vendor] {
			t.Fatalf("Unexpected GPU vendor detected: %v", vendor)
		}
	}
}

func TestGetGPUWithHighestFreeVRAM(t *testing.T) {
	gpus, err := GetGPUInfo()

	gpu, err := GPUList(gpus).GetGPUWithHighestFreeVRAM()
	if err != nil {
		t.Fatalf("Error getting GPU with highest free VRAM: %v", err)
	}

	fmt.Printf("GPU with highest free VRAM: Model: %s, Total VRAM: %d MB, Free VRAM: %d MB, PCI Address: %s, Index: %d\n",
		gpu.Model, gpu.VRAM, gpu.FreeVram, gpu.PCIAddress, gpu.Index)
}
