package matching

import (
	//	"reflect"
	"gitlab.com/nunet/device-management-service/types"
	"golang.org/x/exp/slices"
)

func LocalitiesComparator(lraw interface{}, rraw interface{}, _ ...Preference) types.Comparison {
	// simplified version of Localities comparator
	// which is simply a slice of Locality type;
	// we do not have separate type defined for Localities
	// it takes preference variable where comparison Preference is defined
	// this is the first method that is used to take Preference variable into account
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.([]types.Locality)
	_, rrawok := rraw.([]types.Locality)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.([]types.Locality)
	r := rraw.([]types.Locality)

	interimComparison := make([](map[string]types.Comparison), 0)
	for _, rLocality := range r {
		field := make(map[string]types.Comparison)
		field[rLocality.Kind] = types.Error
		for _, lLocality := range l {
			if lLocality.Kind == rLocality.Kind {
				field[rLocality.Kind] = Compare(lLocality, rLocality)
				// this is to make sure that we have a comparison even if slice dimentiones do not match
			}
		}
		interimComparison = append(interimComparison, field)
	}
	// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right
	var finalComparison []types.Comparison
	for _, c := range interimComparison {
		for _, v := range c { // we know that there is only one value in the map
			finalComparison = append(finalComparison, v)
		}
	}

	if slices.Contains(finalComparison, types.Error) {
		return types.Error
	}
	if slices.Contains(finalComparison, types.Worse) {
		return types.Worse
	}
	if SliceContainsOneValue(finalComparison, types.Equal) {
		return types.Equal
	}
	return types.Better
}
