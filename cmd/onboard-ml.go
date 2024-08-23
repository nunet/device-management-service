package cmd

import (
	"context"
	"fmt"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
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

var onboardMLCmd = &cobra.Command{
	Use:     "onboard-ml",
	Short:   "Setup for Machine Learning with GPU",
	Long:    ``,
	PreRunE: isDMSRunning(networkService),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()

		wsl, err := utils.CheckWSL()
		if err != nil {
			fmt.Println("Error checking WSL:", err)
			return
		}

		vendors, err := resources.ManagerInstance.SystemSpecs().GetGPUVendors()
		if err != nil {
			fmt.Println("Error detecting GPUs:", err)
			return
		}

		// check for GPU vendors
		hasAMD := containsVendor(vendors, types.GPUVendorAMDATI)
		hasNVIDIA := containsVendor(vendors, types.GPUVendorNvidia)
		hasIntel := containsVendor(vendors, types.GPUVendorIntel)

		if !hasAMD && !hasNVIDIA && !hasIntel {
			fmt.Println(`No NVIDIA/AMD/Intel GPU(s) detected...`)
			return
		}

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			fmt.Println("Error creating Docker client:", err)
			return
		}

		imageList, err := cli.ImageList(ctx, docker_types.ImageListOptions{})
		if err != nil {
			fmt.Println("Error listing Docker images:", err)
			return
		}

		if wsl {
			fmt.Printf("You are running on Windows Subsystem for Linux (WSL)\nMake sure that NVIDIA drivers are set up correctly\n\nWARNING: AMD GPUs are not supported on WSL!\n")
		}

		if hasNVIDIA {
			err = pullMultipleImages(ctx, cli, imageList, imagesNVIDIA)
			if err != nil {
				fmt.Println("Error pulling NVIDIA images:", err)
				return
			}
		}

		if hasAMD {
			err = pullMultipleImages(ctx, cli, imageList, imagesAMD)
			if err != nil {
				fmt.Println("Error pulling AMD images:", err)
				return
			}
		}
	},
}

func pullMultipleImages(ctx context.Context, cli *client.Client, imageList []docker_types.ImageSummary, images []string) error {
	for i := 0; i < len(images); i++ {
		if !imageExists(imageList, images[i]) {
			err := pullImage(ctx, cli, images[i])
			if err != nil {
				return fmt.Errorf("unable to pull image %s: %v", images[i], err)
			}
		} else {
			fmt.Printf("Image already pulled: %s\n", images[i])
		}
	}

	return nil
}
