package gorm

import (
	"context"
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/assert"
)

// TestMachineResourcesRepository is a test suite for the MachineResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the MachineResources model behave as expected.
func TestMachineResourcesRepository(t *testing.T) {
	setup()
	defer teardown()

	// Initialize the repository
	machineResourcesRepo := NewMachineResources(db)

	// Test Save method
	createdMachineResources, err := machineResourcesRepo.Save(
		context.Background(),
		types.MachineResources{
			Resources: types.Resources{
				CPU:  types.CPU{Cores: 2, ClockSpeed: 10000},
				RAM:  types.RAM{Size: 4096},
				Disk: types.Disk{Size: 1024},
			},
		},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdMachineResources.ID)

	// Test Get method
	retrievedMachineResources, err := machineResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdMachineResources.CPU, retrievedMachineResources.CPU)

	// Test Save (update) method
	updatedMachineResources := retrievedMachineResources
	updatedMachineResources.CPU.Cores = 4

	_, err = machineResourcesRepo.Save(context.Background(), updatedMachineResources)
	assert.NoError(t, err)
	retrievedMachineResources, err = machineResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedMachineResources.CPU, retrievedMachineResources.CPU)

	// Test History method
	query := machineResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := machineResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = machineResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = machineResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 0)
}

// TestFreeResourcesRepository is a test suite for the FreeResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the FreeResources model behave as expected.
func TestFreeResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	freeResourcesRepo := NewFreeResources(db)

	// Test Save method
	createdFreeResources, err := freeResourcesRepo.Save(
		context.Background(),
		types.FreeResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 4096},
				Disk: types.Disk{Size: 100000},
			},
		},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdFreeResources.ID)

	// Test Get method
	retrievedFreeResources, err := freeResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdFreeResources.CPU, retrievedFreeResources.CPU)

	// Test Save (update) method
	updatedFreeResources := retrievedFreeResources
	updatedFreeResources.CPU.Cores = 4

	_, err = freeResourcesRepo.Save(context.Background(), updatedFreeResources)
	assert.NoError(t, err)
	retrievedFreeResources, err = freeResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedFreeResources.CPU, retrievedFreeResources.CPU)

	// Test History method
	query := freeResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := freeResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = freeResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = freeResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 0)
}

// TestOnboardedResourcesRepository is a test suite for the OnboardedResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the OnboardedResources model behave as expected.
func TestOnboardedResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	onboardedResourcesRepo := NewOnboardedResources(db)

	// Test Save method
	createdOnboardedResources, err := onboardedResourcesRepo.Save(
		context.Background(),
		types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 4096},
				Disk: types.Disk{Size: 100000},
			},
		},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdOnboardedResources.ID)

	// Test Get method
	retrievedOnboardedResources, err := onboardedResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdOnboardedResources.CPU, retrievedOnboardedResources.CPU)

	// Test Save (update) method
	updatedOnboardedResources := retrievedOnboardedResources
	updatedOnboardedResources.CPU.Cores = 4

	_, err = onboardedResourcesRepo.Save(context.Background(), updatedOnboardedResources)
	assert.NoError(t, err)
	retrievedOnboardedResources, err = onboardedResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedOnboardedResources.CPU, retrievedOnboardedResources.CPU)

	// Test History method
	query := onboardedResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := onboardedResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = onboardedResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = onboardedResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 0)
}

// TestAvailableResources is a test suite for the AvailableResources.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the AvailableResources model behave as expected.
func TestAvailableResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	availableResourcesRepo := NewAvailableResources(db)

	// Test Save method
	createdAvailableResources, err := availableResourcesRepo.Save(
		context.Background(),
		types.AvailableResources{},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdAvailableResources.ID)

	// Test Get method
	retrievedAvailableResources, err := availableResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdAvailableResources.ID, retrievedAvailableResources.ID)

	// Test Save (update) method
	updatedAvailableResources := retrievedAvailableResources
	updatedAvailableResources.Vcpu = 4

	_, err = availableResourcesRepo.Save(context.Background(), updatedAvailableResources)
	assert.NoError(t, err)
	retrievedAvailableResources, err = availableResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedAvailableResources.Vcpu, retrievedAvailableResources.Vcpu)

	// Test History method
	query := availableResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := availableResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = availableResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = availableResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 0)
}

func TestResourceAllocationRepository(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	resourceAllocationRepo := NewResourceAllocation(db)

	// Test Save method
	createdResourceAllocation, err := resourceAllocationRepo.Create(
		context.Background(),
		types.ResourceAllocation{
			JobID: "123",
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 10000,
				},
				RAM:  types.RAM{Size: 4096},
				Disk: types.Disk{Size: 100000},
			},
		},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdResourceAllocation.ID)

	// Test Get method
	retrievedResourceAllocation, err := resourceAllocationRepo.Get(context.Background(), createdResourceAllocation.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdResourceAllocation.JobID, retrievedResourceAllocation.JobID)

	// Test Save (update) method
	updatedResourceAllocation := retrievedResourceAllocation
	updatedResourceAllocation.JobID = "456"

	_, err = resourceAllocationRepo.Update(context.Background(), updatedResourceAllocation.ID, updatedResourceAllocation)
	assert.NoError(t, err)
	retrievedResourceAllocation, err = resourceAllocationRepo.Get(context.Background(), updatedResourceAllocation.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedResourceAllocation.JobID, retrievedResourceAllocation.JobID)

	// Test History method
	query := resourceAllocationRepo.GetQuery()
	query.Limit = 3
	history, err := resourceAllocationRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 1)

	// Test Clear method
	err = resourceAllocationRepo.Delete(context.Background(), retrievedResourceAllocation.ID)
	assert.NoError(t, err)
	history, err = resourceAllocationRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 0)
}
