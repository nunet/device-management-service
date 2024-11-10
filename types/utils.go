// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"math"
	"reflect"
	"slices"
)

// IsStrictlyContained checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContained(leftSlice, rightSlice []interface{}) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}

func IsSameShallowType(a, b interface{}) bool {
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)
	result := aType == bType
	return result
}

// round rounds the value to the specified number of decimal places
func round[T float32 | float64](value T, places int) T {
	factor := math.Pow(10, float64(places))
	roundedValue := math.Round(float64(value)*factor) / factor
	return T(roundedValue)
}

func SliceContainsOneValue(slice []Comparison, value Comparison) bool {
	// returns true if all elements in the slice are equal to the given value
	for _, v := range slice {
		if v != value {
			return false
		}
	}
	return true
}

func returnBestMatch(dimension []Comparison) (Comparison, int) {
	bestMatch := None
	bestIndex := -1

	for i, v := range dimension {
		switch v {
		case Equal:
			return v, i // Equal is the best, so return immediately
		case Better:
			if bestMatch != Better { // Prioritize Better only if we haven't seen one yet
				bestMatch = Better
				bestIndex = i
			}
		case Worse:
			if bestMatch == None { // Prioritize Worse only if nothing better found
				bestMatch = Worse
				bestIndex = i
			}
		default:
			// Ignore None
		}
	}

	return bestMatch, bestIndex
}

func removeIndex(slice [][]Comparison, index int) [][]Comparison {
	// removeIndex removes the element at the specified index from each sub-slice in the given slice.
	// If the index is out of bounds for a sub-slice, the function leaves that sub-slice unmodified.
	for i, c := range slice {
		if index < 0 || index >= len(c) {
			// Index is out of bounds, leave the sub-slice unmodified
			continue
		}
		slice[i] = append(c[:index], c[index+1:]...)
	}
	return slice
}

func ConvertTypedSliceToUntypedSlice(typedSlice interface{}) []interface{} {
	s := reflect.ValueOf(typedSlice)
	if s.Kind() != reflect.Slice {
		return nil
	}
	result := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		result[i] = s.Index(i).Interface()
	}
	return result
}

// IsStrictlyContainedInt checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContainedInt(leftSlice, rightSlice []int) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}
