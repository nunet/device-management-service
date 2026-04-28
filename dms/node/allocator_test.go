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
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
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

		config := PortConfig{AvailableRangeFrom: 30000, AvailableRangeTo: 30100}
		testPortAllocator := newPortAllocator(config)

		allocatedPorts, err := testPortAllocator.AllocateRandom(testAllocation1, 5)
		expectedPorts := []int{30000, 30001, 30002, 30003, 30004}

		require.NoError(t, err)
		require.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

		allocatedPorts, err = testPortAllocator.AllocateRandom(testAllocation2, 3)
		expectedPorts = []int{30005, 30006, 30007}

		require.NoError(t, err)
		require.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")
	})

	t.Run("must be able to allocate specific ports", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 31000, AvailableRangeTo: 31200}
		testPortAllocator := newPortAllocator(config)

		// Test successful allocation
		err := testPortAllocator.Allocate(testAllocation1, []int{31005, 31006, 31007})
		assert.NoError(t, err)

		// Verify allocation
		ports, err := testPortAllocator.GetAllocation(testAllocation1)
		assert.NoError(t, err)
		assert.Equal(t, []int{31005, 31006, 31007}, ports)

		// Test allocation of already reserved ports
		err = testPortAllocator.Allocate(testAllocation2, []int{31005, 31006})
		assert.Error(t, err)

		// Test allocation outside range
		err = testPortAllocator.Allocate("testAllocationOutOfRange", []int{8099, 38201})
		assert.Error(t, err)
	})

	t.Run("the port allocation must stay in range", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 30200, AvailableRangeTo: 30205}
		testPortAllocator := newPortAllocator(config)

		_, err := testPortAllocator.AllocateRandom(testAllocation1, 4)
		assert.NoError(t, err)

		_, err = testPortAllocator.AllocateRandom("alloc2", 3)
		assert.Error(t, err, "Expected an error due to insufficient ports")
	})

	t.Run("must return the allocated ports by allocation id", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 30300, AvailableRangeTo: 30400}
		testPortAllocator := newPortAllocator(config)

		_, _ = testPortAllocator.AllocateRandom(testAllocation1, 5)

		allocatedPorts, err := testPortAllocator.GetAllocation(testAllocation1)
		expectedPorts := []int{30300, 30301, 30302, 30303, 30304}

		assert.NoError(t, err)
		assert.Equal(t, expectedPorts, allocatedPorts, "Retrieved ports do not match expected")

		_, err = testPortAllocator.GetAllocation("nonexistent")
		assert.Error(t, err, "Expected an error for a non-existent allocation ID")
	})

	t.Run("must manage the lifecycle of port allocation properly", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 30400, AvailableRangeTo: 30410}
		testPortAllocator := newPortAllocator(config)

		// First random allocation
		randomPorts1, err := testPortAllocator.AllocateRandom("random1", 3)
		assert.NoError(t, err)
		assert.Equal(t, []int{30400, 30401, 30402}, randomPorts1)

		// Specific allocation should work with available ports
		err = testPortAllocator.Allocate("specific1", []int{30405, 30406})
		assert.NoError(t, err)

		// Second random allocation should skip used ports
		randomPorts2, err := testPortAllocator.AllocateRandom("random2", 2)
		assert.NoError(t, err)
		assert.Equal(t, []int{30403, 30404}, randomPorts2)

		// Release first random allocation
		testPortAllocator.Release("random1")

		// Should be able to specifically allocate now-free ports
		err = testPortAllocator.Allocate("specific2", []int{30400, 30401})
		assert.NoError(t, err)

		// Verify all allocations
		ports, err := testPortAllocator.GetAllocation("specific1")
		assert.NoError(t, err)
		assert.Equal(t, []int{30405, 30406}, ports)

		ports, err = testPortAllocator.GetAllocation("random2")
		assert.NoError(t, err)
		assert.Equal(t, []int{30403, 30404}, ports)

		ports, err = testPortAllocator.GetAllocation("specific2")
		assert.NoError(t, err)
		assert.Equal(t, []int{30400, 30401}, ports)

		// Try to allocate already reserved ports should fail
		err = testPortAllocator.Allocate("specific3", []int{30400, 30407})
		assert.Error(t, err)

		// Test Allocated
		allocated := testPortAllocator.isAllocated([]int{30400, 30401, 30403, 30404, 30405, 30406})
		assert.True(t, allocated)

		allocated = testPortAllocator.isAllocated([]int{30409, 30410})
		assert.False(t, allocated)

		// Release everything
		testPortAllocator.Release("specific1")
		testPortAllocator.Release("random2")
		testPortAllocator.Release("specific2")

		// Should be able to allocate the entire range again
		randomPorts3, err := testPortAllocator.AllocateRandom("random3", 11)
		assert.NoError(t, err)
		assert.Equal(t, []int{30400, 30401, 30402, 30403, 30404, 30405, 30406, 30407, 30408, 30409, 30410}, randomPorts3)
	})

	t.Run("must allocate ports within the range", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 30500, AvailableRangeTo: 30504}
		testPortAllocator := newPortAllocator(config)

		allocatedPorts, err := testPortAllocator.AllocateRandom(testAllocation1, 5)
		expectedPorts := []int{30500, 30501, 30502, 30503, 30504}

		assert.NoError(t, err)
		assert.Equal(t, expectedPorts, allocatedPorts, "Allocated ports do not match expected")

		_, err = testPortAllocator.AllocateRandom("alloc2", 1)
		assert.Error(t, err, "Expected an error due to no available ports")
	})

	t.Run("must report the port allocation status correctly", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 30600, AvailableRangeTo: 30700}
		testPortAllocator := newPortAllocator(config)

		err := testPortAllocator.Allocate(testAllocation1, []int{30600, 30601, 30602, 30603})
		assert.NoError(t, err)

		allocated := testPortAllocator.isAllocated([]int{30600, 30601, 30602, 30603})
		assert.True(t, allocated)

		allocated = testPortAllocator.isAllocated([]int{37080, 37081, 37082, 37083, 37084})
		assert.False(t, allocated)
	})

	t.Run("must report the port availability status correctly", func(t *testing.T) {
		config := PortConfig{AvailableRangeFrom: 30700, AvailableRangeTo: 30704}
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

	const testAllocID = "test-alloc"
	ctx := context.Background()
	subs := network.NewSubstrate()

	inRangeResources := types.CommittedResources{
		AllocationID: testAllocID,
		Resources: types.Resources{
			CPU: types.CPU{Cores: 1},
			RAM: types.RAM{Size: 1},
		},
	}
	beyondRangeResources := types.CommittedResources{
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

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		alloc, _, _ := newMockAllocator(t, subs)

		// no commits on start
		assert.Empty(t, alloc.getCommits(), "commits should be empty on start")

		err := alloc.Commit(ctx, testAllocID, inRangeResources, portsInRange, 0, time.Hour)
		assert.NoError(t, err, "commit should not return an error")

		expiry, exists := alloc.getCommit(testAllocID)
		assert.True(t, exists, "commit should exist")
		assert.Greater(t, expiry, time.Now().Unix(), "expiry should be greater than current time")
		assert.Len(t, alloc.getCommits(), 1, "commits should have one entry")

		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")
		assert.Empty(t, alloc.getCommits(), "commits should be empty after uncommit")
	})

	t.Run("uncommitting non-existent allocation", func(t *testing.T) {
		t.Parallel()

		alloc, _, _ := newMockAllocator(t, subs)
		// test uncommitting a non-existent allocation
		err := alloc.Uncommit(ctx, "non-existent-alloc")
		assert.NoError(t, err, "uncommit should not return an error for non-existent allocation")
		_, exists := alloc.getCommit("non-existent-alloc")
		assert.False(t, exists, "commit should not exist for non-existent allocation")
	})

	t.Run("re-committing an already committed allocation", func(t *testing.T) {
		t.Parallel()

		alloc, _, _ := newMockAllocator(t, subs)
		// test committing an already committed allocation with 5 dynamic ports
		err := alloc.Commit(ctx, testAllocID, inRangeResources, nil, 5, 0)
		assert.NoError(t, err, "commit should not return an error")
		err = alloc.Commit(ctx, testAllocID, inRangeResources, nil, 0, 0)
		assert.Error(t, err, "commit should return an error for already committed allocation")
		_, exists := alloc.getCommit(testAllocID)
		assert.True(t, exists, "commit should exist after re-commit")
		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")
	})

	t.Run("committing resources beyond range", func(t *testing.T) {
		t.Parallel()

		alloc, _, _ := newMockAllocator(t, subs)
		// with resources beyond range
		err := alloc.Commit(ctx, testAllocID, beyondRangeResources, portsInRange, 0, 0)
		assert.Error(t, err, "commit should return an error for resources beyond range")
		_, exists := alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")
		err = alloc.Uncommit(ctx, testAllocID)
		assert.NoError(t, err, "uncommit should not return an error for non-existent allocation")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after uncommit")
		isResourceCommitted, err := alloc.resources.IsCommitted(testAllocID)
		assert.NoError(t, err, "IsCommitted should not return an error")
		assert.False(t, isResourceCommitted, "resources should not be committed after failed commit")
		assert.False(t,
			alloc.ports.isAllocated([]int{portRangeFrom + 1, portRangeTo - 1}),
			"ports should not be allocated after failed commit")

		// with out-of-range ports
		err = alloc.Commit(ctx, testAllocID, inRangeResources, portsOutOfRange, 0, 0)
		assert.Error(t, err, "commit should return an error for out-of-range ports")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")
		assert.Empty(t, alloc.ports.allocations, "ports allocations should be empty after failed commit")

		// with out-of-range dynamic ports
		err = alloc.Commit(ctx, testAllocID, inRangeResources, portsInRange, 999999, 0)
		assert.Error(t, err, "commit should return an error for out-of-range dynamic ports")
		_, exists = alloc.getCommit(testAllocID)
		assert.False(t, exists, "commit should not exist after failed commit")
		isResourceCommitted, err = alloc.resources.IsCommitted(testAllocID)
		assert.NoError(t, err, "IsCommitted should not return an error")
		assert.False(t, isResourceCommitted, "resources should not be committed after failed commit")
		assert.Empty(t, alloc.ports.allocations, "ports allocations should be empty")
		assert.Empty(t, alloc.ports.reserved, "ports reserved should be empty")
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
			map[string]types.ContractConfig{},
			eventhandler.New(context.Background(), 1, 1, time.Second, time.Second, func(_ eventhandler.Event) error { return nil }),
			"",
			func(_ string, _ jobtypes.AllocationStatus) {},
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

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{30050: 30050}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor,
			orchHandle, job, nullExecutor,
			map[string]types.ContractConfig{},
			eventhandler.New(
				context.Background(), 1, 1,
				time.Second, time.Second,
				func(_ eventhandler.Event) error { return nil }),
			"",
			func(_ string, _ jobtypes.AllocationStatus) {},
		)
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
		assert.Equal(t, jobs.AllocationStatus("pending"), allocation.Status().Status, "allocation status should be running")

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
			ctx, allocationID, "service", allocActor,
			orchHandle, beyondAvailableJob, nullExecutor,
			map[string]types.ContractConfig{},
			eventhandler.New(
				context.Background(), 1, 1,
				time.Second, time.Second,
				func(_ eventhandler.Event) error { return nil }),
			"",
			func(_ string, _ jobtypes.AllocationStatus) {},
		)
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

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{30000: 30000}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle,
			job, nullExecutor, map[string]types.ContractConfig{},
			eventhandler.New(
				context.Background(), 1, 1,
				time.Second, time.Second,
				func(_ eventhandler.Event) error { return nil }),
			"",
			func(_ string, _ jobtypes.AllocationStatus) {},
		)

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

		assert.Equal(t, jobs.AllocationStatus("pending"), allocation.Status().Status, "allocation status should be pending")

		// stop the allocation
		err = alloc.Stop(ctx)
		assert.NoError(t, err)

		assert.Equal(t, jobs.AllocationStatus("stopped"), allocation.Status().Status, "allocation status should be stopped")

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

		err = alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{30041: 30041}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		anotherAllocationID := "another-allocation"
		anotherResourceToCommit := types.CommittedResources{
			AllocationID: anotherAllocationID,
			Resources:    resrc,
		}
		err = alloc.Commit(ctx, anotherAllocationID, anotherResourceToCommit, map[int]int{30021: 30021}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		// verify commit is stored
		_, exists := alloc.getCommit(allocationID)
		assert.True(t, exists, "commit should exist after commit")
		_, exists = alloc.getCommit(anotherAllocationID)
		assert.True(t, exists, "commit should exist after commit")
		assert.Equal(t, 2, len(alloc.getCommits()), "commits should have two entries")

		// allocate the first allocation
		allocation, err := alloc.Allocate(
			ctx, allocationID, "service", allocActor, orchHandle,
			job, nullExecutor, map[string]types.ContractConfig{},
			eventhandler.New(context.Background(), 1, 1,
				time.Second, time.Second, func(_ eventhandler.Event) error { return nil }),
			"", func(_ string, _ jobtypes.AllocationStatus) {},
		)

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

		assert.Equal(t, jobs.AllocationStatus("stopped"), allocation.Status().Status, "allocation status should be stopped")

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

		err := alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{30030: 30030}, 0, 0)
		assert.NoError(t, err, "commit should not return an error")

		err = alloc.CheckAvailability([]int{30030, 30031}, 0, resrc)
		assert.Error(t, err, "CheckAvailability should return an error for already allocated ports")
		assert.ErrorIs(t, err, ErrPortsBusy, "expected ErrPortsBusy when ports are already allocated")

		err = alloc.CheckAvailability([]int{30031, 30032}, 0, resrc)
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

		// 40 dynamic ports plus 1 static should consume upto 30000 + 40

		err := alloc.Commit(ctx, allocationID, resourceToCommit, map[int]int{30000: 30000}, 40, 0)
		assert.NoError(t, err, "commit should not return an error")

		err = alloc.CheckAvailability([]int{30092, 30093}, 0, resrc)
		assert.NoError(t, err, "CheckAvailability should not return an error for free ports")

		// too many dynamic ports
		err = alloc.CheckAvailability([]int{30094}, 1000, resrc)
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
