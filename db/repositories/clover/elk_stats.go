package repositories_clover

import (
	"github.com/ostafen/clover/v2"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerRepositoryClover is a Clover implementation of the RequestTrackerRepository interface.
type RequestTrackerRepositoryClover struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTrackerRepository creates a new instance of RequestTrackerRepositoryClover.
// It initializes and returns a Clover-based repository for RequestTracker entities.
func NewRequestTrackerRepository(db *clover.DB) repositories.RequestTrackerRepository {
	return &RequestTrackerRepositoryClover{
		NewGenericRepository[types.RequestTracker](db),
	}
}
