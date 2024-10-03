//go:build darwin && arm64

package cpu

import (
	"github.com/shoenig/go-m1cpu"
	"gitlab.com/nunet/device-management-service/types"
)

// GetCPU returns the  types.CPU information for the system
func GetCPU() (types.CPU, error) {
	var (
		totalCompute float64
		totalCores   uint64
	)
	eCompute := float64(m1cpu.ECoreCount()) * m1cpu.ECoreGHz() * 1000000
	pCompute := float64(m1cpu.PCoreCount()) * m1cpu.PCoreGHz() * 1000000
	totalCompute = eCompute + pCompute
	totalCores = uint64(m1cpu.ECoreCount() + m1cpu.PCoreCount())

	cpuInfo := types.CPU{
		Cores:      float32(totalCores),
		ClockSpeed: totalCompute / float64(m1cpu.ECoreCount()+m1cpu.PCoreCount()),
		Compute:    totalCompute,
	}
	return cpuInfo, nil
}
