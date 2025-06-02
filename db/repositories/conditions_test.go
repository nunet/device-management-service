// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEQ(t *testing.T) {
	t.Parallel()
	// Test with string value
	condition := EQ("name", "John")
	assert.Equal(t, "name", condition.Field)
	assert.Equal(t, "=", condition.Operator)
	assert.Equal(t, "John", condition.Value)

	// Test with int value
	condition = EQ("age", 30)
	assert.Equal(t, "age", condition.Field)
	assert.Equal(t, "=", condition.Operator)
	assert.Equal(t, 30, condition.Value)

	// Test with bool value
	condition = EQ("active", true)
	assert.Equal(t, "active", condition.Field)
	assert.Equal(t, "=", condition.Operator)
	assert.Equal(t, true, condition.Value)
}

func TestGT(t *testing.T) {
	t.Parallel()
	// Test with int value
	condition := GT("age", 30)
	assert.Equal(t, "age", condition.Field)
	assert.Equal(t, ">", condition.Operator)
	assert.Equal(t, 30, condition.Value)

	// Test with float value
	condition = GT("price", 19.99)
	assert.Equal(t, "price", condition.Field)
	assert.Equal(t, ">", condition.Operator)
	assert.Equal(t, 19.99, condition.Value)
}

func TestGTE(t *testing.T) {
	t.Parallel()
	// Test with int value
	condition := GTE("age", 30)
	assert.Equal(t, "age", condition.Field)
	assert.Equal(t, ">=", condition.Operator)
	assert.Equal(t, 30, condition.Value)

	// Test with float value
	condition = GTE("price", 19.99)
	assert.Equal(t, "price", condition.Field)
	assert.Equal(t, ">=", condition.Operator)
	assert.Equal(t, 19.99, condition.Value)
}

func TestLT(t *testing.T) {
	t.Parallel()
	// Test with int value
	condition := LT("age", 30)
	assert.Equal(t, "age", condition.Field)
	assert.Equal(t, "<", condition.Operator)
	assert.Equal(t, 30, condition.Value)

	// Test with float value
	condition = LT("price", 19.99)
	assert.Equal(t, "price", condition.Field)
	assert.Equal(t, "<", condition.Operator)
	assert.Equal(t, 19.99, condition.Value)
}

func TestLTE(t *testing.T) {
	t.Parallel()
	// Test with int value
	condition := LTE("age", 30)
	assert.Equal(t, "age", condition.Field)
	assert.Equal(t, "<=", condition.Operator)
	assert.Equal(t, 30, condition.Value)

	// Test with float value
	condition = LTE("price", 19.99)
	assert.Equal(t, "price", condition.Field)
	assert.Equal(t, "<=", condition.Operator)
	assert.Equal(t, 19.99, condition.Value)
}

func TestIN(t *testing.T) {
	t.Parallel()
	// Test with string slice
	values := []interface{}{"apple", "banana", "orange"}
	condition := IN("fruit", values)
	assert.Equal(t, "fruit", condition.Field)
	assert.Equal(t, "IN", condition.Operator)
	assert.Equal(t, values, condition.Value)

	// Test with int slice
	intValues := []interface{}{1, 2, 3}
	condition = IN("id", intValues)
	assert.Equal(t, "id", condition.Field)
	assert.Equal(t, "IN", condition.Operator)
	assert.Equal(t, intValues, condition.Value)

	// Test with empty slice
	emptyValues := []interface{}{}
	condition = IN("category", emptyValues)
	assert.Equal(t, "category", condition.Field)
	assert.Equal(t, "IN", condition.Operator)
	assert.Equal(t, emptyValues, condition.Value)
}

func TestLIKE(t *testing.T) {
	t.Parallel()
	// Test with prefix pattern
	condition := LIKE("name", "John%")
	assert.Equal(t, "name", condition.Field)
	assert.Equal(t, "LIKE", condition.Operator)
	assert.Equal(t, "John%", condition.Value)

	// Test with suffix pattern
	condition = LIKE("email", "%@example.com")
	assert.Equal(t, "email", condition.Field)
	assert.Equal(t, "LIKE", condition.Operator)
	assert.Equal(t, "%@example.com", condition.Value)

	// Test with contains pattern
	condition = LIKE("description", "%keyword%")
	assert.Equal(t, "description", condition.Field)
	assert.Equal(t, "LIKE", condition.Operator)
	assert.Equal(t, "%keyword%", condition.Value)

	// Test with empty pattern
	condition = LIKE("title", "")
	assert.Equal(t, "title", condition.Field)
	assert.Equal(t, "LIKE", condition.Operator)
	assert.Equal(t, "", condition.Value)
}
