package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// StorageVolume represents a repository for CRUD operations on StorageVolume entities.
type StorageVolume interface {
	GenericRepository[types.StorageVolume]
}
