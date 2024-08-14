//go:build darwin && (arm64 || amd64)

package resources

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
)

func GetGPUInfo() ([]types.GPU, error) {
	// GPU Detection not supported on Darwin
	// Currently using github.com/jaypipes/ghw for GPU info
	// See:
	//      https://github.com/jaypipes/ghw/blob/v0.12.0/pkg/gpu/gpu_stub.go
	//      https://github.com/jaypipes/ghw/issues/199#issuecomment-946701616
	zlog.Warn("GPU Detection not supported on Darwin")
	return []types.GPU, nil
}

func GetGPUWithHighestFreeVRAM() (types.GPU, error) {
	// GPU detection not supported on Darwin
	return types.GPU{}, fmt.Errorf("GetGPUWithHighestFreeVRAM not supported on Darwin")
}

func GetNVIDIAGPUInfo() ([]types.GPU, error) {
	// NVIDIA GPU detection not supported on Darwin
	return nil, fmt.Errorf("GetNVIDIAGPUInfo not supported on Darwin")
}

func GetAMDGPUInfo() ([]types.GPU, error) {
	// AMD GPU detection not supported on Darwin
	return nil, fmt.Errorf("GetAMDGPUInfo not supported on Darwin")
}

func GetIntelGPUInfo() ([]types.GPU, error) {
	// Intel GPU detection not supported on Darwin
	return nil, fmt.Errorf("GetIntelGPUInfo not supported on Darwin")
}
