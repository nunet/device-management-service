package repositories

import contract "gitlab.com/nunet/device-management-service/dms/contract"

// Contract represents a repository for CRUD operations on Contract.
type Contract interface {
	GenericRepository[contract.Contract]
}
