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
	"fmt"
	"strconv"

	"gitlab.com/nunet/device-management-service/lib/gpu/xpum"
	"gitlab.com/nunet/device-management-service/types"
)

// intelGPUConnector implements the types.GPUConnector interface for Intel GPUs.
type intelGPUConnector struct {
	deviceCache    []xpum.DeviceBasicInfo
	deviceIndexMap map[string]int32 // UUID to index of deviceCache map
}

var _ types.GPUConnector = (*intelGPUConnector)(nil)

// newIntelGPUConnector creates a new Intel GPU Connector.
func newIntelGPUConnector() (types.GPUConnector, error) {
	connector := &intelGPUConnector{
		deviceIndexMap: make(map[string]int32),
	}

	if err := connector.initialize(); err != nil {
		return nil, fmt.Errorf("initialize Intel GPU connector: %w", err)
	}

	if err := connector.loadDevices(); err != nil {
		return nil, fmt.Errorf("load Intel GPU devices: %w", err)
	}

	return connector, nil
}

// initialize loads the Intel XPUM library and initializes the connector by loading the devices.
func (i *intelGPUConnector) initialize() error {
	ret, err := xpum.Init()
	if err != nil {
		return fmt.Errorf("initialize Intel XPUM: %w", err)
	}
	if ret.Code != xpum.ResultOk {
		return fmt.Errorf("intel XPUM initialization was unsuccessful: %w", ret.Error())
	}

	return nil
}

// loadDevices loads the Intel GPU devices.
func (i *intelGPUConnector) loadDevices() error {
	deviceList, ret := xpum.GetDeviceList()
	if ret.Code != xpum.ResultOk {
		return fmt.Errorf("get Intel GPU device list: %w", ret.Error())
	}

	for _, device := range deviceList {
		i.deviceCache = append(i.deviceCache, device)
		i.deviceIndexMap[device.UUID] = device.DeviceID
	}
	return nil
}

// getTotalVRAM returns the total VRAM for the device with the given deviceID.
func (i *intelGPUConnector) getTotalVRAM(deviceID int32) (float64, error) {
	deviceProps, ret := xpum.GetDeviceProperties(deviceID)
	if ret.Code != xpum.ResultOk {
		return 0, fmt.Errorf("get properties for device %d: %w", deviceID, ret.Error())
	}

	for _, prop := range deviceProps {
		if prop.Name == xpum.DevicePropertyMemoryPhysicalSizeByte {
			totalMemory, err := strconv.ParseFloat(prop.Value, 64)
			if err != nil {
				return 0, fmt.Errorf("parse total memory for device %d: %w", deviceID, err)
			}
			return totalMemory, nil
		}
	}
	return 0, fmt.Errorf("total memory property not found for device %d", deviceID)
}

// GetGPUs returns the Intel GPUs.
func (i *intelGPUConnector) GetGPUs() (types.GPUs, error) {
	gpus := make(types.GPUs, 0, len(i.deviceCache))
	for _, device := range i.deviceCache {
		vram, err := i.getTotalVRAM(device.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("get total VRAM for intel device %d: %w", device.DeviceID, err)
		}

		gpu := types.GPU{
			UUID:       device.UUID,
			Model:      device.DeviceName,
			VRAM:       vram, // VRAM in bytes
			Vendor:     types.GPUVendorIntel,
			PCIAddress: device.PCIBDFAddress,
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// GetGPUUsage returns the GPU usage for the device with the given UUID.
func (i *intelGPUConnector) GetGPUUsage(uuid string) (float64, error) {
	deviceIndex, ok := i.deviceIndexMap[uuid]
	if !ok {
		return 0, fmt.Errorf("intel device with UUID %s not found", uuid)
	}
	device := i.deviceCache[deviceIndex]

	stats, err := xpum.GetDeviceStats(device.DeviceID, 0)
	if err != nil {
		return 0, fmt.Errorf("getting device stats for %d: %w", device.DeviceID, err)
	}

	for _, stat := range stats {
		for _, data := range stat.DataList {
			if data.MetricsType == xpum.StatsMemoryUsed {
				usedMemory := float64(data.Value)
				return usedMemory, nil
			}
		}
	}
	return 0, fmt.Errorf("used memory not found for device %d", device.DeviceID)
}

// Shutdown shuts down the Intel GPU connector.
func (i *intelGPUConnector) Shutdown() error {
	ret := xpum.Shutdown()
	if ret.Code != xpum.ResultOk {
		return fmt.Errorf("shutdown Intel XPUM: %w", ret.Error())
	}

	// Clear the cache
	i.deviceCache = nil
	i.deviceIndexMap = nil
	return nil
}
