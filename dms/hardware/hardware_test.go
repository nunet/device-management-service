package hardware

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultHardwareManager_GetMachineResources(t *testing.T) {
	t.Parallel()

	hm := NewHardwareManager()
	machineResources, err := hm.GetMachineResources()
	require.NoError(t, err)
	require.NotZero(t, machineResources.CPU.Cores)
	require.NotZero(t, machineResources.CPU.ClockSpeed)
	require.NotZero(t, machineResources.RAM.Size)
	require.NotZero(t, machineResources.Disk.Size)
}

func TestDefaultHardwareManager_GetFreeResources(t *testing.T) {
	t.Parallel()
	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	if isPipeline {
		t.Skip("Skipping test as the usage would be 0 in the pipeline")
	}

	hm := NewHardwareManager()
	freeResources, err := hm.GetFreeResources()
	require.NoError(t, err)
	require.NotZero(t, freeResources.CPU.Cores)
	require.NotZero(t, freeResources.CPU.ClockSpeed)
	require.NotZero(t, freeResources.RAM.Size)
	require.NotZero(t, freeResources.Disk.Size)
}

func TestDefaultHardwareManager_GetUsage(t *testing.T) {
	t.Parallel()

	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	if isPipeline {
		t.Skip("Skipping test as the usage would be 0 in the pipeline")
	}

	hm := NewHardwareManager()
	usage, err := hm.GetUsage()
	require.NoError(t, err)
	require.NotZero(t, usage.CPU.Cores)
	require.NotZero(t, usage.CPU.ClockSpeed)
	require.NotZero(t, usage.RAM.Size)
	require.NotZero(t, usage.Disk.Size)
}
