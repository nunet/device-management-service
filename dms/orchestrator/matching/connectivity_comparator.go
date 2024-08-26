package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func ConnectivityComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.Connectivity)
	_, rrawok := rraw.(types.Connectivity)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.Connectivity)
	r := rraw.(types.Connectivity)

	//nolint
	if reflect.DeepEqual(l, r) {
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		return types.Equal
	} else if (utils.IsStrictlyContainedInt(l.Ports, r.Ports)) && (l.VPN && r.VPN || l.VPN && !r.VPN) {
		return types.Better
	} else {
		return types.Worse
	}
}
