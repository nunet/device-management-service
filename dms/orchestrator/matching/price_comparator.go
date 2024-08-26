package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func PriceInformationComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.PriceInformation)
	_, rrawok := rraw.(types.PriceInformation)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.PriceInformation)
	r := rraw.(types.PriceInformation)

	//nolint
	if reflect.DeepEqual(l, r) {
		return types.Equal
	} else if l.Currency == r.Currency {
		//nolint
		if l.TotalPerJob == r.TotalPerJob {
			if l.CurrencyPerHour == r.CurrencyPerHour {
				return types.Equal
			} else if l.CurrencyPerHour < r.CurrencyPerHour {
				return types.Better
			} else {
				return types.Worse
			}
		} else if l.TotalPerJob < r.TotalPerJob {
			if l.CurrencyPerHour <= r.CurrencyPerHour {
				return types.Better
			} else {
				return types.Worse
			}
		} else {
			return types.Worse
		}
	}

	return types.Error
}
