package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/docker/cli/opts"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/dms/resources"
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

func newGPUCapacityCmd() *cobra.Command {
	fnCuda := "cuda-tensor"
	fnRocm := "rocm-hip"

	cmd := &cobra.Command{
		Use:   "capacity",
		Short: "Check availability of NVIDIA/AMD/Intel GPUs",
		Long:  ``,
		Run: func(cmd *cobra.Command, _ []string) {
			cuda, _ := cmd.Flags().GetBool(fnCuda)
			rocm, _ := cmd.Flags().GetBool(fnRocm)

			if !cuda && !rocm {
				fmt.Println(`Error: no flags specified`)
				err := cmd.Help()
				if err != nil {
					cmd.Println(err)
				}
				os.Exit(1)
			}

			machineResources, err := resources.ManagerInstance.SystemSpecs().GetMachineResources()
			if err != nil {
				fmt.Println("Error detecting GPU vendors:", err)
				os.Exit(1)
			}

			if len(machineResources.GPUs) == 0 {
				fmt.Println("No GPU detected...")
				return
			}

			gpuVendorMap := getGPUVendorMap(machineResources.GPUs)
			ctx := context.Background()

			if cuda {
				if len(gpuVendorMap[types.GPUVendorNvidia]) == 0 {
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
					os.Exit(1)
				}

				images, err := cli.ImageList(ctx, docker_types.ImageListOptions{})
				if err != nil {
					fmt.Println("Error listing Docker images:", err)
					os.Exit(1)
				}

				if !imageExists(images, cudaOpts.Image) {
					err := pullImage(ctx, cli, cudaOpts.Image)
					if err != nil {
						fmt.Println("Error pulling CUDA image:", err)
						os.Exit(1)
					}
				}

				err = runDockerContainer(ctx, cli, cudaOpts)
				if err != nil {
					fmt.Println("Error running CUDA container:", err)
					os.Exit(1)
				}
			}

			if rocm {
				if len(gpuVendorMap[types.GPUVendorAMDATI]) == 0 {
					fmt.Println("No AMD GPU(s) detected...")
					os.Exit(1)
				}

				rocmOpts := ContainerOptions{
					Devices:    []string{"/dev/kfd", "/dev/dri"},
					Groups:     []string{"video", "render"},
					Image:      "registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch-amd",
					Command:    []string{"python", "check-rocm-and-hip-availability.py"},
					Entrypoint: []string{""},
				}

				cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
				if err != nil {
					fmt.Println("Error creating Docker client:", err)
					os.Exit(1)
				}

				images, err := cli.ImageList(ctx, docker_types.ImageListOptions{})
				if err != nil {
					fmt.Println("Error listing images:", err)
					os.Exit(1)
				}

				if !imageExists(images, rocmOpts.Image) {
					err := pullImage(ctx, cli, rocmOpts.Image)
					if err != nil {
						fmt.Println("Error pulling ROCm-HIP image:", err)
						os.Exit(1)
					}
				}

				err = runDockerContainer(ctx, cli, rocmOpts)
				if err != nil {
					fmt.Println("Error running ROCm-HIP container:", err)
					os.Exit(1)
				}
			}
		},
	}
	cmd.Flags().BoolP(fnCuda, "c", false, "check CUDA Tensor")
	cmd.Flags().BoolP(fnRocm, "r", false, "check ROCM-HIP")
	return cmd
}

func runDockerContainer(ctx context.Context, cli *client.Client, options ContainerOptions) error {
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
		if err := cli.ContainerRemove(ctx, resp.ID, docker_types.ContainerRemoveOptions{}); err != nil {
			fmt.Printf("WARNING: could not remove container: %v\n", err)
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, docker_types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("cannot start container: %v", err)
	}

	out, err := cli.ContainerAttach(ctx, resp.ID, docker_types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("failed attaching container: %v", err)
	}

	_, err = io.Copy(os.Stdout, out.Reader)
	if err != nil {
		return fmt.Errorf("failed to copy container output: %w", err)
	}

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

func imageExists(images []docker_types.ImageSummary, imageName string) bool {
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return true
			}
		}
	}
	return false
}

func pullImage(ctx context.Context, cli *client.Client, imageName string) error {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	out, err := cli.ImagePull(ctxCancel, imageName, docker_types.ImagePullOptions{})
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

	_, err = io.Copy(os.Stdout, out)
	if err != nil {
		return fmt.Errorf("failed to copy image pull to stdout: %w", err)
	}

	return nil
}
