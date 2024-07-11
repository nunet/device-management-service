package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func GpuComparator(l, r interface{}, preference ...Preference) models.Comparison {
	// comparator for GPU type

	// we want to reason about the inner fields of the GPU type and how they compare between left and right
	// in the future we may want to pass custom preference parameters to the ComplexComparator
	// for now it is probably best to hardcode them;

	comparison := ReturnComplexComparison(l, r)

	if comparison["VRAM"] == models.Error  {
		return models.Error
	} 
	if comparison["VRAM"] == models.Worse  {
		return models.Worse
	}
	if comparison["VRAM"] == models.Better  {
		return models.Better
	}
	if comparison["VRAM"] == models.Equal {
		return models.Equal
	}

	// currently this is a very simple comparison, based on the assumption
	// that more cores / or equal amount of cores and VRAM is acceptable, but nothing less;
	// for more complex comparisons we would need to encode the very specific hardware knowledge;
	// it could be, that we want to compare models of GPUs and rank them in some way;
	// using e.g. benchmarking data from Tom's Hardware or some other source;
	
	return models.Error // error is the default value

}
