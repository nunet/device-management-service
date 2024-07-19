package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func LocalityComparator(lraw interface{}, rraw interface{}, preference ...Preference) models.Comparison {
	// simplified version of (placeholder)
	// comparator for Locality:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(models.Locality)
	_, rrawok := rraw.(models.Locality)
	if !lrawok || !rrawok {
		return models.Error
	}		

	l := lraw.(models.Locality)
	r := rraw.(models.Locality)

	if l.Kind == r.Kind {
		if l.Name == r.Name {
			return models.Equal
		} else {
			return models.Worse
		}
	} else {
		return models.Error
	}	

}