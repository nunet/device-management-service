// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gorm

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/observability"
)

// Note: our GORM implementation does not support:
//
// - Structs with maps
//
// - Structs with nested NAMED structs, e.g.:
//     type ComputerSpecs struct {
//      types.BaseDBModel
//      CPU     int
//      Another AnotherStruct
//     }

// GenericRepositoryGORM is a generic repository implementation using GORM as an ORM.
// It is intended to be embedded in model repositories to provide basic database operations.
type GenericRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB
}

// NewGenericRepository creates a new instance of GenericRepositoryGORM.
// It initializes and returns a repository with the provided GORM database.
func NewGenericRepository[T repositories.ModelType](db *gorm.DB) repositories.GenericRepository[T] {
	endTrace := observability.StartTrace("gorm_db_repo_init_duration")
	defer endTrace()

	logger.Infow("gorm_db_repo_init_success", "repository", fmt.Sprintf("%T", *new(T)))
	return &GenericRepositoryGORM[T]{db: db}
}

// GetQuery returns a clean Query instance for building queries.
func (repo *GenericRepositoryGORM[T]) GetQuery() repositories.Query[T] {
	return repositories.Query[T]{}
}

// Create adds a new record to the repository and returns the created data.
func (repo *GenericRepositoryGORM[T]) Create(ctx context.Context, data T) (T, error) {
	endTrace := observability.StartTrace(ctx, "gorm_db_create_duration")
	defer endTrace()

	err := repo.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		logger.Errorw("gorm_db_create_failure", "error", err)
		return data, handleDBError(err)
	}

	logger.Infow("gorm_db_create_success", "record", fmt.Sprintf("%+v", data))
	return data, handleDBError(err)
}

// Get retrieves a record by its identifier.
func (repo *GenericRepositoryGORM[T]) Get(ctx context.Context, id interface{}) (T, error) {
	endTrace := observability.StartTrace(ctx, "gorm_db_get_duration")
	defer endTrace()

	var result T
	err := repo.db.WithContext(ctx).First(&result, "id = ?", id).Error
	if err != nil {
		logger.Errorw("gorm_db_get_failure", "id", id, "error", err)
		return result, handleDBError(err)
	}

	logger.Infow("gorm_db_get_success", "record", fmt.Sprintf("%+v", result))
	return result, handleDBError(err)
}

// Update modifies a record by its identifier.
func (repo *GenericRepositoryGORM[T]) Update(ctx context.Context, id interface{}, data T) (T, error) {
	endTrace := observability.StartTrace(ctx, "gorm_db_update_duration")
	defer endTrace()

	err := repo.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(data).Error
	if err != nil {
		logger.Errorw("gorm_db_update_failure", "id", id, "error", err)
		return data, handleDBError(err)
	}

	logger.Infow("gorm_db_update_success", "id", id, "data", fmt.Sprintf("%+v", data))
	return data, handleDBError(err)
}

// Delete removes a record by its identifier.
func (repo *GenericRepositoryGORM[T]) Delete(ctx context.Context, id interface{}) error {
	endTrace := observability.StartTrace(ctx, "gorm_db_delete_duration")
	defer endTrace()

	err := repo.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error
	if err != nil {
		logger.Errorw("gorm_db_delete_failure", "id", id, "error", err)
		return err
	}

	logger.Infow("gorm_db_delete_success", "id", id)
	return err
}

// Find retrieves a single record based on a query.
func (repo *GenericRepositoryGORM[T]) Find(
	ctx context.Context,
	query repositories.Query[T],
) (T, error) {
	endTrace := observability.StartTrace(ctx, "gorm_db_find_duration")
	defer endTrace()

	var result T
	db := repo.db.WithContext(ctx).Model(new(T))

	db = applyConditions(db, query)

	err := db.First(&result).Error
	if err != nil {
		logger.Errorw("gorm_db_find_failure", "error", err)
		return result, handleDBError(err)
	}

	logger.Infow("gorm_db_find_success", "record", fmt.Sprintf("%+v", result))
	return result, handleDBError(err)
}

// FindAll retrieves multiple records based on a query.
func (repo *GenericRepositoryGORM[T]) FindAll(
	ctx context.Context,
	query repositories.Query[T],
) ([]T, error) {
	endTrace := observability.StartTrace(ctx, "gorm_db_find_all_duration")
	defer endTrace()

	var results []T
	db := repo.db.WithContext(ctx).Model(new(T))

	db = applyConditions(db, query)

	err := db.Find(&results).Error
	if err != nil {
		logger.Errorw("gorm_db_find_all_failure", "error", err)
		return results, handleDBError(err)
	}

	logger.Infow("gorm_db_find_all_success", "recordCount", len(results))
	return results, handleDBError(err)
}

// applyConditions applies conditions, sorting, limiting, and offsetting to a GORM database query.
// It takes a GORM database instance (db) and a generic query (repositories.Query) as input.
// The function dynamically constructs the WHERE clause based on the provided conditions and instance values.
// It also includes sorting, limiting, and offsetting based on the query parameters.
// The modified GORM database instance is returned.
func applyConditions[T any](db *gorm.DB, query repositories.Query[T]) *gorm.DB {
	// Retrieve the table name using the GORM naming strategy.
	tableName := db.NamingStrategy.TableName(reflect.TypeOf(*new(T)).Name())

	// Apply conditions specified in the query.
	for _, condition := range query.Conditions {
		columnName := db.NamingStrategy.ColumnName(tableName, condition.Field)
		db = db.Where(
			fmt.Sprintf("%s %s ?", columnName, condition.Operator),
			condition.Value,
		)
	}

	// Apply conditions based on non-zero values in the query instance.
	if !repositories.IsEmptyValue(query.Instance) {
		exampleType := reflect.TypeOf(query.Instance)
		exampleValue := reflect.ValueOf(query.Instance)
		for i := 0; i < exampleType.NumField(); i++ {
			fieldName := exampleType.Field(i).Name
			fieldValue := exampleValue.Field(i).Interface()
			if !repositories.IsEmptyValue(fieldValue) {
				columnName := db.NamingStrategy.ColumnName(tableName, fieldName)
				db = db.Where(fmt.Sprintf("%s = ?", columnName), fieldValue)
			}
		}
	}

	// Apply sorting if specified in the query.
	if query.SortBy != "" {
		dir := "ASC"
		if query.SortBy[0] == '-' {
			query.SortBy = query.SortBy[1:]
			dir = "DESC"
		}
		columnName := db.NamingStrategy.ColumnName(tableName, query.SortBy)
		db = db.Order(fmt.Sprintf("%s.%s %s", tableName, columnName, dir))
	}

	// Apply limit if specified in the query.
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	// Apply offset if specified in the query.
	if query.Offset > 0 {
		db = db.Limit(query.Offset)
	}

	return db
}
