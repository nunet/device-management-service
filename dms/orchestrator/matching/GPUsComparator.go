package matching

import (
	"gitlab.com/nunet/device-management-service/models"
	"golang.org/x/exp/slices"
)

func GPUsComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for GPUs type which is just a slice of GPU types:
	// left represent machine capabilities;
	// right represent required capabilities;
	// we need to check if for ech GPU on the right there exist a matching GPU on the left...
    // (since given slices are not ordered...)

	// validate input type
	_, lrawok := lraw.([]models.GPU)
	_, rrawok := rraw.([]models.GPU)
	if !lrawok || !rrawok {
		return models.Error
	}

	l := lraw.([]models.GPU)
	r := rraw.([]models.GPU)

	var interimComparison1 [][]models.Comparison
	for _, rGPU := range r {
		var interimComparison2 []models.Comparison
		for _, lGPU := range l {
			interimComparison2 = append(interimComparison2, Compare(lGPU, rGPU))
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



