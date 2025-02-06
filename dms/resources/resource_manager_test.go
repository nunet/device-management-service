// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/db/repositories"
	cloverRepo "gitlab.com/nunet/device-management-service/db/repositories/clover"
	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewResourceManager(t *testing.T) {
	t.Parallel()

	mockDB, err := cloverRepo.NewMemDB(
		[]string{
			"onboarded_resources",
			"resource_allocation",
		},
	)
	require.NoError(t, err)
	defer mockDB.Close()

	repos := setupManagerRepos(mockDB)

	hm := hardware.NewHardwareManager()
	rm, err := NewResourceManager(repos, hm)
	require.NotNil(t, rm)
	require.NoError(t, err)
}

func TestDefaultManager_CommitResources(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to commit the resources when there is an availability", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.CommittedResources{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		err = rm.CommitResources(context.Background(), demand)
		require.NoError(t, err)

		// Check if the committed resources are stored in the map
		demandFromMap, ok := rm.store.committedResources[demand.AllocationID]
		require.True(t, ok)
		assertResources(t, demand.Resources, demandFromMap.Resources)

		isCommitted, err := rm.IsCommitted(demand.AllocationID)
		require.NoError(t, err)
		require.True(t, isCommitted)
	})

	t.Run("Must return an error when resources are already committed for the allocation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		demand := types.CommittedResources{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 1024},
			},
		}
		rm.store.withCommittedLock(func() {
			rm.store.committedResources[demand.AllocationID] = &types.CommittedResources{
				Resources:    demand.Resources,
				AllocationID: demand.AllocationID,
			}
		})

		err = rm.CommitResources(context.Background(), demand)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources already committed for allocation")
	})

	t.Run("Must return an error when there are insufficient resources to commit", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		// Table tests for insufficient resources
		tests := []struct {
			name   string
			demand types.CommittedResources
			error  bool
		}{
			{
				name: "CPU allocations exceeds",
				demand: types.CommittedResources{
					AllocationID: "alloc1",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      6,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 512},
					},
				},
				error: true,
			},
			{
				name: "RAM allocations exceeds",
				demand: types.CommittedResources{
					AllocationID: "alloc1",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      3,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 4096},
						Disk: types.Disk{Size: 512},
					},
				},
				error: true,
			},
			{
				name: "Disk allocations exceeds",
				demand: types.CommittedResources{
					AllocationID: "alloc1",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      3,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 2048},
					},
				},
				error: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := rm.CommitResources(context.Background(), tt.demand)
				if tt.error {
					require.Error(t, err)
					require.Contains(t, err.Error(), "no free resources: error subtracting")
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("Must be able to update the free resources after committing resources", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.CommittedResources{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		err = rm.CommitResources(context.Background(), demand)
		require.NoError(t, err)

		// Check if the free resources are updated in the store
		updatedFreeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		err = onboardedResources.Resources.Subtract(demand.Resources)
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		require.NoError(t, err)
		assertResources(t, expectedFreeResources.Resources, updatedFreeResources.Resources)
	})
}

func TestDefaultManager_ReleaseCommittedResources(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to release the resources and update free resources", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.CommittedResources{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}
		err = rm.CommitResources(context.Background(), demand)
		require.NoError(t, err)

		err = rm.UncommitResources(context.Background(), demand.AllocationID)

		require.NoError(t, err)

		// Check if the committed resources were removed from the map
		_, ok := rm.store.committedResources[demand.AllocationID]
		require.False(t, ok)

		// check if the resources are added back to the free resources
		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResourcesFromDB.Resources)
	})

	t.Run("Must return an error when resources are not pre-allocated for the allocation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		err = rm.UncommitResources(context.Background(), "alloc1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources not committed for allocation")
	})
}

func TestDefaultManager_AllocateResources(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to allocate the resources when there is an availability", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 2048},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      1,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 512},
				Disk: types.Disk{Size: 512},
			},
		}
		err = rm.CommitResources(context.Background(), types.CommittedResources{
			AllocationID: demand.AllocationID,
			Resources:    demand.Resources,
		})
		require.NoError(t, err)

		err = rm.AllocateResources(context.Background(), demand.AllocationID)
		require.NoError(t, err)

		// Check if the allocations is stored in the map
		demandFromMap, ok := rm.store.allocations[demand.AllocationID]
		require.True(t, ok)
		assertResources(t, demand.Resources, demandFromMap.Resources)

		isAllocated, err := rm.IsAllocated(demand.AllocationID)
		require.NoError(t, err)
		require.True(t, isAllocated)
	})

	t.Run("Must return an error when resources are already allocated for the allocation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockDB, err := cloverRepo.NewMemDB(
			[]string{"resource_allocation", "onboarded_resources"},
		)
		require.NoError(t, err)
		t.Cleanup(func() {
			ctrl.Finish()
			mockDB.Close()
		})

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 2048},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      1,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 1024},
			},
		}
		err = rm.CommitResources(context.Background(), types.CommittedResources{
			AllocationID: demand.AllocationID,
			Resources:    demand.Resources,
		})
		require.NoError(t, err)
		rm.store.withAllocationsLock(func() {
			rm.store.allocations[demand.AllocationID] = demand
		})

		err = rm.AllocateResources(context.Background(), demand.AllocationID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources already allocated for allocation")
	})

	t.Run("Must be able to update the free resources after allocating resources", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		err = rm.CommitResources(context.Background(), types.CommittedResources{Resources: demand.Resources, AllocationID: demand.AllocationID})
		require.NoError(t, err)

		err = rm.AllocateResources(context.Background(), demand.AllocationID)
		require.NoError(t, err)

		// Check if the free resources are updated in the store
		updatedFreeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		err = onboardedResources.Resources.Subtract(demand.Resources)
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		require.NoError(t, err)
		assertResources(t, expectedFreeResources.Resources, updatedFreeResources.Resources)
	})
}

func TestDefaultManager_DeallocateResources(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to deallocate the resources and update free resources", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}
		err = rm.CommitResources(context.Background(), types.CommittedResources{Resources: demand.Resources, AllocationID: demand.AllocationID})
		require.NoError(t, err)
		err = rm.AllocateResources(context.Background(), demand.AllocationID)
		require.NoError(t, err)

		err = rm.DeallocateResources(context.Background(), demand.AllocationID)
		require.NoError(t, err)

		// Check if the allocations is removed from the map
		_, ok := rm.store.allocations[demand.AllocationID]
		require.False(t, ok)

		// check if the resources are added back to the free resources
		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResourcesFromDB.Resources)
	})

	t.Run("Must return an error when resources are not allocated for the allocation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{}, // no database used in deallocation only in-mem store
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		err = rm.DeallocateResources(context.Background(), "alloc1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources not allocated for allocation")
	})
}

func TestDefaultManager_OnboardedResources(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	t.Run("Must be able to get onboarded resources", func(t *testing.T) {
		t.Parallel()
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		onboardedResourcesFromManager, err := rm.GetOnboardedResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, onboardedResourcesFromManager.Resources)
	})

	t.Run("Must be able to update onboarded resources both in store and db", func(t *testing.T) {
		t.Parallel()
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		newOnboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      6,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 3072},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), newOnboardedResources.Resources)
		require.NoError(t, err)

		// Check if the onboarded resources are updated in the database
		onboardedResourcesFromDB := getOnboardedResourcesFromDB(repos.OnboardedResources, t)
		assertResources(t, newOnboardedResources.Resources, onboardedResourcesFromDB.Resources)

		// Check if the onboarded resources are updated in the store
		onboardedResourcesFromStore := rm.store.onboardedResources
		require.NotNil(t, onboardedResourcesFromStore)
		assertResources(t, newOnboardedResources.Resources, onboardedResourcesFromStore.Resources)
	})

	t.Run("Must be able to get onboarded resources from DB if not in store", func(t *testing.T) {
		t.Parallel()
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		// Set the onboarded resources in the store to nil
		_ = rm.store.withOnboardedLock(func() error {
			rm.store.onboardedResources = nil
			return nil
		})

		onboardedResourcesFromManager, err := rm.GetOnboardedResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, onboardedResourcesFromManager.Resources)
	})
}

func TestDefaultManager_FreeResources(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to get free resources", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)
		freeResources := types.FreeResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		updatedFreeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, freeResources.Resources, updatedFreeResources.Resources)
	})

	t.Run("Must be able to update free resources", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			AllocationID: "alloc1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}
		err = rm.CommitResources(context.Background(), types.CommittedResources{Resources: demand.Resources, AllocationID: demand.AllocationID})
		require.NoError(t, err)
		err = rm.AllocateResources(context.Background(), demand.AllocationID)
		require.NoError(t, err)

		updatedFreeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)

		err = onboardedResources.Resources.Subtract(demand.Resources)
		expectedFreeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		require.NoError(t, err)
		assertResources(t, expectedFreeResources.Resources, updatedFreeResources.Resources)

		// Check if the free resources are updated in the store
		assertResources(t, expectedFreeResources.Resources, updatedFreeResources.Resources)
	})
}

func TestDefaultManager_GetTotalAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Must be able to get total allocations", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      7,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 3064},
				Disk: types.Disk{Size: 2048},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demands := []types.ResourceAllocation{
			{
				AllocationID: "alloc1",
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      3,
						ClockSpeed: 10000,
					},
					RAM:  types.RAM{Size: 1024},
					Disk: types.Disk{Size: 512},
				},
			},
			{
				AllocationID: "alloc2",
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 10000,
					},
					RAM:  types.RAM{Size: 1024},
					Disk: types.Disk{Size: 1024},
				},
			},
		}

		var totalDemand types.Resources
		for _, demand := range demands {
			err = rm.CommitResources(context.Background(), types.CommittedResources{AllocationID: demand.AllocationID, Resources: demand.Resources})
			require.NoError(t, err)

			err = rm.AllocateResources(context.Background(), demand.AllocationID)
			require.NoErrorf(t, err, "failed to allocate resources for allocation %s", demand.AllocationID)

			err = totalDemand.Add(demand.Resources)
			require.NoError(t, err)
		}

		actualDemand, err := rm.GetTotalAllocation()
		require.NoError(t, err)
		assertResources(t, totalDemand, actualDemand)
	})

	t.Run("Must be able to get total allocations from DB", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := cloverRepo.NewMemDB(
			[]string{
				"onboarded_resources",
				"resource_allocation",
			},
		)
		require.NoError(t, err)
		defer mockDB.Close()

		repos := setupManagerRepos(mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      7,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 3064},
				Disk: types.Disk{Size: 2048},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		demands := []types.ResourceAllocation{
			{
				AllocationID: "alloc1",
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      3,
						ClockSpeed: 10000,
					},
					RAM:  types.RAM{Size: 1024},
					Disk: types.Disk{Size: 512},
				},
			},
			{
				AllocationID: "alloc2",
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 10000,
					},
					RAM:  types.RAM{Size: 1024},
					Disk: types.Disk{Size: 1024},
				},
			},
		}

		var totalDemand types.Resources
		for _, demand := range demands {
			err = rm.CommitResources(context.Background(), types.CommittedResources{AllocationID: demand.AllocationID, Resources: demand.Resources})
			require.NoError(t, err)

			err = rm.AllocateResources(context.Background(), demand.AllocationID)
			require.NoErrorf(t, err, "failed to allocate resources for allocation %s", demand.AllocationID)

			err = totalDemand.Add(demand.Resources)
			require.NoError(t, err)
		}

		actualDemand, err := rm.GetTotalAllocation()
		require.NoError(t, err)
		assertResources(t, totalDemand, actualDemand)
	})
}

func TestDefaultManager_Concurrency(t *testing.T) {
	t.Parallel()
	const numGoroutines = 25

	t.Run("Allocate resources then deallocate them", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		onboardedResourcesRepo.EXPECT().Save(gomock.Any(), onboardedResources).Return(onboardedResources, nil)
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		var (
			wg    sync.WaitGroup
			mutex sync.Mutex
		)
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()
				demand := types.ResourceAllocation{
					AllocationID: fmt.Sprintf("alloc%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}
				err := rm.CommitResources(context.Background(), types.CommittedResources{AllocationID: demand.AllocationID, Resources: demand.Resources})
				require.NoError(t, err)

				mutex.Lock()
				resourceAllocationRepo.EXPECT().Create(gomock.Any(), demand).Return(demand, nil)
				mutex.Unlock()
				err = rm.AllocateResources(context.Background(), demand.AllocationID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are allocated for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocationID := fmt.Sprintf("alloc%d", i)
			demand, ok := rm.store.allocations[allocationID]
			require.True(t, ok)
			require.Equal(t, allocationID, demand.AllocationID)
		}

		// Check if the free resources are updated correctly
		freeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50 - 0.1*numGoroutines,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048 - 10*numGoroutines},
				Disk: types.Disk{Size: 1024 - 10*numGoroutines},
			},
		}
		assertResources(t, expectedFreeResources.Resources, freeResources.Resources)
		// Deallocate the resources
		for i := 0; i < numGoroutines; i++ {
			allocationID := fmt.Sprintf("alloc%d", i)
			wg.Add(1)
			go func() {
				defer wg.Done()
				mutex.Lock()
				resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
				resourceAllocationRepo.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, query repositories.Query[types.ResourceAllocation]) (types.ResourceAllocation, error) {
					return types.ResourceAllocation{
						BaseDBModel: types.BaseDBModel{
							ID: query.Conditions[0].Value.(string),
						},
					}, nil
				})
				resourceAllocationRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
				mutex.Unlock()
				err := rm.DeallocateResources(context.Background(), allocationID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are deallocated for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocationID := fmt.Sprintf("alloc%d", i)
			_, ok := rm.store.allocations[allocationID]
			require.False(t, ok)
		}

		// Check if the free resources are updated correctly
		freeResources, err = rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResources.Resources)
	})

	t.Run("Concurrent allocation and deallocation", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}
		onboardedResourcesRepo.EXPECT().Save(gomock.Any(), onboardedResources).Return(onboardedResources, nil).Times(1)
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		var wg sync.WaitGroup
		var mutex sync.Mutex
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()
				demand := types.ResourceAllocation{
					AllocationID: fmt.Sprintf("alloc%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}

				err := rm.CommitResources(context.Background(), types.CommittedResources{AllocationID: demand.AllocationID, Resources: demand.Resources})
				require.NoError(t, err)

				mutex.Lock()
				// allocate expectations
				resourceAllocationRepo.EXPECT().Create(gomock.Any(), demand).Return(demand, nil).Times(1)

				// deallocate expectations
				resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{}).Times(1)
				resourceAllocationRepo.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, query repositories.Query[types.ResourceAllocation]) (types.ResourceAllocation, error) {
					return types.ResourceAllocation{
						BaseDBModel: types.BaseDBModel{
							ID: query.Conditions[0].Value.(string),
						},
					}, nil
				}).Times(1)
				resourceAllocationRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				mutex.Unlock()

				err = rm.AllocateResources(context.Background(), demand.AllocationID)
				require.NoError(t, err)

				// Deallocate the resources
				err = rm.DeallocateResources(context.Background(), demand.AllocationID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are deallocated for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocID := fmt.Sprintf("allocation%d", i)
			_, ok := rm.store.allocations[allocID]
			require.False(t, ok)
		}

		// Check if the free resources are updated correctly
		freeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResources.Resources)
	})

	t.Run("Commit resources then uncommit them", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048},
				Disk: types.Disk{Size: 1024},
			},
		}

		onboardedResourcesRepo.EXPECT().Save(gomock.Any(), onboardedResources).Return(onboardedResources, nil)
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()
				demand := types.CommittedResources{
					AllocationID: fmt.Sprintf("alloc%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}
				err := rm.CommitResources(context.Background(), demand)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are committed for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocID := fmt.Sprintf("alloc%d", i)
			demand, ok := rm.store.committedResources[allocID]
			require.True(t, ok)
			require.Equal(t, allocID, demand.AllocationID)
		}

		// Check if the free resources are updated correctly
		freeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50 - 0.1*numGoroutines,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 2048 - 10*numGoroutines},
				Disk: types.Disk{Size: 1024 - 10*numGoroutines},
			},
		}
		assertResources(t, expectedFreeResources.Resources, freeResources.Resources)
		// Uncommit the resources
		for i := 0; i < numGoroutines; i++ {
			allocID := fmt.Sprintf("alloc%d", i)
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := rm.UncommitResources(context.Background(), allocID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are uncommitted for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocID := fmt.Sprintf("alloc%d", i)
			_, ok := rm.store.allocations[allocID]
			require.False(t, ok)
		}
	})

	t.Run("Concurrent commit and uncommit", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      50,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1000000},
				Disk: types.Disk{Size: 1000000},
			},
		}
		onboardedResourcesRepo.EXPECT().Save(gomock.Any(), onboardedResources).Return(onboardedResources, nil).Times(1)
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()
				demand := types.CommittedResources{
					AllocationID: fmt.Sprintf("alloc%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 1},
						Disk: types.Disk{Size: 1},
					},
				}

				err := rm.CommitResources(context.Background(), demand)
				require.NoError(t, err)

				// Deallocate the resources
				err = rm.UncommitResources(context.Background(), demand.AllocationID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are uncommitted for all the allocations
		for i := 0; i < numGoroutines; i++ {
			allocID := fmt.Sprintf("alloc%d", i)
			_, ok := rm.store.committedResources[allocID]
			require.False(t, ok)
		}

		// Check if the free resources are updated correctly
		freeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResources.Resources)
	})
}
