package clover

import (
	"github.com/ostafen/clover/v2"
	"gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// MachineResourcesRepositoryClover is a Clover implementation of the MachineResourcesRepository interface.
type MachineResourcesRepositoryClover struct {
	repositories.GenericEntityRepository[types.MachineResources]
}

// NewMachineResourcesRepository creates a new instance of MachineResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for MachineResources entity.
func NewMachineResourcesRepository(db *clover.DB) repositories.MachineResources {
	return &MachineResourcesRepositoryClover{
		NewGenericEntityRepository[types.MachineResources](db),
	}
}

// FreeResourcesClover is a Clover implementation of the FreeResources interface.
type FreeResourcesClover struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResources creates a new instance of FreeResourcesClover.
// It initializes and returns a Clover-based repository for FreeResources entity.
func NewFreeResources(db *clover.DB) repositories.FreeResources {
	return &FreeResourcesClover{
		NewGenericEntityRepository[types.FreeResources](db),
	}
}

// OnboardedResourcesClover is a Clover implementation of the OnboardedResources interface.
type OnboardedResourcesClover struct {
	repositories.GenericEntityRepository[types.OnboardedResources]
}

// NewOnboardedResources creates a new instance of OnboardedResourcesClover.
// It initializes and returns a Clover-based repository for OnboardedResources entity.
func NewOnboardedResources(db *clover.DB) repositories.OnboardedResources {
	return &OnboardedResourcesClover{
		NewGenericEntityRepository[types.OnboardedResources](db),
	}
}

// ResourceAllocationClover is a Clover implementation of the ResourceAllocation interface.
type ResourceAllocationClover struct {
	repositories.GenericRepository[types.ResourceAllocation]
}

// NewResourceAllocation creates a new instance of ResourceAllocationClover.
// It initializes and returns a Clover-based repository for ResourceAllocation entity.
func NewResourceAllocation(db *clover.DB) repositories.ResourceAllocation {
	return &ResourceAllocationClover{
		NewGenericRepository[types.ResourceAllocation](db),
	}
}
