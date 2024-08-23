package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func CapabilityComparator(_, _ interface{}, _ ...Preference) types.Comparison {
	// TODO: implement the comparison logic for the Capability type
	// after all the fields are implemented
	return types.Error
}
