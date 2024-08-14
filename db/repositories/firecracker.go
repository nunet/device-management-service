package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachineRepository represents a repository for CRUD operations on VirtualMachine entities.
type VirtualMachineRepository interface {
	GenericRepository[types.VirtualMachine]
}
