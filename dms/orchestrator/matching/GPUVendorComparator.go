package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
)

func GPUVendorComparator(l, r interface{}, preference ...Preference) models.Comparison {

	// validate input type
	_, lok := l.(models.GPUVendor)
	_, rok := r.(models.GPUVendor)
	if !lok || !rok {
		return models.Error
	}

	result := models.Error // default answer is error
	if reflect.DeepEqual(l, r) {
		result = models.Equal
	}
	return result

	// This comparison logic just tells if the vendor is the same or not;
	// however, we do not have yet a mechanism for externally defined preferences from a user;
	// in this case, we may need to implement that -- because some compute may prefer one vendor over the other;
	// some compute may be strictly dependent on a specific vendor;
	// technically, this will have to be solved on the resource matching level;
	// but the mechanism will have to be generic...
}