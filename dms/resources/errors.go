package resources

import (
	"errors"
)

var (
	ErrResourcesNotCommitted     = errors.New("resources not committed")
	ErrResourcesNotAllocated     = errors.New("resources not allocated")
	ErrResourcesAlreadyCommitted = errors.New("resources already committed")
	ErrResourcesAlreadyAllocated = errors.New("resources already allocated")
)
