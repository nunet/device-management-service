package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// MachineResources represents a repository for CRUD operations on MachineResources entity.
type MachineResources interface {
	GenericEntityRepository[types.MachineResources]
}

// FreeResources represents a repository for CRUD operations on FreeResources entity.
type FreeResources interface {
	GenericEntityRepository[types.FreeResources]
}

// OnboardedResources represents a repository for CRUD operations on OnboardedResources entity.
type OnboardedResources interface {
	GenericEntityRepository[types.OnboardedResources]
}

// ResourceAllocation represents a repository for CRUD operations on ResourceAllocation entity.
type ResourceAllocation interface {
	GenericRepository[types.ResourceAllocation]
}
