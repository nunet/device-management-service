package repositories_clover

import (
	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/models"
)

// StorageVolumeRepositoryClover is a Clover implementation of the StorageVolumeRepository interface.
type StorageVolumeRepositoryClover struct {
	repositories.GenericRepository[models.StorageVolume]
}

// NewStorageVolumeRepository creates a new instance of StorageVolumeRepositoryClover.
// It initializes and returns a Clover-based repository for StorageVolume entities.
func NewStorageVolumeRepository(db *clover.DB) repositories.StorageVolumeRepository {
	return &StorageVolumeRepositoryClover{
		NewGenericRepository[models.StorageVolume](db),
	}
}
