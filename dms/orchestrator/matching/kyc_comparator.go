package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func KYCComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.KYC)
	_, rrawok := rraw.(types.KYC)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.KYC)
	r := rraw.(types.KYC)

	if reflect.DeepEqual(l, r) {
		return types.Equal
	}

	return types.Error
}
