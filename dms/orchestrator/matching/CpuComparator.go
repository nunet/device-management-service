package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func CpuComparator(l, r interface{}, preference ...Preference) types.Comparison {
	// comparator for CPU type

	// we want to reason about the inner fields of the CPU type and how they compare between left and right

	// validate input type
	lCpu, lok := l.(types.CPU)
	rCpu, rok := r.(types.CPU)
	if !lok || !rok {
		return types.Error
	}

	perf_comparison := NumericComparator(
		(lCpu.Cores * lCpu.Freq),
		(rCpu.Cores * rCpu.Freq),
	)

	arch_comparision := LiteralComparator(lCpu.Arch, rCpu.Arch)

	if arch_comparision == types.Error {
		return types.Error
	} 
	if arch_comparision != types.Equal {
		return types.Worse
	}

	return perf_comparison

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and frequency is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare types.of CPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;
}

