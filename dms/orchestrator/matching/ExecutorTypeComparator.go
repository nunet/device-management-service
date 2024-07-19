package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
)

func ExecutorTypeComparator(l, r interface{}, preference ...Preference) models.Comparison {

	_, lok := l.(models.ExecutorType)
	_, rok := r.(models.ExecutorType)
	if !lok || !rok {
		return models.Error
	}	
	result := models.Error // default answer is error
	if reflect.DeepEqual(l, r) {
		result = models.Equal
	}
	return result
}