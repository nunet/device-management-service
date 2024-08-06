//go:build linux && amd64

package resources

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/models"
	// "gitlab.com/nunet/device-management-service/library/gpudetect"
	// "gitlab.com/nunet/device-management-service/dms/onboarding/gpuinfo"
)

func CheckGPU() ([]models.Gpu, error) {
	var gpuInfo []models.Gpu
	vendors, err := DetectGPUVendors()
	if err != nil {
		return nil, fmt.Errorf("unable to detect GPU Vendor: %v", err)
	}
	foundNVIDIA, foundAMD := false, false
	for _, vendor := range vendors {
		switch vendor {
		case NVIDIA:
			if !foundNVIDIA {
				var gpu models.Gpu
				info, err := GetNVIDIAGPUInfo()
				if err != nil {
					return nil, fmt.Errorf("error getting NVIDIA GPU info: %v", err)
				}
				for _, i := range info {
					gpu.Name = i.GPUName
					gpu.FreeVram = i.FreeMemory
					gpu.TotVram = i.TotalMemory
					gpuInfo = append(gpuInfo, gpu)
				}
				foundNVIDIA = true
			}
		case AMD:
			if !foundAMD {
				var gpu models.Gpu
				info, err := GetAMDGPUInfo()
				if err != nil {
					return nil, fmt.Errorf("error getting AMD GPU info: %v", err)
				}
				for _, i := range info {
					gpu.Name = i.GPUName
					gpu.FreeVram = i.FreeMemory
					gpu.TotVram = i.TotalMemory
					gpuInfo = append(gpuInfo, gpu)
				}
				foundAMD = true
			}
		case Unknown:
			fmt.Println("Unknown GPU(s) detected")
		}
	}
	return gpuInfo, nil
}
