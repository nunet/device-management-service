// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import "reflect"

// Comparable public Comparable interface to be enforced on types that can be compared
type Comparable[T any] interface {
	Compare(other T) (Comparison, error)
}

type Calculable[T any] interface {
	Add(other T) error
	Subtract(other T) error
}

type PreferenceString string

const (
	Hard PreferenceString = "Hard"
	Soft PreferenceString = "Soft"
)

type Preference struct {
	TypeName                  string
	Strength                  PreferenceString
	DefaultComparatorOverride Comparator
}

type Comparator func(l, r any, preference ...Preference) Comparison

// ComplexCompare helper function to return a complex comparison of two complex types
// this uses reflection, could become a performance bottleneck
// with generics, we wouldn't need this function and could use the ComplexCompare method directly
func ComplexCompare(l, r any) ComplexComparison {
	// Complex comparison is a comparison of two complex types
	// Which have nested fields that need to be considered together
	// before a final comparison for the whole complex type can be made
	// it is a helper function used in some type comparators

	complexComparison := make(map[string]Comparison)

	val1 := reflect.ValueOf(l)
	val2 := reflect.ValueOf(r)

	// handle pointers
	if val1.Kind() == reflect.Pointer {
		val1 = val1.Elem()
	}
	if val2.Kind() == reflect.Pointer {
		val2 = val2.Elem()
	}

	// ensure we're working with struct types
	if val1.Kind() != reflect.Struct || val2.Kind() != reflect.Struct {
		// Question: should return error?
		return complexComparison
	}

	for i := range val1.NumField() {
		fieldName := val1.Type().Field(i).Name
		field1 := val1.Field(i)
		field2 := val2.Field(i)

		// compare primitive types directly
		switch field1.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			complexComparison[fieldName] = NumericComparator(field1.Int(), field2.Int())
			continue
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			complexComparison[fieldName] = NumericComparator(field1.Uint(), field2.Uint())
			continue
		case reflect.Float32, reflect.Float64:
			complexComparison[fieldName] = NumericComparator(field1.Float(), field2.Float())
			continue
		case reflect.String:
			complexComparison[fieldName] = LiteralComparator(field1.String(), field2.String())
			continue
		}

		// for struct fields, try to use their Compare method if available
		if field1.Kind() == reflect.Struct {
			// try to find a Compare method on the field
			var field1Ptr reflect.Value
			if field1.CanAddr() {
				field1Ptr = field1.Addr()
			} else {
				ptr := reflect.New(field1.Type())
				ptr.Elem().Set(field1)
				field1Ptr = ptr
			}

			compareMethod := field1Ptr.MethodByName("Compare")
			if compareMethod.IsValid() &&
				compareMethod.Type().NumIn() == 1 &&
				compareMethod.Type().In(0) == field1.Type() {
				result := compareMethod.Call([]reflect.Value{field2})
				if len(result) > 0 {
					comp, ok := result[0].Interface().(Comparison)
					if ok {
						complexComparison[fieldName] = comp
						continue
					}
				}
			}
		}

		// default to None if no comparison could be made
		complexComparison[fieldName] = None
	}

	return complexComparison
}

func NumericComparator[T float64 | float32 | int | int32 | int64 | uint64](l, r T, _ ...Preference) Comparison {
	// comparator for numeric types:
	// left represents machine capabilities;
	// right represents required capabilities;

	switch {
	case l == r:
		return Equal

	case l < r:
		return Worse

	default:
		return Better
	}
}

func LiteralComparator[T ~string](l, r T, _ ...Preference) Comparison {
	// comparator for literal (string-like) types:
	// left represents machine capabilities;
	// right represents required capabilities;
	// which can only be equal or not equal.

	// ComplexCompare the string values
	if l == r {
		return Equal
	}

	return None
}
