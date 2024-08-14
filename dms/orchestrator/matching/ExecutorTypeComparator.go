package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func ExecutorTypeComparator(l, r interface{}, preference ...Preference) types.Comparison {

	_, lok := l.(types.ExecutorType)
	_, rok := r.(types.ExecutorType)
	if !lok || !rok {
		return types.Error
	}	
	result := types.Error // default answer is error
	if reflect.DeepEqual(l, r) {
		result = types.Equal
	}
	return result
}