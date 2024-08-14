package repositories_clover

import (
	"github.com/ostafen/clover/v2"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachineRepositoryClover is a Clover implementation of the VirtualMachineRepository interface.
type VirtualMachineRepositoryClover struct {
	repositories.GenericRepository[types.VirtualMachine]
}

// NewVirtualMachineRepository creates a new instance of VirtualMachineRepositoryClover.
// It initializes and returns a Clover-based repository for VirtualMachine entities.
func NewVirtualMachineRepository(db *clover.DB) repositories.VirtualMachineRepository {
	return &VirtualMachineRepositoryClover{
		NewGenericRepository[types.VirtualMachine](db),
	}
}
