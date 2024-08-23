package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func MemoryComparator(l, r interface{}, _ ...Preference) types.Comparison {
	// comparator for Memory type

	// we want to reason about the inner fields of the Memory type and how they compare between left and right

	// validate input type
	_, lok := l.(types.RAM)
	_, rok := r.(types.RAM)
	if !lok || !rok {
		return types.Error
	}

	comparison := ReturnComplexComparison(l, r)

	if comparison["Size"] == types.Error {
		return types.Error
	}
	if comparison["Size"] == types.Worse {
		return types.Worse
	}

	return comparison["ClockSpeedHz"]

	// currently this is a very simple comparison, based on the assumption
	// that more Size / or equal amount of size and speed is acceptable, but nothing less;
}
