// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.
package glusterfs

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
)

// GlusterFS holds the configuration needed to mount a GlusterFS volume.
type GlusterFS struct {
	servers []string
	name    string

	mu sync.Mutex

	tracker          *storage.VoumeTracker
	allocationID     string
	clientPrivateKey string
	clientPEM        string
	clientCA         string
}

var _ types.Mounter = (*GlusterFS)(nil)

// New creates a new GlusterFS mounter with the provided configuration.
func New(t *storage.VoumeTracker, servers []string, name string, clientPrivateKey, clientPEM, clientCA, allocationID string) (*GlusterFS, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no GlusterFS servers provided")
	}

	if name == "" {
		return nil, fmt.Errorf("no volume provided")
	}

	return &GlusterFS{
		allocationID:     allocationID,
		servers:          servers,
		name:             name,
		tracker:          t,
		clientPrivateKey: clientPrivateKey,
		clientPEM:        clientPEM,
		clientCA:         clientCA,
	}, nil
}

// Mount mounts the GlusterFS volume to the provided targetPath.
// Additional mount options can be passed in the options map.
func (g *GlusterFS) Mount(targetPath string, _ map[string]string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.tracker.IsMounted(targetPath) {
		return fmt.Errorf("%s is already mounted", targetPath)
	}

	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	return g.runGlusterfsClient(targetPath)
}

func (g *GlusterFS) runGlusterfsClient(targetPath string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}
	binds := []string{
		"/dev/:/dev/",
	}
	hostConfig := &container.HostConfig{
		Binds:        binds,
		Privileged:   true,
		NetworkMode:  "host",
		CgroupnsMode: "host",
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   targetPath,
				Target:   "/mounted",
				ReadOnly: false,
				BindOptions: &mount.BindOptions{
					Propagation: mount.PropagationRShared,
				},
				Consistency: mount.ConsistencyDefault,
			},
		},
	}

	envs := []string{
		"GLUSTER_VOLUME=" + g.name,
		"GLUSTER_HOST=" + strings.Join(g.servers, ","),
		"MOUNT_PATH=mounted",
	}

	// if we are supplying tls certs then its a secure connection
	if g.clientPrivateKey != "" {
		clientAuth := []string{
			"GLUSTERFS_PEM=" + g.clientPEM,
			"GLUSTERFS_KEY=" + g.clientPrivateKey,
			"GLUSTERFS_CA=" + g.clientCA,
		}
		envs = append(envs, clientAuth...)
	}

	containerConfig := &container.Config{
		Env:   envs,
		Image: "nunet-glusterfs-client",
	}

	mountingContainerName := fmt.Sprintf("%s_%s", g.allocationID, g.name)

	resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, mountingContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start glusterfs client container: %w", err)
	}

	logReader, err := cli.ContainerLogs(context.Background(), resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to read container logs: %w", err)
	}
	defer logReader.Close()

	logScanner := bufio.NewScanner(logReader)
	for logScanner.Scan() {
		logLine := logScanner.Text()
		if strings.Contains(logLine, "mounted successfully at") {
			fmt.Println(logLine)
			g.tracker.TrackMount(targetPath, g.allocationID, resp.ID)
			return nil
		}

		if strings.Contains(logLine, "failed mounting glusterfs volume") {
			return fmt.Errorf("failed to mount volume: %s", g.name)
		}
	}

	if err := logScanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return nil
}

func (g *GlusterFS) unmountAndKillContainer(containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		return fmt.Errorf("failed to create Docker Client: %w", err)
	}
	execConfig := container.ExecOptions{
		Cmd:          []string{"umount", "/mounted/"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := cli.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	execAttachResp, err := cli.ContainerExecAttach(context.Background(), execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer execAttachResp.Close()

	logScanner := bufio.NewScanner(execAttachResp.Reader)
	for logScanner.Scan() {
		logLine := logScanner.Text()
		fmt.Println(logLine)
		if strings.Contains(logLine, "success") || strings.Contains(logLine, "not mounted") {
			break
		}
	}

	if err := logScanner.Err(); err != nil {
		return fmt.Errorf("error reading exec output: %w", err)
	}

	if err := cli.ContainerKill(context.Background(), containerID, "SIGKILL"); err != nil {
		return fmt.Errorf("failed to kill container %s: %w", containerID, err)
	}

	if err := cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// Unmount unmounts the GlusterFS volume from the provided targetPath.
func (g *GlusterFS) Unmount(targetPath string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.tracker.IsMounted(targetPath) {
		log.Warnf("target path %s is not mounted", targetPath)
		// no need to unmount if it's not mounted
		return nil
	}

	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	info, err := g.tracker.GetMountInfo(targetPath)
	if err != nil {
		return nil
	}

	err = g.unmountAndKillContainer(info.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to unmount volume and kill container: %w", err)
	}

	g.tracker.UntrackMount(targetPath)

	return nil
}
