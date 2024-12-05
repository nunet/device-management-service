// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || amd)

package gpu

import (
	"errors"
	"fmt"
	"sync"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	nvidiaInitOnce sync.Once
	isNvidiaInit   bool

	nvidiaDeviceMap = make(map[string]nvml.Device) // UUID to Device map
)

// initNVML initializes the NVIDIA Management Library.
func initNVML() error {
	if isNvidiaInit {
		return nil
	}

	var err error
	nvidiaInitOnce.Do(func() {
		ret := nvml.Init()
		if !errors.Is(ret, nvml.SUCCESS) {
			err = fmt.Errorf("NVIDIA Management Library not installed, initialized, or configured (reboot recommended for newly installed NVIDIA GPU drivers): %s", nvml.ErrorString(ret))
			return
		}
		isNvidiaInit = true
	})
	return err
}

// shutdownNVML shuts down the NVIDIA Management Library.
func shutdownNVML() { //nolint // #790
	if !isNvidiaInit {
		return
	}
	_ = nvml.Shutdown()
}

// getNVIDIADeviceCount returns the number of NVIDIA devices (GPUs).
func getNVIDIADeviceCount() (int, error) {
	if !isNvidiaInit {
		return 0, errors.New("NVIDIA Management Library not initialized")
	}

	deviceCount, ret := nvml.DeviceGetCount()
	if !errors.Is(ret, nvml.SUCCESS) {
		return 0, fmt.Errorf("failed to get device count: %s", nvml.ErrorString(ret))
	}
	return deviceCount, nil
}

// getNVIDIADeviceHandle returns the handle for the NVIDIA device by its index.
func getNVIDIADeviceHandle(index int) (nvml.Device, error) {
	if !isNvidiaInit {
		return nil, errors.New("NVIDIA Management Library not initialized")
	}

	device, ret := nvml.DeviceGetHandleByIndex(index)
	if !errors.Is(ret, nvml.SUCCESS) {
		return nil, fmt.Errorf("failed to get device handle for device %d: %s", index, nvml.ErrorString(ret))
	}
	return device, nil
}

// getNVIDIADeviceName returns the name of the NVIDIA device.
func getNVIDIADeviceName(device nvml.Device) (string, error) {
	if !isNvidiaInit {
		return "", errors.New("NVIDIA Management Library not initialized")
	}

	name, ret := device.GetName()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("failed to get name for device: %s", nvml.ErrorString(ret))
	}
	return name, nil
}

// getNVIDIADeviceMemory returns the memory information for the NVIDIA device.
func getNVIDIADeviceMemory(device nvml.Device) (nvml.Memory, error) {
	if !isNvidiaInit {
		return nvml.Memory{}, errors.New("NVIDIA Management Library not initialized")
	}

	memory, ret := device.GetMemoryInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return nvml.Memory{}, fmt.Errorf("failed to get NVIDIA GPU memory info: %s", nvml.ErrorString(ret))
	}
	return memory, nil
}

// getNVIDIADeviceUUID returns the UUID of the NVIDIA device.
func getNVIDIADeviceUUID(device nvml.Device) (string, error) {
	if !isNvidiaInit {
		return "", errors.New("NVIDIA Management Library not initialized")
	}

	uuid, ret := device.GetUUID()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("failed to get UUID for device: %s", nvml.ErrorString(ret))
	}
	return uuid, nil
}

// getNVIDIAPCIAddress returns the PCI address for the NVIDIA device.
func getNVIDIAPCIAddress(device nvml.Device) (string, error) {
	pciInfo, ret := device.GetPciInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("failed to get PCI info for device: %s", nvml.ErrorString(ret))
	}
	return convertBusID(pciInfo.BusId), nil
}

// GetNVIDIAGPUs returns the GPU information for NVIDIA GPUs.
func GetNVIDIAGPUs() ([]types.GPU, error) {
	if err := initNVML(); err != nil {
		return nil, err
	}

	deviceCount, err := getNVIDIADeviceCount()
	if err != nil {
		return nil, err
	}

	var gpus []types.GPU
	// Iterate over each device
	for i := 0; i < deviceCount; i++ {
		device, err := getNVIDIADeviceHandle(i)
		if err != nil {
			return nil, err
		}

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

		gpu := types.GPU{
			UUID:       uuid,
			PCIAddress: pciAddress,
			Model:      name,
			VRAM:       float64(memory.Total),
			Vendor:     types.GPUVendorNvidia,
		}
		gpus = append(gpus, gpu)

		// Add the device to the device map
		nvidiaDeviceMap[uuid] = device
	}

	return gpus, nil
}

// GetNVIDIADeviceUsage returns the GPU usage for the device with the given UUID.
func GetNVIDIADeviceUsage(uuid string) (float64, error) {
	if err := initNVML(); err != nil {
		return 0, err
	}

	device, ok := nvidiaDeviceMap[uuid]
	if !ok {
		return 0, fmt.Errorf("nvidia device with UUID %s not found", uuid)
	}

	memory, err := getNVIDIADeviceMemory(device)
	if err != nil {
		return 0, err
	}

	return float64(memory.Used), nil
}
