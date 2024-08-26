package matching

import "gitlab.com/nunet/device-management-service/types"

func StorageComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of (placeholder)
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.Storage)
	_, rrawok := rraw.(types.Storage)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.Storage)
	r := rraw.(types.Storage)

	if l.Type == r.Type {
		if l.Size*l.Amount == r.Size*l.Amount {
			return types.Equal
		} else if l.Size*l.Amount > r.Size*l.Amount {
			return types.Better
		}
		return types.Worse
	}

	if l.Type == types.SSD_STORAGE_TYPE && r.Type == types.HDD_STORAGE_TYPE {
		if l.Size*l.Amount == r.Size*r.Amount {
			return types.Better
		} else if l.Size*l.Amount > r.Size*r.Amount {
			return types.Better
		}
		return types.Worse
	} else if l.Type == types.HDD_STORAGE_TYPE && r.Type == types.SSD_STORAGE_TYPE {
		return types.Worse
	}

	return types.Error
}
