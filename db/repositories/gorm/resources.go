// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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

// OnboardedResourcesGORM is a GORM implementation of the OnboardedResources interface.
type OnboardedResourcesGORM struct {
	repositories.GenericEntityRepository[types.OnboardedResources]
}

// NewOnboardedResources creates a new instance of OnboardedResourcesGORM.
// It initializes and returns a GORM-based repository for OnboardedResources entity.
func NewOnboardedResources(db *gorm.DB) repositories.OnboardedResources {
	return &OnboardedResourcesGORM{
		NewGenericEntityRepository[types.OnboardedResources](db),
	}
}

// ResourceAllocationGORM is a GORM implementation of the ResourceAllocation interface.
type ResourceAllocationGORM struct {
	repositories.GenericRepository[types.ResourceAllocation]
}

// NewResourceAllocation creates a new instance of ResourceAllocationGORM.
// It initializes and returns a GORM-based repository for ResourceAllocation entities.
func NewResourceAllocation(db *gorm.DB) repositories.ResourceAllocation {
	return &ResourceAllocationGORM{
		NewGenericRepository[types.ResourceAllocation](db),
	}
}
