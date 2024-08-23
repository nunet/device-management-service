package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func JobTypeComparator(l, r interface{}, _ ...Preference) types.Comparison {
	// validate input type
	_, lok := l.(types.JobType)
	_, rok := r.(types.JobType)
	if !lok || !rok {
		return types.Error
	}

	result := types.Error // default answer is error
	if reflect.DeepEqual(l, r) {
		result = types.Equal
	}
	return result
}
