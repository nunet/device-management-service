//go:build darwin && amd64

package resources

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"gitlab.com/nunet/device-management-service/types"
)

// totalRamInMB fetches total memory installed on host machine
func totalRamInMB() uint64 {
	v, _ := mem.VirtualMemory()

	ramInMB := v.Total / 1024 / 1024

	return ramInMB
}

// totalCPUInMHz fetches compute capacity of the host machine
func totalCPUInMHz() float64 {
	var totalCompute float64
	cpus, _ := cpu.Info()
	for cpu := range cpus {
		totalCompute += float64(cpus[cpu].Cores) * cpus[cpu].Mhz
	}
	return totalCompute
}

// fetches the max clock speed of a single core
// Assuming all cores have the same clock speed
func Hz_per_cpu() float64 {
	cpu, _ := cpu.Info()
	return cpu[0].Mhz
}

// GetTotalProvisioned returns Provisioned struct with provisioned memory and CPU.
func GetTotalProvisioned() *types.Provisioned {
	var totalCores int32
	cpus, _ := cpu.Info()
	for cpu := range cpus {
		totalCores += cpus[cpu].Cores
	}

	provisioned := &types.Provisioned{
		CPU:      totalCPUInMHz(),
		Memory:   totalRamInMB(),
		NumCores: uint64(totalCores),
	}
	return provisioned
}
