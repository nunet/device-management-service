package matching

import (
	//	"reflect"
	"golang.org/x/exp/slices"
	"gitlab.com/nunet/device-management-service/models"
)

func LocalitiesComparator(lraw interface{}, rraw interface{}, preference ...Preference) models.Comparison {
	// simplified version of Localities comparator
	// which is simply a slice of Locality type;
	// we do not have separate type defined for Localities
	// it takes preference variable where comparison Preference is defined
	// this is the first method that is used to take Preference variable into account
	// left represent machine capabilities;
	// right represent required capabilities;

	l := lraw.([]models.Locality)
	r := rraw.([]models.Locality)

	var interimComparison [](map[string]models.Comparison)
	for _, rLocality := range r {
		field := make(map[string]models.Comparison)
		field[rLocality.Kind] = models.Error
		for _, lLocality := range l {
			if lLocality.Kind == rLocality.Kind {
				field[rLocality.Kind] = Compare(lLocality, rLocality)
				// this is to make sure that we have a comparison even if slice dimentiones do not match
			}
		}
		interimComparison = append(interimComparison, field)
	}
		// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right
	var finalComparison []models.Comparison
	for _, c := range interimComparison {
		for _, v := range c { // we know that there is only one value in the map
			finalComparison = append(finalComparison, v)
		}
	}

	if slices.Contains(finalComparison, models.Error) {
		return models.Error
	}
	if slices.Contains(finalComparison, models.Worse) {
		return models.Worse
	}
	if SliceContainsOneValue(finalComparison, models.Equal) {
		return models.Equal
	}
	return models.Better
}