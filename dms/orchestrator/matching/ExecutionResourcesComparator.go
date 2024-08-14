package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func ExecutionResourcesComparator(l, r interface{}, preference ...Preference) types.Comparison {
	// comparator for types.ExecutionResources type
	// Current implementation of the type has four fields: CPU, Memory, Disk, GPUs
	// we consider that all fields have to be 'Better' or 'Equal' 
	// for the comparison to be 'Better' or 'Equal'
	// else we return 'Worse'

	// validate input type
	_, lok := l.(types.ExecutionResources)
	_, rok := r.(types.ExecutionResources)
	if !lok || !rok {
		return types.Error
	}

	comparison := ReturnComplexComparison(l, r)
	
	if comparison["CPU"] == types.Error || 
		comparison["Memory"] == types.Error ||
		comparison["Disk"] == types.Error || 
		comparison["GPUs"] == types.Error {
			return types.Error
	}

	if comparison["CPU"] == types.Worse || 
		comparison["Memory"] == types.Worse ||
		comparison["Disk"] == types.Worse || 
		comparison["GPUs"] == types.Worse {
			return types.Worse
}

	if comparison["CPU"] == types.Equal && 
		comparison["Memory"] == types.Equal &&
		comparison["Disk"] == types.Equal && 
		comparison["GPUs"] == types.Equal {
			return types.Equal
	}

	return types.Better // if non above returns, then the result is better

}
