package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

func NumericComparator(lraw, rraw interface{}, preference ...Preference) types.Comparison {
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

	if reflect.DeepEqual(l, r) {
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		result = types.Equal
	} else if l < r {
		// if declared machine numeric capability
		// is less than jobs required capability
		// then the result of comparison in 'Less'
		// ("less is required than available")
		result = types.Worse
	} else if l > r {
		// if declared machine numeric capability
		// is more than jobs required numeric capability
		// then the result of comparison is 'More'
		// ("more is required than available")
		result = types.Better
	}
	return result
}
