// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gpu

import (
	"sync"

	logging "github.com/ipfs/go-log/v2"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	gpuIndexCache map[string]int // UUID to gpuCache index map
	gpuCache      []types.GPU    // Cache of GPUs

	mu  sync.Mutex
	log = logging.Logger("hardware/gpu")
)

func copyCache() []types.GPU {
	gpuCopy := make([]types.GPU, len(gpuCache))
	copy(gpuCopy, gpuCache)
	return gpuCopy
}

// GetGPUs returns the GPUs in the system
func GetGPUs() ([]types.GPU, error) {
	// Check if the GPUs are cached
	if len(gpuCache) > 0 {
		return copyCache(), nil
	}

	mu.Lock()
	defer mu.Unlock()

	nvidiaGPUs, err := GetNVIDIAGPUs()
	if err != nil {
		log.Warnf("couldn't get NVIDIA GPUs: %v", err)
	}
	gpuCache = append(gpuCache, nvidiaGPUs...)

	amdGPUs, err := GetAMDGPUs()
	if err != nil {
		log.Warnf("couldn't get AMD GPUs: %v", err)
	}
	gpuCache = append(gpuCache, amdGPUs...)

	intelGPUs, err := GetIntelGPUs()
	if err != nil {
		log.Warnf("couldn't get Intel GPUs: %v", err)
	}
	gpuCache = append(gpuCache, intelGPUs...)

	// Assign index to GPUs
	// Note: The index is internal to dms and is not the same as the device index
	gpuCache = assignIndexToGPUs(gpuCache)

	// Cache the GPU index
	gpuIndexCache = make(map[string]int)
	for i, gpu := range gpuCache {
		gpuIndexCache[gpu.UUID] = i
	}

	return copyCache(), nil
}

// GetGPUUsage returns the GPU usage based on the specified uuid.
// if uuid is empty, it returns the usage of all GPUs.
func GetGPUUsage(uuid ...string) ([]types.GPU, error) {
	// Get the GPUs based on the UUID
	var gpus []types.GPU
	if len(uuid) == 0 {
		// copy the GPU cache
		gpus = make([]types.GPU, len(gpuCache))
		copy(gpus, gpuCache)
	} else {
		// Get the GPUs based on the UUID
		for _, u := range uuid {
			if index, ok := gpuIndexCache[u]; ok {
				gpus = append(gpus, gpuCache[index])
			}
		}
	}

	// Get the GPU usage
	for i, gpu := range gpus {
		switch gpu.Vendor {
		case types.GPUVendorNvidia:
			usage, err := GetNVIDIADeviceUsage(gpu.UUID)
			if err != nil {
				log.Warnf("get NVIDIA GPU usage: %v", err)
			}
			gpus[i].VRAM = usage
		case types.GPUVendorAMDATI:
			usage, err := GetAMDDeviceUsage(gpu.UUID)
			if err != nil {
				log.Warnf("get AMD GPU usage: %v", err)
			}
			gpus[i].VRAM = usage
		case types.GPUVendorIntel:
			usage, err := GetIntelDeviceUsage(gpu.UUID)
			if err != nil {
				log.Warnf("get Intel GPU usage: %v", err)
			}
			gpus[i].VRAM = usage
		}
	}

	return gpus, nil
}
