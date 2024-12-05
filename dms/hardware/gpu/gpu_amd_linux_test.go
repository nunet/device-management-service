// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gpu

import (
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/require"
)

// We cannot run the tests in parallel as we are using a global cache for the GPUs

func TestGPU(t *testing.T) {
	t.Run("must return the GPU information if the GPU is available", func(t *testing.T) {
		gpus, err := GetGPUs()
		require.NoError(t, err)

		if len(gpus) == 0 {
			t.Skip("No GPUs available")
		}

		for i, gpu := range gpus {
			usage, err := GetGPUUsage(gpu.UUID)
			require.NoError(t, err)

			require.Equal(t, gpu.UUID, usage[i].UUID)
			require.Equal(t, gpu.Model, usage[i].Model)
			require.Equal(t, gpu.Vendor, usage[i].Vendor)
			require.Equal(t, gpu.Index, usage[i].Index)
			require.Equal(t, gpu.PCIAddress, usage[i].PCIAddress)

			// The usage should be less than the total VRAM
			require.Greater(t, gpu.VRAM, usage[i].VRAM)
		}
	})
}

func TestGPUCache(t *testing.T) {
	// Initialize the cache
	// If the cache is set, the function must return the cached GPUs
	gpuCache = []types.GPU{
		{
			UUID:       "123",
			Model:      "model",
			Vendor:     types.GPUVendorNvidia,
			Index:      1,
			PCIAddress: "pci",
			VRAM:       1024,
		},
		{
			UUID:       "456",
			Model:      "model",
			Vendor:     types.GPUVendorAMDATI,
			Index:      2,
			PCIAddress: "pci",
			VRAM:       1024,
		},
	}
	gpuIndexCache = map[string]int{
		"123": 0,
		"456": 1,
	}

	gpus, err := GetGPUs()
	require.NoError(t, err)
	// Ensure the gpus are the same as the cached gpus
	requireGPUsEqual(t, gpuCache, gpus)

	// Copy the cache
	cacheCopy := make([]types.GPU, len(gpuCache))
	copy(cacheCopy, gpuCache)

	// Ensure gpus are returned safely
	//
	// We change every field of the gpus to ensure that the function returns a copy of the gpus
	for i := range gpus {
		gpus[i].UUID = "uuid"
		gpus[i].Model = "model"
		gpus[i].Vendor = types.GPUVendorIntel
		gpus[i].Index = 3
		gpus[i].PCIAddress = "pci"
		gpus[i].VRAM = 2048
	}

	// get the gpus again
	newGpus, err := GetGPUs()
	require.NoError(t, err)
	// ensure the gpuCache is not changed and the gpus are the same as before
	requireGPUsEqual(t, cacheCopy, newGpus)
}

func requireGPUsEqual(t *testing.T, expected, actual []types.GPU) {
	t.Helper()

	require.Len(t, actual, len(expected))

	// Create a map of the expected GPUs
	expectedMap := make(map[string]types.GPU)
	for _, gpu := range expected {
		expectedMap[gpu.UUID] = gpu
	}

	// Compare the actual GPUs with the expected GPUs
	counter := 0
	for _, gpu := range actual {
		expectedGPU, ok := expectedMap[gpu.UUID]
		if ok {
			counter++
			require.Equal(t, expectedGPU.Index, gpu.Index)
			require.Equal(t, expectedGPU.Model, gpu.Model)
			require.Equal(t, expectedGPU.PCIAddress, gpu.PCIAddress)
			require.Equal(t, expectedGPU.UUID, gpu.UUID)
			require.Equal(t, expectedGPU.VRAM, gpu.VRAM)
			require.Equal(t, expectedGPU.Vendor, gpu.Vendor)
		}
	}

	// Ensure we've traversed all the expected GPUs
	require.Equal(t, counter, len(expected))
}
