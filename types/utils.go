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
	// while i feel that there could be some weird matrix sorting algorithm that could be used here
	// i can't think of any right now, so i will just iterate over the matrix and return matches
	// in somewhat manual way

	for i, v := range dimension {
		if v == Equal {
			return v, i // selecting an equal match is the most efficient match
		}
	}
	for i, v := range dimension {
		if v == Better {
			return v, i // selecting a better is also not bad
		}
	}
	for i, v := range dimension {
		if v == Worse {
			return v, i // this is just for sport
		}
	}
	for i, v := range dimension {
		if v == Error {
			return v, i // this is just for sport
		}
	}
	return Error, -1
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
