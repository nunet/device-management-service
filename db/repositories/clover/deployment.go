package repositories_clover

import (
	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlatRepositoryClover is a Clover implementation of the DeploymentRequestFlatRepository interface.
type DeploymentRequestFlatRepositoryClover struct {
	repositories.GenericRepository[types.DeploymentRequestFlat]
}

// NewDeploymentRequestFlatRepository creates a new instance of DeploymentRequestFlatRepositoryClover.
// It initializes and returns a Clover-based repository for DeploymentRequestFlat entities.
func NewDeploymentRequestFlatRepository(db *clover.DB) repositories.DeploymentRequestFlatRepository {
	return &DeploymentRequestFlatRepositoryClover{
		NewGenericRepository[types.DeploymentRequestFlat](db),
	}
}
