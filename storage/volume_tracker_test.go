package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test constants
const (
	testBasePath        = "/mnt/test"
	testAllocationID    = "alloc-test"
	testContainerID     = "container-test"
	errFailedFindVolume = "failed to find volume"
)

func TestVolumeTracker(t *testing.T) {
	t.Parallel()

	t.Run("NewVolumeTracker_ShouldInitializeEmptyMap", func(t *testing.T) {
		t.Parallel()
		tracker := NewVolumeTracker()
		assert.NotNil(t, tracker, "NewVolumeTracker should not return nil")
		assert.NotNil(t, tracker.mounts, "NewVolumeTracker mounts map should be initialized")
		assert.Empty(t, tracker.mounts, "NewVolumeTracker mounts map should be empty initially")
	})

	t.Run("TrackMount_Operations", func(t *testing.T) {
		t.Parallel()

		t.Run("ShouldStoreVolumeInfo_WhenValidPathAndIDs", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/basic"
			allocID := testAllocationID
			containerID := testContainerID

			tracker.TrackMount(path, allocID, containerID)

			info, err := tracker.GetMountInfo(path)
			assert.NoError(t, err, "GetMountInfo should not return an error for a tracked mount")
			assert.Equal(t, allocID, info.AllocationID, "Tracked AllocationID does not match")
			assert.Equal(t, containerID, info.ContainerID, "Tracked ContainerID does not match")
			assert.True(t, tracker.IsMounted(path), "IsMounted should return true for a tracked mount")
		})

		t.Run("ShouldOverwriteExistingMount_WhenSamePathIsUsed", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/overwrite"
			allocID1 := testAllocationID + "-old"
			containerID1 := testContainerID + "-old"
			allocID2 := testAllocationID + "-new"
			containerID2 := testContainerID + "-new"

			tracker.TrackMount(path, allocID1, containerID1) // Initial mount
			tracker.TrackMount(path, allocID2, containerID2) // Overwrite

			info, err := tracker.GetMountInfo(path)
			assert.NoError(t, err)
			assert.Equal(t, allocID2, info.AllocationID, "AllocationID should be overwritten")
			assert.Equal(t, containerID2, info.ContainerID, "ContainerID should be overwritten")
			assert.Len(t, tracker.mounts, 1, "Mounts map should still contain only one entry for the path")
		})

		t.Run("ShouldAcceptEmptyValues_ForPathAndIDs", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()

			// Test empty path
			emptyPath := ""
			tracker.TrackMount(emptyPath, testAllocationID+"-empty-path", testContainerID+"-empty-path")
			info, err := tracker.GetMountInfo(emptyPath)
			assert.NoError(t, err)
			assert.Equal(t, testAllocationID+"-empty-path", info.AllocationID)

			// Test empty allocation ID
			emptyAllocPath := testBasePath + "/empty-alloc"
			tracker.TrackMount(emptyAllocPath, "", testContainerID+"-empty-alloc")
			info, err = tracker.GetMountInfo(emptyAllocPath)
			assert.NoError(t, err)
			assert.Equal(t, "", info.AllocationID)

			// Test empty container ID
			emptyContainerPath := testBasePath + "/empty-container"
			tracker.TrackMount(emptyContainerPath, testAllocationID+"-empty-container", "")
			info, err = tracker.GetMountInfo(emptyContainerPath)
			assert.NoError(t, err)
			assert.Equal(t, "", info.ContainerID)
		})
	})

	t.Run("UntrackMount_Operations", func(t *testing.T) {
		t.Parallel()

		t.Run("ShouldRemoveMount_WhenPathExists", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/untrack"
			allocID := testAllocationID + "-untrack"
			containerID := testContainerID + "-untrack"

			tracker.TrackMount(path, allocID, containerID)
			assert.True(t, tracker.IsMounted(path), "Mount should be tracked before untracking")

			tracker.UntrackMount(path)
			assert.False(t, tracker.IsMounted(path), "IsMounted should return false after untracking")

			_, err := tracker.GetMountInfo(path)
			assert.Error(t, err, "GetMountInfo should return an error after untracking")
			assert.Equal(t, errFailedFindVolume, err.Error(), "Error message should match expected")
		})

		t.Run("ShouldNotPanic_WhenPathDoesNotExist", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/non-existent"

			// Untracking a non-existent mount should not cause a panic or error.
			assert.NotPanics(t, func() { tracker.UntrackMount(path) }, "Untracking non-existent mount should not panic")
			assert.False(t, tracker.IsMounted(path), "IsMounted should be false for a non-existent mount")
		})
	})

	t.Run("IsMounted_Operations", func(t *testing.T) {
		t.Parallel()

		t.Run("ShouldReturnFalse_WhenPathNotTracked", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/is-mounted-non-existent"

			assert.False(t, tracker.IsMounted(path), "IsMounted should return false for a non-existent mount")
		})
	})

	t.Run("GetMountInfo_Operations", func(t *testing.T) {
		t.Parallel()

		t.Run("ShouldReturnError_WhenPathNotTracked", func(t *testing.T) {
			t.Parallel()
			tracker := NewVolumeTracker()
			path := testBasePath + "/get-info-non-existent"

			_, err := tracker.GetMountInfo(path)
			assert.Error(t, err, "GetMountInfo should return an error for a non-existent mount")
			assert.Equal(t, errFailedFindVolume, err.Error(), "Error message should match expected")
		})
	})

	t.Run("ShouldHandleConcurrentAccess_WithoutRaceConditions", func(t *testing.T) {
		t.Parallel()
		tracker := NewVolumeTracker()
		numGoroutines := 100
		var wg sync.WaitGroup

		// Concurrent writes (TrackMount and UntrackMount)
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				path := fmt.Sprintf(testBasePath+"/concurrent_%d", idx)
				allocID := fmt.Sprintf(testAllocationID+"_%d", idx)
				containerID := fmt.Sprintf(testContainerID+"_%d", idx)

				tracker.TrackMount(path, allocID, containerID)

				if idx%2 == 0 { // Untrack some of them
					tracker.UntrackMount(path)
				}
			}(i)
		}
		wg.Wait()

		// Concurrent reads (IsMounted and GetMountInfo)
		// After writes have settled, check consistency.
		reads := 0
		muReads := sync.Mutex{}
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				path := fmt.Sprintf(testBasePath+"/concurrent_%d", idx)
				if tracker.IsMounted(path) {
					info, err := tracker.GetMountInfo(path)
					// Only assert if it was expected to be mounted (not untracked)
					if idx%2 != 0 {
						assert.NoError(t, err)
						assert.Equal(t, fmt.Sprintf(testAllocationID+"_%d", idx), info.AllocationID)
						muReads.Lock()
						reads++
						muReads.Unlock()
					}
				} else {
					// Should be unmounted if idx % 2 == 0
					assert.True(t, idx%2 == 0, "Path %s should have been unmounted but IsMounted is false", path)
				}
			}(i)
		}
		wg.Wait()

		// Check final count of mounts
		expectedMounts := numGoroutines / 2
		if numGoroutines%2 != 0 { // if numGoroutines is odd, one more will be mounted
			expectedMounts = numGoroutines/2 + 1
		}
		assert.Equal(t, expectedMounts, len(tracker.mounts), "Number of final mounts does not match expected after concurrency test")
		assert.Equal(t, expectedMounts, reads, "Number of successful reads does not match expected mounted items")
	})
}
