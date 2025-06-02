// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/db/clover"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

const (
	// Allocation IDs for tests
	stableDiffusionAlloc = "stable-diffusion"
	llmInferenceAlloc    = "llm-inference"
	videoTranscodeAlloc  = "video-transcode"
	nginxAlloc           = "nginx"
)

// Define a struct to hold the resource manager dependencies
type resManagerDeps struct {
	repos    resources.ManagerRepos
	hardware types.HardwareManager
}

// setupDefaultManagerDeps creates a new DefaultManager with in-memory repositories for testing
// multiplier is used to scale the resources for the mock hardware manager
func setupDefaultManagerDeps(t *testing.T, multiplier uint32) resManagerDeps {
	t.Helper()

	if multiplier == 0 {
		multiplier = 1 // Default to 1 if not specified
	}

	db, err := clover.NewMemDB(
		[]string{
			"onboarded_resources",
			"resource_allocation",
		})
	require.NoError(t, err)

	onboardedRepo := clover.NewGenericEntityRepository[types.OnboardedResources](db)
	allocRepo := clover.NewGenericRepository[types.ResourceAllocation](db)

	// Create resources with proper unit conversions
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	ramSize, _ := convert.ParseBytesWithDefaultUnit(8*multiplier, "GiB")
	diskSize, _ := convert.ParseBytesWithDefaultUnit(100*multiplier, "GiB")

	machineResources := types.MachineResources{
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4 * float32(multiplier),
				ClockSpeed: cpuClockSpeed,
			},
			RAM: types.RAM{
				Size: ramSize,
			},
			Disk: types.Disk{
				Size: diskSize,
			},
			GPUs: types.GPUs{}, // No GPUs
		},
	}

	freeResources := machineResources.Resources

	mockHW := hardware.NewMockHardwareManager(
		machineResources,
		freeResources,
		types.Resources{}, // No usage initially
	)

	// Create manager repos
	repos := resources.ManagerRepos{
		OnboardedResources: onboardedRepo,
		ResourceAllocation: allocRepo,
	}

	return resManagerDeps{
		repos:    repos,
		hardware: mockHW,
	}
}

func TestNewResourceManager(t *testing.T) {
	t.Parallel()
	// Test with valid hardware manager
	deps := setupDefaultManagerDeps(t, 1)
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test with nil hardware manager
	repos := resources.ManagerRepos{}
	nilManager, err := resources.NewResourceManager(repos, nil)
	require.Error(t, err)
	require.Nil(t, nilManager)
}

func TestUpdateAndGetOnboardedResources(t *testing.T) {
	t.Parallel()
	deps := setupDefaultManagerDeps(t, 1)
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	ctx := context.Background()

	// Define base resources
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	ramSize, _ := convert.ParseBytesWithDefaultUnit(8, "GiB")
	diskSize, _ := convert.ParseBytesWithDefaultUnit(100, "GiB")

	onboardedResources := types.Resources{
		CPU: types.CPU{
			Cores:      4,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: ramSize,
		},
		Disk: types.Disk{
			Size: diskSize,
		},
	}

	// Update onboarded resources
	err = manager.UpdateOnboardedResources(ctx, onboardedResources)
	require.NoError(t, err)

	// Verify onboarded resources were updated
	onboarded, err := manager.GetOnboardedResources(ctx)
	require.NoError(t, err)
	require.True(t, onboarded.Resources.Equal(onboardedResources))

	// Verify free resources match onboarded resources (no allocations yet)
	free, err := manager.GetFreeResources(ctx)
	require.NoError(t, err)
	require.True(t, free.Resources.Equal(onboardedResources))
}

func TestCommitAndAllocateResources(t *testing.T) {
	t.Parallel()
	deps := setupDefaultManagerDeps(t, 1)
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	ctx := context.Background()

	// Define base resources and update onboarded resources
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	ramSize, _ := convert.ParseBytesWithDefaultUnit(8, "GiB")
	diskSize, _ := convert.ParseBytesWithDefaultUnit(100, "GiB")

	onboardedResources := types.Resources{
		CPU: types.CPU{
			Cores:      4,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: ramSize,
		},
		Disk: types.Disk{
			Size: diskSize,
		},
	}
	err = manager.UpdateOnboardedResources(ctx, onboardedResources)
	require.NoError(t, err)

	// Define smaller resources for commitment
	smallRAMSize, _ := convert.ParseBytesWithDefaultUnit(2, "GiB")
	smallDiskSize, _ := convert.ParseBytesWithDefaultUnit(20, "GiB")
	smallResources := types.Resources{
		CPU: types.CPU{
			Cores:      1,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: smallRAMSize,
		},
		Disk: types.Disk{
			Size: smallDiskSize,
		},
	}

	// Test committing resources
	commitment := types.CommittedResources{
		AllocationID: nginxAlloc,
		Resources:    smallResources,
	}
	err = manager.CommitResources(ctx, commitment)
	require.NoError(t, err)

	// Verify resources are committed
	committed, err := manager.IsCommitted(nginxAlloc)
	require.NoError(t, err)
	require.True(t, committed)

	// Verify free resources are reduced
	free, err := manager.GetFreeResources(ctx)
	require.NoError(t, err)

	expectedFree := onboardedResources
	err = expectedFree.Subtract(smallResources)
	require.NoError(t, err)
	require.True(t, free.Resources.Equal(expectedFree))

	// Test allocating resources
	err = manager.AllocateResources(ctx, nginxAlloc)
	require.NoError(t, err)

	// Verify resources are allocated
	allocated, err := manager.IsAllocated(nginxAlloc)
	require.NoError(t, err)
	require.True(t, allocated)

	// Verify resources are no longer committed
	committed, err = manager.IsCommitted(nginxAlloc)
	require.NoError(t, err)
	require.False(t, committed)

	// Verify total allocation
	totalAllocation, err := manager.GetTotalAllocation()
	require.NoError(t, err)
	require.True(t, totalAllocation.Equal(smallResources))

	// Test deallocating resources
	err = manager.DeallocateResources(ctx, nginxAlloc)
	require.NoError(t, err)

	// Verify resources are no longer allocated
	allocated, err = manager.IsAllocated(nginxAlloc)
	require.NoError(t, err)
	require.False(t, allocated)

	// Verify total allocation is back to zero
	totalAllocation, err = manager.GetTotalAllocation()
	require.NoError(t, err)
	require.True(t, totalAllocation.Equal(types.Resources{}))
}

func TestResourceManagerErrorCases(t *testing.T) {
	deps := setupDefaultManagerDeps(t, 1)
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	ctx := context.Background()

	// Define base resources and update onboarded resources
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	ramSize, _ := convert.ParseBytesWithDefaultUnit(8, "GiB")
	diskSize, _ := convert.ParseBytesWithDefaultUnit(100, "GiB")

	onboardedResources := types.Resources{
		CPU: types.CPU{
			Cores:      4,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: ramSize,
		},
		Disk: types.Disk{
			Size: diskSize,
		},
	}
	err = manager.UpdateOnboardedResources(ctx, onboardedResources)
	require.NoError(t, err)

	t.Run("Test committing resources with invalid allocation ID", func(t *testing.T) {
		// Test committing resources with empty allocation ID
		invalidCommitment := types.CommittedResources{
			AllocationID: "", // Empty allocation ID
			Resources:    onboardedResources,
		}
		err = manager.CommitResources(ctx, invalidCommitment)
		require.Error(t, err)
	})

	t.Run("Test committing resources beyond capacity", func(t *testing.T) {
		largeCPUCores := float32(10)
		largeRAMSize, _ := convert.ParseBytesWithDefaultUnit(16, "GiB")
		largeDiskSize, _ := convert.ParseBytesWithDefaultUnit(200, "GiB")

		largeResources := types.Resources{
			CPU: types.CPU{
				Cores:      largeCPUCores,
				ClockSpeed: cpuClockSpeed,
			},
			RAM: types.RAM{
				Size: largeRAMSize,
			},
			Disk: types.Disk{
				Size: largeDiskSize,
			},
		}

		largeCommitment := types.CommittedResources{
			AllocationID: stableDiffusionAlloc,
			Resources:    largeResources,
		}
		err = manager.CommitResources(ctx, largeCommitment)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrNoFreeResources)
	})

	t.Run("Allocate/Committing operations with unknown allocation", func(t *testing.T) {
		// Test allocating resources that aren't committed
		err = manager.AllocateResources(ctx, "unknown-allocation")
		require.Error(t, err)
		require.ErrorIs(t, err, resources.ErrResourcesNotCommitted)

		// Test deallocating resources that aren't allocated
		err = manager.DeallocateResources(ctx, "unknown-allocation")
		require.Error(t, err)
		require.ErrorIs(t, err, resources.ErrResourcesNotAllocated)

		// Test uncommitting resources that aren't committed
		err = manager.UncommitResources(ctx, "unknown-allocation")
		require.Error(t, err)
		require.ErrorIs(t, err, resources.ErrResourcesNotCommitted)
	})

	t.Run("Test committing duplicate resources", func(t *testing.T) {
		smallRAMSize, _ := convert.ParseBytesWithDefaultUnit(2, "GiB")
		smallDiskSize, _ := convert.ParseBytesWithDefaultUnit(20, "GiB")
		smallResources := types.Resources{
			CPU: types.CPU{
				Cores:      1,
				ClockSpeed: cpuClockSpeed,
			},
			RAM: types.RAM{
				Size: smallRAMSize,
			},
			Disk: types.Disk{
				Size: smallDiskSize,
			},
		}

		// Commit resources first time
		commitment := types.CommittedResources{
			AllocationID: llmInferenceAlloc,
			Resources:    smallResources,
		}
		err = manager.CommitResources(ctx, commitment)
		require.NoError(t, err)

		// Test committing the same resources again - should fail
		err = manager.CommitResources(ctx, commitment)
		require.Error(t, err)
		require.ErrorIs(t, err, resources.ErrResourcesAlreadyCommitted)

		// Test uncommit resources - happy path
		err = manager.UncommitResources(ctx, llmInferenceAlloc)
		require.NoError(t, err)

		// Verify resources are no longer committed
		committed, err := manager.IsCommitted(llmInferenceAlloc)
		require.NoError(t, err)
		require.False(t, committed)
	})
}

// TestUnableToDecreaseOnboardedResources caller can not decrease
// onboarded resources if it's being used by allocations
func TestUnableToDecreaseOnboardedResources(t *testing.T) {
	t.Parallel()
	deps := setupDefaultManagerDeps(t, 1)
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	ctx := context.Background()

	// Define initial resources and update onboarded resources
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	initialRAMSize, _ := convert.ParseBytesWithDefaultUnit(8, "GiB")
	initialDiskSize, _ := convert.ParseBytesWithDefaultUnit(100, "GiB")

	onboardedResources := types.Resources{
		CPU: types.CPU{
			Cores:      4,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: initialRAMSize,
		},
		Disk: types.Disk{
			Size: initialDiskSize,
		},
	}
	err = manager.UpdateOnboardedResources(ctx, onboardedResources)
	require.NoError(t, err)

	// Commit and allocate some resources
	smallRAMSize, _ := convert.ParseBytesWithDefaultUnit(2, "GiB")
	smallDiskSize, _ := convert.ParseBytesWithDefaultUnit(20, "GiB")
	allocatedResources := types.Resources{
		CPU: types.CPU{
			Cores:      2,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: smallRAMSize,
		},
		Disk: types.Disk{
			Size: smallDiskSize,
		},
	}

	commitment := types.CommittedResources{
		AllocationID: videoTranscodeAlloc,
		Resources:    allocatedResources,
	}
	err = manager.CommitResources(ctx, commitment)
	require.NoError(t, err)

	err = manager.AllocateResources(ctx, videoTranscodeAlloc)
	require.NoError(t, err)

	// Try to update onboarded resources to less than what's already allocated
	smallerRAMSize, _ := convert.ParseBytesWithDefaultUnit(1, "GiB")
	smallerResources := types.Resources{
		CPU: types.CPU{
			Cores:      1, // Less than allocated
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: smallerRAMSize, // Less than allocated
		},
		Disk: types.Disk{
			Size: smallDiskSize, // Same as allocated
		},
	}

	// This should fail because we can't reduce resources below what's already allocated
	err = manager.UpdateOnboardedResources(ctx, smallerResources)
	require.Error(t, err)

	// Verify onboarded resources haven't changed
	onboarded, err := manager.GetOnboardedResources(ctx)
	require.NoError(t, err)
	require.True(t, onboarded.Resources.Equal(onboardedResources))
}

// TestConcurrency tests a mix of concurrent operations (commit, allocate, deallocate)
func TestConcurrency(t *testing.T) {
	t.Parallel()
	deps := setupDefaultManagerDeps(t, 1000) // Use a larger multiplier for concurrency tests
	manager, err := resources.NewResourceManager(deps.repos, deps.hardware)
	require.NoError(t, err)
	ctx := context.Background()

	// Get machine resources from the hardware manager
	machineResources, err := deps.hardware.GetMachineResources()
	require.NoError(t, err)

	// Use the machine resources as onboarded resources
	onboardedResources := machineResources.Resources
	err = manager.UpdateOnboardedResources(ctx, onboardedResources)
	require.NoError(t, err)

	// Define smaller resources for allocation
	cpuClockSpeed, _ := convert.ParseSIWithDefaultUnit(2.5, "GHz")
	smallRAMSize, _ := convert.ParseBytesWithDefaultUnit(1, "GiB")
	smallDiskSize, _ := convert.ParseBytesWithDefaultUnit(2, "GiB")
	smallResources := types.Resources{
		CPU: types.CPU{
			Cores:      1,
			ClockSpeed: cpuClockSpeed,
		},
		RAM: types.RAM{
			Size: smallRAMSize,
		},
		Disk: types.Disk{
			Size: smallDiskSize,
		},
	}

	// Number of concurrent operations per type
	numConcurrent := 10

	// Create allocation IDs for each operation type
	commitIDs := make([]string, numConcurrent)
	allocateIDs := make([]string, numConcurrent)
	deallocateIDs := make([]string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		commitIDs[i] = fmt.Sprintf("commit-alloc-%d", i)
		allocateIDs[i] = fmt.Sprintf("allocate-alloc-%d", i)
		deallocateIDs[i] = fmt.Sprintf("deallocate-alloc-%d", i)
	}

	// Pre-commit resources for allocate operations
	for _, id := range allocateIDs {
		commitment := types.CommittedResources{
			AllocationID: id,
			Resources:    smallResources,
		}
		err = manager.CommitResources(ctx, commitment)
		require.NoError(t, err)
	}

	// Pre-allocate resources for deallocate operations
	for _, id := range deallocateIDs {
		commitment := types.CommittedResources{
			AllocationID: id,
			Resources:    smallResources,
		}
		err = manager.CommitResources(ctx, commitment)
		require.NoError(t, err)

		err = manager.AllocateResources(ctx, id)
		require.NoError(t, err)
	}

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(numConcurrent * 3) // 3 types of operations

	// Track errors from goroutines
	errChan := make(chan error, numConcurrent*3)

	// Launch concurrent commit operations
	for i, id := range commitIDs {
		go func(idx int, allocID string) {
			defer wg.Done()

			commitment := types.CommittedResources{
				AllocationID: allocID,
				Resources:    smallResources,
			}

			if err := manager.CommitResources(ctx, commitment); err != nil {
				errChan <- fmt.Errorf("commit goroutine %d failed: %w", idx, err)
				return
			}
		}(i, id)
	}

	// Launch concurrent allocate operations
	for i, id := range allocateIDs {
		go func(idx int, allocID string) {
			defer wg.Done()

			if err := manager.AllocateResources(ctx, allocID); err != nil {
				errChan <- fmt.Errorf("allocate goroutine %d failed: %w", idx, err)
				return
			}
		}(i, id)
	}

	// Launch concurrent deallocate operations
	for i, id := range deallocateIDs {
		go func(idx int, allocID string) {
			defer wg.Done()

			if err := manager.DeallocateResources(ctx, allocID); err != nil {
				errChan <- fmt.Errorf("deallocate goroutine %d failed: %w", idx, err)
				return
			}
		}(i, id)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check for errors
	errs := make([]error, 0, numConcurrent*3)
	for err := range errChan {
		errs = append(errs, err)
	}
	require.Empty(t, errs, "concurrent mixed operations failed: %v", errs)

	// Verify final state
	// 1. All commit operations should have succeeded
	for _, id := range commitIDs {
		committed, err := manager.IsCommitted(id)
		require.NoError(t, err)
		require.True(t, committed, "ID %s should be committed", id)
	}

	// 2. All allocate operations should have succeeded
	for _, id := range allocateIDs {
		allocated, err := manager.IsAllocated(id)
		require.NoError(t, err)
		require.True(t, allocated, "ID %s should be allocated", id)
	}

	// 3. All deallocate operations should have succeeded
	for _, id := range deallocateIDs {
		allocated, err := manager.IsAllocated(id)
		require.NoError(t, err)
		require.False(t, allocated, "ID %s should not be allocated", id)
	}

	// Clean up - uncommit and deallocate remaining resources
	for _, id := range commitIDs {
		err = manager.UncommitResources(ctx, id)
		require.NoError(t, err)
	}

	for _, id := range allocateIDs {
		err = manager.DeallocateResources(ctx, id)
		require.NoError(t, err)
	}
}
