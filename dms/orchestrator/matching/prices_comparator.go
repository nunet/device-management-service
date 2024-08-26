package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func PricesInformationComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	// validate input type
	_, lrawok := lraw.([]types.PriceInformation)
	_, rrawok := rraw.([]types.PriceInformation)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.([]types.PriceInformation)
	r := rraw.([]types.PriceInformation)

	if reflect.DeepEqual(l, r) {
		return types.Equal
	}

	comparison := types.Error
	for _, lPrice := range l {
		for _, rPrice := range r {
			if comparison = Compare(lPrice, rPrice); comparison != types.Error {
				return comparison
			}
		}
	}

	return comparison
}
