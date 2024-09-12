package resources

import (
	"testing"

	"github.com/shoenig/go-m1cpu"
	"github.com/stretchr/testify/require"
)

func TestNewAmd64DarwinSystemSpecs(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs(newStore())
	require.NotNil(t, systemSpecs)
}

func TestAmd64DarwinSystemSpecs_GetTotalMemory(t *testing.T) {
	t.Parallel()
	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	ram, err := getRAM()
	require.NoError(t, err)
	require.Greater(t, ram.Size, uint64(0))
	// other fields as needed
}

func TestAmd64DarwinSystemSpecs_GetTotalStorage(t *testing.T) {
	t.Parallel()
	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	disk, err := getDisk()
	require.NoError(t, err)
	require.Greater(t, disk.Size, uint64(0))
	// other fields as needed
}

func TestAmd64DarwinSystemSpecs_GetCPUInfo(t *testing.T) {
	t.Parallel()
	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	cpuInfo, err := getCPU()
	require.NoError(t, err)

	require.Greater(t, cpuInfo.Cores, float32(0))
	require.Greater(t, cpuInfo.ClockSpeed, int64(0))
	require.Greater(t, cpuInfo.Compute, float64(0))
}

func TestAmd64DarwinSystemSpecs_GetMachineResources(t *testing.T) {
	t.Parallel()
	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs(newStore())
	resources, err := systemSpecs.GetMachineResources()
	require.NoError(t, err)
	require.Greater(t, resources.RAM.Size, uint64(0))
	require.Greater(t, resources.Disk.Size, uint64(0))
}

func TestAmd64DarwinSystemSpecs_GetGPUs(t *testing.T) {
	t.Parallel()
	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	gpus, err := getGPUs()
	require.NoError(t, err)
	require.Empty(t, gpus) // GPUs are not supported on Darwin yet
}
