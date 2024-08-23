package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func DiskComparator(l, r interface{}, _ ...Preference) types.Comparison {
	// comparator for Memory type

	// we want to reason about the inner fields of the Memory type and how they compare between left and right

	// validate input type
	_, lok := l.(types.Disk)
	_, rok := r.(types.Disk)
	if !lok || !rok {
		return types.Error
	}

	comparison := ReturnComplexComparison(l, r)

	if comparison["Type"] == types.Error {
		return types.Error
	}
	if comparison["Type"] != types.Equal {
		return types.Worse
	}

	return comparison["Size"]

	// currently this is a very simple comparison, based on the assumption
	// that more Size / or equal amount of size and speed is acceptable, but nothing less;
}
