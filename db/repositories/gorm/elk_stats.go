package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerGORM is a GORM implementation of the RequestTracker interface.
type RequestTrackerGORM struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTracker creates a new instance of RequestTrackerGORM.
// It initializes and returns a GORM-based repository for RequestTracker entities.
func NewRequestTracker(db *gorm.DB) repositories.RequestTracker {
	return &RequestTrackerGORM{
		NewGenericRepository[types.RequestTracker](db),
	}
}
