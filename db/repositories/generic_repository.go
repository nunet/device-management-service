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

// EQ creates a QueryCondition for equality comparison.
// It takes a field name and a value and returns a QueryCondition with the equality operator.
func EQ(field string, value interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: "=", Value: value}
}

// GT creates a QueryCondition for greater-than comparison.
// It takes a field name and a value and returns a QueryCondition with the greater-than operator.
func GT(field string, value interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: ">", Value: value}
}

// GTE creates a QueryCondition for greater-than or equal comparison.
// It takes a field name and a value and returns a QueryCondition with the greater-than or equal operator.
func GTE(field string, value interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: ">=", Value: value}
}

// LT creates a QueryCondition for less-than comparison.
// It takes a field name and a value and returns a QueryCondition with the less-than operator.
func LT(field string, value interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: "<", Value: value}
}

// LTE creates a QueryCondition for less-than or equal comparison.
// It takes a field name and a value and returns a QueryCondition with the less-than or equal operator.
func LTE(field string, value interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: "<=", Value: value}
}

// IN creates a QueryCondition for an "IN" comparison.
// It takes a field name and a slice of values and returns a QueryCondition with the "IN" operator.
func IN(field string, values []interface{}) QueryCondition {
	return QueryCondition{Field: field, Operator: "IN", Value: values}
}

// LIKE creates a QueryCondition for a "LIKE" comparison.
// It takes a field name and a pattern and returns a QueryCondition with the "LIKE" operator.
func LIKE(field, pattern string) QueryCondition {
	return QueryCondition{Field: field, Operator: "LIKE", Value: pattern}
}
