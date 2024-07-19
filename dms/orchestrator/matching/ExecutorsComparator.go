package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"
)

func ExecutorsComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for Executors types:
	// left represent machine capabilities;
	// right represent required capabilities;
	var result models.Comparison
	result = models.Error // error is the default value

	// validate input type
	_, lrawok := lraw.(models.Executors)
	_, rrawok := rraw.(models.Executors)
	if !lrawok || !rrawok {
		return models.Error
	}

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
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'		
		result = models.Better
	} else if utils.IsStrictlyContained(r, l) {
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		result = models.Worse
	}

	return result
}
