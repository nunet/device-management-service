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
func newSystemSpecs() *darwinSystemSpecs {
	return &darwinSystemSpecs{}
}

var _ SystemSpecs = (*darwinSystemSpecs)(nil)

// GetSpecInfo returns the detailed specifications of the system
// TODO: implement the method
// https://gitlab.com/nunet/device-management-service/-/issues/537
func (d darwinSystemSpecs) GetSpecInfo() (types.SpecInfo, error) {
	// TODO implement me
	panic("implement me")
}

// GetGPUVendors returns the GPU vendors for the system
// This function is not supported on Darwin systems
func (d darwinSystemSpecs) GetGPUVendors() ([]types.GPUVendor, error) {
	zlog.Warn("GPUs are not supported on Darwin yet")
	return []types.GPUVendor{}, nil
}

// GetGPUs returns the GPUs for the system
func (d darwinSystemSpecs) GetGPUs(vendor ...types.GPUVendor) ([]types.GPU, error) {
	zlog.Warn("GPUs are not supported on Darwin yet")
	return []types.GPU{}, nil
}

// GetTotalMemory returns the total memory available on the system
func (d darwinSystemSpecs) GetTotalMemory() (uint64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("failed to get total memory: %w", err)
	}

	ramInMB := vm.Total / 1024 / 1024
	return ramInMB, nil
}

// GetTotalStorage returns the total storage available on the system
func (d darwinSystemSpecs) GetTotalStorage() (uint64, error) {
	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return 0, fmt.Errorf("failed to get partitions: %w", err)
	}

	var totalStorage uint64
	for p := range partitions {
		usage, err := disk.UsageWithContext(context.Background(), partitions[p].Mountpoint)
		if err != nil {
			return 0, fmt.Errorf("failed to get disk usage: %w", err)
		}
		totalStorage += usage.Total
	}

	return totalStorage, nil
}

// GetCPUInfo returns the CPU information for the system
func (d darwinSystemSpecs) GetCPUInfo() (types.CPUInfo, error) {
	var (
		totalCompute float64
		totalCores   uint64
	)
	eCompute := float64(m1cpu.ECoreCount()) * m1cpu.ECoreGHz() * 1000
	pCompute := float64(m1cpu.PCoreCount()) * m1cpu.PCoreGHz() * 1000
	totalCompute = eCompute + pCompute
	totalCores = uint64(m1cpu.ECoreCount() + m1cpu.PCoreCount())

	cpuInfo := types.CPUInfo{
		NumCores:   totalCores,
		MHzPerCore: totalCompute / float64(m1cpu.ECoreCount()+m1cpu.PCoreCount()),
		Compute:    totalCompute,
	}
	return cpuInfo, nil
}

// GetProvisionedResources returns the total resources available on the system
func (d darwinSystemSpecs) GetProvisionedResources() (types.Resources, error) {
	cpuInfo, err := d.GetCPUInfo()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get CPU info: %w", err)
	}

	totalMemory, err := d.GetTotalMemory()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get total memory: %w", err)
	}

	totalStorage, err := d.GetTotalStorage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get total storage: %w", err)
	}

	totalResources := types.Resources{
		CPU:  cpuInfo.Compute,
		RAM:  totalMemory,
		Disk: totalStorage,
	}
	return totalResources, nil
}
