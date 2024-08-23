package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func JobTypesComparator(lraw, rraw interface{}, _ ...Preference) types.Comparison {
	// comparator for JobTypes type:
	// left represent machine capabilities;
	// right represent required capabilities;
	// if machine capabilities contain oll the required capabilities, then we are good to go

	// validate input type
	_, lrawok := lraw.(types.JobTypes)
	_, rrawok := rraw.(types.JobTypes)
	if !lrawok || !rrawok {
		return types.Error
	}

	result := types.Error // error is the default value

	// we know that interfaces here are slices, so need to assert first
	l := utils.ConvertTypedSliceToUntypedSlice(lraw)
	r := utils.ConvertTypedSliceToUntypedSlice(rraw)

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

		// TODO: this comparator does not take into account options when several job types are available and several job types are required
		// in the same data structure; this is why the test fails;
	}

	return result
}
