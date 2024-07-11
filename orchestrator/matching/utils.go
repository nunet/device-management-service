package matching

import (
	"reflect"
	"gitlab.com/nunet/device-management-service/models"
)


func ReturnComplexComparison(l, r interface{}) models.ComplexComparison {
	vl := reflect.ValueOf(l)
	vr := reflect.ValueOf(r)
	var complexComparison models.ComplexComparison = make(models.ComplexComparison)
	for i := 0; i < vl.NumField(); i++ {
		innerTypeName := vl.Type().Field(i).Name
		valueL := vl.Field(i).Interface()
		valueR := vr.Field(i).Interface()
		complexComparison[innerTypeName] = Compare(valueL, valueR)
	}
	return complexComparison
}

func removeIndex(slice [][]models.Comparison, index int) [][]models.Comparison {
	for i, c := range slice {
		slice[i] = append(c[:index], c[index+1:]...)
	}
	return slice
}

func returnBestMatch(dimension []models.Comparison) (models.Comparison, int) {

	// while i feel that there could be some weird matrix sorting algorithm that could be used here
	// i can't think of any right now, so i will just iterate over the matrix and return matches
	// in somewhat manual way

	for i, v := range dimension {
		if v == models.Equal {
			return v, i // selecting an equal match is the most efficient match
		}
	}
	for i, v := range dimension {
		if v == models.Better {
			return v, i // selecting a better is also not bad
		}
	}
	for i, v := range dimension {
		if v == models.Worse {
			return v, i // this is just for sport
		}	
	}
	for i, v := range dimension {
		if v == models.Error {
			return v, i // this is just for sport
		}
	}
	return models.Error, -1
}

func SliceContainsOneValue(slice []models.Comparison, value models.Comparison) bool {
	result := true
	for _, v := range slice {
		if v != value {
			result = result && false
		}
	}
	return result
}