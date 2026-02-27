// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package gpu

import (
	"errors"
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"gitlab.com/nunet/device-management-service/types"
)

// nvidiaGPUConnector implements the types.GPUConnector interface for NVIDIA GPUs.
type nvidiaGPUConnector struct {
	deviceCache         []nvml.Device
	deviceCacheIndexMap map[string]int // UUID to deviceCache index map
}

var _ types.GPUConnector = (*nvidiaGPUConnector)(nil)

// newNVIDIAGPUConnector creates a new NVIDIA GPU Connector.
func newNVIDIAGPUConnector() (types.GPUConnector, error) {
	connector := &nvidiaGPUConnector{
		deviceCacheIndexMap: make(map[string]int),
	}

	if err := connector.initialise(); err != nil {
		return nil, fmt.Errorf("initialize NVIDIA GPU connector: %w", err)
	}

	if err := connector.loadDevices(); err != nil {
		return nil, fmt.Errorf("load NVIDIA GPU devices: %w", err)
	}

	return connector, nil
}

// getNVIDIADeviceCount returns the number of NVIDIA devices (GPUs).
func getNVIDIADeviceCount() (int, error) {
	deviceCount, ret := nvml.DeviceGetCount()
	if !errors.Is(ret, nvml.SUCCESS) {
		return 0, fmt.Errorf("get device count: %s", nvml.ErrorString(ret))
	}
	return deviceCount, nil
}

// getNVIDIADeviceHandle returns the handle for the NVIDIA device by its index.
func getNVIDIADeviceHandle(index int) (nvml.Device, error) {
	device, ret := nvml.DeviceGetHandleByIndex(index)
	if !errors.Is(ret, nvml.SUCCESS) {
		return nil, fmt.Errorf("get device handle for device %d: %s", index, nvml.ErrorString(ret))
	}
	return device, nil
}

// getNVIDIADeviceName returns the name of the NVIDIA device.
func getNVIDIADeviceName(device nvml.Device) (string, error) {
	name, ret := device.GetName()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("get name for device: %s", nvml.ErrorString(ret))
	}
	return name, nil
}

// getNVIDIADeviceMemory returns the memory information for the NVIDIA device.
func getNVIDIADeviceMemory(device nvml.Device) (nvml.Memory, error) {
	memory, ret := device.GetMemoryInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return nvml.Memory{}, fmt.Errorf("get NVIDIA GPU memory info: %s", nvml.ErrorString(ret))
	}
	return memory, nil
}

// getNVIDIADeviceUUID returns the UUID of the NVIDIA device.
func getNVIDIADeviceUUID(device nvml.Device) (string, error) {
	uuid, ret := device.GetUUID()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("get UUID for device: %s", nvml.ErrorString(ret))
	}
	return uuid, nil
}

// getNVIDIACoreCount returns the number of GPU cores (CUDA cores) for the NVIDIA device.
func getNVIDIACoreCount(device nvml.Device) (int, error) {
	cores, ret := device.GetNumGpuCores()
	if !errors.Is(ret, nvml.SUCCESS) {
		return 0, fmt.Errorf("get GPU core count: %s", nvml.ErrorString(ret))
	}
	return cores, nil
}

// getNVIDIAPCIAddress returns the PCI address for the NVIDIA device.
func getNVIDIAPCIAddress(device nvml.Device) (string, error) {
	pciInfo, ret := device.GetPciInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("get PCI info for device: %s", nvml.ErrorString(ret))
	}
	return convertBusID(pciInfo.BusId), nil
}

// initialise initialises the nvidia gpu connector
func (n *nvidiaGPUConnector) initialise() error {
	ret := nvml.Init()
	if !errors.Is(ret, nvml.SUCCESS) {
		return fmt.Errorf("initialize nvml: %s", nvml.ErrorString(ret))
	}

	return nil
}

// loadDevices loads the NVIDIA GPU devices.
func (n *nvidiaGPUConnector) loadDevices() error {
	deviceCount, err := getNVIDIADeviceCount()
	if err != nil {
		return err
	}

	// Iterate over each device
	for i := 0; i < deviceCount; i++ {
		device, err := getNVIDIADeviceHandle(i)
		if err != nil {
			return err
		}

		uuid, err := getNVIDIADeviceUUID(device)
		if err != nil {
			return err
		}

		n.deviceCache = append(n.deviceCache, device)
		n.deviceCacheIndexMap[uuid] = i
	}

	return nil
}

// GetGPUs returns the GPU information for NVIDIA GPUs.
func (n *nvidiaGPUConnector) GetGPUs() (types.GPUs, error) {
	if len(n.deviceCache) == 0 {
		return nil, nil
	}

	gpus := make(types.GPUs, 0, len(n.deviceCache))
	for _, device := range n.deviceCache {
		name, err := getNVIDIADeviceName(device)
		if err != nil {
			return nil, err
		}

		memory, err := getNVIDIADeviceMemory(device)
		if err != nil {
			return nil, err
		}

		pciAddress, err := getNVIDIAPCIAddress(device)
		if err != nil {
			return nil, err
		}

		uuid, err := getNVIDIADeviceUUID(device)
		if err != nil {
			return nil, err
		}

		var cores uint32
		coreCount, err := getNVIDIACoreCount(device)
		if err != nil {
			log.Debugw("could not get GPU core count, defaulting to 0",
				"error", err, "uuid", uuid)
		} else {
			cores = uint32(coreCount)
		}

		gpu := types.GPU{
			UUID:       uuid,
			PCIAddress: pciAddress,
			Model:      name,
			VRAM:       memory.Total,
			Cores:      cores,
			Vendor:     types.GPUVendorNvidia,
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// GetGPUUsage returns the GPU usage for the device with the given UUID.
func (n *nvidiaGPUConnector) GetGPUUsage(uuid string) (uint64, error) {
	deviceIndex, ok := n.deviceCacheIndexMap[uuid]
	if !ok {
		return 0, fmt.Errorf("nvidia device with UUID %s not found", uuid)
	}
	device := n.deviceCache[deviceIndex]

	memory, err := getNVIDIADeviceMemory(device)
	if err != nil {
		return 0, err
	}

	return memory.Used, nil
}

// Shutdown shuts down the NVIDIA Management Library.
func (n *nvidiaGPUConnector) Shutdown() error {
	ret := nvml.Shutdown()
	if !errors.Is(ret, nvml.SUCCESS) {
		return fmt.Errorf("shutdown nvml: %s", nvml.ErrorString(ret))
	}

	// clear the cache
	n.deviceCache = nil
	n.deviceCacheIndexMap = nil
	return nil
}
