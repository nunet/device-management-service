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
	"fmt"
	"sync"

	goamdsmi "gitlab.com/nunet/device-management-service/lib/gpu/amdsmi"
	"gitlab.com/nunet/device-management-service/types"
)

// Initialize AMD SMI once using sync.Once
var (
	amdInitOnce sync.Once
	isAMDInit   bool

	amdProcessorMap = make(map[string]goamdsmi.ProcessorHandle) // UUID to processor map
)

// initializeAMD ensures that goamdsmi.Init() is called only once.
func initializeAMD() error {
	if isAMDInit {
		return nil
	}

	var initErr error
	amdInitOnce.Do(func() {
		status, err := goamdsmi.Init()
		if err != nil {
			initErr = fmt.Errorf("initialize AMD SMI: %w", err)
			return
		}

		if status.Code != goamdsmi.StatusSuccess {
			initErr = fmt.Errorf("AMD SMI initialization was unsuccessful: %w", status.Error())
			return
		}
		isAMDInit = true
	})
	return initErr
}

// GetAMDGPUs returns the GPU information for AMD GPUs.
func GetAMDGPUs() ([]types.GPU, error) {
	// Initialize AMD SMI
	if err := initializeAMD(); err != nil {
		return nil, err
	}

	// Retrieve socket handles
	sockets, ret := goamdsmi.GetSocketHandles()
	if ret.Code != goamdsmi.StatusSuccess {
		return nil, fmt.Errorf("get socket handles: %w", ret.Error())
	}

	var gpus []types.GPU

	// Iterate over each socket
	for _, socket := range sockets {
		// Retrieve processor handles for the current socket
		processors, ret := goamdsmi.GetProcessorHandles(socket)
		if ret.Code != goamdsmi.StatusSuccess {
			return nil, fmt.Errorf("get processor handles: %w", ret.Error())
		}

		// Iterate over each processor
		for _, processor := range processors {
			boardInfo, ret := goamdsmi.GetGPUBoardInfo(processor)
			if ret.Code != goamdsmi.StatusSuccess {
				return nil, fmt.Errorf("get board info: %w", ret.Error())
			}

			vRAM, ret := goamdsmi.GetGPUVRAM(processor)
			if ret.Code != goamdsmi.StatusSuccess {
				return nil, fmt.Errorf("get GPU VRAM: %w", ret.Error())
			}

			bdfID, ret := goamdsmi.GetGPUBDFID(processor)
			if ret.Code != goamdsmi.StatusSuccess {
				return nil, fmt.Errorf("get GPU BDFID: %w", ret.Error())
			}

			uuid, ret := goamdsmi.GetGPUUUID(processor)
			if ret.Code != goamdsmi.StatusSuccess {
				return nil, fmt.Errorf("failed to get GPU UUID: %w", ret.Error())
			}

			gpu := types.GPU{
				UUID:       uuid,
				Model:      boardInfo.ProductName,
				VRAM:       types.ConvertMibToBytes(float64(vRAM.Total)),
				Vendor:     types.GPUVendorAMDATI,
				PCIAddress: bdfIDToPCIAddress(bdfID),
			}
			gpus = append(gpus, gpu)

			// Add the processor to the processor map
			amdProcessorMap[uuid] = processor
		}
	}

	return gpus, nil
}

// GetAMDDeviceUsage returns the GPU usage for the device with the given UUID.
func GetAMDDeviceUsage(uuid string) (float64, error) {
	if !isAMDInit {
		return 0, fmt.Errorf("AMD SMI not initialized")
	}

	// Initialize AMD SMI
	if err := initializeAMD(); err != nil {
		return 0, err
	}

	processor, ok := amdProcessorMap[uuid]
	if !ok {
		return 0, fmt.Errorf("AMD device with UUID %s not found", uuid)
	}

	vram, ret := goamdsmi.GetGPUVRAM(processor)
	if ret.Code != goamdsmi.StatusSuccess {
		return 0, fmt.Errorf("get GPU usage: %w", ret.Error())
	}

	return types.ConvertMibToBytes(float64(vram.Used)), nil
}
