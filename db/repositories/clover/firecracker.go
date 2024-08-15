package repositories_clover

import (
	"github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachineClover is a Clover implementation of the VirtualMachine interface.
type VirtualMachineClover struct {
	repositories.GenericRepository[types.VirtualMachine]
}

// NewVirtualMachine creates a new instance of VirtualMachineClover.
// It initializes and returns a Clover-based repository for VirtualMachine entities.
func NewVirtualMachine(db *clover.DB) repositories.VirtualMachine {
	return &VirtualMachineClover{
		NewGenericRepository[types.VirtualMachine](db),
	}
}
