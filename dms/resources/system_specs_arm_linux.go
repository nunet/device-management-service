//go:build linux && (arm || arm64)

package resources

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"gitlab.com/nunet/device-management-service/types"
)

// linuxSystemSpecs implements the SystemSpecs interface for Linux systems
type linuxSystemSpecs struct {
	store *store
}

// newSystemSpecs returns a new instance of linuxSystemSpecs
func newSystemSpecs(store *store) *linuxSystemSpecs {
	return &linuxSystemSpecs{
		store: store,
	}
}

var _ types.SystemSpecs = (*linuxSystemSpecs)(nil)

// getRAM returns the types.RAM information for the system
func getRAM() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %s", err)
	}

	return types.RAM{
		Size: v.Total,
	}, nil
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

	return types.Disk{
		Size: totalStorage,
	}, nil
}

// GetCPU returns the CPU information for the system
func getCPU() (types.CPU, error) {
	cores, err := cpu.Info()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %s", err)
	}

	var totalCompute float64
	for i := 0; i < len(cores); i++ {
		totalCompute += cores[i].Mhz
	}

	return types.CPU{
		Compute:    totalCompute,
		Cores:      float32(len(cores)),
		ClockSpeed: cores[0].Mhz * 1000000,
	}, nil
}

// TODO: move the following functions to the `gpu` sub-package
// https://gitlab.com/nunet/device-management-service/-/issues/546
// assignIndexToGPUs assigns an index to each GPU in the list starting from 0
func assignIndexToGPUs(gpus []types.GPU) []types.GPU {
	for i := range gpus {
		gpus[i].Index = i
	}
	return gpus
}

func (l *linuxSystemSpecs) GetMachineResources() (types.MachineResources, error) {
	var (
		ok               bool
		machineResources types.MachineResources
	)
	l.store.withMachineResourcesRLock(func() {
		if l.store.machineResources != nil {
			machineResources = *l.store.machineResources
			ok = true
		}
	})
	if ok {
		return machineResources, nil
	}

	cpuDetails, err := getCPU()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get CPU: %s", err)
	}

	ram, err := getRAM()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get RAM: %s", err)
	}

	diskDetails, err := getDisk()
	if err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get DISK: %s", err)
	}

	machineResources = types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuDetails,
			RAM:  ram,
			Disk: diskDetails,
			GPUs: []types.GPU{},
		},
	}
	l.store.withMachineResourcesLock(func() {
		l.store.machineResources = &machineResources
	})
	// TODO: do we wanna store it in the db?
	return machineResources, nil
}
