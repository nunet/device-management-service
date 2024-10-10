package hardware

import (
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/dms/hardware/cpu"
	"gitlab.com/nunet/device-management-service/dms/hardware/gpu"
	"gitlab.com/nunet/device-management-service/types"
)

// defaultHardwareManager manages the machine's hardware resources.
type defaultHardwareManager struct {
	machineResources *types.MachineResources
	mu               sync.Mutex
}

// NewHardwareManager creates a new instance of defaultHardwareManager.
func NewHardwareManager() types.HardwareManager {
	return &defaultHardwareManager{}
}

var _ types.HardwareManager = (*defaultHardwareManager)(nil)

// GetMachineResources returns the resources of the machine in a thread-safe manner.
func (m *defaultHardwareManager) GetMachineResources() (types.MachineResources, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.machineResources != nil {
		return *m.machineResources, nil
	}

	var err error
	var cpuDetails types.CPU
	var ram types.RAM
	var gpus []types.GPU
	var diskDetails types.Disk

	if cpuDetails, err = cpu.GetCPU(); err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get CPU: %w", err)
	}

	if ram, err = GetRAM(); err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get RAM: %w", err)
	}

	if gpus, err = gpu.GetGPUs(); err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get GPUs: %w", err)
	}

	if diskDetails, err = GetDisk(); err != nil {
		return types.MachineResources{}, fmt.Errorf("failed to get Disk: %w", err)
	}

	m.machineResources = &types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuDetails,
			RAM:  ram,
			Disk: diskDetails,
			GPUs: gpus,
		},
	}

	return *m.machineResources, nil
}

// GetUsage returns the usage of the machine.
func (m *defaultHardwareManager) GetUsage() (types.Resources, error) {
	cpuDetails, err := cpu.GetUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	ram, err := GetRAMUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get RAM usage: %w", err)
	}

	diskDetails, err := GetDiskUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get Disk usage: %w", err)
	}

	gpus, err := gpu.GetGPUUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get GPU usage: %w", err)
	}

	return types.Resources{
		CPU:  cpuDetails,
		RAM:  ram,
		Disk: diskDetails,
		GPUs: gpus,
	}, nil
}

// GetFreeResources returns the free resources of the machine.
func (m *defaultHardwareManager) GetFreeResources() (types.Resources, error) {
	usage, err := m.GetUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get usage: %w", err)
	}

	availableResources, err := m.GetMachineResources()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to get machine resources: %w", err)
	}

	if err := availableResources.Subtract(usage); err != nil {
		return types.Resources{}, fmt.Errorf("no free resources: %w", err)
	}

	return availableResources.Resources, nil
}
