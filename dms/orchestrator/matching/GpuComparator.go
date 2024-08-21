package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func GpuComparator(l, r interface{}, preference ...Preference) types.Comparison {
	// comparator for GPU type

	// we want to reason about the inner fields of the GPU type and how they compare between left and right
	// in the future we may want to pass custom preference parameters to the ComplexComparator
	// for now it is probably best to hardcode them;

	// validate input type
	_, lok := l.(types.GPU)
	_, rok := r.(types.GPU)
	if !lok || !rok {
		return types.Error
	}

	comparison := ReturnComplexComparison(l, r)

	if comparison["TotalVRAM"] == types.Error {
		return types.Error
	}
	if comparison["TotalVRAM"] == types.Worse {
		return types.Worse
	}
	if comparison["TotalVRAM"] == types.Better {
		return types.Better
	}
	if comparison["TotalVRAM"] == types.Equal {
		return types.Equal
	}

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and VRAM is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare types.of GPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;

	return types.Error // error is the default value

}
