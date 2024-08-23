package clover

import (
	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlatClover is a Clover implementation of the DeploymentRequestFlat interface.
type DeploymentRequestFlatClover struct {
	repositories.GenericRepository[types.DeploymentRequestFlat]
}

// NewDeploymentRequestFlat creates a new instance of DeploymentRequestFlatClover.
// It initializes and returns a Clover-based repository for DeploymentRequestFlat entities.
func NewDeploymentRequestFlat(db *clover.DB) repositories.DeploymentRequestFlat {
	return &DeploymentRequestFlatClover{
		NewGenericRepository[types.DeploymentRequestFlat](db),
	}
}
