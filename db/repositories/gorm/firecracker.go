package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachineGORM is a GORM implementation of the VirtualMachine interface.
type VirtualMachineGORM struct {
	repositories.GenericRepository[types.VirtualMachine]
}

// NewVirtualMachine creates a new instance of VirtualMachineGORM.
// It initializes and returns a GORM-based repository for VirtualMachine entities.
func NewVirtualMachine(db *gorm.DB) repositories.VirtualMachine {
	return &VirtualMachineGORM{
		NewGenericRepository[types.VirtualMachine](db),
	}
}
