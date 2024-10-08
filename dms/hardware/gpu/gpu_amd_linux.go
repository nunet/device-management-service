//go:build linux && amd64

package gpu

import (
	"fmt"
	"sync"

	"github.com/jaypipes/ghw"

	"gitlab.com/nunet/device-management-service/types"
)

var (
	metadata map[types.GPUVendor][]types.GPUMetadata
	mu       sync.Mutex
)

// fetchGPUMetadata returns the GPU metadata for all GPUs
// TODO: Use one single library to fetch GPU information or improve the match criteria
// https://gitlab.com/nunet/device-management-service/-/issues/548
// TODO: write tests by mocking the gpu snapshot
// https://gitlab.com/nunet/device-management-service/-/issues/534
func fetchGPUMetadata() (map[types.GPUVendor][]types.GPUMetadata, error) {
	if metadata != nil {
		return metadata, nil
	}

	mu.Lock()
	defer mu.Unlock()

	metadata = make(map[types.GPUVendor][]types.GPUMetadata)
	gpuInfo, err := ghw.GPU()
	if err != nil {
		return nil, err
	}

	for _, card := range gpuInfo.GraphicsCards {
		if card.DeviceInfo == nil {
			continue
		}
		pciAddress := card.Address
		vendor := types.ParseGPUVendor(card.DeviceInfo.Vendor.Name)
		metadata[vendor] = append(metadata[vendor], types.GPUMetadata{PCIAddress: pciAddress})
	}

	return metadata, nil
}

// GetGPUs returns the GPUs based on the specified vendors. If no vendors are provided, it returns the information of all the GPUs
func GetGPUs(vendors ...types.GPUVendor) ([]types.GPU, error) {
	return getGPUsHelper(fetchGPUMetadata, assignIndexToGPUs, map[types.GPUVendor]func(metadata []types.GPUMetadata) ([]types.GPU, error){
		types.GPUVendorIntel:  getIntelGPUs,
		types.GPUVendorNvidia: getNVIDIAGPUs,
		types.GPUVendorAMDATI: getAMDGPUs,
	}, vendors...)
}

// GetGPUUsage returns the GPU usage based on the specified vendors. If no vendors are provided, it returns the information of all the GPUs
func GetGPUUsage(vendors ...types.GPUVendor) ([]types.GPU, error) {
	return getGPUsHelper(fetchGPUMetadata, assignIndexToGPUs, map[types.GPUVendor]func(metadata []types.GPUMetadata) ([]types.GPU, error){
		types.GPUVendorIntel:  getIntelGPUUsage,
		types.GPUVendorNvidia: getNVIDIAGPUUsage,
		types.GPUVendorAMDATI: getAMDGPUUsage,
	}, vendors...)
}

// getGPUsHelper is a helper function to avoid code duplication in GetGPUs and GetGPUUsage
func getGPUsHelper(fetchMetadata func() (map[types.GPUVendor][]types.GPUMetadata, error), assignFunc func([]types.GPU) []types.GPU, fetchFuncs map[types.GPUVendor]func(metadata []types.GPUMetadata) ([]types.GPU, error), vendors ...types.GPUVendor) ([]types.GPU, error) {
	var gpus []types.GPU

	gpuMetadata, err := fetchMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GPU metadata: %v", err)
	}

	// Helper function to fetch and append GPU info
	fetchAndAppendGPUs := func(fetchFunc func(metadata []types.GPUMetadata) ([]types.GPU, error), vendor types.GPUVendor) {
		vendorMetadata, ok := gpuMetadata[vendor]
		if !ok {
			// TODO: log a warning here
			return
		}

		gpuList, err := fetchFunc(vendorMetadata)
		if err != nil {
			// TODO: log a warning here
			return
		}
		gpus = append(gpus, gpuList...)
	}

	if len(vendors) == 0 {
		// No specific vendor requested, fetch all types of GPUs
		for vendor, fetchFunc := range fetchFuncs {
			fetchAndAppendGPUs(fetchFunc, vendor)
		}
	} else {
		// Fetch GPUs for the specified vendor only
		for _, vendor := range vendors {
			fetchFunc, ok := fetchFuncs[vendor]
			if !ok {
				return nil, fmt.Errorf("unsupported GPU vendor: %v", vendor)
			}
			fetchAndAppendGPUs(fetchFunc, vendor)
		}
	}

	// Assign index to GPUs and return
	// Note: The index is internal to dms and is not the same as the device index
	return assignFunc(gpus), nil
}
