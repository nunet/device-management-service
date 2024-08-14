package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func CapabilityComparator(l, r interface{}, preference ...Preference) types.Comparison {
	var result types.Comparison
	result = types.Error // error is the default value
	// TODO: implement the comparison logic for the Capability type
	// after all the fields are implemented
	return result
} 