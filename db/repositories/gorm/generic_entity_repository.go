package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

const (
	createdAtField = "CreatedAt"
)

// GenericEntityRepositoryGORM is a generic single entity repository implementation using GORM as an ORM.
// It is intended to be embedded in single entity model repositories to provide basic database operations.
type GenericEntityRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB // db is the GORM database instance.
}

// NewGenericEntityRepository creates a new instance of GenericEntityRepositoryGORM.
// It initializes and returns a repository with the provided GORM database, primary key field, and value.
func NewGenericEntityRepository[T repositories.ModelType](
	db *gorm.DB,
) repositories.GenericEntityRepository[T] {
	return &GenericEntityRepositoryGORM[T]{db: db}
}

// GetQuery returns a clean Query instance for building queries.
func (repo *GenericEntityRepositoryGORM[T]) GetQuery() repositories.Query[T] {
	return repositories.Query[T]{}
}

// Save creates or updates the record to the repository and returns the new/updated data.
func (repo *GenericEntityRepositoryGORM[T]) Save(ctx context.Context, data T) (T, error) {
	err := repo.db.WithContext(ctx).Create(&data).Error
	return data, handleDBError(err)
}

// Get retrieves the record from the repository.
func (repo *GenericEntityRepositoryGORM[T]) Get(ctx context.Context) (T, error) {
	var result T

	query := repo.GetQuery()
	query.SortBy = fmt.Sprintf("-%s", createdAtField)

	db := repo.db.WithContext(ctx)
	db = applyConditions(db, query)

	err := db.First(&result).Error

	return result, handleDBError(err)
}

// Clear removes the record with its history from the repository.
func (repo *GenericEntityRepositoryGORM[T]) Clear(ctx context.Context) error {
	return repo.db.WithContext(ctx).Delete(new(T), "id IS NOT NULL").Error
}

// History retrieves previous records from the repository constrained by the query.
func (repo *GenericEntityRepositoryGORM[T]) History(
	ctx context.Context,
	query repositories.Query[T],
) ([]T, error) {
	var results []T

	db := repo.db.WithContext(ctx).Model(new(T))
	db = applyConditions(db, query)

	err := db.Find(&results).Error
	return results, handleDBError(err)
}
