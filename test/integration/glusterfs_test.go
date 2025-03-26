package itest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const glusterContainerName = "gluster-container"

func createDirectories() {
	dirs := []string{
		"/etc/glusterfs",
		"/var/lib/glusterd",
		"/var/log/glusterfs",
		"/glusterfs_data",
	}

	for _, dir := range dirs {
		_ = os.MkdirAll(dir, 0o755)
	}
}

func runGlusterContainer() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if c.Names[0] == "/"+glusterContainerName {
			if c.State == "running" {
				fmt.Println("Container is already running.")
				return nil
			}

			if err := cli.ContainerStart(context.Background(), c.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start stopped container: %w", err)
			}
			fmt.Println("Container restarted successfully.")
			return nil
		}
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/etc/glusterfs:/etc/glusterfs:z",
			"/var/lib/glusterd:/var/lib/glusterd:z",
			"/var/log/glusterfs:/var/log/glusterfs:z",
			"/sys/fs/cgroup:/sys/fs/cgroup:rw",
			"/dev/:/dev",
			"/glusterfs_data:/data",
		},
		Privileged:   true,
		NetworkMode:  "host",
		CgroupnsMode: "host",
		Mounts:       []mount.Mount{},
	}

	containerConfig := &container.Config{
		Image: "ghcr.io/gluster/gluster-containers:fedora",
		Tty:   true,
	}

	resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, glusterContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Println("Container started successfully with ID:", resp.ID)

	return nil
}

func runGlusterCommands() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	commands := [][]string{
		{"gluster", "volume", "create", "nunet_vol", hostname + ":/data/brick2", "force"},
		{"gluster", "volume", "start", "nunet_vol"},
	}

	for _, cmd := range commands {
		execConfig := container.ExecOptions{
			Cmd:          cmd,
			AttachStdout: true,
			AttachStderr: true,
		}

		execIDResp, err := cli.ContainerExecCreate(context.Background(), glusterContainerName, execConfig)
		if err != nil {
			return fmt.Errorf("failed to create exec instance: %w", err)
		}

		if err := cli.ContainerExecStart(context.Background(), execIDResp.ID, container.ExecStartOptions{}); err != nil {
			return fmt.Errorf("failed to start exec command: %w", err)
		}
	}

	fmt.Println("Gluster volume setup completed successfully.")
	return nil
}

func deleteGlusterContainer() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if c.Names[0] == "/"+glusterContainerName {
			if err := cli.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true}); err != nil {
				return fmt.Errorf("failed to remove container: %w", err)
			}
			fmt.Println("Container deleted successfully.")
			return nil
		}
	}

	fmt.Println("Container not found.")
	return nil
}

func pullGlusterImage() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}

	imageName := "ghcr.io/gluster/gluster-containers:fedora"
	images, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				fmt.Println("Image is already present.")
				return nil
			}
		}
	}

	out, err := cli.ImagePull(context.Background(), imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer out.Close()
	fmt.Println("Image pulled successfully.")
	return nil
}

// runBinaryInContainer executes a binary inside the container and redirects output to a file.
//
//nolint:unused // to be used later for testing
func runBinaryInContainer(containerID string, binaryPath string, args []string, envVars []string, outputFilePath string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}

	cmd := []string{"sh", "-c", fmt.Sprintf("%s %s > %s 2>&1", binaryPath, strings.Join(args, " "), outputFilePath)}

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: false,
		AttachStderr: false,
		Env:          envVars,
		Detach:       true,
	}

	execIDResp, err := cli.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	err = cli.ContainerExecStart(context.Background(), execIDResp.ID, container.ExecStartOptions{Detach: true})
	if err != nil {
		return fmt.Errorf("failed to start exec command in background: %w", err)
	}

	fmt.Printf("Binary %s executed inside container %s, output redirected to %s\n", binaryPath, containerID, outputFilePath)
	return nil
}
