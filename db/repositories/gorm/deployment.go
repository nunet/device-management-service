package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlatRepositoryGORM is a GORM implementation of the DeploymentRequestFlatRepository interface.
type DeploymentRequestFlatRepositoryGORM struct {
	repositories.GenericRepository[types.DeploymentRequestFlat]
}

// NewDeploymentRequestFlatRepository creates a new instance of DeploymentRequestFlatRepositoryGORM.
// It initializes and returns a GORM-based repository for DeploymentRequestFlat entities.
func NewDeploymentRequestFlatRepository(db *gorm.DB) repositories.DeploymentRequestFlatRepository {
	return &DeploymentRequestFlatRepositoryGORM{
		NewGenericRepository[types.DeploymentRequestFlat](db),
	}
}
