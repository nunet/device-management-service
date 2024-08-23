package resources

import (
	"context"
	"fmt"
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	gormRepo "gitlab.com/nunet/device-management-service/db/repositories/gorm"
)

// setupManagerRepos prepares a full structure of ManagerRepos to be
// used by tests.
func setupManagerRepos(t *testing.T, db *gorm.DB) ManagerRepos {
	err := db.AutoMigrate(
		&types.FreeResources{},
		&types.OnboardedResources{},
		&types.RequiredResources{},
		&types.VirtualMachine{},
		&types.Services{},
	)
	assert.NoError(t, err)

	return ManagerRepos{
		FreeResources:      gormRepo.NewFreeResources(db),
		OnboardedResources: gormRepo.NewOnboardedResources(db),
		RequiredResources:  gormRepo.NewRequiredResources(db),
		VirtualMachine:     gormRepo.NewVirtualMachine(db),
		Services:           gormRepo.NewServices(db),
	}
}

// setupMockRunningVMs sets up the mock running VMs using the repository
func setupMockRunningVMs(repo repositories.VirtualMachine, vms []types.VirtualMachine, t *testing.T) {
	t.Helper()

	ctx := context.Background()
	for _, vm := range vms {
		_, err := repo.Create(ctx, vm)
		assert.NoError(t, err)
	}
}

// setupMockRunningServices sets up the mock running services using the repository
func setupMockRunningServices(repo repositories.Services, services []types.Services, t *testing.T) {
	t.Helper()

	ctx := context.Background()
	for _, service := range services {
		_, err := repo.Create(ctx, service)
		assert.NoError(t, err)
	}
}

// setupMockRequiredResources sets up the mock required resources using the repository
func setupMockRequiredResources(repo repositories.RequiredResources, requiredResources []types.RequiredResources, t *testing.T) {
	t.Helper()

	ctx := context.Background()
	for _, reqRes := range requiredResources {
		_, err := repo.Create(ctx, reqRes)
		assert.NoError(t, err)
	}
}

// setUpOnboardedResources sets up the mock onboarded resources using the repository
func setUpOnboardedResources(repo repositories.OnboardedResources, onboardedResources types.OnboardedResources, t *testing.T) {
	t.Helper()

	ctx := context.Background()
	_, err := repo.Save(ctx, onboardedResources)
	assert.NoError(t, err)
}

// setUpFreeResources sets up the mock free resources using the repository
func setUpFreeResources(repo repositories.FreeResources, freeResources types.FreeResources, t *testing.T) {
	t.Helper()

	ctx := context.Background()
	_, err := repo.Save(ctx, freeResources)
	assert.NoError(t, err)
}

// getFreeResourcesFromDB gets the free resources using the repository
func getFreeResourcesFromDB(repo repositories.FreeResources) (types.FreeResources, error) {
	ctx := context.Background()
	freeResources, err := repo.Get(ctx)
	if err != nil {
		return types.FreeResources{},
			fmt.Errorf(
				"error getting FreeRes from DB. Err: %w",
				err,
			)
	}
	return freeResources, nil
}
