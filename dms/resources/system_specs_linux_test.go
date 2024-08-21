package resources

import (
	"fmt"
	"gitlab.com/nunet/device-management-service/types"
	"testing"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/assert"
)

func TestLinuxNewSystemSpecs(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	assert.NotNil(t, systemSpecs)
}

func TestLinuxSystemSpecs_GetTotalMemory(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	totalMemory, err := systemSpecs.GetTotalMemory()
	assert.NoError(t, err)
	assert.Greater(t, totalMemory, uint64(0))
}

func TestLinuxSystemSpecs_GetTotalStorage(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	totalStorage, err := systemSpecs.GetTotalStorage()
	assert.NoError(t, err)
	assert.Greater(t, totalStorage, uint64(0))
}

func TestLinuxSystemSpecs_GetCPUInfo(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	cpuInfo, err := systemSpecs.GetCPUInfo()
	assert.NoError(t, err)

	expectedCPUInfo := getLinuxCPUInfo(t)
	assert.Equal(t, expectedCPUInfo.NumCores, cpuInfo.NumCores)
	assert.Equal(t, expectedCPUInfo.MHzPerCore, cpuInfo.MHzPerCore)
	assert.Equal(t, expectedCPUInfo.Compute, cpuInfo.Compute)
}

func TestLinuxSystemSpecs_GetProvisionedResources(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	resources, err := systemSpecs.GetProvisionedResources()
	assert.NoError(t, err)

	expectedCPUInfo := getLinuxCPUInfo(t)
	assert.Equal(t, expectedCPUInfo.Compute, resources.CPU)
	assert.Greater(t, resources.RAM, uint64(0))
	assert.Greater(t, resources.Disk, uint64(0))
}

// TODO: we need to mock the gpu details so that the library can pick it up
// https://gitlab.com/nunet/device-management-service/-/issues/534
func TestLinuxSystemSpecs_GetGPUVendors(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	gpuVendors, err := systemSpecs.GetGPUVendors()
	assert.NoError(t, err)
	fmt.Println(gpuVendors)
}

// TODO: we need to mock the gpu details so that the library can pick it up
// https://gitlab.com/nunet/device-management-service/-/issues/534
func TestLinuxSystemSpecs_GetGPUs(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()

	gpuVendors, err := systemSpecs.GetGPUVendors()
	assert.NoError(t, err)

	gpuVendorMap := make(map[types.GPUVendor]struct{})
	for _, vendor := range gpuVendors {
		gpuVendorMap[vendor] = struct{}{}
	}

	gpus, err := systemSpecs.GetGPUs()
	assert.NoError(t, err)

	// Ensure that all the GPUs belong to the vendors we got from GetGPUVendors
	for _, gpu := range gpus {
		assert.Contains(t, gpuVendorMap, gpu.Vendor)
	}
}

func Test_PrettyPrintGPUs(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs()
	// get specific vendor GPUs by passing the vendor as an argument. Example:
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorNvidia)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorAMDATI)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorIntel)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorNvidia, models.GPUVendorAMDATI)
	//
	// If no vendors are provided, it returns the information of all the GPUs
	gpus, err := systemSpecs.GetGPUs()
	assert.NoError(t, err)

	fmt.Println("GPU Details:")
	for _, gpu := range gpus {
		fmt.Printf("Model: %s, Total VRAM: %d MB, Free VRAM: %d MB, Used VRAM: %d MB, Vendor: %s, PCI Address: %s, Index: %d\n",
			gpu.Model, gpu.TotalVRAM, gpu.FreeVRAM, gpu.UsedVRAM, gpu.Vendor, gpu.PCIAddress, gpu.Index)
	}
}

func getLinuxCPUInfo(t *testing.T) types.CPUInfo {
	t.Helper()

	cores, err := cpu.Info()
	assert.NoError(t, err)
	var totalCompute float64
	for i := 0; i < len(cores); i++ {
		totalCompute += cores[i].Mhz
	}

	return types.CPUInfo{
		Compute:    totalCompute,
		NumCores:   uint64(len(cores)),
		MHzPerCore: cores[0].Mhz,
	}
}
