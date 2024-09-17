package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentRequestFlatGORM is a GORM implementation of the DeploymentRequestFlat interface.
type DeploymentRequestFlatGORM struct {
	repositories.GenericRepository[types.DeploymentRequestFlat]
}

// NewDeploymentRequestFlat creates a new instance of DeploymentRequestFlatGORM.
// It initializes and returns a GORM-based repository for DeploymentRequestFlat entities.
func NewDeploymentRequestFlat(db *gorm.DB) repositories.DeploymentRequestFlat {
	return &DeploymentRequestFlatGORM{
		NewGenericRepository[types.DeploymentRequestFlat](db),
	}
}
