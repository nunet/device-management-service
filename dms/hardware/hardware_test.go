// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware

import (
	"testing"

	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/require"
)

func newMockHardwareManager(gpuManager types.GPUManager) *defaultHardwareManager {
	return &defaultHardwareManager{
		gpuManager: gpuManager,
	}
}

func TestDefaultHardwareManager(t *testing.T) {
	t.Parallel()

	t.Run("must be able to get machine resources", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockGPUManager := NewMockGPUManager(ctrl)
		hm := newMockHardwareManager(mockGPUManager)

		mockGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		mockGPUManager.EXPECT().GetGPUs().Return(mockGPUs, nil).Times(1)
		machineResources, err := hm.GetMachineResources()
		require.NoError(t, err)
		require.NotZero(t, machineResources.CPU.Cores)
		require.NotZero(t, machineResources.CPU.ClockSpeed)
		require.NotZero(t, machineResources.RAM.Size)
		require.NotZero(t, machineResources.Disk.Size)
		require.Equal(t, mockGPUs, machineResources.GPUs)
	})

	t.Run("must be able to get free resources", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockGPUManager := NewMockGPUManager(ctrl)
		hm := newMockHardwareManager(mockGPUManager)

		mockGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		mockGPUManager.EXPECT().GetGPUs().Return(mockGPUs, nil).Times(1)
		mockGPUManager.EXPECT().GetGPUUsage().Return(mockGPUs, nil).Times(1)
		freeResources, err := hm.GetFreeResources()
		require.NoError(t, err)
		require.NotZero(t, freeResources.CPU.Cores)
		require.NotZero(t, freeResources.CPU.ClockSpeed)
		require.NotZero(t, freeResources.RAM.Size)
		require.NotZero(t, freeResources.Disk.Size)
		require.Equal(t, mockGPUs, freeResources.GPUs)
		require.Zero(t, freeResources.GPUs[0].VRAM)
	})

	t.Run("must be able to get usage", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockGPUManager := NewMockGPUManager(ctrl)
		hm := newMockHardwareManager(mockGPUManager)

		mockGPUs := types.GPUs{
			{
				Index:  0,
				UUID:   "GPU-0",
				Model:  "Tesla V100",
				Vendor: types.GPUVendorNvidia,
				VRAM:   100000,
			},
		}
		mockGPUManager.EXPECT().GetGPUUsage().Return(mockGPUs, nil).Times(1)
		usage, err := hm.GetUsage()
		require.NoError(t, err)
		require.NotZero(t, usage.CPU.Cores)
		require.NotZero(t, usage.CPU.ClockSpeed)
		require.NotZero(t, usage.RAM.Size)
		require.NotZero(t, usage.Disk.Size)
		require.Equal(t, mockGPUs, usage.GPUs)
	})
}
