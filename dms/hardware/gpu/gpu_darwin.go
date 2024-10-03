package gpu

import "gitlab.com/nunet/device-management-service/types"

func GetGPUs(_ ...types.GPUVendor) ([]types.GPU, error) {
	// GPUs are not supported on Darwin yet
	return []types.GPU{}, nil
}
