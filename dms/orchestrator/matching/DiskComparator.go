package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func DiskComparator(l, r interface{}, preference ...Preference) models.Comparison {
	// comparator for Memory type

	// we want to reason about the inner fields of the Memory type and how they compare between left and right

	// validate input type
	_, lok := l.(models.Disk)
	_, rok := r.(models.Disk)
	if !lok || !rok {
		return models.Error
	}

	comparison := ReturnComplexComparison(l, r)


	if comparison["Type"] == models.Error {
		return models.Error
	} 
	if comparison["Type"] != models.Equal {
		return models.Worse
	}

	return comparison["Size"]

	// currently this is a very simple comparison, based on the assumption
	// that more Size / or equal amount of size and speed is acceptable, but nothing less;
}



