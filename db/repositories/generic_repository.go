// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package repositories

import (
	"context"
)

// QueryCondition is a struct representing a query condition.
type QueryCondition struct {
	Field    string      // Field specifies the database or struct field to which the condition applies.
	Operator string      // Operator defines the comparison operator (e.g., "=", ">", "<").
	Value    interface{} // Value is the expected value for the given field.
}

type ModelType interface{}

// Query is a struct that wraps both the instance of type T and additional query parameters.
// It is used to construct queries with conditions, sorting, limiting, and offsetting.
type Query[T any] struct {
	Instance   T                // Instance is an optional object of type T used to build conditions from its fields.
	Conditions []QueryCondition // Conditions represent the conditions applied to the query.
	SortBy     string           // SortBy specifies the field by which the query results should be sorted.
	Limit      int              // Limit specifies the maximum number of results to return.
	Offset     int              // Offset specifies the number of results to skip before starting to return data.
}

// GenericRepository is an interface defining basic CRUD operations and standard querying methods.
type GenericRepository[T ModelType] interface {
	// Create adds a new record to the repository.
	Create(ctx context.Context, data T) (T, error)
	// Get retrieves a record by its identifier.
	Get(ctx context.Context, id interface{}) (T, error)
	// Update modifies a record by its identifier.
	Update(ctx context.Context, id interface{}, data T) (T, error)
	// Delete removes a record by its identifier.
	Delete(ctx context.Context, id interface{}) error
	// Find retrieves a single record based on a query.
	Find(ctx context.Context, query Query[T]) (T, error)
	// FindAll retrieves multiple records based on a query.
	FindAll(ctx context.Context, query Query[T]) ([]T, error)
	// GetQuery returns an empty query instance for the repository's type.
	GetQuery() Query[T]
}
