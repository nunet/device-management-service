package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func ExecutorsComparator(lraw, rraw interface{}, _ ...Preference) types.Comparison {
	// comparator for Executors types:
	// left represent machine capabilities;
	// right represent required capabilities;
	var result types.Comparison
	result = types.Error // error is the default value

	// validate input type
	ll, lrawok := lraw.(types.Executors)
	rr, rrawok := rraw.(types.Executors)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := make([]interface{}, 0)
	r := make([]interface{}, 0)
	for _, v := range ll {
		l = append(l, v)
	}
	for _, v := range rr {
		r = append(r, v)
	}

	if !utils.IsSameShallowType(l, r) {
		result = types.Error
	}

	switch {
	case reflect.DeepEqual(l, r):
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		result = types.Equal

	case utils.IsStrictlyContained(l, r):
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'
		result = types.Better

	case utils.IsStrictlyContained(r, l):
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		result = types.Worse
	}

	return result
}
