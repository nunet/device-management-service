package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlat represents a repository for CRUD operations on DeploymentRequestFlat entities.
type DeploymentRequestFlat interface {
	GenericRepository[types.DeploymentRequestFlat]
}
