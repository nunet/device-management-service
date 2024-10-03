//go:build amd64 && darwin

package cpu

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"

	"gitlab.com/nunet/device-management-service/types"
)

// GetCPU returns the types.CPU information for the system
func GetCPU() (types.CPU, error) {
	cpus, err := cpu.Info()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %w", err)
	}

	var (
		totalCompute float64
		totalCores   uint64
	)
	for c := range cpus {
		totalCompute += float64(cpus[c].Cores) * cpus[c].Mhz
		totalCores += uint64(cpus[c].Cores)
	}

	cpuInfo := types.CPU{
		Cores:      float32(totalCores),
		ClockSpeed: float64(cpus[0].Mhz) * 1000000,
		Compute:    totalCompute,
	}
	return cpuInfo, nil
}
