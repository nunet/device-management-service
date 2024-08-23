package gorm

import (
	"gitlab.com/nunet/device-management-service/types"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// FreeResourcesGORM is a GORM implementation of the FreeResourcesRepository interface.
type FreeResourcesGORM struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResources creates a new instance of FreeResourcesRepositoryGORM.
// It initializes and returns a GORM-based repository for FreeResources entity.
func NewFreeResources(db *gorm.DB) repositories.FreeResources {
	return &FreeResourcesGORM{
		NewGenericEntityRepository[types.FreeResources](db),
	}
}

// OnboardedResourcesRepositoryGORM is a GORM implementation of the OnboardedResourcesRepository interface.
type OnboardedResourcesRepositoryGORM struct {
	repositories.GenericEntityRepository[types.OnboardedResources]
}

// NewOnboardedResources creates a new instance of OnboardedResourcesRepositoryGORM.
// It initializes and returns a GORM-based repository for OnboardedResources entity.
func NewOnboardedResources(db *gorm.DB) repositories.OnboardedResources {
	return &OnboardedResourcesRepositoryGORM{
		NewGenericEntityRepository[types.OnboardedResources](db),
	}
}

// RequiredResourcesRepositoryGORM is a GORM implementation of the RequiredResourcesRepository interface.
type RequiredResourcesRepositoryGORM struct {
	repositories.GenericRepository[types.RequiredResources]
}

// NewRequiredResources creates a new instance of RequiredResourcesRepositoryGORM.
// It initializes and returns a GORM-based repository for RequiredResources entities.
func NewRequiredResources(db *gorm.DB) repositories.RequiredResources {
	return &RequiredResourcesRepositoryGORM{
		NewGenericRepository[types.RequiredResources](db),
	}
}
