// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/lib/hardware/cpu"
	"gitlab.com/nunet/device-management-service/lib/hardware/gpu"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// defaultHardwareManager manages the machine's hardware resources.
type defaultHardwareManager struct {
	gpuManager types.GPUManager
}

// NewHardwareManager creates a new instance of defaultHardwareManager.
func NewHardwareManager() types.HardwareManager {
	return &defaultHardwareManager{
		gpuManager: gpu.NewGPUManager(),
	}
}

var _ types.HardwareManager = (*defaultHardwareManager)(nil)

// GetMachineResources returns the resources of the machine in a thread-safe manner.
func (m *defaultHardwareManager) GetMachineResources() (types.MachineResources, error) {
	var err error
	var cpuDetails types.CPU
	var ram types.RAM
	var gpus []types.GPU
	var diskDetails types.Disk

	if cpuDetails, err = cpu.GetCPU(); err != nil {
		return types.MachineResources{}, fmt.Errorf("get CPU: %w", err)
	}

	if ram, err = GetRAM(); err != nil {
		return types.MachineResources{}, fmt.Errorf("get RAM: %w", err)
	}

	if gpus, err = m.gpuManager.GetGPUs(); err != nil {
		return types.MachineResources{}, fmt.Errorf("get GPUs: %w", err)
	}

	if diskDetails, err = GetDisk(); err != nil {
		return types.MachineResources{}, fmt.Errorf("get Disk: %w", err)
	}

	return types.MachineResources{
		Resources: types.Resources{
			CPU:  cpuDetails,
			RAM:  ram,
			Disk: diskDetails,
			GPUs: gpus,
		},
	}, nil
}

// GetUsage returns the usage of the machine.
func (m *defaultHardwareManager) GetUsage() (types.Resources, error) {
	cpuDetails, err := cpu.GetUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("get CPU usage: %w", err)
	}
	// Log CPU usage with "accounting" and "metric" labels
	log.Debugw("cpu_usage_computed",
		"labels", string(observability.LabelAccounting),
		"usage", cpuDetails)

	ram, err := GetRAMUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("get RAM usage: %w", err)
	}
	// Log RAM usage
	log.Debugw("ram_usage_computed",
		"labels", string(observability.LabelAccounting),
		"usedMemoryBytes", ram.Size)

	diskDetails, err := GetDiskUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("get Disk usage: %w", err)
	}
	// Log disk usage
	log.Debugw("disk_usage_computed",
		"labels", string(observability.LabelAccounting),
		"usedStorageBytes", diskDetails.Size)

	gpus, err := m.gpuManager.GetGPUUsage()
	if err != nil {
		return types.Resources{}, fmt.Errorf("get GPU usage: %w", err)
	}
	// Log GPU usage
	for _, gpuItem := range gpus {
		log.Debugw("gpu_usage_computed",
			"labels", string(observability.LabelAccounting),
			"gpuUUID", gpuItem.UUID,
			"vendor", gpuItem.Vendor,
			"usedVRAM", gpuItem.VRAM)
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
		return types.Resources{}, fmt.Errorf("get usage: %w", err)
	}

	availableResources, err := m.GetMachineResources()
	if err != nil {
		return types.Resources{}, fmt.Errorf("get machine resources: %w", err)
	}

	log.Debugf("system resource usage: %+v\nsystem resource available: %+v", usage, availableResources)

	if err := availableResources.Subtract(usage); err != nil {
		return types.Resources{}, fmt.Errorf("no free resources: %w", err)
	}

	return availableResources.Resources, nil
}

// CheckCapacity checks if the machine has enough resources to commit/allocate.
func (m *defaultHardwareManager) CheckCapacity(resources types.Resources) (bool, error) {
	// Check if there are enough free resources on the machine to allocate
	freeResources, err := m.GetFreeResources()
	if err != nil {
		return false, fmt.Errorf("get free resources: %w", err)
	}

	if err := freeResources.Subtract(resources); err != nil {
		return false, fmt.Errorf("no free resources on the machine: %w", err)
	}

	return true, nil
}

// Shutdown shuts down the hardware manager.
func (m *defaultHardwareManager) Shutdown() error {
	// shutdown gpu manager
	if err := m.gpuManager.Shutdown(); err != nil {
		return fmt.Errorf("shutdown gpu manager: %w", err)
	}

	return nil
}
