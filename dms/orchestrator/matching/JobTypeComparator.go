package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
)

func JobTypeComparator(l, r interface{}, preference ...Preference) models.Comparison {

	// validate input type
	_, lok := l.(models.JobType)
	_, rok := r.(models.JobType)
	if !lok || !rok {
		return models.Error
	}

	result := models.Error // default answer is error
	if reflect.DeepEqual(l, r) {
		result = models.Equal
	}
	return result
}