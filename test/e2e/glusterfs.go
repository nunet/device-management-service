package e2e

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const containerName = "gluster-container"

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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if c.Names[0] == "/"+containerName {
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
		Image: "ghcr.io/gluster/gluster-containers:centos",
		Tty:   true,
	}

	resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, containerName)
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
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

		execIDResp, err := cli.ContainerExecCreate(context.Background(), containerName, execConfig)
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
