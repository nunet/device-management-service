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
	"sync"

	goamdsmi "gitlab.com/nunet/device-management-service/lib/amdsmi"
	"gitlab.com/nunet/device-management-service/types"
)

// Initialize AMD SMI once using sync.Once
var (
	initOnce sync.Once
	initErr  error
)

// initializeAMD ensures that goamdsmi.Init() is called only once.
func initializeAMD() error {
	initOnce.Do(func() {
		ok, err := goamdsmi.Init()
		if err != nil {
			initErr = fmt.Errorf("failed to initialize AMD SMI: %w", err)
			return
		}

		if !ok {
			initErr = fmt.Errorf("AMD SMI initialization was unsuccessful")
			return
		}
	})
	return initErr
}

// fetchAMDGPUs is a helper function that retrieves GPU information based on the VRAM selector.
// The selector determines which VRAM field (Total or Used) to populate in the types.GPU struct.
func fetchAMDGPUs(vRAMSelector func(vRAM goamdsmi.VRAM) float64) ([]types.GPU, error) {
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

			gpu := types.GPU{
				Model:      boardInfo.ProductName,
				VRAM:       vRAMSelector(vRAM),
				Vendor:     types.GPUVendorAMDATI,
				PCIAddress: bdfIDToPCIAddress(bdfID),
			}
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// getAMDGPUs returns the GPU information for AMD GPUs, specifically the total VRAM.
func getAMDGPUs() ([]types.GPU, error) {
	return fetchAMDGPUs(func(vRAM goamdsmi.VRAM) float64 {
		return types.ConvertMibToBytes(float64(vRAM.Total))
	})
}

// getAMDGPUUsage returns the GPU usage for AMD GPUs, specifically the used VRAM.
func getAMDGPUUsage() ([]types.GPU, error) {
	return fetchAMDGPUs(func(vRAM goamdsmi.VRAM) float64 {
		return types.ConvertMibToBytes(float64(vRAM.Used))
	})
}

// bdfIDToPCIAddress converts a 64-bit BDFID to a standard PCI address string.
// The PCI address format is 'domain:bus:device.function'.
//
// Taken from the AMD SMI library:
// BDFID = ((DOMAIN & 0xffffffff) << 32) | ((BUS & 0xff) << 8) |
// ((DEVICE & 0x1f) <<3 ) | (FUNCTION & 0x7)
//
// | Name     | Field   |
// ---------- | ------- |
// | Domain   | [64:32] |
// | Reserved | [31:16] |
// | Bus      | [15: 8] |
// | Device   | [ 7: 3] |
// | Function | [ 2: 0] |
func bdfIDToPCIAddress(bdfID uint64) string {
	// Extract Domain: Bits [63:32]
	domain := (bdfID >> 32) & 0xFFFFFFFF

	// Extract Bus: Bits [15:8]
	bus := (bdfID >> 8) & 0xFF

	// Extract Device: Bits [7:3]
	device := (bdfID >> 3) & 0x1F

	// Extract Function: Bits [2:0]
	function := bdfID & 0x7

	// Format each component into hexadecimal with appropriate padding
	// Domain: 4 hex digits, Bus: 2 hex digits, Device: 2 hex digits, Function: 1 hex digit
	return fmt.Sprintf("%04X:%02X:%02X.%X", domain, bus, device, function)
}
