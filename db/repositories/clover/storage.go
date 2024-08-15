package repositories_clover

import (
	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// StorageVolumeClover is a Clover implementation of the StorageVolume interface.
type StorageVolumeClover struct {
	repositories.GenericRepository[types.StorageVolume]
}

// NewStorageVolume creates a new instance of StorageVolumeClover.
// It initializes and returns a Clover-based repository for StorageVolume entities.
func NewStorageVolume(db *clover.DB) repositories.StorageVolume {
	return &StorageVolumeClover{
		NewGenericRepository[types.StorageVolume](db),
	}
}
