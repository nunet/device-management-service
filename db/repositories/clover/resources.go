package repositories_clover

import (
	"github.com/ostafen/clover/v2"
	"gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// FreeResourcesRepositoryClover is a Clover implementation of the FreeResourcesRepository interface.
type FreeResourcesRepositoryClover struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResourcesRepository creates a new instance of FreeResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for FreeResources entity.
func NewFreeResourcesRepository(db *clover.DB) repositories.FreeResources {
	return &FreeResourcesRepositoryClover{
		NewGenericEntityRepository[types.FreeResources](db),
	}
}

// OnboardedResourcesRepositoryClover is a Clover implementation of the OnboardedResourcesRepository interface.
type OnboardedResourcesRepositoryClover struct {
	repositories.GenericEntityRepository[types.OnboardedResources]
}

// NewOnboardedResourcesRepository creates a new instance of OnboardedResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for OnboardedResources entity.
func NewOnboardedResourcesRepository(db *clover.DB) repositories.OnboardedResources {
	return &OnboardedResourcesRepositoryClover{
		NewGenericEntityRepository[types.OnboardedResources](db),
	}
}

// RequiredResourcesRepositoryClover is a Clover implementation of the RequiredResourcesRepository interface.
type RequiredResourcesRepositoryClover struct {
	repositories.GenericRepository[types.RequiredResources]
}

// NewRequiredResourcesRepository creates a new instance of RequiredResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for RequiredResources entities.
func NewRequiredResourcesRepository(db *clover.DB) repositories.RequiredResources {
	return &RequiredResourcesRepositoryClover{
		NewGenericRepository[types.RequiredResources](db),
	}
}
