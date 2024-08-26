package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
)

func TimeInformationComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.TimeInformation)
	_, rrawok := rraw.(types.TimeInformation)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.TimeInformation)
	r := rraw.(types.TimeInformation)

	if reflect.DeepEqual(l, r) {
		return types.Equal
	}
	lTotalTime := totalTime(l)
	rTotalTime := totalTime(r)

	//nolint
	if lTotalTime == rTotalTime {
		return types.Equal
	} else if lTotalTime < rTotalTime {
		return types.Worse
	}
	return types.Better
}

func totalTime(timeInfo types.TimeInformation) int {
	switch timeInfo.Units {
	case "seconds":
		return timeInfo.MaxTime
	case "minutes":
		return timeInfo.MaxTime * 60
	case "hours":
		return timeInfo.MaxTime * 60 * 60
	case "days":
		return timeInfo.MaxTime * 60 * 60 * 24
	default:
		return timeInfo.MaxTime
	}
}
