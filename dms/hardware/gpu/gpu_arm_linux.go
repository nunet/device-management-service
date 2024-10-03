//go:build linux && (arm || arm64)

package gpu

// GetGPUs returns the GPUs based on the specified vendors. If no vendors are provided, it returns the information of all the GPUs
func GetGPUs(vendors ...types.GPUVendor) ([]types.GPU, error) {
	// TODO: Implement this function
	return nil, nil
}
