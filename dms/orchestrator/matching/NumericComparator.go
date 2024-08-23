package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

func NumericComparator(lraw, rraw interface{}, _ ...Preference) types.Comparison {
	// comparator for  numeric types:
	// left represent machine capabilities;
	// right represent required capabilities;

	var result types.Comparison
	result = types.Error // error is the default value

	// validate input type
	l, lnumeric := validate.ConvertNumericToFloat64(lraw)
	r, rnumeric := validate.ConvertNumericToFloat64(rraw)
	if !lnumeric || !rnumeric {
		result = types.Error
	}

	switch {
	case reflect.DeepEqual(l, r):
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		result = types.Equal

	case l < r:
		// if declared machine numeric capability
		// is less than job's required capability
		// then the result of comparison is 'Worse'
		result = types.Worse

	case l > r:
		// if declared machine numeric capability
		// is more than job's required numeric capability
		// then the result of comparison is 'Better'
		result = types.Better
	}

	return result
}
