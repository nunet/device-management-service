package gorm

import (
	"gitlab.com/nunet/device-management-service/types"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// MachineResourcesGORM is a GORM implementation of the MachineResources interface.
type MachineResourcesGORM struct {
	repositories.GenericEntityRepository[types.MachineResources]
}

// NewMachineResources creates a new instance of MachineResourcesGORM.
// It initializes and returns a GORM-based repository for MachineResources entity.
func NewMachineResources(db *gorm.DB) repositories.MachineResources {
	return &MachineResourcesGORM{
		NewGenericEntityRepository[types.MachineResources](db),
	}
}

// FreeResourcesGORM is a GORM implementation of the FreeResources interface.
type FreeResourcesGORM struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResources creates a new instance of FreeResourcesGORM.
// It initializes and returns a GORM-based repository for FreeResources entity.
func NewFreeResources(db *gorm.DB) repositories.FreeResources {
	return &FreeResourcesGORM{
		NewGenericEntityRepository[types.FreeResources](db),
	}
}

// OnboardedResourcesRepositoryGORM is a GORM implementation of the OnboardedResources interface.
type OnboardedResourcesRepositoryGORM struct {
	repositories.GenericEntityRepository[types.OnboardedResources]
}

// NewOnboardedResources creates a new instance of OnboardedResourcesGORM.
// It initializes and returns a GORM-based repository for OnboardedResources entity.
func NewOnboardedResources(db *gorm.DB) repositories.OnboardedResources {
	return &OnboardedResourcesRepositoryGORM{
		NewGenericEntityRepository[types.OnboardedResources](db),
	}
}

// RequiredResourcesRepositoryGORM is a GORM implementation of the RequiredResources interface.
type RequiredResourcesRepositoryGORM struct {
	repositories.GenericRepository[types.RequiredResources]
}

// NewRequiredResources creates a new instance of RequiredResourcesGORM.
// It initializes and returns a GORM-based repository for RequiredResources entities.
func NewRequiredResources(db *gorm.DB) repositories.RequiredResources {
	return &RequiredResourcesRepositoryGORM{
		NewGenericRepository[types.RequiredResources](db),
	}
}
