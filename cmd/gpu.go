// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/types"
)

func newGPUCommand(hardwareManager types.HardwareManager, dockerClient docker.ClientInterface) *cobra.Command {
	gpuCmd := &cobra.Command{
		Use:   "gpu <operation>",
		Short: "Manage GPU resources",
		Long: `Available operations:
- list: List all available GPUs
- test: Test GPU deployment by running a docker container with GPU resources
`,
	}

	// Add subcommands
	gpuCmd.AddCommand(newGPUListCommand(hardwareManager))
	gpuCmd.AddCommand(newGPUTestCommand(hardwareManager, dockerClient))

	return gpuCmd
}

func newGPUListCommand(hardwareManager types.HardwareManager) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available GPUs",
		RunE: func(_ *cobra.Command, _ []string) error {
			machineResources, err := hardwareManager.GetMachineResources()
			if err != nil {
				return fmt.Errorf("error getting machine resources: %w", err)
			}
			machineUsage, err := hardwareManager.GetUsage()
			if err != nil {
				return fmt.Errorf("error getting GPU usage: %w", err)
			}

			if len(machineResources.GPUs) == 0 {
				log.Info("No GPUs detected on the host")
				return nil
			}

			if len(machineResources.GPUs) != len(machineUsage.GPUs) {
				return fmt.Errorf("GPU and GPU usage counts do not match. This is a bug")
			}

			fmt.Println("GPU Details:")
			for i, g := range machineResources.GPUs {
				fmt.Printf("Model: %s, Total VRAM: %.2f GB, Used VRAM: %.2f GB, Vendor: %s, PCI Address: %s, Index: %d\n",
					g.Model, g.VRAMInGB(), machineUsage.GPUs[i].VRAMInGB(), g.Vendor, g.PCIAddress, g.Index)
			}
			return nil
		},
	}
}

func newGPUTestCommand(hardwareManager types.HardwareManager, dockerClient docker.ClientInterface) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test GPU deployment by running a Docker container with GPU resources",
		RunE: func(_ *cobra.Command, _ []string) error {
			machineResources, err := hardwareManager.GetMachineResources()
			if err != nil {
				return fmt.Errorf("getting machine resources: %v", err)
			}

			if len(machineResources.GPUs) == 0 {
				return fmt.Errorf("no GPUs found")
			}

			maxFreeVRAMGpu, err := machineResources.GPUs.MaxFreeVRAMGPU()
			if err != nil {
				return fmt.Errorf("getting GPU with highest free VRAM: %v", err)
			}
			fmt.Printf("Selected Vendor: %s, Device: %+v\n", maxFreeVRAMGpu.Vendor, maxFreeVRAMGpu)

			if maxFreeVRAMGpu.Vendor == types.GPUVendorNvidia {
				// Check if NVIDIA container toolkit is installed
				// We specifically look for the nvidia-container-toolkit executable because:
				// 1. It's the name of the main package installed via apt (nvidia-container-toolkit)
				// 2. It's the most reliable indicator of a proper toolkit installation
				// 3. Checking for this single file reduces the risk of false positives
				_, err = os.Stat("/usr/bin/nvidia-container-toolkit")
				if os.IsNotExist(err) {
					return fmt.Errorf("nvidia container toolkit is not installed. Please install it before running this command")
				}
			}
			imageName := "ubuntu:20.04"

			if !dockerClient.IsInstalled(context.Background()) {
				return fmt.Errorf("docker is not installed or running. Cannot run GPU deployment test")
			}

			fmt.Printf("Creating the docker conainer for the image: %s\n", imageName)
			containerConfig := &container.Config{
				Image:        imageName,
				User:         "root",
				Tty:          true,         // Enable TTY
				AttachStdout: true,         // Attach stdout
				AttachStderr: true,         // Attach stderr
				Entrypoint:   []string{""}, // Set entrypoint to run shell commands
				Cmd: []string{
					// This will show both the integrated and discrete GPUs
					"sh", "-c",
					"apt-get update && apt-get install -y pciutils && lspci | grep 'VGA compatible controller'",
				},
			}

			var hostConfig *container.HostConfig
			switch maxFreeVRAMGpu.Vendor {
			case types.GPUVendorNvidia:
				hostConfig = &container.HostConfig{
					AutoRemove: true,
					Resources: container.Resources{
						DeviceRequests: []container.DeviceRequest{
							{
								Driver:       "nvidia",
								Count:        -1,
								Capabilities: [][]string{{"gpu"}},
							},
						},
					},
				}
			case types.GPUVendorAMDATI:
				hostConfig = &container.HostConfig{
					AutoRemove: true,
					Binds: []string{
						"/dev/kfd:/dev/kfd",
						"/dev/dri:/dev/dri",
					},
					Resources: container.Resources{
						Devices: []container.DeviceMapping{
							{
								PathOnHost:        "/dev/kfd",
								PathInContainer:   "/dev/kfd",
								CgroupPermissions: "rwm",
							},
							{
								PathOnHost:        "/dev/dri",
								PathInContainer:   "/dev/dri",
								CgroupPermissions: "rwm",
							},
						},
					},
					GroupAdd: []string{"video"},
				}
			case types.GPUVendorIntel:
				hostConfig = &container.HostConfig{
					AutoRemove: true,
					Binds: []string{
						"/dev/dri:/dev/dri",
					},
					Resources: container.Resources{
						Devices: []container.DeviceMapping{
							{
								PathOnHost:        "/dev/dri",
								PathInContainer:   "/dev/dri",
								CgroupPermissions: "rwm",
							},
						},
					},
				}
			default:
				return fmt.Errorf("unknown GPU vendor: %s", maxFreeVRAMGpu.Vendor)
			}

			containerID, err := dockerClient.CreateContainer(context.Background(),
				containerConfig,
				hostConfig,
				nil,
				nil,
				"nunet-gpu-test",
				true,
			)
			if err != nil {
				return fmt.Errorf("pulling Docker image: %v", err)
			}

			fmt.Println("Container created with ID: ", containerID)

			if err := dockerClient.StartContainer(context.Background(), "nunet-gpu-test"); err != nil {
				return fmt.Errorf("starting docker container: %v", err)
			}

			ctx := context.Background()
			// Wait for the container to finish execution
			statusCh, errCh := dockerClient.WaitContainer(ctx, containerID)
			select {
			case err := <-errCh:
				if err != nil {
					fmt.Printf("Container exited with error: %v\n", err)
				}
			case <-statusCh:
				fmt.Println("Container execution completed.")
			}

			reader, err := dockerClient.GetOutputStream(ctx, containerID, "", true)
			if err != nil {
				return fmt.Errorf("getting output stream: %v", err)
			}

			// Print the output stream
			if _, err := os.Stdout.ReadFrom(reader); err != nil {
				return fmt.Errorf("reading output stream: %v", err)
			}

			return nil
		},
	}
}
