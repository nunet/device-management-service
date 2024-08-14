package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerRepositoryGORM is a GORM implementation of the RequestTrackerRepository interface.
type RequestTrackerRepositoryGORM struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTrackerRepository creates a new instance of RequestTrackerRepositoryGORM.
// It initializes and returns a GORM-based repository for RequestTracker entities.
func NewRequestTrackerRepository(db *gorm.DB) repositories.RequestTrackerRepository {
	return &RequestTrackerRepositoryGORM{
		NewGenericRepository[types.RequestTracker](db),
	}
}
