//go:build darwin && arm64

package library

import (
	"github.com/shirou/gopsutil/mem"
	"github.com/shoenig/go-m1cpu"
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
	ecompute := float64(m1cpu.ECoreCount()) * m1cpu.ECoreGHz() * 1000
	pcompute := float64(m1cpu.PCoreCount()) * m1cpu.PCoreGHz() * 1000
	totalCompute = ecompute + pcompute

	return totalCompute
}

// fetches the max clock speed of a single core
// Assuming all cores have the same clock speed
func Hz_per_cpu() float64 {

	return totalCPUInMHz() / float64(m1cpu.ECoreCount()+m1cpu.PCoreCount())
}

// GetTotalProvisioned returns Provisioned struct with provisioned memory and CPU.
func GetTotalProvisioned() *types.Provisioned {
	cores := m1cpu.ECoreCount() + m1cpu.PCoreCount()

	provisioned := &types.Provisioned{
		CPU:      totalCPUInMHz(),
		Memory:   totalRamInMB(),
		NumCores: uint64(cores),
	}
	return provisioned
}
