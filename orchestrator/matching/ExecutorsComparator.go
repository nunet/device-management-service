package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"
)

func ExecutorsComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for  numeric types:
	// left represent machine capabilities;
	// right represent required capabilities;
	var result models.Comparison
	result = models.Error // error is the default value

	var l []interface{} 
	l = lraw.([]interface{})
	var r []interface{}
	r = rraw.([]interface{})

	if !utils.IsSameShallowType(l, r) {
		result = models.Error
	}
	if reflect.DeepEqual(l, r) {
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		result = models.Equal
	} else if utils.IsStrictlyContained(l, r) {
		result = models.Worse
	} else if utils.IsStrictlyContained(r, l) {
		// if declared machine numeric capability
		// is more than jobs required numeric capability
		// then the result of comparison is 'More'
		// ("more is required than available")
		result = models.Better
	}

	return result
}
