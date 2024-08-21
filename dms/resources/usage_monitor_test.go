package resources

import (
	"context"
	"gitlab.com/nunet/device-management-service/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestGetUsage tests the GetUsage method
func TestGetUsage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCPU := types.CPUInfo{
		NumCores:   5,
		MHzPerCore: 10000,
		Compute:    50000,
	}

	t.Run("Must be able to get the usage from running services and VMs", func(t *testing.T) {
		t.Parallel()

		mockDB, err := gorm.Open(sqlite.Open("file:test1?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		// VM Usage:
		// 1. VM1: 1 CPU, 1024 MB
		// 2. VM2: 1 CPU, 1024 MB
		// Total: 2 * 10000 = 20,000 CPU, 2048 MB
		mockVMs := []types.VirtualMachine{
			{
				VCPUCount:  1,
				MemSizeMib: 1024,
				State:      "running",
			},
			{
				VCPUCount:  1,
				MemSizeMib: 1024,
				State:      "running",
			},
			{
				VCPUCount:  1,
				MemSizeMib: 1024,
				State:      "stopped",
			},
		}
		setupMockRunningVMs(repos.VirtualMachine, mockVMs, t)

		// Container Usage:
		// This is nothing but the resource requirements of the services
		// 1. Service1: 10000 CPU, 512 MB
		// 2. Service2: 20000 CPU, 512 MB
		// Total: 30000 CPU, 1024 MB
		// Total Usage: 30,000 CPU, 1024 MB
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

		mockSystemSpecs := NewMockSystemSpecs(ctrl)
		um := &defaultUsageMonitor{
			systemSpecs:           mockSystemSpecs,
			vmRepo:                repos.VirtualMachine,
			serviceRepo:           repos.Services,
			requiredResourcesRepo: repos.RequiredResources,
		}

		mockSystemSpecs.EXPECT().GetCPUInfo().Return(mockCPU, nil).Times(1)
		usage, err := um.GetUsage(context.Background())
		assert.NoError(t, err)

		expectedUsage := types.Resources{
			CPU: 2*mockCPU.MHzPerCore + 30000, // 2 VMs + 2 Services
			RAM: 2048 + 1024,                  // 2 VMs + 2 Services
		}
		assert.Equal(t, expectedUsage, usage)
	})

	t.Run("Must be able to get the usage when there are no running VMs and services", func(t *testing.T) {
		t.Parallel()

		mockDB, err := gorm.Open(sqlite.Open("file:test2?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		setupMockRunningServices(repos.Services, []types.Services{}, t)
		setupMockRequiredResources(repos.RequiredResources, []types.RequiredResources{}, t)
		setupMockRunningVMs(repos.VirtualMachine, []types.VirtualMachine{}, t)

		mockSystemSpecs := NewMockSystemSpecs(ctrl)
		um := &defaultUsageMonitor{
			systemSpecs:           mockSystemSpecs,
			vmRepo:                repos.VirtualMachine,
			serviceRepo:           repos.Services,
			requiredResourcesRepo: repos.RequiredResources,
		}

		mockSystemSpecs.EXPECT().GetCPUInfo().Return(mockCPU, nil).Times(1)
		usage, err := um.GetUsage(context.Background())
		assert.NoError(t, err)

		expectedUsage := types.Resources{}
		assert.Equal(t, expectedUsage, usage)
	})

	t.Run("Must be able to get the usage when there are only VMs", func(t *testing.T) {
		t.Parallel()

		mockDB, err := gorm.Open(sqlite.Open("file:test3?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		mockVMs := []types.VirtualMachine{
			{
				VCPUCount:  1,
				MemSizeMib: 1024,
				State:      "running",
			},
		}
		setupMockRunningVMs(repos.VirtualMachine, mockVMs, t)
		setupMockRequiredResources(repos.RequiredResources, []types.RequiredResources{}, t)
		setupMockRunningServices(repos.Services, []types.Services{}, t)

		mockSystemSpecs := NewMockSystemSpecs(ctrl)
		um := &defaultUsageMonitor{
			systemSpecs:           mockSystemSpecs,
			vmRepo:                repos.VirtualMachine,
			serviceRepo:           repos.Services,
			requiredResourcesRepo: repos.RequiredResources,
		}

		mockSystemSpecs.EXPECT().GetCPUInfo().Return(mockCPU, nil).Times(1)

		usage, err := um.GetUsage(context.Background())
		assert.NoError(t, err)

		// VM Usage:
		// 1. VM1: 1 CPU, 1024 MB
		// Total: 1 * 10000 = 10,000 CPU, 1024 MB
		expectedUsage := types.Resources{
			CPU: mockCPU.MHzPerCore,
			RAM: 1024,
		}
		assert.Equal(t, expectedUsage, usage)
	})

	t.Run("Must be able to get the usage when there are only services", func(t *testing.T) {
		t.Parallel()

		mockDB, err := gorm.Open(sqlite.Open("file:test4?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		mockServices := []types.Services{
			{
				ResourceRequirements: 1,
				JobStatus:            "running",
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
		}
		setupMockRunningServices(repos.Services, mockServices, t)
		setupMockRequiredResources(repos.RequiredResources, mockRequiredResources, t)
		setupMockRunningVMs(repos.VirtualMachine, []types.VirtualMachine{}, t)

		mockSystemSpecs := NewMockSystemSpecs(ctrl)
		um := &defaultUsageMonitor{
			vmRepo:                repos.VirtualMachine,
			serviceRepo:           repos.Services,
			requiredResourcesRepo: repos.RequiredResources,
			systemSpecs:           mockSystemSpecs,
		}

		mockSystemSpecs.EXPECT().GetCPUInfo().Return(mockCPU, nil).Times(1)
		usage, err := um.GetUsage(context.Background())
		assert.NoError(t, err)

		// Container Usage:
		// This is nothing but the resource requirements of the services
		// 1. Service1: 10000 CPU, 512 MB
		// Total: 10000 CPU, 512 MB
		expectedUsage := types.Resources{
			CPU: 10000,
			RAM: 512,
		}
		assert.Equal(t, expectedUsage, usage)
	})
}

// TestGetContainerUsage tests the getContainerUsage method
func TestGetContainerUsage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Must get only running services", func(t *testing.T) {
		mockDB, err := gorm.Open(sqlite.Open("file:test6?mode=memory&cache=shared"), &gorm.Config{})
		assert.NoError(t, err)

		repos := setupManagerRepos(t, mockDB)

		mockServices := []types.Services{
			{
				ResourceRequirements: 1,
				JobStatus:            "running",
			},
			{
				ResourceRequirements: 2,
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
					RAM: 1024,
				},
			},
		}

		setupMockRunningServices(repos.Services, mockServices, t)
		setupMockRequiredResources(repos.RequiredResources, mockRequiredResources, t)

		um := &defaultUsageMonitor{
			serviceRepo:           repos.Services,
			requiredResourcesRepo: repos.RequiredResources,
		}

		usage, err := um.getContainerUsage(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, types.Resources{
			CPU: 10000,
			RAM: 512,
		}, usage)
	})
}
