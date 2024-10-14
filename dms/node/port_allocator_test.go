package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllocatePortsSuccess tests successful allocation of ports.
func TestAllocatePortsSuccess(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	allocatedPorts, err := allocator.Allocate("alloc1", 5)
	expectedPorts := []int{8000, 8001, 8002, 8003, 8004}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

	allocatedPorts, err = allocator.Allocate("alloc2", 3)
	expectedPorts = []int{8005, 8006, 8007}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")
}

// TestAllocatePortsInsufficientPorts tests allocation when there are not enough ports available.
func TestAllocatePortsInsufficientPorts(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8005}
	allocator := NewPortAllocator(config)

	_, err := allocator.Allocate("alloc1", 4)
	assert.NoError(t, err)

	_, err = allocator.Allocate("alloc2", 3)
	assert.Error(t, err, "Expected an error due to insufficient ports")
}

// TestGetAllocationsSuccess tests retrieving the allocated ports by allocation ID.
func TestGetAllocationsSuccess(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
	allocator := NewPortAllocator(config)

	_, _ = allocator.Allocate("alloc1", 5)

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

// TestAllocateExactPortRange tests allocation when the requested ports exactly match the available range.
func TestAllocateExactPortRange(t *testing.T) {
	config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8004}
	allocator := NewPortAllocator(config)

	allocatedPorts, err := allocator.Allocate("alloc1", 5)
	expectedPorts := []int{8000, 8001, 8002, 8003, 8004}

	assert.NoError(t, err)
	assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

	_, err = allocator.Allocate("alloc2", 1)
	assert.Error(t, err, "Expected an error due to no available ports")
}
