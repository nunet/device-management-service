package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func CapabilityComparator(l, r interface{}, preference ...Preference) models.Comparison {
	var result models.Comparison
	result = models.Error // error is the default value
	// TODO: implement the comparison logic for the Capability type
	// after all the fields are implemented
	return result
} 