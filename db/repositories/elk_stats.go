package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerRepository represents a repository for CRUD operations on RequestTracker entities.
type RequestTrackerRepository interface {
	GenericRepository[types.RequestTracker]
}
