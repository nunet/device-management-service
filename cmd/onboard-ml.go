package cmd

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/utils"
)

var imagesNVIDIA = []string{
	"registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/tensorflow",
	"registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch",
}

var imagesAMD = []string{
	"registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/tensorflow-amd",
	"registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch-amd",
}

func newOnboardMLCmd(afs afero.Afero, dockerClient *docker.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "onboard-ml",
		Short: "Setup for Machine Learning with GPU",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wsl, err := utils.CheckWSL(afs)
			if err != nil {
				return fmt.Errorf("could not check WSL: %w", err)
			}

			machineResources, err := resources.ManagerInstance.SystemSpecs().GetMachineResources()
			if err != nil {
				return fmt.Errorf("could not fetch GPU vendors: %w", err)
			}

			if len(machineResources.GPUs) == 0 {
				return fmt.Errorf("no GPU detected")
			}

			gpuVendorMap := getGPUVendorMap(machineResources.GPUs)

			if wsl {
				fmt.Fprintf(cmd.OutOrStdout(), "You are running on Windows Subsystem for Linux (WSL)\nMake sure that NVIDIA drivers are set up correctly\n\nWARNING: AMD GPUs are not supported on WSL!\n")
			}

			if len(gpuVendorMap[types.GPUVendorNvidia]) > 0 {
				for _, image := range imagesNVIDIA {
					digest, err := dockerClient.PullImage(cmd.Context(), image)
					if err != nil {
						return fmt.Errorf("could not pull image %s: %w", image, err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Pulled Nvidia image %s with digest %s\n", image, digest)
				}
			}
			if len(gpuVendorMap[types.GPUVendorAMDATI]) > 0 {
				for _, image := range imagesAMD {
					digest, err := dockerClient.PullImage(cmd.Context(), image)
					if err != nil {
						return fmt.Errorf("could not pull image %s: %w", image, err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Pulled AMD image %s with digest %s\n", image, digest)
				}
			}
			return nil
		},
	}
}
