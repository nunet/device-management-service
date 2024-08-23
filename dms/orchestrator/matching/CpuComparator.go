package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func CPUComparator(l, r interface{}, _ ...Preference) types.Comparison {
	// comparator for CPU type

	// we want to reason about the inner fields of the CPU type and how they compare between left and right

	// validate input type
	lCPU, lok := l.(types.CPU)
	rCPU, rok := r.(types.CPU)
	if !lok || !rok {
		return types.Error
	}

	perfComparison := NumericComparator(
		(int64(lCPU.Cores) * lCPU.ClockSpeedHz),
		(int64(rCPU.Cores) * rCPU.ClockSpeedHz),
	)

	archComparison := LiteralComparator(lCPU.Architecture, rCPU.Architecture)

	if archComparison == types.Error {
		return types.Error
	}
	if archComparison != types.Equal {
		return types.Worse
	}

	return perfComparison

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and frequency is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare types.of CPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;
}
