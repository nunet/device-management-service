// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
	isXPUMInit   bool

	intelDeviceIDMap = make(map[string]int32) // UUID to DeviceID map
)

// initializeXPUM ensures that xpum.InitIntel() is called only once.
func initializeXPUM() error {
	if isXPUMInit {
		return nil
	}

	var xpumInitErr error
	xpumInitOnce.Do(func() {
		ok, err := xpum.InitIntel()
		if err != nil {
			xpumInitErr = fmt.Errorf("initialize Intel XPUM: %w", err)
			return
		}
		if !ok {
			xpumInitErr = fmt.Errorf("intel XPUM initialization was unsuccessful")
			return
		}
		isXPUMInit = true
	})
	return xpumInitErr
}

// getIntelTotalVRAM returns the total VRAM for the device with the given deviceID.
func getIntelTotalVRAM(deviceID int32) (float64, error) {
	if !isXPUMInit {
		return 0, fmt.Errorf("intel XPUM not initialized")
	}

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
}

// GetIntelGPUs returns the GPU information for Intel GPUs.
func GetIntelGPUs() ([]types.GPU, error) {
	if err := initializeXPUM(); err != nil {
		return nil, err
	}

	deviceList, err := xpum.GetDeviceList()
	if err != nil {
		return nil, fmt.Errorf("retrieve Intel GPU device list: %w", err)
	}

	gpus := make([]types.GPU, 0, len(deviceList))
	for _, device := range deviceList {
		vram, err := getIntelTotalVRAM(device.DeviceID)
		if err != nil {
			return nil, err
		}

		gpu := types.GPU{
			UUID:       device.UUID,
			Model:      device.DeviceName,
			VRAM:       vram, // VRAM in bytes
			Vendor:     types.GPUVendorIntel,
			PCIAddress: device.PCIBDFAddress,
		}
		gpus = append(gpus, gpu)

		// Add the device to the device map
		intelDeviceIDMap[device.UUID] = device.DeviceID
	}
	return gpus, nil
}

// GetIntelDeviceUsage returns the GPU usage for the device with the given UUID.
func GetIntelDeviceUsage(uuid string) (float64, error) {
	if !isXPUMInit {
		return 0, fmt.Errorf("intel XPUM not initialized")
	}

	deviceID, ok := intelDeviceIDMap[uuid]
	if !ok {
		return 0, fmt.Errorf("intel device with UUID %s not found", uuid)
	}

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
}
