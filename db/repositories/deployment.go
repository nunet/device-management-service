package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlatRepository represents a repository for CRUD operations on DeploymentRequestFlat entities.
type DeploymentRequestFlatRepository interface {
	GenericRepository[types.DeploymentRequestFlat]
}
