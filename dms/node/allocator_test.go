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
	"github.com/stretchr/testify/require"
)

func TestPortAllocator(t *testing.T) {
	t.Parallel()

	testAllocation1 := "alloc1"
	testAllocation2 := "alloc2"

	t.Run("must be able to allocate random ports in range", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 3000, AvailableRangeTo: 3100}
		testPortAllocator := newPortAllocator(config)

		allocatedPorts, err := testPortAllocator.AllocateRandom(testAllocation1, 5)
		expectedPorts := []int{3000, 3001, 3002, 3003, 3004}

		require.NoError(t, err)
		require.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

		allocatedPorts, err = testPortAllocator.AllocateRandom(testAllocation2, 3)
		expectedPorts = []int{3005, 3006, 3007}

		require.NoError(t, err)
		require.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")
	})

	t.Run("must be able to allocate specific ports", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 3100, AvailableRangeTo: 3200}
		testPortAllocator := newPortAllocator(config)

		// Test successful allocation
		err := testPortAllocator.Allocate(testAllocation1, []int{3105, 3106, 3107})
		assert.NoError(t, err)

		// Verify allocation
		ports, err := testPortAllocator.GetAllocation(testAllocation1)
		assert.NoError(t, err)
		assert.Equal(t, []int{3105, 3106, 3107}, ports)

		// Test allocation of already reserved ports
		err = testPortAllocator.Allocate(testAllocation2, []int{8104, 8106})
		assert.Error(t, err)

		// Test allocation outside range
		err = testPortAllocator.Allocate("alloc3", []int{8099, 8201})
		assert.Error(t, err)
	})

	t.Run("the port allocation must stay in range", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 3200, AvailableRangeTo: 3205}
		testPortAllocator := newPortAllocator(config)

		_, err := testPortAllocator.AllocateRandom(testAllocation1, 4)
		assert.NoError(t, err)

		_, err = testPortAllocator.AllocateRandom("alloc2", 3)
		assert.Error(t, err, "Expected an error due to insufficient ports")
	})

	t.Run("must return the allocated ports by allocation id", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 3300, AvailableRangeTo: 3400}
		testPortAllocator := newPortAllocator(config)

		_, _ = testPortAllocator.AllocateRandom(testAllocation1, 5)

		allocatedPorts, err := testPortAllocator.GetAllocation(testAllocation1)
		expectedPorts := []int{3300, 3301, 3302, 3303, 3304}

		assert.NoError(t, err)
		assert.Equal(t, expectedPorts, allocatedPorts, "Retrieved ports do not match expected")

		_, err = testPortAllocator.GetAllocation("nonexistent")
		assert.Error(t, err, "Expected an error for a non-existent allocation ID")
	})

	t.Run("must manage the lifecycle of port allocation properly", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 3400, AvailableRangeTo: 3410}
		testPortAllocator := newPortAllocator(config)

		// First random allocation
		randomPorts1, err := testPortAllocator.AllocateRandom("random1", 3)
		assert.NoError(t, err)
		assert.Equal(t, []int{3400, 3401, 3402}, randomPorts1)

		// Specific allocation should work with available ports
		err = testPortAllocator.Allocate("specific1", []int{3405, 3406})
		assert.NoError(t, err)

		// Second random allocation should skip used ports
		randomPorts2, err := testPortAllocator.AllocateRandom("random2", 2)
		assert.NoError(t, err)
		assert.Equal(t, []int{3403, 3404}, randomPorts2)

		// Release first random allocation
		testPortAllocator.Release("random1")

		// Should be able to specifically allocate now-free ports
		err = testPortAllocator.Allocate("specific2", []int{3400, 3401})
		assert.NoError(t, err)

		// Verify all allocations
		ports, err := testPortAllocator.GetAllocation("specific1")
		assert.NoError(t, err)
		assert.Equal(t, []int{3405, 3406}, ports)

		ports, err = testPortAllocator.GetAllocation("random2")
		assert.NoError(t, err)
		assert.Equal(t, []int{3403, 3404}, ports)

		ports, err = testPortAllocator.GetAllocation("specific2")
		assert.NoError(t, err)
		assert.Equal(t, []int{3400, 3401}, ports)

		// Try to allocate already reserved ports should fail
		err = testPortAllocator.Allocate("specific3", []int{3400, 3407})
		assert.Error(t, err)

		// Test Allocated
		allocated := testPortAllocator.isAllocated([]int{3400, 3401, 3403, 3404, 3405, 3406})
		assert.True(t, allocated)

		allocated = testPortAllocator.isAllocated([]int{3409, 3410})
		assert.False(t, allocated)

		// Release everything
		testPortAllocator.Release("specific1")
		testPortAllocator.Release("random2")
		testPortAllocator.Release("specific2")

		// Should be able to allocate the entire range again
		randomPorts3, err := testPortAllocator.AllocateRandom("random3", 11)
		assert.NoError(t, err)
		assert.Equal(t, []int{3400, 3401, 3402, 3403, 3404, 3405, 3406, 3407, 3408, 3409, 3410}, randomPorts3)
	})

	t.Run("must allocate ports within the range", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 3500, AvailableRangeTo: 3504}
		testPortAllocator := newPortAllocator(config)

		allocatedPorts, err := testPortAllocator.AllocateRandom(testAllocation1, 5)
		expectedPorts := []int{3500, 3501, 3502, 3503, 3504}

		assert.NoError(t, err)
		assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

		_, err = testPortAllocator.AllocateRandom("alloc2", 1)
		assert.Error(t, err, "Expected an error due to no available ports")
	})

	t.Run("must report the port allocation status correctly", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 3600, AvailableRangeTo: 3700}
		testPortAllocator := newPortAllocator(config)

		err := testPortAllocator.Allocate(testAllocation1, []int{3600, 3601, 3602, 3603})
		assert.NoError(t, err)

		allocated := testPortAllocator.isAllocated([]int{3600, 3601, 3602, 3603})
		assert.True(t, allocated)

		allocated = testPortAllocator.isAllocated([]int{7080, 7081, 7082, 7083, 7084})
		assert.False(t, allocated)
	})

	t.Run("must report the port availability status correctly", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 3700, AvailableRangeTo: 3704}
		testPortAllocator := newPortAllocator(config)

		available := testPortAllocator.portsAvailable(5)
		assert.True(t, available)

		_, err := testPortAllocator.AllocateRandom(testAllocation1, 3)
		assert.NoError(t, err)

		available = testPortAllocator.portsAvailable(3)
		assert.False(t, available)

		available = testPortAllocator.portsAvailable(2)
		assert.True(t, available)
	})
}
