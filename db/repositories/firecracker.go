package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// VirtualMachine represents a repository for CRUD operations on VirtualMachine entities.
type VirtualMachine interface {
	GenericRepository[types.VirtualMachine]
}
