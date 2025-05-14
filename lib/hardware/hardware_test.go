// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/utils/convert"
)

const (
	minimalDisk = 2 // GiB (for both usage and total)
)

// TestGetMachineResources verifies machine resources are not 0
func TestGetMachineResources(t *testing.T) {
	t.Parallel()
	hwManager := NewHardwareManager()
	assert.NotNil(t, hwManager, "Hardware manager should not be nil")
	resources, err := hwManager.GetMachineResources()
	require.NoError(t, err, "GetMachineResources should not return an error")

	// Verify that CPU, RAM, and Disk resources are valid
	assert.Greater(t, resources.CPU.Cores, float32(0), "CPU cores should be greater than 0")

	// Check RAM against system page size in GiB
	require.NoError(t, err, "Failed to convert page size to GiB")
	assert.GreaterOrEqual(t, resources.RAM.Size, uint64(syscall.Getpagesize()), "RAM size should be greater or equal to page size in GiB")

	// Disk size should be at least minimalDisk GiB
	minimalDiskBytes, err := convert.ParseBytesWithDefaultUnit(minimalDisk, "GiB")
	require.NoError(t, err, "Failed to convert minimalDisk to bytes")
	assert.GreaterOrEqual(t, resources.Disk.Size, minimalDiskBytes, "Disk size should be at least minimalDisk GiB")

	err = hwManager.Shutdown()
	require.NoError(t, err, "Shutdown should not return an error")
}

// TestGetUsage checks if resource usage is reasonable
func TestGetUsage(t *testing.T) {
	t.Parallel()
	hwManager := NewHardwareManager()
	assert.NotNil(t, hwManager, "Hardware manager should not be nil")

	usage, err := hwManager.GetUsage()
	require.NoError(t, err, "GetUsage should not return an error")

	machineResources, err := hwManager.GetMachineResources()
	require.NoError(t, err, "GetMachineResources should not return an error")

	// Usage should meet minimal requirements
	assert.GreaterOrEqual(t, usage.RAM.Size, uint64(syscall.Getpagesize()), "RAM usage should be at least system page size in GiB")
	assert.Greater(t, usage.CPU.Cores, float32(0), "CPU usage should be greater than 0")

	minimalDiskBytes, err := convert.ParseBytesWithDefaultUnit(minimalDisk, "GiB")
	require.NoError(t, err, "Failed to convert minimalDisk to bytes")
	assert.GreaterOrEqual(t, usage.Disk.Size, minimalDiskBytes, "Disk usage should be at least minimalDisk GiB")

	// Usage should be less than or equal to total resources
	assert.LessOrEqual(t, usage.CPU.Cores, machineResources.CPU.Cores,
		"CPU usage should be less than or equal to total CPU")
	assert.LessOrEqual(t, usage.RAM.Size, machineResources.RAM.Size,
		"RAM usage should be less than or equal to total RAM")
	assert.LessOrEqual(t, usage.Disk.Size, machineResources.Disk.Size,
		"Disk usage should be less than or equal to total Disk")

	err = hwManager.Shutdown()
	require.NoError(t, err, "Shutdown should not return an error")
}

// TestGetFreeResources verifies that free resources are correctly calculated
//
// Note: don't run it in parallel because it might make
// resources 100% used
func TestGetFreeResources(t *testing.T) {
	hwManager := NewHardwareManager()
	assert.NotNil(t, hwManager, "Hardware manager should not be nil")

	freeResources, err := hwManager.GetFreeResources()
	require.NoError(t, err, "GetFreeResources should not return an error")

	// Verify free resources are reasonable
	// Disk and RAM are ALWAYS greater than 0 on powered-on machines
	// Also the number of cores
	assert.Greater(t, freeResources.RAM.Size, uint64(0),
		"Free RAM should be greater than 0")

	assert.Greater(t, freeResources.Disk.Size, uint64(0),
		"Free Disk should be greater than 0")

	assert.Greater(t, freeResources.CPU.Cores, float32(0),
		"Free CPU cores should be greater than 0")

	err = hwManager.Shutdown()
	require.NoError(t, err, "Shutdown should not return an error")
}

// TestCheckCapacity verifies capacity checking logic
//
// Note: don't run it in parallel because it might make
// resources 100% used
func TestCheckCapacity(t *testing.T) {
	hwManager := NewHardwareManager()
	assert.NotNil(t, hwManager, "Hardware manager should not be nil")

	freeResources, err := hwManager.GetFreeResources()
	require.NoError(t, err, "GetFreeResources should not return an error")

	// Test CheckCapacity with a half amount of resources (should succeed)
	smallResources := freeResources
	// TODO: not testing CPU cores until we have a better logic there
	// https://gitlab.com/nunet/device-management-service/-/issues/1051
	smallResources.CPU.Cores = 0
	smallResources.RAM.Size = freeResources.RAM.Size / 2
	smallResources.Disk.Size = freeResources.Disk.Size / 2

	hasCapacity, err := hwManager.CheckCapacity(smallResources)
	require.NoError(t, err, "CheckCapacity should not return an error for half resources")
	assert.True(t, hasCapacity, "Should have capacity for half resources")

	// Test with resources larger than available (should fail)
	largeResources := freeResources
	largeResources.RAM.Size = freeResources.RAM.Size + 1
	largeResources.Disk.Size = freeResources.Disk.Size + 1

	hasCapacity, err = hwManager.CheckCapacity(largeResources)
	require.Error(t, err, "CheckCapacity should return as there is not enough capacity")
	assert.False(t, hasCapacity, "Should not have capacity for resources larger than available")

	err = hwManager.Shutdown()
	require.NoError(t, err, "Shutdown should not return an error")
}

func TestTODOFixCPU(t *testing.T) {
	t.Parallel()
	t.Skip("CPU Cores Usage is kinda broken, so some assertions could be improved. See #1051")
}
