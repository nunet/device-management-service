package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func StoragesComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.([]types.Storage)
	_, rrawok := rraw.([]types.Storage)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.([]types.Storage)
	r := rraw.([]types.Storage)

	rAcc := map[string]map[string]int{
		"ssd": {
			"size":   0,
			"amount": 0,
		},
		"hdd": {
			"size":   0,
			"amount": 0,
		},
	}

	for _, rstrg := range r {
		if rstrg.Type == "ssd" {
			rAcc["ssd"]["size"] += rstrg.Size * rstrg.Amount
		} else if rstrg.Type == "hdd" {
			rAcc["hdd"]["size"] += rstrg.Size * rstrg.Amount
		}
	}

	lAcc := map[string]map[string]int{
		"ssd": {
			"size":   0,
			"amount": 0,
		},
		"hdd": {
			"size":   0,
			"amount": 0,
		},
	}

	for _, lstrg := range l {
		if lstrg.Type == "ssd" {
			lAcc["ssd"]["size"] += lstrg.Size * lstrg.Amount
		} else if lstrg.Type == "hdd" {
			lAcc["hdd"]["size"] += lstrg.Size * lstrg.Amount
		}
	}

	// compare
	totalRequestedSSD := rAcc["ssd"]["size"]
	totalRequestedHDD := rAcc["hdd"]["size"]

	totalAvailableSSD := lAcc["ssd"]["size"]
	totalAvailableHDD := lAcc["hdd"]["size"]

	// if hdd is being requested but we don't have it
	//nolint
	if totalAvailableHDD == 0 && totalRequestedHDD > 0 {
		// if ssd is better than ssd and hdd combined
		if totalAvailableSSD >= totalRequestedSSD+totalRequestedHDD {
			return types.Better
		}

		return types.Worse
	} else if totalAvailableSSD < totalRequestedSSD {
		return types.Worse
	} else if totalAvailableSSD == totalRequestedSSD && totalAvailableHDD == totalRequestedHDD {
		return types.Equal
	} else if totalAvailableSSD >= totalRequestedSSD && totalAvailableHDD >= totalRequestedHDD {
		return types.Better
	}
	return types.Error
}
