//go:build darwin && amd64

package resources

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"gitlab.com/nunet/device-management-service/types"
)

// darwinSystemSpecs is a struct that implements the SystemSpecs interface for Darwin amd64 systems
type darwinSystemSpecs struct{}

// newSystemSpecs returns a new instance of darwinSystemSpecs
func newSystemSpecs(_ *store) *darwinSystemSpecs {
	return &darwinSystemSpecs{}
}

var _ types.SystemSpecs = (*darwinSystemSpecs)(nil)

// getGPUs returns the GPUs for the system
func getGPUs(vendor ...types.GPUVendor) ([]types.GPU, error) {
	zlog.Warn("GPUs are not supported on Darwin yet")
	return []types.GPU{}, nil
}

// getRAM returns the types.RAM for the system
func getRAM() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("failed to get total memory: %w", err)
	}

	return types.RAM{Size: v.Total}, nil
}

// getDisk returns the types.Disk for the system
func getDisk() (types.Disk, error) {
	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return types.Disk{}, fmt.Errorf("failed to get partitions: %w", err)
	}

	var totalStorage uint64
	for p := range partitions {
		usage, err := disk.UsageWithContext(context.Background(), partitions[p].Mountpoint)
		if err != nil {
			return types.Disk{}, fmt.Errorf("failed to get disk usage: %w", err)
		}
		totalStorage += usage.Total
	}

	return totalStorage, nil
}

// getCPU returns the types.CPU information for the system
func getCPU() (types.CPU, error) {
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
		ClockSpeed: int64(cpus[0].Mhz) * 1000000,
		Compute:    totalCompute,
	}
	return cpuInfo, nil
}

// GetMachineResources returns the total resources available on the system
func (d darwinSystemSpecs) GetMachineResources() (types.MachineResources, error) {
	cpuInfo, err := d.GetCPU()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get CPU info: %w", err)
	}

	ram, err := getRAM()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get total memory: %w", err)
	}

	diskInfo, err := getDisk()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get total storage: %w", err)
	}

	resources := types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuInfo,
			RAM:  ram,
			Disk: diskInfo,
		},
	}
	return totalResources, nil
}
