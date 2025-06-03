// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestPortAllocator(t *testing.T) {
	t.Parallel()

	const (
		testAllocation1 = "alloc1"
		testAllocation2 = "alloc2"
	)

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
		err = testPortAllocator.Allocate(testAllocation2, []int{3105, 3106})
		assert.Error(t, err)

		// Test allocation outside range
		err = testPortAllocator.Allocate("testAllocationOutOfRange", []int{8099, 8201})
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

func TestAllocatorCommit(t *testing.T) {
	t.Parallel()

	t.Run("must start the allocator", func(t *testing.T) {
		t.Parallel()

		const testAllocID = "test-alloc" // Scoped constant for this test

		ctx := context.Background()

		subs := network.NewSubstrate()
		alloc, _, _ := newMockAllocator(t, subs)

		err := alloc.Run()
		require.NoError(t, err, "Run should not return an error")
		defer func() {
			err := alloc.Stop(ctx)
			assert.NoError(t, err, "Stop should not return an error")
		}()

		inRangeCommit := types.CommittedResources{
			AllocationID: testAllocID,
			Resources: types.Resources{
				CPU: types.CPU{Cores: 1},
				RAM: types.RAM{Size: 1},
			},
		}
		beyondRangeCommit := types.CommittedResources{
			AllocationID: testAllocID,
			Resources: types.Resources{
				CPU:  types.CPU{Cores: MockTotalCPU + 1},
				RAM:  types.RAM{Size: MockTotalRAM + 1},
				Disk: types.Disk{Size: MockTotalDisk + 1},
			},
		}

		portsInRange := map[int]int{
			portRangeFrom + 1: portRangeFrom + 1,
			portRangeTo - 1:   portRangeTo - 1,
		}
		portsOutOfRange := map[int]int{
			portRangeFrom - 100: portRangeFrom - 100,
			portRangeTo + 100:   portRangeTo + 100,
		}

		// no commits on start
		assert.Empty(t, alloc.getCommits(), "commits should be empty on start")

		// test committing resources within range
		err = alloc.Commit(ctx, testAllocID, inRangeCommit, portsInRange, 0, 0)

		assert.NoError(t, err, "commit should not return an error")

		expiry, exists := alloc.getCommit(testAllocID)
		assert.True(t, exists, "commit should exist")
		assert.Equal(t, int64(0), expiry, "expiry should be 0")
		assert.Len(t, alloc.getCommits(), 1, "commits should have one entry")

		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")
		assert.Empty(t, alloc.getCommits(), "commits should be empty after uncommit")

		// test uncommitting a non-existent allocation
		err = alloc.Uncommit(ctx, "non-existent-alloc")
		assert.NoError(t, err, "uncommit should not return an error for non-existent allocation")
		_, exists = alloc.getCommit("non-existent-alloc")
		assert.False(t, exists, "commit should not exist for non-existent allocation")

		// test committing an already committed allocation with 5 dynamic ports
		err = alloc.Commit(ctx, testAllocID, inRangeCommit, nil, 5, 0)
		assert.NoError(t, err, "commit should not return an error")
		err = alloc.Commit(ctx, testAllocID, inRangeCommit, nil, 0, 0)
		assert.Error(t, err, "commit should return an error for already committed allocation")
		_, exists = alloc.getCommit(testAllocID)
		assert.True(t, exists, "commit should exist after re-commit")
		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")

		// test committing resources beyond range
		err = alloc.Commit(ctx, testAllocID, beyondRangeCommit, portsInRange, 0, 0)
		assert.Error(t, err, "commit should return an error for resources beyond range")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")
		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error for non-existent allocation")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")

		// test committing resources with out-of-range ports
		err = alloc.Commit(ctx, testAllocID, inRangeCommit, portsOutOfRange, 0, 0)
		assert.Error(t, err, "commit should return an error for out-of-range ports")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")

		// test committing resources with out-of-range dynamic ports
		err = alloc.Commit(ctx, testAllocID, inRangeCommit, portsInRange, 999999, 0)
		assert.Error(t, err, "commit should return an error for out-of-range dynamic ports")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")
	})
}

func TestAllocatorAllocate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const allocationID = "test-allocation"

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 2},
		Disk: types.Disk{Size: 10},
	}

	job := jobs.Job{
		Resources: resrc,
		Execution: types.SpecConfig{
			Type: "null",
		},
	}

	resourceToCommit := types.CommittedResources{
		AllocationID: allocationID,
		Resources:    resrc,
	}

	t.Run("must fail if allocation is not committed", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, vNet, priv := newMockAllocator(t, subs)

		allocActor, _, _, _ := newActor(t, priv, vNet)
		defer func() {
			err := allocActor.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		allocation, err := alloc.Allocate(
			ctx,
			allocationID,
			"service",
			allocActor,
			orchHandle,
			job,
			nullExecutor,
		)
		assert.Error(t, err, "allocate should return an error for resources not committed")
		assert.Nil(t, allocation, "allocation should be nil on failure")
		assert.Contains(t, err.Error(), "allocation not committed", "error message should indicate resources not committed")
	})

	t.Run("successful allocate with prior commit", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, vNet, priv := newMockAllocator(t, subs)

		allocActor, _, _, _ := newActor(t, priv, vNet)
		defer func() {
			err := allocActor.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{3050: 3050}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle, job, nullExecutor)
		assert.NoError(t, err, "allocate should not return an error")
		assert.NotNil(t, allocation, "allocation should not be nil on success")

		err = alloc.Run()
		require.NoError(t, err, "Run should not return an error")

		defer func() {
			err := alloc.Stop(ctx)
			assert.NoError(t, err, "Stop should not return an error")
		}()

		assert.NoError(t, err, "allocate should not return an error")
		assert.NotNil(t, allocation, "allocation should not be nil on success")
		assert.Equal(t, allocation.ID, allocationID, "allocation ID should match")
		assert.Equal(t, allocation.Job.Resources.CPU.Cores, job.Resources.CPU.Cores, "CPU cores should match")
		assert.Equal(t, jobs.AllocationStatus("pending"), allocation.Status(ctx).Status, "allocation status should be running")

		// verify allocation is stored
		allocInst, err := alloc.GetAllocation(allocationID)
		assert.NoError(t, err, "GetAllocation should not return an error")
		assert.NotNil(t, allocInst, "allocation should not be nil")
		assert.Equal(t, allocInst.ID, allocationID, "allocation ID should match")

		alloc.lock.Lock()
		_, exists := alloc.allocations[allocationID]
		alloc.lock.Unlock()
		assert.True(t, exists, "allocation should be stored in allocator")

		err = alloc.Uncommit(ctx, allocationID)
		assert.NoError(t, err)

		err = alloc.Release(ctx, allocationID)
		assert.NoError(t, err)

		// verify allocation is removed
		allocInst, err = alloc.GetAllocation(allocationID)
		assert.Error(t, err, "GetAllocation should return an error for non-existent allocation")
		assert.Nil(t, allocInst, "allocation should be nil after release")
	})

	t.Run("must fail if hardware capacity is insufficient", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, vNet, priv := newMockAllocator(t, subs)

		allocActor, _, _, _ := newActor(t, priv, vNet)
		defer func() {
			err := allocActor.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		// commit a reasonable resource first
		err = alloc.Commit(ctx, allocationID, resourceToCommit, nil, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		// verify allocation is stored
		_, exists := alloc.getCommit(allocationID)
		assert.True(t, exists, "allocation should be stored in allocator")

		beyondAvailableJob := jobs.Job{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 100},
				RAM: types.RAM{Size: 100},
			},
			Execution: types.SpecConfig{
				Type: "null",
			},
		}

		// try to allocate too much resources
		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle, beyondAvailableJob, nullExecutor)
		assert.ErrorContains(t, err, types.ErrNoFreeResources.Error())
		assert.Nil(t, allocation)
	})
}

func TestAllocator_Stop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const allocationID = "test-allocation"

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	job := jobs.Job{
		Resources: resrc,
		Execution: types.SpecConfig{
			Type: "null",
		},
	}

	resourceToCommit := types.CommittedResources{
		AllocationID: allocationID,
		Resources:    resrc,
	}

	t.Run("releasing a stopped allocation", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, vNet, priv := newMockAllocator(t, subs)

		allocActor, _, _, _ := newActor(t, priv, vNet)
		defer func() {
			err := allocActor.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{3000: 3000}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle, job, nullExecutor)

		assert.NoError(t, err, "allocate should not return an error")
		assert.NotNil(t, allocation, "allocation should not be nil on success")
		assert.Equal(t, allocation.ID, allocationID, "allocation ID should match")
		assert.Equal(t, allocation.Job.Resources.CPU.Cores, job.Resources.CPU.Cores, "CPU cores should match")

		_, exists := alloc.getCommit(allocationID)
		assert.False(t, exists, "commit should not exist after allocation")

		// verify allocation is stored
		allocInst, err := alloc.GetAllocation(allocationID)
		assert.NoError(t, err, "GetAllocation should not return an error")
		assert.NotNil(t, allocInst, "allocation should not be nil")
		assert.Equal(t, allocInst.ID, allocationID, "allocation ID should match")
		assert.Equal(t, 1, len(alloc.GetAllocations()), "allocations should have one entry")

		alloc.lock.Lock()
		_, exists = alloc.allocations[allocationID]
		alloc.lock.Unlock()
		assert.True(t, exists, "allocation should be stored in allocator")

		assert.Equal(t, jobs.AllocationStatus("pending"), allocation.Status(ctx).Status, "allocation status should be pending")

		// stop the allocation
		err = alloc.Stop(ctx)
		assert.NoError(t, err)

		assert.Equal(t, jobs.AllocationStatus("stopped"), allocation.Status(ctx).Status, "allocation status should be stopped")

		err = alloc.Release(ctx, allocationID)
		assert.NoError(t, err)

		// verify allocation is removed
		allocInst, err = alloc.GetAllocation(allocationID)
		assert.Error(t, err, "GetAllocation should return an error for non-existent allocation")
		assert.Nil(t, allocInst, "allocation should be nil after release")
		assert.Equal(t, 0, len(alloc.GetAllocations()), "allocations should be empty after release")
	})

	t.Run("stop must clear unallocated commits", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, vNet, priv := newMockAllocator(t, subs)

		allocActor, _, _, _ := newActor(t, priv, vNet)
		defer func() {
			err := allocActor.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{3041: 3041}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		anotherAllocationID := "another-allocation"
		anotherResourceToCommit := types.CommittedResources{
			AllocationID: anotherAllocationID,
			Resources:    resrc,
		}
		err = alloc.Commit(ctx, anotherAllocationID, anotherResourceToCommit, map[int]int{3021: 3021}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		// verify commit is stored
		_, exists := alloc.getCommit(allocationID)
		assert.True(t, exists, "commit should exist after commit")
		_, exists = alloc.getCommit(anotherAllocationID)
		assert.True(t, exists, "commit should exist after commit")
		assert.Equal(t, 2, len(alloc.getCommits()), "commits should have two entries")

		// allocate the first allocation
		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle, job, nullExecutor)

		assert.NoError(t, err, "allocate should not return an error")
		assert.NotNil(t, allocation, "allocation should not be nil on success")
		assert.Equal(t, allocation.ID, allocationID, "allocation ID should match")
		assert.Equal(t, allocation.Job.Resources.CPU.Cores, job.Resources.CPU.Cores, "CPU cores should match")

		_, exists = alloc.getCommit(allocationID)
		assert.False(t, exists, "commit should not exist after allocation")
		_, exists = alloc.getCommit(anotherAllocationID)
		assert.True(t, exists, "commit should exist after allocation")

		// verify allocation is stored
		allocInst, err := alloc.GetAllocation(allocationID)
		assert.NoError(t, err, "GetAllocation should not return an error")
		assert.NotNil(t, allocInst, "allocation should not be nil")
		assert.Equal(t, allocInst.ID, allocationID, "allocation ID should match")
		assert.Equal(t, 1, len(alloc.GetAllocations()), "allocations should have one entry")

		// stop the allocation
		err = alloc.Stop(ctx)
		assert.NoError(t, err)

		assert.Equal(t, jobs.AllocationStatus("stopped"), allocation.Status(ctx).Status, "allocation status should be stopped")

		// verify commit is removed
		_, exists = alloc.getCommit(anotherAllocationID)
		assert.False(t, exists, "commit should not exist after stop")
		assert.Equal(t, 0, len(alloc.getCommits()), "commits should be empty after stop")

		err = alloc.Release(ctx, allocationID)
		assert.NoError(t, err)

		// verify allocation is removed
		allocInst, err = alloc.GetAllocation(allocationID)
		assert.Error(t, err, "GetAllocation should return an error for non-existent allocation")
		assert.Nil(t, allocInst, "allocation should be nil after release")
		assert.Equal(t, 0, len(alloc.GetAllocations()), "allocations should be empty after release")
	})
}

func TestAllocator_CheckAvailability(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const allocationID = "test-allocation"

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	resourceToCommit := types.CommittedResources{
		AllocationID: allocationID,
		Resources:    resrc,
	}

	t.Run("specific ports", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, _, _ := newMockAllocator(t, subs)
		defer func() {
			err := alloc.Stop(ctx)
			assert.NoError(t, err, "Stop should not return an error")
		}()

		err := alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{3030: 3030}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		err = alloc.CheckAvailability([]int{3030, 3031}, 0, resrc)
		assert.Error(t, err, "CheckAvailability should return an error for already allocated ports")
		assert.ErrorIs(t, err, ErrPortsBusy, "expected ErrPortsBusy when ports are already allocated")

		err = alloc.CheckAvailability([]int{3031, 3032}, 0, resrc)
		assert.NoError(t, err, "CheckAvailability should not return an error for free ports")
	})

	t.Run("dynamic ports", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, _, _ := newMockAllocator(t, subs)
		defer func() {
			err := alloc.Stop(ctx)
			assert.NoError(t, err, "Stop should not return an error")
		}()

		// 40 dynamic ports plus 1 static should consume upto 3000 + 40

		err := alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{3000: 3000}, 40, 0)
		assert.NoError(t, err, "commit should not return an error")

		err = alloc.CheckAvailability([]int{3092, 3093}, 0, resrc)
		assert.NoError(t, err, "CheckAvailability should not return an error for free ports")

		// too many dynamic ports
		err = alloc.CheckAvailability([]int{3094}, 1000, resrc)
		assert.Error(t, err, "CheckAvailability should return an error for unavailable dynamic ports")
		assert.ErrorIs(t, err, ErrDynamicPortsNotAvailable, "expected ErrDynamicPortsNotAvailable when dynamic ports are unavailable")
	})

	t.Run("compute resources", func(t *testing.T) {
		t.Parallel()

		subs := network.NewSubstrate()
		alloc, _, _ := newMockAllocator(t, subs)
		defer func() {
			err := alloc.Stop(ctx)
			assert.NoError(t, err, "Stop should not return an error")
		}()

		// onboarded 4 cpu cores and 4GB RAM - committing 1 cpu core and 1GB RAM
		err := alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		// check availability for 1 cpu core and 1GB RAM
		err = alloc.CheckAvailability([]int{}, 0, resrc)
		assert.NoError(t, err, "CheckAvailability should not return an error for available resources")

		tooMuchResource := types.Resources{
			CPU:  types.CPU{Cores: MockTotalCPU + 1},
			RAM:  types.RAM{Size: MockTotalRAM + 1},
			Disk: types.Disk{Size: MockTotalDisk + 1},
		}
		err = alloc.CheckAvailability([]int{}, 0, tooMuchResource)
		assert.Error(t, err, "CheckAvailability should return an error for unavailable resources")
		assert.ErrorIs(t, err, ErrResourcesNotAvailable, "expected ErrResourcesNotAvailable when resources are unavailable")
	})
}
