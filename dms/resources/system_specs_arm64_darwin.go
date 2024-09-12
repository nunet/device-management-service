//go:build darwin && arm64

package resources

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shoenig/go-m1cpu"
	"gitlab.com/nunet/device-management-service/types"
)

// darwinSystemSpecs is a struct that implements the SystemSpecs interface for Darwin arm64 systems
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
	vm, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %w", err)
	}

	return types.RAM{Size: vm.Total}, nil
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

	return types.Disk{Size: totalStorage}, nil
}

// getCPU returns the  types.CPU information for the system
func getCPU() (types.CPU, error) {
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

// GetMachineResources returns the total resources available on the system
func (d darwinSystemSpecs) GetMachineResources() (types.MachineResources, error) {
	cpuInfo, err := getCPU()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get CPU info: %w", err)
	}

	ram, err := getRAM()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get total memory: %w", err)
	}

	diskInfo, err := getDisk()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get total storage: %w", err)
	}

	return types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuInfo,
			RAM:  ram,
			Disk: diskInfo,
		},
	}, nil
}
