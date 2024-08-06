//go:build darwin && (amd64 || arm64)

package resources

import (
	"gitlab.com/nunet/device-management-service/models"
)

func CheckGPU() ([]models.Gpu, error) {
	// GPU Detection not supported on Darwin
	// Currently using github.com/jaypipes/ghw for GPU info
	// See:
	//      https://github.com/jaypipes/ghw/blob/v0.12.0/pkg/gpu/gpu_stub.go
	//      https://github.com/jaypipes/ghw/issues/199#issuecomment-946701616
	zlog.Warn("GPU Detection not supported on Darwin")
	return []models.Gpu{}, nil
}
