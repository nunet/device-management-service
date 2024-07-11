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
		result = models.Better
	} else if utils.IsStrictlyContained(r, l) {
		// if declared machine numeric capability
		// is more than jobs required numeric capability
		// then the result of comparison is 'More'
		// ("more is required than available")
		result = models.Worse

		// TODO: this comparator does not take into account options when several job types are available and several job types are required
		// in the same data structure; this is why the test fails;
	}


	return result
}
