package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// FreeResources represents a repository for CRUD operations on FreeResources entity.
type FreeResources interface {
	GenericEntityRepository[types.FreeResources]
}

// OnboardedResources represents a repository for CRUD operations on OnboardedResources entity.
type OnboardedResources interface {
	GenericEntityRepository[types.OnboardedResources]
}

// RequiredResources represents a repository for CRUD operations on RequiredResources entities.
type RequiredResources interface {
	GenericRepository[types.RequiredResources]
}

// AvailableResources represents a repository for CRUD operations on AvailableResources entity.
type AvailableResources interface {
	GenericEntityRepository[types.AvailableResources]
}
