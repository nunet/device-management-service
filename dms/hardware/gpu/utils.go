package gpu

import "gitlab.com/nunet/device-management-service/types"

// assignIndexToGPUs assigns an index to each GPU in the list starting from 0
func assignIndexToGPUs(gpus []types.GPU) []types.GPU {
	for i := range gpus {
		gpus[i].Index = i
	}
	return gpus
}
