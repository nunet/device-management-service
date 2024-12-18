// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllocateRandomSuccess tests successful random allocation of ports.
func TestAllocateRandomSuccess(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	allocatedPorts, err := allocator.AllocateRandom("alloc1", 5)
	expectedPorts := []int{8000, 8001, 8002, 8003, 8004}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

	allocatedPorts, err = allocator.AllocateRandom("alloc2", 3)
	expectedPorts = []int{8005, 8006, 8007}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")
}

// TestAllocateSpecificPorts tests allocation of specific ports
func TestAllocateSpecificPorts(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	// Test successful allocation
	err := allocator.AllocatePorts("alloc1", []int{8005, 8006, 8007})
	assert.NoError(t, err)

	// Verify allocation
	ports, err := allocator.GetAllocation("alloc1")
	assert.NoError(t, err)
	assert.Equal(t, []int{8005, 8006, 8007}, ports)

	// Test allocation of already reserved ports
	err = allocator.AllocatePorts("alloc2", []int{8004, 8006})
	assert.Error(t, err)

	// Test allocation outside range
	err = allocator.AllocatePorts("alloc3", []int{7999, 8000})
	assert.Error(t, err)
}

// TestAllocateRandomInsufficientPorts tests allocation when there are not enough ports available.
func TestAllocateRandomInsufficientPorts(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8005}
	allocator := NewPortAllocator(config)

	_, err := allocator.AllocateRandom("alloc1", 4)
	assert.NoError(t, err)

	_, err = allocator.AllocateRandom("alloc2", 3)
	assert.Error(t, err, "Expected an error due to insufficient ports")
}

// TestGetAllocationsSuccess tests retrieving the allocated ports by allocation ID.
func TestGetAllocationsSuccess(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	_, _ = allocator.AllocateRandom("alloc1", 5)

	allocatedPorts, err := allocator.GetAllocation("alloc1")
	expectedPorts := []int{8000, 8001, 8002, 8003, 8004}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Retrieved ports do not match expected")
}

// TestGetAllocationsNotFound tests retrieval with a non-existing allocation ID.
func TestGetAllocationsNotFound(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	_, err := allocator.GetAllocation("nonexistent")
	assert.Error(t, err, "Expected an error for a non-existent allocation ID")
}

// TestNestedAllocation tests the interaction between random and specific allocations
// It basically test a real scenario workflow of allocation and releases
func TestNestedAllocation(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8010}
	allocator := NewPortAllocator(config)

	// First random allocation
	randomPorts1, err := allocator.AllocateRandom("random1", 3)
	assert.NoError(t, err)
	assert.Equal(t, []int{8000, 8001, 8002}, randomPorts1)

	// Specific allocation should work with available ports
	err = allocator.AllocatePorts("specific1", []int{8005, 8006})
	assert.NoError(t, err)

	// Second random allocation should skip used ports
	randomPorts2, err := allocator.AllocateRandom("random2", 2)
	assert.NoError(t, err)
	assert.Equal(t, []int{8003, 8004}, randomPorts2)

	// Release first random allocation
	allocator.Release("random1")

	// Should be able to specifically allocate now-free ports
	err = allocator.AllocatePorts("specific2", []int{8000, 8001})
	assert.NoError(t, err)

	// Verify all allocations
	ports, err := allocator.GetAllocation("specific1")
	assert.NoError(t, err)
	assert.Equal(t, []int{8005, 8006}, ports)

	ports, err = allocator.GetAllocation("random2")
	assert.NoError(t, err)
	assert.Equal(t, []int{8003, 8004}, ports)

	ports, err = allocator.GetAllocation("specific2")
	assert.NoError(t, err)
	assert.Equal(t, []int{8000, 8001}, ports)

	// Try to allocate already reserved ports should fail
	err = allocator.AllocatePorts("specific3", []int{8000, 8007})
	assert.Error(t, err)

	// Release everything
	allocator.Release("specific1")
	allocator.Release("random2")
	allocator.Release("specific2")

	// Should be able to allocate the entire range again
	randomPorts3, err := allocator.AllocateRandom("random3", 11)
	assert.NoError(t, err)
	assert.Equal(t, []int{8000, 8001, 8002, 8003, 8004, 8005, 8006, 8007, 8008, 8009, 8010}, randomPorts3)
}

// TestAllocateExactPortRange tests allocation when the requested ports
// exactly match the available range.
func TestAllocateExactPortRange(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8004}
	allocator := NewPortAllocator(config)

	allocatedPorts, err := allocator.AllocateRandom("alloc1", 5)
	expectedPorts := []int{8000, 8001, 8002, 8003, 8004}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

	_, err = allocator.AllocateRandom("alloc2", 1)
	assert.Error(t, err, "Expected an error due to no available ports")
}
