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

	logging "github.com/ipfs/go-log/v2"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	// TODO: refactor cache usage while implementing #712 (comment from !682)
	gpuCache map[types.GPUVendor][]types.GPU
	mu       sync.Mutex
	log      = logging.Logger("hardware/gpu")
)

// GetGPUs returns the GPU information based on the specified vendors.
// If no vendors are provided, it returns the information of all the GPUs.
func GetGPUs(vendors ...types.GPUVendor) ([]types.GPU, error) {
	return getGPUsHelper(assignIndexToGPUs, map[types.GPUVendor]func() ([]types.GPU, error){
		types.GPUVendorIntel:  getIntelGPUs,
		types.GPUVendorNvidia: getNVIDIAGPUs,
		types.GPUVendorAMDATI: getAMDGPUs,
	}, false, vendors...)
}

// GetGPUUsage returns the GPU usage based on the specified vendors.
// If no vendors are provided, it returns the information of all the GPUs.
func GetGPUUsage(vendors ...types.GPUVendor) ([]types.GPU, error) {
	return getGPUsHelper(assignIndexToGPUs, map[types.GPUVendor]func() ([]types.GPU, error){
		types.GPUVendorIntel:  getIntelGPUUsage,
		types.GPUVendorNvidia: getNVIDIAGPUUsage,
		types.GPUVendorAMDATI: getAMDGPUUsage,
	}, false, vendors...)
}

// getGPUsHelper is a helper function to avoid code duplication in GetGPUs and GetGPUUsage.
// It fetches GPU information from different vendors and aggregates the results.
func getGPUsHelper(assignFunc func([]types.GPU) []types.GPU,
	fetchFuncs map[types.GPUVendor]func() ([]types.GPU, error),
	useCache bool,
	vendors ...types.GPUVendor,
) ([]types.GPU, error) {
	var gpus []types.GPU

	mu.Lock()
	defer mu.Unlock()

	if gpuCache == nil {
		gpuCache = make(map[types.GPUVendor][]types.GPU)
	}

	// Helper function to fetch and append GPU info for a vendor
	fetchAndAppendGPUs := func(fetchFunc func() ([]types.GPU, error), vendor types.GPUVendor) {
		gpuList, err := fetchFunc()
		if err != nil {
			log.Warnf("error fetching %v GPUs: %v", vendor, err)
			return
		}
		log.Infof("fetched %v GPUs: %v", vendor, gpuList)
		gpuCache[vendor] = gpuList
		gpus = append(gpus, gpuList...)
	}

	if len(vendors) == 0 {
		// No specific vendor requested, fetch all types of GPUs
		for vendor, fetchFunc := range fetchFuncs {
			if !useCache || len(gpuCache[vendor]) == 0 {
				fetchAndAppendGPUs(fetchFunc, vendor)
			} else {
				log.Infof("using cached %v GPUs", vendor)
				gpus = append(gpus, gpuCache[vendor]...)
			}
		}
	} else {
		// Fetch GPUs for the specified vendor only
		for _, vendor := range vendors {
			fetchFunc, ok := fetchFuncs[vendor]
			if !ok {
				return nil, fmt.Errorf("unsupported GPU vendor: %v", vendor)
			}
			if !useCache || len(gpuCache[vendor]) == 0 {
				fetchAndAppendGPUs(fetchFunc, vendor)
			} else {
				log.Infof("using cached %v GPUs", vendor)
				gpus = append(gpus, gpuCache[vendor]...)
			}
		}
	}

	// Assign index to GPUs and return
	// Note: The index is internal to dms and is not the same as the device index
	return assignFunc(gpus), nil
}
