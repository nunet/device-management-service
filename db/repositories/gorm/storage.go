package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// StorageVolumeGORM is a GORM implementation of the StorageVolume interface.
type StorageVolumeGORM struct {
	repositories.GenericRepository[types.StorageVolume]
}

// NewStorageVolume creates a new instance of StorageVolumeGORM.
// It initializes and returns a GORM-based repository for StorageVolume entities.
func NewStorageVolume(db *gorm.DB) repositories.StorageVolume {
	return &StorageVolumeGORM{
		NewGenericRepository[types.StorageVolume](db),
	}
}
