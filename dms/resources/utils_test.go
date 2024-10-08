package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	gormRepo "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/types"
)

// setupManagerRepos prepares a full structure of ManagerRepos to be
// used by tests.
func setupManagerRepos(t *testing.T, db *gorm.DB) ManagerRepos {
	err := db.AutoMigrate(
		&types.FreeResources{},
		&types.OnboardedResources{},
		&types.ResourceAllocation{},
	)
	require.NoError(t, err)

	return ManagerRepos{
		FreeResources:      gormRepo.NewFreeResources(db),
		OnboardedResources: gormRepo.NewOnboardedResources(db),
		ResourceAllocation: gormRepo.NewResourceAllocation(db),
	}
}

// setUpFreeResources sets up the mock free resources using the repository
func setUpFreeResources(repo repositories.FreeResources, freeResources types.FreeResources, t *testing.T) {
	t.Helper()
	_, err := repo.Save(context.Background(), freeResources)
	require.NoError(t, err)
}

// getFreeResourcesFromDB gets the free resources using the repository
func getFreeResourcesFromDB(repo repositories.FreeResources, t *testing.T) types.FreeResources {
	t.Helper()
	freeResources, err := repo.Get(context.Background())
	require.NoError(t, err)
	return freeResources
}

// getOnboardedResourcesFromDB gets the onboarded resources using the repository
func getOnboardedResourcesFromDB(repo repositories.OnboardedResources, t *testing.T) types.OnboardedResources {
	t.Helper()
	onboardedResources, err := repo.Get(context.Background())
	require.NoError(t, err)
	return onboardedResources
}

func assertResources(t *testing.T, expected, actual types.Resources) {
	t.Helper()

	require.Equal(t, expected.CPU.Cores, actual.CPU.Cores)
	require.Equal(t, expected.CPU.ClockSpeed, actual.CPU.ClockSpeed)
	require.Equal(t, expected.RAM, actual.RAM)
	require.Equal(t, expected.Disk, actual.Disk)
	// TODO: GPU
}

// newMockResourceManager creates a new mockResourceManager
func newMockResourceManager(
	repos ManagerRepos,
	hardware *MockHardwareManager,
	t *testing.T,
) *DefaultManager {
	t.Helper()

	return &DefaultManager{
		repos:    repos,
		store:    newStore(),
		hardware: hardware,
	}
}

// newMockManagerRepos creates a new mock ManagerRepos
func newMockManagerRepos(t *testing.T,
	freeResources repositories.FreeResources,
	onboardedResources repositories.OnboardedResources,
	resourceAllocation repositories.ResourceAllocation,
) ManagerRepos {
	t.Helper()

	return ManagerRepos{
		FreeResources:      freeResources,
		OnboardedResources: onboardedResources,
		ResourceAllocation: resourceAllocation,
	}
}
