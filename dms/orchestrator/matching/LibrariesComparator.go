package matching

import (
	"gitlab.com/nunet/device-management-service/types"
	"golang.org/x/exp/slices"
)

func LibrariesComparator(lraw, rraw interface{}, _ ...Preference) types.Comparison {
	// comparator for Libraries slices (of different lengths) of Library types:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.([]types.Library)
	_, rrawok := rraw.([]types.Library)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.([]types.Library)
	r := rraw.([]types.Library)

	interimComparison1 := make([][]types.Comparison, 0)
	for _, rLibrary := range r {
		var interimComparison2 []types.Comparison
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

	finalComparison := make([]types.Comparison, 0)
	for i := 0; i < len(interimComparison1); i++ {
		// we need to find the best match for each GPU on the right
		if len(interimComparison1[i]) < i {
			break
		}
		c := interimComparison1[i]
		bestMatch, index := returnBestMatch(c)
		finalComparison = append(finalComparison, bestMatch)
		interimComparison1 = removeIndex(interimComparison1, index)
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
