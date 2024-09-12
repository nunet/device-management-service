package resources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinuxNewSystemSpecs(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs(newStore())
	require.NotNil(t, systemSpecs)
}

func TestLinuxSystemSpecs_getRam(t *testing.T) {
	t.Parallel()

	ram, err := getRAM()
	require.NoError(t, err)
	require.Greater(t, ram.Size, uint64(0))
	// other fields as needed
}

func TestLinuxSystemSpecs_getDisk(t *testing.T) {
	t.Parallel()

	diskInfo, err := getDisk()
	require.NoError(t, err)
	require.Greater(t, diskInfo.Size, uint64(0))
}

func TestLinuxSystemSpecs_getCPU(t *testing.T) {
	t.Parallel()

	cpuInfo, err := getCPU()
	require.NoError(t, err)

	require.Greater(t, cpuInfo.Cores, float32(0))
	require.Greater(t, cpuInfo.ClockSpeed, int64(0))
	require.Greater(t, cpuInfo.Compute, float64(0))
	// other fields as needed
}

func TestLinuxSystemSpecs_GetMachineResources(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs(newStore())
	resources, err := systemSpecs.GetMachineResources()
	require.NoError(t, err)

	require.Greater(t, resources.CPU.Cores, float32(0))
	require.Greater(t, resources.CPU.ClockSpeed, int64(0))
	require.Greater(t, resources.CPU.Compute, float64(0))
	require.Greater(t, resources.RAM.Size, uint64(0))
	require.Greater(t, resources.Disk.Size, uint64(0))
}

// TODO: we need to mock the gpu details so that the library can pick it up
// https://gitlab.com/nunet/device-management-service/-/issues/534
func TestLinuxSystemSpecs_getGPUs(t *testing.T) {
	t.Parallel()
	t.Skipf("Skipping test as it requires gpu snapshots")
}

func Test_PrettyPrintGPUs(t *testing.T) {
	t.Parallel()

	systemSpecs := newSystemSpecs(newStore())
	// get specific vendor GPUs by passing the vendor as an argument. Example:
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorNvidia)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorAMDATI)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorIntel)
	// gpus, err := systemSpecs.GetGPUs(models.GPUVendorNvidia, models.GPUVendorAMDATI)
	//
	// If no vendors are provided, it returns the information of all the GPUs
	resources, err := systemSpecs.GetMachineResources()
	require.NoError(t, err)

	fmt.Println("GPU Details:")
	for _, gpu := range resources.GPUs {
		fmt.Printf("Model: %s, Total VRAM: %d MB, Free VRAM: %d MB, Used VRAM: %d MB, Vendor: %s, PCI Address: %s, Index: %d\n",
			gpu.Model, gpu.TotalVRAM, gpu.FreeVRAM, gpu.UsedVRAM, gpu.Vendor, gpu.PCIAddress, gpu.Index)
	}
}
