package cpu

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"

	"gitlab.com/nunet/device-management-service/types"
)

// GetCPU returns the CPU information for the system
func GetCPU() (types.CPU, error) {
	cores, err := cpu.Info()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %s", err)
	}

	var totalCompute float64
	for i := 0; i < len(cores); i++ {
		totalCompute += cores[i].Mhz
	}

	return types.CPU{
		Cores:      float32(len(cores)),
		ClockSpeed: cores[0].Mhz * 1000000,
	}, nil
}
