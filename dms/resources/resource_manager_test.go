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
		rm := newMockResourceManager(repos, hm, t)
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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)

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
		rm := newMockResourceManager(repos, hm, t)
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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)
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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		freeResourcesFromDB := getFreeResourcesFromDB(repos.FreeResources, t)
		err = onboardedResources.Resources.Subtract(demand.Resources)
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		require.NoError(t, err)
		assertResources(t, expectedFreeResources.Resources, freeResourcesFromDB.Resources)
	})

	t.Run("Must fail when there are not enough resources in the machine", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_DefaultManager_AllocateResources5?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm := newMockResourceManager(repos, hm, t)

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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)

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
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)

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
		rm := newMockResourceManager(repos, hm, t)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		err = rm.UpdateOnboardedResources(context.Background(), newOnboardedResources)
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
		rm := newMockResourceManager(repos, hm, t)
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 10000,
				},
				RAM: types.RAM{Size: 2048},
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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
		setUpFreeResources(repos.FreeResources, freeResources, t)

		rm := newMockResourceManager(repos, hm, t)

		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, freeResources.Resources, freeResourcesFromDB.Resources)
		require.NotNil(t, rm.store.freeResources)
	})

	t.Run("Must be able to update the store from DB if empty", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_FreeResources2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)

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
		setUpFreeResources(repos.FreeResources, freeResources, t)

		rm := newMockResourceManager(repos, hm, t)

		freeResourcesFromDB, err := rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, freeResources.Resources, freeResourcesFromDB.Resources)
		require.NotNil(t, rm.store.freeResources)

		// Set the free resources in the store to nil
		rm.store.withFreeLock(func() {
			rm.store.freeResources = nil
		})

		// Get the free resources again
		freeResourcesFromDB, err = rm.GetFreeResources(context.Background())
		require.NoError(t, err)
		assertResources(t, freeResources.Resources, freeResourcesFromDB.Resources)
		require.NotNil(t, rm.store.freeResources)
	})

	t.Run("Must be able to update free resources", func(t *testing.T) {
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
		freeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		setUpFreeResources(repos.FreeResources, freeResources, t)
		hm := NewMockHardwareManager(ctrl)

		rm := newMockResourceManager(repos, hm, t)
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

		freeResourcesFromDB := getFreeResourcesFromDB(repos.FreeResources, t)
		err = onboardedResources.Subtract(demand.Resources)
		require.NoError(t, err)
		expectedFreeResources := types.FreeResources{
			Resources: onboardedResources.Resources,
		}
		require.NoError(t, err)
		assertResources(t, expectedFreeResources.Resources, freeResourcesFromDB.Resources)

		// Check if the free resources are updated in the store
		require.NotNil(t, rm.store.freeResources)
		updatedFreeResources := *rm.store.freeResources
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
		rm := newMockResourceManager(repos, hm, t)

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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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

	t.Run("Must be able to get total allocations from DB if not in store", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})
		mockDB, err := gorm.Open(sqlite.Open("file:test_GetTotalDemand2?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)
		hm := NewMockHardwareManager(ctrl)
		rm := newMockResourceManager(repos, hm, t)

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

		err = rm.UpdateOnboardedResources(context.Background(), onboardedResources)
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

		// Set the allocations in the store to nil
		rm.store.withAllocationsLock(func() {
			rm.store.allocations = make(map[string]types.ResourceAllocation)
		})

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
		freeResourcesRepo := NewMockGenericEntityRepository[types.FreeResources](ctrl)
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, freeResourcesRepo, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm := newMockResourceManager(repos, hm, t)

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), repositories.Query[types.ResourceAllocation]{}).Return(nil, nil)
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
		freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
			return freeResources, nil
		})
		err := rm.UpdateOnboardedResources(context.Background(), onboardedResources)
		require.NoError(t, err)

		var wg sync.WaitGroup
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

				freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
					return freeResources, nil
				})
				resourceAllocationRepo.EXPECT().Create(gomock.Any(), demand).Return(demand, nil)
				hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
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

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{}).Times(numGoroutines)
		resourceAllocationRepo.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, query repositories.Query[types.ResourceAllocation]) (types.ResourceAllocation, error) {
			return types.ResourceAllocation{
				BaseDBModel: types.BaseDBModel{
					ID: query.Conditions[0].Value.(string),
				},
			}, nil
		}).Times(numGoroutines)
		resourceAllocationRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(numGoroutines)
		freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
			return freeResources, nil
		}).Times(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			jobID := fmt.Sprintf("job%d", i)
			wg.Add(1)
			go func() {
				defer wg.Done()

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
		freeResourcesRepo := NewMockGenericEntityRepository[types.FreeResources](ctrl)
		onboardedResourcesRepo := NewMockGenericEntityRepository[types.OnboardedResources](ctrl)
		resourceAllocationRepo := NewMockGenericRepository[types.ResourceAllocation](ctrl)

		repos := newMockManagerRepos(t, freeResourcesRepo, onboardedResourcesRepo, resourceAllocationRepo)
		hm := NewMockHardwareManager(ctrl)
		rm := newMockResourceManager(repos, hm, t)

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{})
		resourceAllocationRepo.EXPECT().FindAll(gomock.Any(), repositories.Query[types.ResourceAllocation]{}).Return(nil, nil)
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
		freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
			return freeResources, nil
		})
		err := rm.UpdateOnboardedResources(context.Background(), onboardedResources)
		require.NoError(t, err)

		var wg sync.WaitGroup

		resourceAllocationRepo.EXPECT().GetQuery().Return(repositories.Query[types.ResourceAllocation]{}).Times(numGoroutines)
		resourceAllocationRepo.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, query repositories.Query[types.ResourceAllocation]) (types.ResourceAllocation, error) {
			return types.ResourceAllocation{
				BaseDBModel: types.BaseDBModel{
					ID: query.Conditions[0].Value.(string),
				},
			}, nil
		}).Times(numGoroutines)
		resourceAllocationRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(numGoroutines)
		freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
			return freeResources, nil
		}).Times(numGoroutines)
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

				freeResourcesRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, freeResources types.FreeResources) (types.FreeResources, error) {
					return freeResources, nil
				})
				resourceAllocationRepo.EXPECT().Create(gomock.Any(), demand).Return(demand, nil)
				hm.EXPECT().GetFreeResources().Return(onboardedResources.Resources, nil)
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
}
