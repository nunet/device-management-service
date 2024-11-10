// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
