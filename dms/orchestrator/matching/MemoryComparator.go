package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func MemoryComparator(l, r interface{}, preference ...Preference) models.Comparison {
	// comparator for Memory type

	// we want to reason about the inner fields of the Memory type and how they compare between left and right

	// validate input type
	_, lok := l.(models.Memory)
	_, rok := r.(models.Memory)
	if !lok || !rok {
		return models.Error
	}

	comparison := ReturnComplexComparison(l, r)


	if comparison["Size"] == models.Error {
		return models.Error
	} 
	if comparison["Size"] == models.Worse {
		return models.Worse
	}

	return comparison["Speed"]

	// currently this is a very simple comparison, based on the assumption
	// that more Size / or equal amount of size and speed is acceptable, but nothing less;
}


