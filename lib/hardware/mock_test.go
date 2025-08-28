// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/types"
)

type mockResourcesDeps struct {
	machineResources types.MachineResources
	freeResources    types.Resources
	usedResources    types.Resources
}

func setupMockResources(t *testing.T) mockResourcesDeps {
	t.Helper()

	return mockResourcesDeps{
		machineResources: types.MachineResources{
			Resources: types.Resources{
				CPU:  types.CPU{Cores: 8},
				RAM:  types.RAM{Size: 16},
				Disk: types.Disk{Size: 500},
				GPUs: types.GPUs{
					{Index: 0, VRAM: 8},
				},
			},
		},
		freeResources: types.Resources{
			CPU:  types.CPU{Cores: 4},
			RAM:  types.RAM{Size: 8},
			Disk: types.Disk{Size: 200},
			GPUs: types.GPUs{
				{Index: 0, VRAM: 4},
			},
		},
		usedResources: types.Resources{
			CPU:  types.CPU{Cores: 4},
			RAM:  types.RAM{Size: 8},
			Disk: types.Disk{Size: 300},
			GPUs: types.GPUs{
				{Index: 0, VRAM: 4},
			},
		},
	}
}

func TestMockHardwareManager(t *testing.T) {
	t.Parallel()
	deps := setupMockResources(t)
	manager := hardware.NewMockHardwareManager(
		deps.machineResources, deps.freeResources, deps.usedResources,
	)

	t.Run("GetMachineResources", func(t *testing.T) {
		t.Parallel()
		result, err := manager.GetMachineResources()
		assert.NoError(t, err)
		assert.Equal(t, deps.machineResources, result)
	})

	t.Run("GetFreeResources", func(t *testing.T) {
		t.Parallel()
		result, err := manager.GetFreeResources()
		assert.NoError(t, err)
		assert.Equal(t, deps.freeResources, result)
	})

	t.Run("GetUsage", func(t *testing.T) {
		t.Parallel()
		result, err := manager.GetUsage()
		assert.NoError(t, err)
		assert.Equal(t, deps.usedResources, result)
	})

	err := manager.Shutdown()
	assert.NoError(t, err)
}

func TestMockHardwareManagerCheckCapacity(t *testing.T) {
	t.Parallel()
	deps := setupMockResources(t)
	manager := hardware.NewMockHardwareManager(
		deps.machineResources, deps.freeResources, deps.usedResources,
	)

	// Test success case
	successResources := types.Resources{
		CPU:  types.CPU{Cores: 2},
		RAM:  types.RAM{Size: 4},
		Disk: types.Disk{Size: 100},
		GPUs: types.GPUs{
			{Index: 0, VRAM: 2},
		},
	}

	result, err := manager.CheckCapacity(successResources)
	assert.NoError(t, err)
	assert.True(t, result)

	// Test failure case
	failureResources := types.Resources{
		CPU:  types.CPU{Cores: 8}, // More than free
		RAM:  types.RAM{Size: 4},
		Disk: types.Disk{Size: 100},
		GPUs: types.GPUs{
			{Index: 0, VRAM: 2},
		},
	}

	result, err = manager.CheckCapacity(failureResources)
	assert.Error(t, err)
	assert.ErrorIs(t, err, types.ErrNoFreeResources)
	assert.False(t, result)

	err = manager.Shutdown()
	assert.NoError(t, err)
}
