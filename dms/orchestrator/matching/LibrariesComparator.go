package matching

import (
	"gitlab.com/nunet/device-management-service/models"
	"golang.org/x/exp/slices"
)

func LibrariesComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for Libraries slices (of different lengths) of Library types:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.([]models.Library)
	_, rrawok := rraw.([]models.Library)
	if !lrawok || !rrawok {
		return models.Error
	}	

	l := lraw.([]models.Library)
	r := rraw.([]models.Library)

	var interimComparison1 [][]models.Comparison
	for _, rLibrary := range r {
		var interimComparison2 []models.Comparison
		for _, lLibrary := range l {
			interimComparison2 = append(interimComparison2, Compare(lLibrary, rLibrary))
		}
		// this matrix structure will hold the comparison results for each GPU on the right
		// with each GPU on the left in the order they are in the slices
		// first dimension represents left GPUs
		// second dimension represents right GPUs
		interimComparison1 = append(interimComparison1, interimComparison2)
	}
		// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right

	var finalComparison []models.Comparison
	var consideredIndexes []int
	for i := 0; i < len(interimComparison1); i++ {
		// we need to find the best match for each GPU on the right
		if len(interimComparison1[i]) < i {
			break
		}
		c := interimComparison1[i]
		bestMatch, index := returnBestMatch(c)
		finalComparison = append(finalComparison, bestMatch)
		consideredIndexes = append(consideredIndexes, index)
		interimComparison1 = removeIndex(interimComparison1, index)
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

