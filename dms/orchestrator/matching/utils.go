package matching

import (
	"reflect"
	"gitlab.com/nunet/device-management-service/types"
)


func ReturnComplexComparison(l, r interface{}) types.ComplexComparison {
	// Complex comparison is a comparison of two complex types
	// Which have nested fields that need to be considered together
	// before a final comparison for the whole complex type can be made
	// it is a helper function used in some type comparators
	vl := reflect.ValueOf(l)
	vr := reflect.ValueOf(r)
	var complexComparison types.ComplexComparison = make(types.ComplexComparison)
	for i := 0; i < vl.NumField(); i++ {
		innerTypeName := vl.Type().Field(i).Name
		valueL := vl.Field(i).Interface()
		valueR := vr.Field(i).Interface()
		complexComparison[innerTypeName] = Compare(valueL, valueR)
	}
	return complexComparison
}

func removeIndex(slice [][]types.Comparison, index int) [][]types.Comparison {
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

func returnBestMatch(dimension []types.Comparison) (types.Comparison, int) {

	// while i feel that there could be some weird matrix sorting algorithm that could be used here
	// i can't think of any right now, so i will just iterate over the matrix and return matches
	// in somewhat manual way

	for i, v := range dimension {
		if v == types.Equal {
			return v, i // selecting an equal match is the most efficient match
		}
	}
	for i, v := range dimension {
		if v == types.Better {
			return v, i // selecting a better is also not bad
		}
	}
	for i, v := range dimension {
		if v == types.Worse {
			return v, i // this is just for sport
		}	
	}
	for i, v := range dimension {
		if v == types.Error {
			return v, i // this is just for sport
		}
	}
	return types.Error, -1
}

func SliceContainsOneValue(slice []types.Comparison, value types.Comparison) bool {
	// returns true if all elements in the slice are equal to the given value
	for _, v := range slice {
		if v != value {
			return false
		}
	}
	return true
}
	