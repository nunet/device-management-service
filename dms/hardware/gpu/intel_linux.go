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
	"fmt"
	"strconv"
	"sync"

	"gitlab.com/nunet/device-management-service/lib/xpum"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	xpumInitOnce sync.Once
	xpumInitErr  error
)

// initializeXPUM ensures that xpum.InitIntel() is called only once.
func initializeXPUM() error {
	xpumInitOnce.Do(func() {
		ok, err := xpum.InitIntel()
		if err != nil {
			xpumInitErr = fmt.Errorf("initialize Intel XPUM: %w", err)
			return
		}
		if !ok {
			xpumInitErr = fmt.Errorf("intel XPUM initialization was unsuccessful")
		}
	})
	return xpumInitErr
}

// fetchIntelGPUs is a helper function that retrieves GPU information based on the vRAMSelector.
// The selector determines which VRAM value (e.g., total or used) to populate in the types.GPU struct.
func fetchIntelGPUs(vRAMSelector func(deviceID int32) (float64, error)) ([]types.GPU, error) {
	if err := initializeXPUM(); err != nil {
		return nil, err
	}

	deviceList, err := xpum.GetDeviceList()
	if err != nil {
		return nil, fmt.Errorf("retrieve Intel GPU device list: %w", err)
	}

	gpus := make([]types.GPU, 0, len(deviceList))
	for _, device := range deviceList {
		// Use the vRAMSelector to get the VRAM value (total or used))
		vram, err := vRAMSelector(device.DeviceID)
		if err != nil {
			return nil, err
		}

		gpu := types.GPU{
			Model:      device.DeviceName,
			VRAM:       vram, // VRAM in bytes
			Vendor:     types.GPUVendorIntel,
			PCIAddress: device.PCIBDFAddress,
		}
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

// getIntelGPUs returns the GPU information for Intel GPUs, specifically the total VRAM.
func getIntelGPUs() ([]types.GPU, error) {
	return fetchIntelGPUs(func(deviceID int32) (float64, error) {
		deviceProps, err := xpum.GetDeviceProperties(deviceID)
		if err != nil {
			return 0, fmt.Errorf("get properties for device %d: %w", deviceID, err)
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
	})
}

// getIntelGPUUsage returns the GPU usage for Intel GPUs, specifically the used VRAM.
func getIntelGPUUsage() ([]types.GPU, error) {
	return fetchIntelGPUs(func(deviceID int32) (float64, error) {
		stats, err := xpum.GetDeviceStats(deviceID, 0)
		if err != nil {
			return 0, fmt.Errorf("getting device stats for %d: %w", deviceID, err)
		}

		for _, stat := range stats {
			for _, data := range stat.DataList {
				if data.MetricsType == xpum.StatsMemoryUsed {
					usedMemory := float64(data.Value)
					return usedMemory, nil
				}
			}
		}
		return 0, fmt.Errorf("used memory not found for device %d", deviceID)
	})
}
