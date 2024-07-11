package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func ExecutionResourcesComparator(l, r interface{}, preference ...Preference) models.Comparison {
	// comparator for models.ExecutionResources type
	// Current implementation of the type has four fields: CPU, Memory, Disk, GPUs
	// we consider that all fields have to be 'Better' or 'Equal' 
	// for the comparison to be 'Better' or 'Equal'
	// else we return 'Worse'

	comparison := ReturnComplexComparison(l, r)
	
	if comparison["CPU"] == models.Error || 
		comparison["Memory"] == models.Error ||
		comparison["Disk"] == models.Error || 
		comparison["GPUs"] == models.Error {
			return models.Error
	}

	if comparison["CPU"] == models.Worse || 
		comparison["Memory"] == models.Worse ||
		comparison["Disk"] == models.Worse || 
		comparison["GPUs"] == models.Worse {
			return models.Worse
}

	if comparison["CPU"] == models.Equal && 
		comparison["Memory"] == models.Equal &&
		comparison["Disk"] == models.Equal && 
		comparison["GPUs"] == models.Equal {
			return models.Equal
	}

	return models.Better // if non above returns, then the result is better

}
