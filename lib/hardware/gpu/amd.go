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

	goamdsmi "gitlab.com/nunet/device-management-service/lib/hardware/gpu/amdsmi"
	"gitlab.com/nunet/device-management-service/types"
)

// amdGPUConnector implements the types.GPUConnector interface for AMD GPUs.
type amdGPUConnector struct {
	processorIndexCache map[string]int // UUID to index map
	processorCache      []goamdsmi.ProcessorHandle
}

var _ types.GPUConnector = (*amdGPUConnector)(nil)

// newAMDGPUConnector returns a new AMD GPU Connector.
func newAMDGPUConnector() (types.GPUConnector, error) {
	connector := &amdGPUConnector{
		processorIndexCache: make(map[string]int),
	}

	if err := connector.initialize(); err != nil {
		return nil, fmt.Errorf("initialize AMD GPU connector: %w", err)
	}

	if err := connector.loadProcessorHandles(); err != nil {
		return nil, fmt.Errorf("load processor handles: %w", err)
	}

	return connector, nil
}

// initialize initializes the AMD SMI library and loads the processors.
func (a *amdGPUConnector) initialize() error {
	status, err := goamdsmi.Init()
	if err != nil {
		return err
	}

	if status.Code != goamdsmi.StatusSuccess {
		return fmt.Errorf("AMD SMI initialization was unsuccessful: %w", status.Error())
	}

	return nil
}

// loadProcessorHandles loads the processor handles for the AMD GPUs.
func (a *amdGPUConnector) loadProcessorHandles() error {
	// Retrieve socket handles
	sockets, ret := goamdsmi.GetSocketHandles()
	if ret.Code != goamdsmi.StatusSuccess {
		return fmt.Errorf("get socket handles: %w", ret.Error())
	}

	// Iterate over each socket
	for _, socket := range sockets {
		// Retrieve processor handles for the current socket
		processors, ret := goamdsmi.GetProcessorHandles(socket)
		if ret.Code != goamdsmi.StatusSuccess {
			return fmt.Errorf("get processor handles: %w", ret.Error())
		}

		// Iterate over each processor
		for i, processor := range processors {
			uuid, ret := goamdsmi.GetGPUUUID(processor)
			if ret.Code != goamdsmi.StatusSuccess {
				return fmt.Errorf("get GPU UUID: %w", ret.Error())
			}

			// Add the processor to the processor map
			a.processorIndexCache[uuid] = i
			a.processorCache = append(a.processorCache, processor)
		}
	}

	return nil
}

// GetGPUs returns the AMD GPUs in the system.
func (a *amdGPUConnector) GetGPUs() (types.GPUs, error) {
	gpus := make(types.GPUs, len(a.processorCache))
	for _, processor := range a.processorCache {
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
			return nil, fmt.Errorf("get GPU UUID: %w", ret.Error())
		}

		// Get compute unit count; default to 0 if unavailable
		var cores uint32
		asicInfo, ret := goamdsmi.GetGPUASICInfo(processor)
		if ret.Code == goamdsmi.StatusSuccess && asicInfo.NumComputeUnits != 0xFFFFFFFF {
			cores = asicInfo.NumComputeUnits
		}

		gpu := types.GPU{
			UUID:       uuid,
			Model:      boardInfo.ProductName,
			VRAM:       types.ConvertMibToBytes(uint64(vRAM.Total)),
			Cores:      cores,
			Vendor:     types.GPUVendorAMDATI,
			PCIAddress: bdfIDToPCIAddress(bdfID),
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// GetGPUUsage returns the GPU usage for the device with the given UUID.
func (a *amdGPUConnector) GetGPUUsage(uuid string) (uint64, error) {
	if len(uuid) == 0 {
		return 0, fmt.Errorf("no UUID provided")
	}

	processorIndex, ok := a.processorIndexCache[uuid]
	if !ok {
		return 0, fmt.Errorf("AMD gpu device with UUID %s not found", uuid)
	}

	processor := a.processorCache[processorIndex]
	vram, ret := goamdsmi.GetGPUVRAM(processor)
	if ret.Code != goamdsmi.StatusSuccess {
		return 0, fmt.Errorf("get GPU usage: %w", ret.Error())
	}

	return types.ConvertMibToBytes(uint64(vram.Used)), nil
}

// Shutdown shuts down the AMD SMI library.
func (a *amdGPUConnector) Shutdown() error {
	ret := goamdsmi.Shutdown()
	if ret.Code != goamdsmi.StatusSuccess {
		return fmt.Errorf("shutdown amdsmi: %w", ret.Error())
	}

	// clear the cache
	a.processorCache = nil
	a.processorIndexCache = nil
	return nil
}
