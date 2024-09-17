package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTracker represents a repository for CRUD operations on RequestTracker entities.
type RequestTracker interface {
	GenericRepository[types.RequestTracker]
}
