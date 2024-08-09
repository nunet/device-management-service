package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/models"
)

// ContainerOptions set parameters for running a Docker container (NVIDIA/AMD/Intel)
type ContainerOptions struct {
	UseGPUs    bool
	Devices    []string
	Groups     []string
	Image      string
	Command    []string
	Entrypoint []string
}

var flagCudaTensor, flagRocmHip, flagIntelXPU bool

var gpuCapacityCmd = &cobra.Command{
	Use:     "capacity",
	Short:   "Check availability of NVIDIA/AMD/Intel GPUs",
	Long:    ``,
	PreRunE: isDMSRunning(networkService),
	Run: func(cmd *cobra.Command, args []string) {
		cuda, _ := cmd.Flags().GetBool("cuda-tensor")
		rocm, _ := cmd.Flags().GetBool("rocm-hip")
		intelXPU, _ := cmd.Flags().GetBool("intel-xpu")

		if !cuda && !rocm && !intelXPU {
			fmt.Println(`Error: no flags specified`)
			cmd.Help()
			return
		}

		vendors, err := resources.DetectGPUVendors()
		if err != nil {
			fmt.Println("Error detecting GPU vendors:", err)
			return
		}

		hasAMD := containsVendor(vendors, models.GPUVendorAMDATI)
		hasNVIDIA := containsVendor(vendors, models.GPUVendorNvidia)
		hasIntel := containsVendor(vendors, models.GPUVendorIntel)

		if !hasAMD && !hasNVIDIA && !hasIntel {
			fmt.Println("No NVIDIA/AMD/Intel GPU(s) detected...")
			return
		}

		ctx := context.Background()

		if cuda {
			if !hasNVIDIA {
				fmt.Println("No NVIDIA GPU(s) detected...")
				return
			}

			cudaOpts := ContainerOptions{
				UseGPUs:    true,
				Image:      "registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch",
				Command:    []string{"python", "check-cuda-and-tensor-cores-availability.py"},
				Entrypoint: []string{""},
			}

			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				fmt.Println("Error creating Docker client:", err)
				return
			}

			images, err := cli.ImageList(ctx, types.ImageListOptions{})
			if err != nil {
				fmt.Println("Error listing Docker images:", err)
				return
			}

			if !imageExists(images, cudaOpts.Image) {
				err := pullImage(cli, ctx, cudaOpts.Image)
				if err != nil {
					fmt.Println("Error pulling CUDA image:", err)
					return
				}
			}

			err = runDockerContainer(cli, ctx, cudaOpts)
			if err != nil {
				fmt.Println("Error running CUDA container:", err)
				return
			}
		}

		if rocm {
			if !hasAMD {
				fmt.Println("No AMD GPU(s) detected...")
				return
			}

			rocmOpts := ContainerOptions{
				Devices:    []string{"/dev/kfd", "/dev/dri"},
				Groups:     []string{"video"},
				Image:      "registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch-amd",
				Command:    []string{"python", "check-rocm-and-hip-availability.py"},
				Entrypoint: []string{""},
			}

			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				fmt.Println("Error creating Docker client:", err)
				return
			}

			images, err := cli.ImageList(ctx, types.ImageListOptions{})
			if err != nil {
				fmt.Println("Error listing images:", err)
				return
			}

			if !imageExists(images, rocmOpts.Image) {
				err := pullImage(cli, ctx, rocmOpts.Image)
				if err != nil {
					fmt.Println("Error pulling ROCm-HIP image:", err)
					return
				}
			}

			err = runDockerContainer(cli, ctx, rocmOpts)
			if err != nil {
				fmt.Println("Error running ROCm-HIP container:", err)
				return
			}
		}

		if intelXPU {
			if !hasIntel {
				fmt.Println("No Intel GPU(s) detected...")
				return
			}

			intelXPUOpts := ContainerOptions{
				Devices:    []string{"/dev/dri"},
				Image:      "registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch-intel",
				Command:    []string{"python", "check-intel-xpu-availability.py"},
				Entrypoint: []string{""},
			}

			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				fmt.Println("Error creating Docker client:", err)
				return
			}

			images, err := cli.ImageList(ctx, types.ImageListOptions{})
			if err != nil {
				fmt.Println("Error listing images:", err)
				return
			}

			if !imageExists(images, intelXPUOpts.Image) {
				err := pullImage(cli, ctx, intelXPUOpts.Image)
				if err != nil {
					fmt.Println("Error pulling Intel XPU image:", err)
					return
				}
			}

			err = runDockerContainer(cli, ctx, intelXPUOpts)
			if err != nil {
				fmt.Println("Error running Intel XPU container:", err)
				return
			}
		}
	},
}

func runDockerContainer(cli *client.Client, ctx context.Context, options ContainerOptions) error {
	if options.Image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	config := &container.Config{
		Image:      options.Image,
		Entrypoint: options.Entrypoint,
		Cmd:        options.Command,
		Tty:        true,
	}

	hostConfig := &container.HostConfig{}

	if options.UseGPUs {
		gpuOpts := opts.GpuOpts{}
		if err := gpuOpts.Set("all"); err != nil {
			return fmt.Errorf("failed setting GPU opts: %v", err)
		}
		hostConfig.DeviceRequests = gpuOpts.Value()
	}

	for _, device := range options.Devices {
		hostConfig.Devices = append(hostConfig.Devices, container.DeviceMapping{
			PathOnHost:        device,
			PathInContainer:   device,
			CgroupPermissions: "rwm",
		})
	}

	hostConfig.GroupAdd = options.Groups

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("cannot create container: %v", err)
	}

	defer func() {
		if err := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); err != nil {
			fmt.Printf("WARNING: could not remove container: %v\n", err)
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("cannot start container: %v", err)
	}

	out, err := cli.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("failed attaching container: %v", err)
	}

	io.Copy(os.Stdout, out.Reader)

	waitCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitResult := <-waitCh:
		if waitResult.Error != nil {
			return fmt.Errorf("container exit error: %s", waitResult.Error.Message)
		}
	case err := <-errCh:
		return fmt.Errorf("error waiting for container: %v", err)
	}

	return nil
}

func imageExists(images []types.ImageSummary, imageName string) bool {
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return true
			}
		}
	}
	return false
}

func pullImage(cli *client.Client, ctx context.Context, imageName string) error {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	out, err := cli.ImagePull(ctxCancel, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull image %s: %v", imageName, err)
	}

	// define interrupt to stop image pull
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		fmt.Println("signal: interrupt")
		cancel()
	}()

	fmt.Printf("Pulling image: %s\nThis may take some time...\n", imageName)
	defer out.Close()

	io.Copy(os.Stdout, out)

	return nil
}
