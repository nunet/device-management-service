package resources

import (
	"testing"

	"github.com/shoenig/go-m1cpu"
	"github.com/stretchr/testify/require"
)

func TestArm64DarwinNewSystemSpecs(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs(newStore())
	require.NotNil(t, systemSpecs)
}

func TestArm64DarwinSystemSpecs_getDisk(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	diskInfo, err := getDisk()
	require.NoError(t, err)
	require.Greater(t, diskInfo.Size, uint64(0))
	// other fields as needed
}

func TestArm64DarwinSystemSpecs_getRam(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	ram, err := getRAM()
	require.NoError(t, err)
	require.Greater(t, ram.Size, uint64(0))
}

func TestArm64DarwinSystemSpecs_getCPU(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	cpuInfo, err := getCPU()
	require.NoError(t, err)
	require.Greater(t, cpuInfo.Cores, float32(0))
	require.Greater(t, cpuInfo.ClockSpeed, float64(0))
}

func TestArm64DarwinSystemSpecs_GetMachineResources(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs(newStore())
	resources, err := systemSpecs.GetMachineResources()
	require.NoError(t, err)
	require.Greater(t, resources.CPU.Cores, float32(0))
	require.Greater(t, resources.CPU.ClockSpeed, float64(0))
	require.Greater(t, resources.CPU.Compute, float64(0))
	require.Greater(t, resources.RAM.Size, uint64(0))
	require.Greater(t, resources.Disk.Size, uint64(0))
}

func TestArm64DarwinSystemSpecs_getGPUs(t *testing.T) {
	t.Parallel()
	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs(newStore())
	resources, err := systemSpecs.GetMachineResources()
	require.NoError(t, err)
	require.Empty(t, resources.GPUs) // GPUs are not supported on Darwin yet
}
