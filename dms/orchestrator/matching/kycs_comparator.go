package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func KYCsComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.([]types.KYC)
	_, rrawok := rraw.([]types.KYC)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.([]types.KYC)
	r := rraw.([]types.KYC)

	//nolint
	if reflect.DeepEqual(l, r) {
		return types.Equal
	} else if len(r) == 0 && len(l) != 0 {
		return types.Better
	} else {
		for _, lkyc := range l {
			for _, rkyc := range r {
				if comp := Compare(lkyc, rkyc); comp == types.Equal {
					return types.Equal
				}
			}
		}
	}

	return types.Error
}
