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
	"sync"

	goamdsmi "gitlab.com/nunet/device-management-service/lib/amdsmi"
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
		ok, err := goamdsmi.Init()
		if err != nil {
			initErr = fmt.Errorf("failed to initialize AMD SMI: %w", err)
			return
		}

		if !ok {
			initErr = fmt.Errorf("AMD SMI initialization was unsuccessful")
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
	sockets, err := goamdsmi.GetSocketHandles()
	if err != nil {
		return nil, fmt.Errorf("failed to get socket handles: %w", err)
	}

	var gpus []types.GPU

	// Iterate over each socket
	for _, socket := range sockets {
		// Retrieve processor handles for the current socket
		processors, err := goamdsmi.GetProcessorHandles(socket)
		if err != nil {
			return nil, fmt.Errorf("failed to get processor handles: %w", err)
		}

		// Iterate over each processor
		for _, processor := range processors {
			boardInfo, err := goamdsmi.GetGPUBoardInfo(processor)
			if err != nil {
				return nil, fmt.Errorf("failed to get board info: %w", err)
			}

			vRAM, err := goamdsmi.GetGPUVRAM(processor)
			if err != nil {
				return nil, fmt.Errorf("failed to get GPU VRAM: %w", err)
			}

			bdfID, err := goamdsmi.GetGPUBDFID(processor)
			if err != nil {
				return nil, fmt.Errorf("failed to get GPU BDFID: %w", err)
			}

			uuid, err := goamdsmi.GetGPUUUID(processor)
			if err != nil {
				return nil, fmt.Errorf("failed to get GPU UUID: %w", err)
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

	vram, err := goamdsmi.GetGPUVRAM(processor)
	if err != nil {
		return 0, fmt.Errorf("failed to get GPU usage: %w", err)
	}

	return types.ConvertMibToBytes(float64(vram.Used)), nil
}
