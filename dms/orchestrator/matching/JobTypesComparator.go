package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"
)

func JobTypesComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for JobTypes type:
	// left represent machine capabilities;
	// right represent required capabilities;
	// if machine capabilities contain oll the required capabilities, then we are good to go

	// validate input type
	_, lrawok := lraw.(models.JobTypes)
	_, rrawok := rraw.(models.JobTypes)
	if !lrawok || !rrawok {
		return models.Error
	}

	var result models.Comparison
	result = models.Error // error is the default value

	// we know that interfaces here are slices, so need to assert first
	var l []interface{} = utils.ConvertTypedSliceToUntypedSlice(lraw)
	var r []interface{} = utils.ConvertTypedSliceToUntypedSlice(rraw)

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

		// TODO: this comparator does not take into account options when several job types are available and several job types are required
		// in the same data structure; this is why the test fails;
	}


	return result
}
