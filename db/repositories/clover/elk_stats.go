package clover

import (
	"github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerClover is a Clover implementation of the RequestTracker interface.
type RequestTrackerClover struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTracker creates a new instance of RequestTrackerClover.
// It initializes and returns a Clover-based repository for RequestTracker entities.
func NewRequestTracker(db *clover.DB) repositories.RequestTracker {
	return &RequestTrackerClover{
		NewGenericRepository[types.RequestTracker](db),
	}
}
