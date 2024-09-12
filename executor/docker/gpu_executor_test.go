package docker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
)

func TestGPUDeployment(t *testing.T) {
	ensureDockerSetup(t)
	t.Log("Starting GPU Deployment Test")

	// Get the GPU with the highest free VRAM
	t.Log("Fetching GPU with highest free VRAM")
	machineResources, err := resources.ManagerInstance.SystemSpecs().GetMachineResources()
	assert.NoError(t, err)

	if len(machineResources.GPUs) == 0 {
		t.Skipf("No GPUs detected on the host")
	}

	maxFreeVRAMGpu, err := machineResources.GPUs.GetGPUWithHighestFreeVRAM()
	if err != nil {
		t.Fatalf("Error getting GPU with highest free VRAM: %v", err)
	}
	t.Logf("Selected Vendor: %s, Device: %+v", maxFreeVRAMGpu.Vendor, maxFreeVRAMGpu)

	// Determine the appropriate image name based on the GPU vendor
	imageName := "registry.gitlab.com/nunet/ml-on-gpu/ml-on-gpu-service/develop/pytorch"
	switch maxFreeVRAMGpu.Vendor {
	case types.GPUVendorNvidia:
		t.Log("NVIDIA GPU selected, no suffix needed for image name")
	case types.GPUVendorAMDATI:
		t.Log("AMD GPU selected, adding -amd suffix to image name")
		imageName += "-amd"
	case types.GPUVendorIntel:
		t.Log("Intel GPU selected, adding -intel suffix to image name")
		imageName += "-intel"
	default:
		t.Fatalf("Unknown GPU vendor: %s", maxFreeVRAMGpu.Vendor)
	}
	t.Logf("Chosen image name: %s", imageName)

	// Create Docker client
	t.Log("Creating Docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Error creating Docker client: %v", err)
	}
	t.Log("Docker client created successfully")

	// Pull the Docker image
	t.Logf("Pulling Docker image: %s", imageName)
	if err := pullImage(t, cli, imageName); err != nil {
		t.Fatalf("Error pulling image: %v", err)
	}
	t.Log("Docker image pulled successfully")

	// Run the container
	t.Log("Running the container")
	if err := runContainer(t, cli, imageName, maxFreeVRAMGpu.Vendor, maxFreeVRAMGpu); err != nil {
		t.Fatalf("Error running container: %v", err)
	}
	t.Log("Container deployed successfully.")
}

func pullImage(t *testing.T, cli *client.Client, imageName string) error {
	out, err := cli.ImagePull(context.Background(), imageName, docker_types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}
	defer out.Close()

	dec := json.NewDecoder(out)
	for {
		var v map[string]interface{}
		if err := dec.Decode(&v); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error decoding image pull output: %v", err)
		}

		if status, ok := v["status"].(string); ok {
			if progress, ok := v["progress"].(string); ok {
				t.Logf("%s: %s", status, progress)
			} else {
				t.Log(status)
			}
		}
	}
	return nil
}

func runContainer(t *testing.T, cli *client.Client, imageName string, vendor types.GPUVendor, _ types.GPU) error {
	ctx := context.Background()

	containerConfig := &container.Config{
		Image:        imageName,
		User:         "root",
		Tty:          true,         // Enable TTY
		AttachStdout: true,         // Attach stdout
		AttachStderr: true,         // Attach stderr
		Entrypoint:   []string{""}, // Set entrypoint to run shell commands
		Cmd: []string{
			"sh", "-c",
			"apt-get update && apt-get install -y pciutils && lspci",
		},
	}

	var hostConfig *container.HostConfig

	switch vendor {
	case types.GPUVendorNvidia:
		hostConfig = &container.HostConfig{
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
		hostConfig = &container.HostConfig{ // Critical configuration for AMD GPUs
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
		hostConfig = &container.HostConfig{ // Critical configuration for discrete Intel GPUs
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
		return fmt.Errorf("unknown GPU vendor: %s", vendor)
	}

	t.Log("Creating container")
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("unable to create container: %v", err)
	}

	t.Log("Starting container..updating container and installing pciutils to run lspci command..")
	if err := cli.ContainerStart(ctx, resp.ID, docker_types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("unable to start container: %v", err)
	}

	t.Log("Fetching container logs")
	out, err := cli.ContainerLogs(ctx, resp.ID, docker_types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return fmt.Errorf("unable to get container logs: %v", err)
	}
	defer out.Close()

	buf := make([]byte, 1024)
	for {
		n, err := out.Read(buf)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading container logs: %v", err)
		}
	}

	t.Log("Container logs fetched successfully, lspci command executed to verify GPU access inside the container.")
	return nil
}
