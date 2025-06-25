// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package repositories

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
