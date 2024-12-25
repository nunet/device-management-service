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
	"testing"

	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/require"
)

func newMockGPUManager(t *testing.T, connectors gpuConnectors) gpuManager {
	t.Helper()

	return gpuManager{
		gpuCache:      []types.GPU{},
		gpuIndexCache: make(map[string]int),
		connectors:    connectors,
		initialised:   true,
	}
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

func TestGPU(t *testing.T) {
	t.Parallel()

	t.Run("must be able to return gpus", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockIntelGPUConnector := NewMockGPUConnector(ctrl)
		mockNVIDIAGPUConnector := NewMockGPUConnector(ctrl)
		mockAMDGPUConnector := NewMockGPUConnector(ctrl)
		mockGPUManager := newMockGPUManager(t, gpuConnectors{
			nvidia: mockNVIDIAGPUConnector,
			amd:    mockAMDGPUConnector,
			intel:  mockIntelGPUConnector,
		})

		nvidiaGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		amdGPUs := types.GPUs{
			{
				Index:  1,
				UUID:   "GPU-1",
				Model:  "Radeon VII",
				Vendor: types.GPUVendorAMDATI,
				VRAM:   100000,
			},
		}
		intelGPUs := types.GPUs{
			{
				Index:  2,
				UUID:   "GPU-2",
				Model:  "UHD Graphics 630",
				Vendor: types.GPUVendorIntel,
				VRAM:   100000,
			},
		}

		mockNVIDIAGPUConnector.EXPECT().GetGPUs().Return(nvidiaGPUs, nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUs().Return(amdGPUs, nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUs().Return(intelGPUs, nil).Times(1)
		gpus, err := mockGPUManager.GetGPUs()
		require.NoError(t, err)
		require.Len(t, gpus, 3)

		expectedGPUs := append(append(nvidiaGPUs, amdGPUs...), intelGPUs...)
		requireGPUsEqual(t, expectedGPUs, gpus)
	})

	t.Run("must be able to return gpu usage", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockIntelGPUConnector := NewMockGPUConnector(ctrl)
		mockNVIDIAGPUConnector := NewMockGPUConnector(ctrl)
		mockAMDGPUConnector := NewMockGPUConnector(ctrl)
		mockGPUManager := newMockGPUManager(t, gpuConnectors{
			nvidia: mockNVIDIAGPUConnector,
			amd:    mockAMDGPUConnector,
			intel:  mockIntelGPUConnector,
		})

		nvidiaGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		amdGPUs := types.GPUs{
			{
				Index:  1,
				UUID:   "GPU-1",
				Model:  "Radeon VII",
				Vendor: types.GPUVendorAMDATI,
				VRAM:   100000,
			},
		}
		intelGPUs := types.GPUs{
			{
				Index:  2,
				UUID:   "GPU-2",
				Model:  "UHD Graphics 630",
				Vendor: types.GPUVendorIntel,
				VRAM:   100000,
			},
		}

		mockNVIDIAGPUConnector.EXPECT().GetGPUs().Return(nvidiaGPUs, nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUs().Return(amdGPUs, nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUs().Return(intelGPUs, nil).Times(1)
		gpus, err := mockGPUManager.GetGPUs()
		require.NoError(t, err)
		require.Len(t, gpus, 3)

		expectedGPUs := append(append(nvidiaGPUs, amdGPUs...), intelGPUs...)
		requireGPUsEqual(t, expectedGPUs, gpus)

		mockNVIDIAGPUConnector.EXPECT().GetGPUUsage("GPU-0").Return(float64(5000), nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUUsage("GPU-1").Return(float64(5000), nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUUsage("GPU-2").Return(float64(5000), nil).Times(1)
		usage, err := mockGPUManager.GetGPUUsage("GPU-0", "GPU-1", "GPU-2")
		require.NoError(t, err)
		require.Len(t, usage, 3)

		require.Equal(t, 5000.0, usage[0].VRAM)
		require.Equal(t, 5000.0, usage[1].VRAM)
		require.Equal(t, 5000.0, usage[2].VRAM)
	})

	t.Run("must be able to shutdown the gpu manager", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockIntelGPUConnector := NewMockGPUConnector(ctrl)
		mockNVIDIAGPUConnector := NewMockGPUConnector(ctrl)
		mockAMDGPUConnector := NewMockGPUConnector(ctrl)
		mockGPUManager := newMockGPUManager(t, gpuConnectors{
			nvidia: mockNVIDIAGPUConnector,
			amd:    mockAMDGPUConnector,
			intel:  mockIntelGPUConnector,
		})

		// Populate the cache
		nvidiaGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		amdGPUs := types.GPUs{
			{
				Index:  1,
				UUID:   "GPU-1",
				Model:  "Radeon VII",
				Vendor: types.GPUVendorAMDATI,
				VRAM:   100000,
			},
		}
		intelGPUs := types.GPUs{
			{
				Index:  2,
				UUID:   "GPU-2",
				Model:  "UHD Graphics 630",
				Vendor: types.GPUVendorIntel,
				VRAM:   100000,
			},
		}

		mockNVIDIAGPUConnector.EXPECT().GetGPUs().Return(nvidiaGPUs, nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUs().Return(amdGPUs, nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUs().Return(intelGPUs, nil).Times(1)
		gpus, err := mockGPUManager.GetGPUs()
		require.NoError(t, err)
		require.Len(t, gpus, 3)

		// Shutdown the GPU manager
		mockNVIDIAGPUConnector.EXPECT().Shutdown().Return(nil).Times(1)
		mockAMDGPUConnector.EXPECT().Shutdown().Return(nil).Times(1)
		mockIntelGPUConnector.EXPECT().Shutdown().Return(nil).Times(1)
		err = mockGPUManager.Shutdown()
		require.NoError(t, err)

		// Ensure the cache is cleared
		require.Len(t, mockGPUManager.gpuCache, 0)
		require.Len(t, mockGPUManager.gpuIndexCache, 0)

		// Ensure the shutdown
		require.False(t, mockGPUManager.initialised)

		_, err = mockGPUManager.GetGPUs()
		require.ErrorContains(t, err, ErrNotInitialised.Error())

		_, err = mockGPUManager.GetGPUUsage()
		require.ErrorContains(t, err, ErrNotInitialised.Error())

		err = mockGPUManager.Shutdown()
		require.ErrorContains(t, err, ErrNotInitialised.Error())
	})

	t.Run("must return the gpu slice safely", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockIntelGPUConnector := NewMockGPUConnector(ctrl)
		mockNVIDIAGPUConnector := NewMockGPUConnector(ctrl)
		mockAMDGPUConnector := NewMockGPUConnector(ctrl)
		mockGPUManager := newMockGPUManager(t, gpuConnectors{
			nvidia: mockNVIDIAGPUConnector,
			amd:    mockAMDGPUConnector,
			intel:  mockIntelGPUConnector,
		})

		nvidiaGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		amdGPUs := types.GPUs{
			{
				Index:  1,
				UUID:   "GPU-1",
				Model:  "Radeon VII",
				Vendor: types.GPUVendorAMDATI,
				VRAM:   100000,
			},
		}
		intelGPUs := types.GPUs{
			{
				Index:  2,
				UUID:   "GPU-2",
				Model:  "UHD Graphics 630",
				Vendor: types.GPUVendorIntel,
				VRAM:   100000,
			},
		}

		mockNVIDIAGPUConnector.EXPECT().GetGPUs().Return(nvidiaGPUs, nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUs().Return(amdGPUs, nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUs().Return(intelGPUs, nil).Times(1)
		gpus, err := mockGPUManager.GetGPUs()
		require.NoError(t, err)
		require.Len(t, gpus, 3)

		expectedGPUs := append(append(nvidiaGPUs, amdGPUs...), intelGPUs...)
		requireGPUsEqual(t, expectedGPUs, gpus)

		// Modify the return value
		for i := range gpus {
			gpus[i].VRAM = 0
		}

		// The next call will use the cache and not call the GetGPUs method of the device managers
		gpus, err = mockGPUManager.GetGPUs()
		require.NoError(t, err)

		// Ensure it matches the original cache value
		requireGPUsEqual(t, expectedGPUs, gpus)
	})

	t.Run("must not fail on concurrent access", func(t *testing.T) {
		t.Parallel()

		const numGoRoutine = 100

		ctrl := gomock.NewController(t)
		mockIntelGPUConnector := NewMockGPUConnector(ctrl)
		mockNVIDIAGPUConnector := NewMockGPUConnector(ctrl)
		mockAMDGPUConnector := NewMockGPUConnector(ctrl)
		mockGPUManager := newMockGPUManager(t, gpuConnectors{
			nvidia: mockNVIDIAGPUConnector,
			amd:    mockAMDGPUConnector,
			intel:  mockIntelGPUConnector,
		})

		nvidiaGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		amdGPUs := types.GPUs{
			{
				Index:  1,
				UUID:   "GPU-1",
				Model:  "Radeon VII",
				Vendor: types.GPUVendorAMDATI,
				VRAM:   100000,
			},
		}
		intelGPUs := types.GPUs{
			{
				Index:  2,
				UUID:   "GPU-2",
				Model:  "UHD Graphics 630",
				Vendor: types.GPUVendorIntel,
				VRAM:   100000,
			},
		}

		mockNVIDIAGPUConnector.EXPECT().GetGPUs().Return(nvidiaGPUs, nil).Times(1)
		mockAMDGPUConnector.EXPECT().GetGPUs().Return(amdGPUs, nil).Times(1)
		mockIntelGPUConnector.EXPECT().GetGPUs().Return(intelGPUs, nil).Times(1)

		mockNVIDIAGPUConnector.EXPECT().GetGPUUsage("GPU-0").Return(float64(5000), nil).Times(numGoRoutine)
		mockAMDGPUConnector.EXPECT().GetGPUUsage("GPU-1").Return(float64(5000), nil).Times(numGoRoutine)
		mockIntelGPUConnector.EXPECT().GetGPUUsage("GPU-2").Return(float64(5000), nil).Times(numGoRoutine)

		var wg sync.WaitGroup
		wg.Add(numGoRoutine * 2)
		for i := 0; i < numGoRoutine; i++ {
			go func() {
				gpus, err := mockGPUManager.GetGPUs()
				require.NoError(t, err)
				require.Len(t, gpus, 3)
				wg.Done()
			}()

			go func() {
				usage, err := mockGPUManager.GetGPUUsage("GPU-0", "GPU-1", "GPU-2")
				require.NoError(t, err)
				require.Len(t, usage, 3)
				wg.Done()
			}()
		}
		wg.Wait()
	})
}
