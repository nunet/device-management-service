package repositories_clover

import (
	"context"
	"gitlab.com/nunet/device-management-service/types"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// TestFreeResourcesRepository is a test suite for the FreeResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the FreeResources model behave as expected.
func TestFreeResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	freeResourcesRepo := NewFreeResourcesRepository(db)

	// Test Save method
	createdFreeResources, err := freeResourcesRepo.Save(
		context.Background(),
		types.FreeResources{
			Resources: types.Resources{
				CPU:  2.0,
				RAM:  4096,
				Disk: 100000,
			},
		},
	)
	assert.NoError(t, err)

	// Test Get method
	retrievedFreeResources, err := freeResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdFreeResources.CPU, retrievedFreeResources.CPU)

	// Test Save (update) method
	updatedFreeResources := retrievedFreeResources
	updatedFreeResources.CPU = 4.0

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
	assert.Len(t, history, 0)
}

// TestRequiredResourcesRepository is a test suite for the RequiredResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the RequiredResources model behave as expected.
func TestRequiredResourcesRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	requiredResourcesRepo := NewRequiredResourcesRepository(db)

	// Test Create method
	createdRequiredResources, err := requiredResourcesRepo.Create(
		context.Background(),
		types.RequiredResources{
			JobID: 1,
			Resources: types.Resources{
				CPU: 2000,
				RAM: 4096,
			},
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdRequiredResources.ID)

	// Test Get method
	retrievedRequiredResources, err := requiredResourcesRepo.Get(
		context.Background(),
		createdRequiredResources.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdRequiredResources.ID, retrievedRequiredResources.ID)
	assert.Equal(t, createdRequiredResources.JobID, retrievedRequiredResources.JobID)
	assert.Equal(t, createdRequiredResources.CPU, retrievedRequiredResources.CPU)
	assert.Equal(t, createdRequiredResources.RAM, retrievedRequiredResources.RAM)

	// Test Update method
	updatedRequiredResources := retrievedRequiredResources
	updatedRequiredResources.CPU = 3000
	updatedRequiredResources.RAM = 8192

	_, err = requiredResourcesRepo.Update(
		context.Background(),
		updatedRequiredResources.ID,
		updatedRequiredResources,
	)
	assert.NoError(t, err)
	retrievedRequiredResources, err = requiredResourcesRepo.Get(
		context.Background(),
		createdRequiredResources.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, updatedRequiredResources.CPU, retrievedRequiredResources.CPU)
	assert.Equal(t, updatedRequiredResources.RAM, retrievedRequiredResources.RAM)

	// Test Delete method
	err = requiredResourcesRepo.Delete(context.Background(), updatedRequiredResources.ID)
	assert.NoError(t, err)

	// Test Find method
	requiredResources1, err := requiredResourcesRepo.Create(
		context.Background(),
		types.RequiredResources{JobID: 2, Resources: types.Resources{CPU: 1000, RAM: 2048}},
	)
	assert.NoError(t, err)

	query := requiredResourcesRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.EQ("JobID", requiredResources1.JobID),
	)
	foundRequiredResources, err := requiredResourcesRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, requiredResources1.JobID, foundRequiredResources.JobID)
	assert.Equal(t, requiredResources1.CPU, foundRequiredResources.CPU)
	assert.Equal(t, requiredResources1.RAM, foundRequiredResources.RAM)

	// Test FindAll method
	requiredResources2, err := requiredResourcesRepo.Create(
		context.Background(),
		types.RequiredResources{JobID: 3, Resources: types.Resources{CPU: 4000, RAM: 16384}},
	)
	assert.NoError(t, err)

	allRequiredResources, err := requiredResourcesRepo.FindAll(
		context.Background(),
		requiredResourcesRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allRequiredResources, 2)

	// Clean up created records
	err = requiredResourcesRepo.Delete(context.Background(), requiredResources1.ID)
	assert.NoError(t, err)
	err = requiredResourcesRepo.Delete(context.Background(), requiredResources2.ID)
	assert.NoError(t, err)
}

// TestOnboardedResourcesRepository is a test suite for the OnboardedResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the OnboardedResources model behave as expected.
func TestOnboardedResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	onboardedResourcesRepo := NewOnboardedResourcesRepository(db)

	// Test Save method
	createdOnboardedResources, err := onboardedResourcesRepo.Save(
		context.Background(),
		types.OnboardedResources{
			Resources: types.Resources{
				CPU:  2.0,
				RAM:  4096,
				Disk: 100000,
			},
		},
	)
	assert.NoError(t, err)

	// Test Get method
	retrievedOnboardedResources, err := onboardedResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdOnboardedResources.CPU, retrievedOnboardedResources.CPU)

	// Test Save (update) method
	updatedOnboardedResources := retrievedOnboardedResources
	updatedOnboardedResources.CPU = 4.0

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
	assert.Len(t, history, 0)
}
