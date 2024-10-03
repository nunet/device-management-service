package cpu

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCPU(t *testing.T) {
	cpu, err := GetCPU()
	require.NoError(t, err)
	require.Greater(t, cpu.Cores, float32(0))
	require.Greater(t, cpu.ClockSpeed, float64(0))
	require.Greater(t, cpu.Compute, float64(0))
}
