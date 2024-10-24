// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && amd64

package gpu

import (
	"errors"
	"fmt"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"gitlab.com/nunet/device-management-service/types"
)

// initNVML initializes the NVIDIA Management Library.
func initNVML() error {
	ret := nvml.Init()
	if !errors.Is(ret, nvml.SUCCESS) {
		return fmt.Errorf("NVIDIA Management Library not installed, initialized, or configured (reboot recommended for newly installed NVIDIA GPU drivers): %s", nvml.ErrorString(ret))
	}
	return nil
}

// shutdownNVML shuts down the NVIDIA Management Library.
func shutdownNVML() {
	_ = nvml.Shutdown()
}

// getNVIDIADeviceCount returns the number of NVIDIA devices (GPUs).
func getNVIDIADeviceCount() (int, error) {
	deviceCount, ret := nvml.DeviceGetCount()
	if !errors.Is(ret, nvml.SUCCESS) {
		return 0, fmt.Errorf("failed to get device count: %s", nvml.ErrorString(ret))
	}
	return deviceCount, nil
}

// getNVIDIADeviceHandle returns the handle for the NVIDIA device by its index.
func getNVIDIADeviceHandle(index int) (nvml.Device, error) {
	device, ret := nvml.DeviceGetHandleByIndex(index)
	if !errors.Is(ret, nvml.SUCCESS) {
		return nil, fmt.Errorf("failed to get device handle for device %d: %s", index, nvml.ErrorString(ret))
	}
	return device, nil
}

// getNVIDIADeviceName returns the name of the NVIDIA device.
func getNVIDIADeviceName(device nvml.Device) (string, error) {
	name, ret := device.GetName()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("failed to get name for device: %s", nvml.ErrorString(ret))
	}
	return name, nil
}

// getNVIDIADeviceMemory returns the memory information for the NVIDIA device.
func getNVIDIADeviceMemory(device nvml.Device) (nvml.Memory, error) {
	memory, ret := device.GetMemoryInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return nvml.Memory{}, fmt.Errorf("failed to get NVIDIA GPU memory info: %s", nvml.ErrorString(ret))
	}
	return memory, nil
}

// convertBusID converts the BusId array to a correctly formatted PCI address string.
func convertBusID(busID [32]int8) string {
	busIDBytes := make([]byte, len(busID))
	for i, b := range busID {
		busIDBytes[i] = byte(b)
	}

	busIDStr := strings.TrimRight(string(busIDBytes), "\x00")

	// Check if the string starts with extra zero groups and correct the format
	if strings.HasPrefix(busIDStr, "00000000:") {
		// Trim it to the correct format: "0000:XX:YY.Z"
		return "0000" + busIDStr[8:]
	}

	return busIDStr
}

// getNVIDIAPCIAddress returns the PCI address for the NVIDIA device.
func getNVIDIAPCIAddress(device nvml.Device) (string, error) {
	pciInfo, ret := device.GetPciInfo()
	if !errors.Is(ret, nvml.SUCCESS) {
		return "", fmt.Errorf("failed to get PCI info for device: %s", nvml.ErrorString(ret))
	}
	return convertBusID(pciInfo.BusId), nil
}

// getNVIDIAGPUs returns the GPU information for NVIDIA GPUs.
func getNVIDIAGPUs() ([]types.GPU, error) {
	if err := initNVML(); err != nil {
		return nil, err
	}
	defer shutdownNVML()

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

		gpu := types.GPU{
			PCIAddress: pciAddress,
			Model:      name,
			VRAM:       float64(memory.Total),
			Vendor:     types.GPUVendorNvidia,
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// getNVIDIAGPUUsage returns the GPU usage for NVIDIA GPUs.
func getNVIDIAGPUUsage() ([]types.GPU, error) {
	if err := initNVML(); err != nil {
		return nil, err
	}
	defer shutdownNVML()

	deviceCount, err := getNVIDIADeviceCount()
	if err != nil {
		return nil, err
	}

	var gpus []types.GPU
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

		gpu := types.GPU{
			PCIAddress: pciAddress,
			Model:      name,
			VRAM:       float64(memory.Used),
			Vendor:     types.GPUVendorNvidia,
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}
