package resources

import (
	"context"
	"gitlab.com/nunet/device-management-service/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewResourceManager(t *testing.T) {
	t.Parallel()

	mockDB, err := gorm.Open(sqlite.Open("file:test_newResourceManager?mode=memory&cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	repos := setupManagerRepos(t, mockDB)

	rm := newResourceManager(repos)
	assert.NotNil(t, rm)
}

func TestUpdateAndGetFreeResources(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must be able to update and get free resources", func(t *testing.T) {
		mockDB, err := gorm.Open(sqlite.Open("file:test_updateAndGetFreeResources1?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		// setup free resources in the database
		setUpFreeResources(repos.FreeResources, types.FreeResources{}, t)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: 50000,
				RAM: 2048,
			},
		}
		setUpOnboardedResources(repos.OnboardedResources, onboardedResources, t)

		mockUsageMonitor := NewMockUsageMonitor(ctrl)
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			repos:        repos,
			usageMonitor: mockUsageMonitor,
			systemSpecs:  mockSystemSpecs,
		}

		mockUsageMonitor.EXPECT().GetUsage(gomock.Any()).Return(types.Resources{
			CPU: 30000,
			RAM: 1024,
		}, nil).Times(1)

		freeResources, err := rm.UpdateFreeResources(context.Background())
		assert.NoError(t, err)

		expectedFreeResources := types.FreeResources{
			Resources: types.Resources{
				CPU: 20000,
				RAM: 1024,
			},
		}
		assert.Equal(t, expectedFreeResources, freeResources)

		// Check if the free resources are updated in the database
		time.Sleep(1 * time.Second)
		freeResourcesFromDB, err := getFreeResourcesFromDB(repos.FreeResources)
		assert.NoError(t, err)
		assert.Equal(t, expectedFreeResources.CPU, freeResourcesFromDB.CPU)
		assert.Equal(t, expectedFreeResources.RAM, freeResourcesFromDB.RAM)
	})
}

func TestOnboardedResources(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must be able to get onboarded resources", func(t *testing.T) {
		mockDB, err := gorm.Open(sqlite.Open("file:test_OnboardedResources1?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: 50000,
				RAM: 2048,
			},
		}
		setUpOnboardedResources(repos.OnboardedResources, onboardedResources, t)

		rm := &defaultManager{
			repos: repos,
		}

		onboardedResourcesFromDB, err := rm.GetOnboardedResources(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, onboardedResources.CPU, onboardedResourcesFromDB.CPU)
		assert.Equal(t, onboardedResources.RAM, onboardedResourcesFromDB.RAM)
	})

	t.Run("Must be able to update onboarded resources", func(t *testing.T) {
		mockDB, err := gorm.Open(sqlite.Open("file:test_OnboardedResources2?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: 50000,
				RAM: 2048,
			},
		}
		setUpOnboardedResources(repos.OnboardedResources, onboardedResources, t)

		rm := &defaultManager{
			repos: repos,
		}

		newOnboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: 60000,
				RAM: 3072,
			},
		}
		err = rm.UpdateOnboardedResources(context.Background(), newOnboardedResources)
		assert.NoError(t, err)

		// Check if the onboarded resources are updated in the database
		onboardedResourcesFromDB, err := rm.GetOnboardedResources(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, newOnboardedResources.CPU, onboardedResourcesFromDB.CPU)
		assert.Equal(t, newOnboardedResources.RAM, onboardedResourcesFromDB.RAM)
	})
}

func TestRequiredResources(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must be able to get required resources", func(t *testing.T) {
		mockDB, err := gorm.Open(sqlite.Open("file:test_RequiredResources1?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		// Required resources of the services:
		// 1. Service1: 10000 CPU, 512 MB
		// 2. Service2: 20000 CPU, 512 MB
		// 3. Service3: 20000 CPU, 512 MB
		// Total: 50000 CPU, 1536 MB
		mockServices := []types.Services{
			{
				ResourceRequirements: 1,
				JobStatus:            "running",
			},
			{
				ResourceRequirements: 2,
				JobStatus:            "running",
			},
			{
				ResourceRequirements: 3,
				JobStatus:            "stopped",
			},
		}
		mockRequiredResources := []types.RequiredResources{
			{
				JobID: 1,
				Resources: types.Resources{
					CPU: 10000,
					RAM: 512,
				},
			},
			{
				JobID: 2,
				Resources: types.Resources{
					CPU: 20000,
					RAM: 512,
				},
			},
			{
				JobID: 3,
				Resources: types.Resources{
					CPU: 20000,
					RAM: 512,
				},
			},
		}
		setupMockRunningServices(repos.Services, mockServices, t)
		setupMockRequiredResources(repos.RequiredResources, mockRequiredResources, t)

		rm := &defaultManager{
			repos: repos,
		}

		expectedRequiredResources := types.Resources{
			CPU: 50000,
			RAM: 1536,
		}
		requiredResources, err := rm.GetRequiredResources(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedRequiredResources.CPU, requiredResources.CPU)
		assert.Equal(t, expectedRequiredResources.RAM, requiredResources.RAM)
	})
}

func TestSystemSpecs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must be able to get cpu info", func(t *testing.T) {
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			systemSpecs: mockSystemSpecs,
		}

		mockSystemSpecs.EXPECT().GetCPUInfo().Return(types.CPUInfo{
			NumCores:   5,
			MHzPerCore: 10000,
			Compute:    50000,
		}, nil).Times(1)

		cpuInfo, err := rm.SystemSpecs().GetCPUInfo()
		assert.NoError(t, err)
		assert.Equal(t, uint64(5), cpuInfo.NumCores)
		assert.Equal(t, float64(10000), cpuInfo.MHzPerCore)
		assert.Equal(t, float64(50000), cpuInfo.Compute)
	})

	t.Run("Must be able to get total memory", func(t *testing.T) {
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			systemSpecs: mockSystemSpecs,
		}

		mockSystemSpecs.EXPECT().GetTotalMemory().Return(uint64(2048), nil).Times(1)

		totalMemory, err := rm.SystemSpecs().GetTotalMemory()
		assert.NoError(t, err)
		assert.Equal(t, uint64(2048), totalMemory)
	})

	t.Run("Must be able to get provisioned resources", func(t *testing.T) {
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			systemSpecs: mockSystemSpecs,
		}

		mockSystemSpecs.EXPECT().GetProvisionedResources().Return(types.Resources{
			CPU: 50000,
			RAM: 2048,
		}, nil).Times(1)

		provisionedResources, err := rm.SystemSpecs().GetProvisionedResources()
		assert.NoError(t, err)
		assert.Equal(t, float64(50000), provisionedResources.CPU)
		assert.Equal(t, uint64(2048), provisionedResources.RAM)
	})

	t.Run("Must be able to get gpu vendors", func(t *testing.T) {
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			systemSpecs: mockSystemSpecs,
		}

		mockSystemSpecs.EXPECT().GetGPUVendors().Return([]types.GPUVendor{types.GPUVendorNvidia}, nil).Times(1)

		gpuVendors, err := rm.SystemSpecs().GetGPUVendors()
		assert.NoError(t, err)
		assert.Equal(t, []types.GPUVendor{types.GPUVendorNvidia}, gpuVendors)
	})

	t.Run("Must be able to get gpu info", func(t *testing.T) {
		mockSystemSpecs := NewMockSystemSpecs(ctrl)

		rm := &defaultManager{
			systemSpecs: mockSystemSpecs,
		}

		mockGPU := types.GPU{
			Vendor:    types.GPUVendorNvidia,
			Model:     "GTX 1080",
			TotalVRAM: 8192,
		}
		mockSystemSpecs.EXPECT().GetGPUs().Return([]types.GPU{mockGPU}, nil).Times(1)

		gpuInfo, err := rm.SystemSpecs().GetGPUs()
		assert.NoError(t, err)
		assert.Equal(t, []types.GPU{mockGPU}, gpuInfo)
	})
}

func TestUsageMonitor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must be able to get usage", func(t *testing.T) {
		mockUsageMonitor := NewMockUsageMonitor(ctrl)

		rm := &defaultManager{
			usageMonitor: mockUsageMonitor,
		}

		mockUsageMonitor.EXPECT().GetUsage(gomock.Any()).Return(types.Resources{
			CPU: 30000,
			RAM: 1024,
		}, nil).Times(1)

		usage, err := rm.UsageMonitor().GetUsage(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, float64(30000), usage.CPU)
		assert.Equal(t, uint64(1024), usage.RAM)
	})
}
