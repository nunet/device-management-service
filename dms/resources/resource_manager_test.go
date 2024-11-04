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

	"gitlab.com/nunet/device-management-service/dms/hardware"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewResourceManager(t *testing.T) {
	t.Parallel()

	mockDB, err := gorm.Open(sqlite.Open("file:test_newResourceManager?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	repos := setupManagerRepos(t, mockDB)

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
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_CommitResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
		err = rm.CommitResources(context.Background(), demand)
		require.NoError(t, err)

		// Check if the committed resources are stored in the map
		demandFromMap, ok := rm.store.committedResources[demand.JobID]
		require.True(t, ok)
		assertResources(t, demand.Resources, demandFromMap.Resources)
	})

	t.Run("Must return an error when resources are already committed for the job", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_CommitResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		demand := types.CommittedResources{
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 1024},
			},
		}
		rm.store.withCommittedLock(func() {
			rm.store.committedResources[demand.JobID] = &types.CommittedResources{
				Resources: demand.Resources,
				JobID:     demand.JobID,
			}
		})

		err = rm.CommitResources(context.Background(), demand)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources already committed for job")
	})

	t.Run("Must return an error when there are insufficient resources to commit", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_CommitResources3?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
					JobID: "job1",
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
					JobID: "job1",
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
					JobID: "job1",
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_CommitResources4?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
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

	t.Run("Must fail when there are not enough resources in the machine", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_CommitResources5?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		// setting a very high unrealistic resources to onboard
		// this is so that the test can skip this check
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 5000,
				},
				RAM:  types.RAM{Size: 50000},
				Disk: types.Disk{Size: 50000},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		// Since this demand is higher than the actual resources on the machine
		// it shouldn't have free resources to allocate
		demand := types.CommittedResources{
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      4,
					ClockSpeed: 4000,
				},
				RAM:  types.RAM{Size: 40000},
				Disk: types.Disk{Size: 40000},
			},
		}

		hm.EXPECT().GetFreeResources().Return(types.Resources{}, fmt.Errorf("no free resources on the machine"))
		err = rm.CommitResources(context.Background(), demand)
		require.ErrorContains(t, err, "no free resources on the machine")
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_ReleaseResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}
		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)

		err = rm.CommitResources(context.Background(), demand)
		require.NoError(t, err)

		err = rm.UnCommittedResources(context.Background(), demand.JobID)

		require.NoError(t, err)

		// Check if the committed resources were removed from the map
		_, ok := rm.store.committedResources[demand.JobID]
		require.False(t, ok)

		// check if the resources are added back to the free resources
		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResourcesFromDB.Resources)
	})

	t.Run("Must return an error when resources are not pre-allocated for the job", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_ReleaseCommittedResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		err = rm.UnCommittedResources(context.Background(), "job1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources not committed for job")
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
		err = rm.AllocateResources(context.Background(), demand)
		require.NoError(t, err)

		// Check if the allocations is stored in the map
		demandFromMap, ok := rm.store.allocations[demand.JobID]
		require.True(t, ok)
		assertResources(t, demand.Resources, demandFromMap.Resources)
	})

	t.Run("Must return an error when resources are already allocated for the job", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		demand := types.ResourceAllocation{
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 1024},
			},
		}
		rm.store.withAllocationsLock(func() {
			rm.store.allocations[demand.JobID] = demand
		})

		err = rm.AllocateResources(context.Background(), demand)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources already allocated for job")
	})

	t.Run("Must return an error when there are insufficient resources to allocate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources3?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			demand types.ResourceAllocation
			error  bool
		}{
			{
				name: "CPU allocations exceeds",
				demand: types.ResourceAllocation{
					JobID: "job1",
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
				demand: types.ResourceAllocation{
					JobID: "job1",
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
				demand: types.ResourceAllocation{
					JobID: "job1",
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
				err := rm.AllocateResources(context.Background(), tt.demand)
				if tt.error {
					require.Error(t, err)
					require.Contains(t, err.Error(), "no free resources: error subtracting")
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("Must be able to update the free resources after allocating resources", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources4?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
		err = rm.AllocateResources(context.Background(), demand)
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

	t.Run("Must fail when there are not enough resources in the machine", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources6?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		// setting a very high unrealistic resources to onboard
		// this is so that the test can skip this check
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 5000,
				},
				RAM:  types.RAM{Size: 50000},
				Disk: types.Disk{Size: 50000},
			},
		}

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources.Resources)
		require.NoError(t, err)

		// Since this demand is higher than the actual resources on the machine
		// it shouldn't have free resources to allocate
		demand := types.ResourceAllocation{
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      4,
					ClockSpeed: 4000,
				},
				RAM:  types.RAM{Size: 40000},
				Disk: types.Disk{Size: 40000},
			},
		}

		hm.EXPECT().GetFreeResources().Return(types.Resources{}, fmt.Errorf("no free resources on the machine"))
		err = rm.AllocateResources(context.Background(), demand)
		require.ErrorContains(t, err, "no free resources on the machine")
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_DeallocateResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}
		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
		err = rm.AllocateResources(context.Background(), demand)
		require.NoError(t, err)

		err = rm.DeallocateResources(context.Background(), demand.JobID)
		require.NoError(t, err)

		// Check if the allocations is removed from the map
		_, ok := rm.store.allocations[demand.JobID]
		require.False(t, ok)

		// check if the resources are added back to the free resources
		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResourcesFromDB.Resources)
	})

	t.Run("Must return an error when resources are not allocated for the job", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_DeallocateResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm, err := NewResourceManager(repos, hm)
		require.NoError(t, err)

		err = rm.DeallocateResources(context.Background(), "job1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "resources not allocated for job")
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_OnboardedResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_OnboardedResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_OnboardedResources3?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_FreeResources1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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

	t.Run("Must be able to up to date free resources", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_FreeResources3?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
			JobID: "job1",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      3,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 1024},
				Disk: types.Disk{Size: 512},
			},
		}

		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
		err = rm.AllocateResources(context.Background(), demand)
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_GetTotalDemand1?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
				JobID: "job1",
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
				JobID: "job2",
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
		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil).Times(len(demands))
		for _, demand := range demands {
			err = rm.AllocateResources(context.Background(), demand)
			require.NoErrorf(t, err, "failed to allocate resources for job %s", demand.JobID)
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
		mockDB, err := gorm.Open(sqlite.Open("file:test_GetTotalDemand2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
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
				JobID: "job1",
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
				JobID: "job2",
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
		hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil).Times(len(demands))
		for _, demand := range demands {
			err = rm.AllocateResources(context.Background(), demand)
			require.NoErrorf(t, err, "failed to allocate resources for job %s", demand.JobID)
			err = totalDemand.Add(demand.Resources)
			require.NoError(t, err)
		}

		// Create a new instance of the manager to test if the allocations are loaded from the DB
		repos = setupManagerRepos(t, mockDB)
		rm, err = NewResourceManager(repos, hm)
		require.NoError(t, err)
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

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), gomock.Any()).Return([]types.ResourceAllocation{}, nil)
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
					JobID: fmt.Sprintf("job%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}
				mutex.Lock()
				rm.hardware.(*MockHardwareManager).EXPECT().GetFreeResources().DoAndReturn(func() (types.Resources, error) {
					return onboardedResources.Resources, nil
				})
				resourceAllocationRepo.EXPECT().Create(gomock.Any(), demand).Return(demand, nil)
				mutex.Unlock()
				err := rm.AllocateResources(context.Background(), demand)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are allocated for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			demand, ok := rm.store.allocations[jobID]
			require.True(t, ok)
			require.Equal(t, jobID, demand.JobID)
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
			jobID := fmt.Sprintf("job%d", i)
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
				err := rm.DeallocateResources(context.Background(), jobID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are deallocated for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			_, ok := rm.store.allocations[jobID]
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
		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), gomock.Any()).Return([]types.ResourceAllocation{}, nil)
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
					JobID: fmt.Sprintf("job%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}

				mutex.Lock()
				// allocate expectations
				rm.hardware.(*MockHardwareManager).EXPECT().GetFreeResources().DoAndReturn(func() (types.Resources, error) {
					return onboardedResources.Resources, nil
				}).Times(1)
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

				err := rm.AllocateResources(context.Background(), demand)
				require.NoError(t, err)

				// Deallocate the resources
				err = rm.DeallocateResources(context.Background(), demand.JobID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are deallocated for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			_, ok := rm.store.allocations[jobID]
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

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), gomock.Any()).Return([]types.ResourceAllocation{}, nil)
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
				demand := types.CommittedResources{
					JobID: fmt.Sprintf("job%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}
				mutex.Lock()
				rm.hardware.(*MockHardwareManager).EXPECT().GetFreeResources().DoAndReturn(func() (types.Resources, error) {
					return onboardedResources.Resources, nil
				})
				mutex.Unlock()
				err := rm.CommitResources(context.Background(), demand)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are committed for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			demand, ok := rm.store.committedResources[jobID]
			require.True(t, ok)
			require.Equal(t, jobID, demand.JobID)
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
			jobID := fmt.Sprintf("job%d", i)
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := rm.UnCommittedResources(context.Background(), jobID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are uncommitted for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			_, ok := rm.store.allocations[jobID]
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
		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), gomock.Any()).Return([]types.ResourceAllocation{}, nil)
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
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()
				demand := types.CommittedResources{
					JobID: fmt.Sprintf("job%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      0.1,
							ClockSpeed: 10000,
						},
						RAM:  types.RAM{Size: 10},
						Disk: types.Disk{Size: 10},
					},
				}

				rm.hardware.(*MockHardwareManager).EXPECT().GetFreeResources().DoAndReturn(func() (types.Resources, error) {
					return onboardedResources.Resources, nil
				})
				err := rm.CommitResources(context.Background(), demand)
				require.NoError(t, err)

				// Deallocate the resources
				err = rm.UnCommittedResources(context.Background(), demand.JobID)
				require.NoError(t, err)
			}()
		}

		wg.Wait()

		// Check if the resources are uncommitted for all the jobs
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			_, ok := rm.store.committedResources[jobID]
			require.False(t, ok)
		}

		// Check if the free resources are updated correctly
		freeResources, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, onboardedResources.Resources, freeResources.Resources)
	})
}
