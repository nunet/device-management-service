package repositories

import (
	"gitlab.com/nunet/device-management-service/models"
)

// StorageVolumeRepository represents a repository for CRUD operations on StorageVolume entities.
type StorageVolumeRepository interface {
	GenericRepository[models.StorageVolume]
}
