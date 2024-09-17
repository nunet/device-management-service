package repositories

import (
	"context"
)

// GenericEntityRepository is an interface defining basic CRUD operations for repositories handling a single record.
type GenericEntityRepository[T ModelType] interface {
	// Save adds or updates a single record in the repository.
	Save(ctx context.Context, data T) (T, error)
	// Get retrieves the single record from the repository.
	Get(ctx context.Context) (T, error)
	// Clear removes the record and its history from the repository.
	Clear(ctx context.Context) error
	// History retrieves previous records from the repository constrained by the query.
	History(ctx context.Context, qiery Query[T]) ([]T, error)
	// GetQuery returns an empty query instance for the repository's type.
	GetQuery() Query[T]
}
