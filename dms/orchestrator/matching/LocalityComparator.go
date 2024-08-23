package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func LocalityComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// comparator for Locality:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.Locality)
	_, rrawok := rraw.(types.Locality)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.Locality)
	r := rraw.(types.Locality)

	if l.Kind == r.Kind {
		if l.Name == r.Name {
			return types.Equal
		}
		return types.Worse
	}

	return types.Error
}
