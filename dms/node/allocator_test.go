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

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
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

func TestAllocator(t *testing.T) {
	t.Parallel()

	testAllocationID := "alloc1"

	t.Run("must be able to create an allocator via the default constructor", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 8000, AvailableRangeTo: 8100}
		testPortAllocator := newPortAllocator(config)
		ctrl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		mockNetwork := NewMockNetwork(ctrl)
		fs := afero.Afero{Fs: afero.NewMemMapFs()}

		vt := storage.NewVolumeTracker()

		testAllocator := newAllocator(vt, testPortAllocator, resourceManager, hardwareManager, mockNetwork, fs, t.TempDir(), "testHost")

		assert.NotNil(t, testAllocator)
	})

	t.Run("must be able to commit then uncommit an allocation", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 8100, AvailableRangeTo: 8200}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)

		// Commit
		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment := map[int]int{
			8100: 10,
			8101: 10,
			8102: 10,
			8103: 10,
			8104: 10,
		}

		// expectations for commit
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		// commit
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment, 10, time.Now().Add(5*time.Minute).Unix())
		require.NoError(t, err)

		// expectations for uncommit
		testAllocator.resources.(*MockResourceManager).EXPECT().UncommitResources(gomock.Any(), testAllocationID).Return(nil).Times(1)
		// uncommit
		err = testAllocator.Uncommit(context.Background(), testAllocationID)
		require.NoError(t, err)
	})

	t.Run("must be able to create then release allocations", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 8200, AvailableRangeTo: 8300}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)

		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment := map[int]int{
			8200: 10,
			8201: 10,
			8202: 10,
			8203: 10,
			8204: 10,
		}
		// expectations for commit
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(2)
		// commit
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment, 10, time.Now().Add(5*time.Minute).Unix())
		require.NoError(t, err)

		mockActor := NewMockActor(ctrl)
		mockJob := jobs.Job{
			Resources: resourceCommitment.Resources,
			Execution: types.SpecConfig{
				Type: "docker",
			},
		}
		mockExecutor := NewMockExecutor(ctrl)
		// expectations for allocate
		testAllocator.resources.(*MockResourceManager).EXPECT().AllocateResources(gomock.Any(), testAllocationID).Return(nil).Times(1)
		mockActor.EXPECT().AddBehavior(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockActor.EXPECT().Start().Return(nil).Times(1)
		// allocate
		allocation, err := testAllocator.Allocate(
			context.Background(), testAllocationID,
			jobtypes.AllocationTypeService, mockActor,
			actor.Handle{}, mockJob, mockExecutor,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		// Ensure that the allocation is in the state
		allocations := testAllocator.GetAllocations()
		require.Equal(t, 1, len(allocations))

		// Ensure that we can get the allocation by ID
		allocation, err = testAllocator.GetAllocation(testAllocationID)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		// expectation for release
		mockActor.EXPECT().Stop().Return(nil).Times(1)
		mockExecutor.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().DeallocateResources(gomock.Any(), testAllocationID).Return(nil).Times(1)
		// release
		err = testAllocator.Release(context.Background(), testAllocationID)
		require.NoError(t, err)
	})

	t.Run("must clear commits when expired", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 8300, AvailableRangeTo: 8400}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)

		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment := map[int]int{
			8300: 10,
			8301: 10,
			8302: 10,
			8303: 10,
			8304: 10,
		}
		// expectations for commit
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		// commit
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment, 10, time.Now().Unix())
		require.NoError(t, err)

		// Ensure that the commit is in the state
		require.Equal(t, 1, len(testAllocator.commits))

		// Start the allocator and wait for it to remove the expired commit
		err = testAllocator.Run()
		require.NoError(t, err)

		// expectations for uncommit when expired
		testAllocator.resources.(*MockResourceManager).EXPECT().UncommitResources(gomock.Any(), testAllocationID).Return(nil).Times(1)
		time.Sleep(clearCommitsFrequency + 1*time.Second)

		// Ensure that the commit is no longer in the state
		require.Equal(t, 0, len(testAllocator.commits))
	})

	t.Run("must stop the allocations and clear commits on stop", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 8400, AvailableRangeTo: 8500}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)
		testAllocation2 := "alloc2"

		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment1 := map[int]int{
			8400: 10,
			8401: 10,
			8402: 10,
			8403: 10,
			8404: 10,
		}
		portCommitment2 := map[int]int{
			8420: 10,
		}
		// expectations for commit
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		// commit
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment1, 10, time.Now().Add(5*time.Minute).Unix())
		require.NoError(t, err)

		// expectations for commit
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		// commit
		err = testAllocator.Commit(context.Background(), testAllocation2, resourceCommitment, portCommitment2, 10, time.Now().Add(5*time.Minute).Unix())
		require.NoError(t, err)

		mockActor := NewMockActor(ctrl)
		mockJob := jobs.Job{
			Resources: resourceCommitment.Resources,
			Execution: types.SpecConfig{
				Type: "docker",
			},
		}
		mockExecutor := NewMockExecutor(ctrl)
		// expectations for allocate
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().AllocateResources(gomock.Any(), testAllocationID).Return(nil).Times(1)
		mockActor.EXPECT().AddBehavior(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockActor.EXPECT().Start().Return(nil).Times(1)
		// allocate
		allocation, err := testAllocator.Allocate(
			context.Background(), testAllocationID,
			jobtypes.AllocationTypeService, mockActor,
			actor.Handle{}, mockJob, mockExecutor,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		// expectations for stop
		mockActor.EXPECT().Stop().Return(nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().UncommitResources(gomock.Any(), testAllocation2).Return(nil).Times(1)
		// stop the allocator
		err = testAllocator.Stop(context.Background())
		require.NoError(t, err)

		// ensure that the allocations are stopped
		for _, a := range testAllocator.allocations {
			assert.Equal(t, jobs.AllocationStopped, a.Status(context.Background()).Status)
		}

		// ensure that the commits are cleared
		assert.Equal(t, 0, len(testAllocator.commits))
	})

	t.Run("must be able to check the availability properly", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 5200, AvailableRangeTo: 5210}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)

		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment1 := map[int]int{
			5200: 10,
			5201: 10,
			5202: 10,
			5203: 10,
			5204: 10,
		}
		portsCommitment1 := []int{5200, 5201, 5202, 5203, 5204}
		portsCommitment2 := []int{5220}
		// expectations for commit
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(true, nil).Times(1)
		testAllocator.resources.(*MockResourceManager).EXPECT().CommitResources(gomock.Any(), resourceCommitment).Return(nil).Times(1)
		// commit
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment1, 5, time.Now().Add(5*time.Minute).Unix())
		require.NoError(t, err)

		// assert the port availability
		err = testAllocator.CheckAvailability(portsCommitment1, 1, resourceCommitment.Resources)
		require.Error(t, err, ErrPortsBusy)

		// assert the dynamic port availability
		err = testAllocator.CheckAvailability(portsCommitment2, 5, resourceCommitment.Resources)
		require.Error(t, err, ErrDynamicPortsNotAvailable)

		// assert the resource availability
		testAllocator.resources.(*MockResourceManager).EXPECT().GetFreeResources(gomock.Any()).Return(types.FreeResources{Resources: resourceCommitment.Resources}, nil).Times(1)
		err = testAllocator.CheckAvailability(portsCommitment2, 1, resourceCommitment.Resources)
		require.NoError(t, err)

		// out of resources
		testAllocator.resources.(*MockResourceManager).EXPECT().GetFreeResources(gomock.Any()).Return(types.FreeResources{Resources: types.Resources{}}, nil).Times(1)
		err = testAllocator.CheckAvailability(portsCommitment2, 1, resourceCommitment.Resources)
		require.Error(t, err, ErrResourcesNotAvailable)
	})

	t.Run("must fail to commit an allocation if there is no hardware capacity", func(t *testing.T) {
		t.Parallel()

		config := PortConfig{AvailableRangeFrom: 5100, AvailableRangeTo: 5110}
		ctrl := gomock.NewController(t)
		testAllocator := createTestAllocator(t, config, ctrl)

		resourceCommitment := types.CommittedResources{
			Resources: types.Resources{
				CPU: types.CPU{Cores: 10},
				RAM: types.RAM{Size: 10000000000},
			},
		}
		portCommitment := map[int]int{
			5100: 10,
			5101: 10,
			5102: 10,
			5103: 10,
			5104: 10,
		}
		testAllocator.hardware.(*MockHardwareManager).EXPECT().CheckCapacity(resourceCommitment.Resources).Return(false, nil).Times(1)
		err := testAllocator.Commit(context.Background(), testAllocationID, resourceCommitment, portCommitment, 10, time.Now().Add(5*time.Minute).Unix())
		require.Error(t, err, ErrNoHardwareCapacity)
	})
}

func createTestAllocator(t *testing.T, portConfig PortConfig, ctrl *gomock.Controller) allocator {
	t.Helper()

	testPortAllocator := newPortAllocator(portConfig)
	resourceManager := NewMockResourceManager(ctrl)
	hardwareManager := NewMockHardwareManager(ctrl)
	mockNetwork := NewMockNetwork(ctrl)
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	ctx, cancel := context.WithCancel(context.Background())

	return allocator{
		ports:       testPortAllocator,
		resources:   resourceManager,
		hardware:    hardwareManager,
		network:     mockNetwork,
		fs:          fs,
		allocations: make(map[string]*jobs.Allocation),
		hostID:      "testHost",
		workDir:     t.TempDir(),
		commits:     make(map[string]int64),
		ctx:         ctx,
		cancel:      cancel,
	}
}
