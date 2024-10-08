package cpu

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCPU(t *testing.T) {
	cpu, err := GetCPU()
	require.NoError(t, err)
	require.Greater(t, cpu.Cores, float32(0))
	require.Greater(t, cpu.ClockSpeed, float64(0))
}

func TestGetCPUUsage(t *testing.T) {
	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	if isPipeline {
		t.Skip("Skipping test as the usage would be 0 in the pipeline")
	}
	cpu, err := GetUsage()
	require.NoError(t, err)
	require.Greater(t, cpu.Cores, float32(0))
	require.Greater(t, cpu.ClockSpeed, float64(0))
}
