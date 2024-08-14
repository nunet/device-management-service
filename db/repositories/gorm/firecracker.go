package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachineRepositoryGORM is a GORM implementation of the VirtualMachineRepository interface.
type VirtualMachineRepositoryGORM struct {
	repositories.GenericRepository[types.VirtualMachine]
}

// NewVirtualMachineRepository creates a new instance of VirtualMachineRepositoryGORM.
// It initializes and returns a GORM-based repository for VirtualMachine entities.
func NewVirtualMachineRepository(db *gorm.DB) repositories.VirtualMachineRepository {
	return &VirtualMachineRepositoryGORM{
		NewGenericRepository[types.VirtualMachine](db),
	}
}
