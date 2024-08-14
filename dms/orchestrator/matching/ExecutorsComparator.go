package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func ExecutorsComparator(lraw, rraw interface{}, preference ...Preference) types.Comparison {
	// comparator for Executors types:
	// left represent machine capabilities;
	// right represent required capabilities;
	var result types.Comparison
	result = types.Error // error is the default value

	// validate input type
	_, lrawok := lraw.(types.Executors)
	_, rrawok := rraw.(types.Executors)
	if !lrawok || !rrawok {
		return types.Error
	}

	var l []interface{} 
	l = lraw.([]interface{})
	var r []interface{}
	r = rraw.([]interface{})

	if !utils.IsSameShallowType(l, r) {
		result = types.Error
	}
	if reflect.DeepEqual(l, r) {
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		result = types.Equal
	} else if utils.IsStrictlyContained(l, r) {
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'		
		result = types.Better
	} else if utils.IsStrictlyContained(r, l) {
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		result = types.Worse
	}

	return result
}
