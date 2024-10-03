package hardware

import (
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/dms/hardware/cpu"
	"gitlab.com/nunet/device-management-service/dms/hardware/gpu"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	machineResources *types.MachineResources
	mu               sync.Mutex
)

// GetMachineResources returns the resources of the machine in a thread-safe manner.
func GetMachineResources() (types.MachineResources, error) {
	mu.Lock()
	defer mu.Unlock()

	if machineResources != nil {
		return *machineResources, nil
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

	machineResources = &types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuDetails,
			RAM:  ram,
			Disk: diskDetails,
			GPUs: gpus,
		},
	}

	return *machineResources, nil
}
