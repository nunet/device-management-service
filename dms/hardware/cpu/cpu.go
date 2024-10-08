package cpu

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"

	"gitlab.com/nunet/device-management-service/types"
)

// GetUsage returns the CPU usage for the system
func GetUsage() (types.CPU, error) {
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU usage: %s", err)
	}

	cpuInfo, err := GetCPU()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %s", err)
	}

	usedCores := float64(cpuInfo.Cores) * cpuUsage[0] / 100
	cpuInfo.Cores = float32(usedCores)

	return cpuInfo, nil
}
