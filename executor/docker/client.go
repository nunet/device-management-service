// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"gitlab.com/nunet/device-management-service/observability"
	"go.uber.org/multierr"
)

type ClientInterface interface {
	IsInstalled(ctx context.Context) bool
	CreateContainer(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		platform *v1.Platform,
		name string,
		pullImage bool,
	) (string, error)
	InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error)
	FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error)
	StartContainer(ctx context.Context, containerID string) error
	WaitContainer(
		ctx context.Context,
		containerID string,
	) (<-chan container.WaitResponse, <-chan error)
	PauseContainer(ctx context.Context, containerID string) error
	ResumeContainer(ctx context.Context, containerID string) error
	StopContainer(
		ctx context.Context,
		containerID string,
		options container.StopOptions,
	) error
	RemoveContainer(ctx context.Context, containerID string) error
	RemoveObjectsWithLabel(ctx context.Context, label string, value string) error
	FindContainer(ctx context.Context, label string, value string) (string, error)
	GetImage(ctx context.Context, imageName string) (image.Summary, error)
	PullImage(ctx context.Context, imageName string) (string, error)
	GetOutputStream(
		ctx context.Context,
		containerID string,
		since string,
		follow bool,
	) (io.ReadCloser, error)
	Exec(ctx context.Context, containerID string, cmd []string) (int, string, string, error)
	CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error
}

// Client wraps the Docker client to provide high-level operations on Docker containers and networks.
type Client struct {
	client *client.Client // Embed the Docker client.
}

// Ensure that Client implements the ClientInterface.
var _ ClientInterface = (*Client)(nil)

// NewDockerClient initializes a new Docker client with environment variables and API version negotiation.
func NewDockerClient() (*Client, error) {
	log.Debugw("docker_client_init_started",
		"labels", string(observability.LabelDeployment))

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHostFromEnv())
	if err != nil {
		log.Errorw("docker_client_init_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return nil, err
	}

	log.Debugw("docker_client_init_success",
		"labels", string(observability.LabelDeployment))
	return &Client{client: c}, nil
}

// IsInstalled checks if Docker is installed and reachable by pinging the Docker daemon.
func (c *Client) IsInstalled(ctx context.Context) bool {
	log.Debugw("docker_client_is_installed_check_started",
		"labels", string(observability.LabelNode))

	_, err := c.client.Ping(ctx)
	if err != nil {
		log.Errorw("docker_client_is_installed_failure",
			"labels", string(observability.LabelNode),
			"error", err)
		return false
	}

	log.Debugw("docker_client_is_installed_success",
		"labels", string(observability.LabelNode))
	return true
}

// CreateContainer creates a new Docker container with the specified configuration.
func (c *Client) CreateContainer(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *v1.Platform,
	name string,
	pullImage bool,
) (string, error) {
	log.Infow("docker_create_container_started",
		"labels", string(observability.LabelDeployment),
		"image", config.Image,
		"name", name)

	if pullImage {
		_, err := c.PullImage(ctx, config.Image)
		if err != nil {
			log.Errorw("docker_create_container_failure",
				"labels", string(observability.LabelDeployment),
				"error", err)
			return "", err
		}
	}

	resp, err := c.client.ContainerCreate(
		ctx,
		config,
		hostConfig,
		networkingConfig,
		platform,
		name,
	)
	if err != nil {
		log.Errorw("docker_create_container_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return "", err
	}

	log.Infow("docker_create_container_success",
		"labels", string(observability.LabelDeployment),
		"containerID", resp.ID)
	return resp.ID, nil
}

// InspectContainer returns detailed information about a Docker container.
func (c *Client) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	log.Infow("docker_inspect_container_started", "containerID", id)
	return c.client.ContainerInspect(ctx, id)
}

// FollowLogs tails the logs of a specified container, returning separate readers for stdout and stderr.
func (c *Client) FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error) {
	log.Infow("docker_follow_logs_started", "containerID", id)

	cont, err := c.InspectContainer(ctx, id)
	if err != nil {
		log.Errorw("docker_follow_logs_failure", "error", err)
		return nil, nil, errors.Wrap(err, "failed to get container")
	}

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	logsReader, err := c.client.ContainerLogs(ctx, cont.ID, logOptions)
	if err != nil {
		log.Errorw("docker_follow_logs_failure", "error", err)
		return nil, nil, errors.Wrap(err, "failed to get container logs")
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	go func() {
		defer logsReader.Close()
		defer stdoutW.Close()
		defer stderrW.Close()

		// Copy logs to stdout and stderr
		if cont.Config.Tty {
			// If TTY is enabled, everything goes to stdout
			_, err = io.Copy(stdoutW, logsReader)
		} else {
			// If TTY is not enabled, use stdcopy.StdCopy to demultiplex the streams
			_, err = stdcopy.StdCopy(stdoutW, stderrW, logsReader)
		}
		if err != nil {
			log.Errorf("failed to copy logs: %v", err)
		}
	}()

	return stdoutR, stderrR, nil
}

// StartContainer starts a specified Docker container.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	log.Infow("docker_start_container_started",
		"labels", string(observability.LabelDeployment),
		"containerID", containerID)
	return c.client.ContainerStart(ctx, containerID, container.StartOptions{})
}

// WaitContainer waits for a container to stop, returning channels for the result and errors.
func (c *Client) WaitContainer(
	ctx context.Context,
	containerID string,
) (<-chan container.WaitResponse, <-chan error) {
	log.Infow("docker_wait_container_started", "containerID", containerID)
	return c.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
}

// PauseContainer pauses the main process of the given container without terminating it.
func (c *Client) PauseContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerPause(ctx, containerID)
}

// ResumeContainer resumes the process execution within the container
func (c *Client) ResumeContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerUnpause(ctx, containerID)
}

// StopContainer stops a running Docker container with a specified timeout.
func (c *Client) StopContainer(
	ctx context.Context,
	containerID string,
	options container.StopOptions,
) error {
	log.Infow("docker_stop_container_started",
		"labels", string(observability.LabelAllocation),
		"containerID", containerID)
	return c.client.ContainerStop(ctx, containerID, options)
}

// RemoveContainer removes a Docker container, optionally forcing removal and removing associated volumes.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	log.Infow("docker_remove_container_started",
		"labels", string(observability.LabelDeployment),
		"containerID", containerID)
	return c.client.ContainerRemove(
		ctx,
		containerID,
		container.RemoveOptions{RemoveVolumes: true, Force: true},
	)
}

// removeContainers removes all containers matching the specified filters.
func (c *Client) removeContainers(ctx context.Context, filterz filters.Args) error {
	log.Infow("docker_remove_containers_started",
		"labels", string(observability.LabelDeployment))

	containers, err := c.client.ContainerList(
		ctx,
		container.ListOptions{All: true, Filters: filterz},
	)
	if err != nil {
		log.Errorw("docker_remove_containers_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return err
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(containers))
	for _, container := range containers {
		wg.Add(1)
		go func(container types.Container, wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			errCh <- c.RemoveContainer(ctx, container.ID)
		}(container, &wg, errCh)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs error
	for err := range errCh {
		errs = multierr.Append(errs, err)
	}

	if errs != nil {
		log.Errorw("docker_remove_containers_failure",
			"labels", string(observability.LabelDeployment),
			"error", errs)
	} else {
		log.Infow("docker_remove_containers_success",
			"labels", string(observability.LabelDeployment))
	}
	return errs
}

// removeNetworks removes all networks matching the specified filters.
func (c *Client) removeNetworks(ctx context.Context, filterz filters.Args) error {
	log.Infow("docker_remove_networks_started",
		"labels", string(observability.LabelDeployment))

	networks, err := c.client.NetworkList(ctx, network.ListOptions{Filters: filterz})
	if err != nil {
		log.Errorw("docker_remove_networks_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return err
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(networks))
	for _, n := range networks {
		wg.Add(1)
		go func(network network.Inspect, wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			errCh <- c.client.NetworkRemove(ctx, network.ID)
		}(n, &wg, errCh)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs error
	for err := range errCh {
		errs = multierr.Append(errs, err)
	}

	if errs != nil {
		log.Errorw("docker_remove_networks_failure",
			"labels", string(observability.LabelDeployment),
			"error", errs)
	} else {
		log.Infow("docker_remove_networks_success",
			"labels", string(observability.LabelDeployment))
	}
	return errs
}

// RemoveObjectsWithLabel removes all Docker containers and networks with a specific label.
func (c *Client) RemoveObjectsWithLabel(ctx context.Context, label string, value string) error {
	log.Infow("docker_remove_objects_with_label_started",
		"labels", string(observability.LabelDeployment),
		"label", label, "value", value)

	filterz := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", label, value)),
	)

	containerErr := c.removeContainers(ctx, filterz)
	networkErr := c.removeNetworks(ctx, filterz)

	if containerErr != nil || networkErr != nil {
		log.Errorw("docker_remove_objects_with_label_failure",
			"labels", string(observability.LabelDeployment),
			"containerErr", containerErr,
			"networkErr", networkErr)
	}

	log.Infow("docker_remove_objects_with_label_success",
		"labels", string(observability.LabelDeployment))
	return multierr.Combine(containerErr, networkErr)
}

// GetOutputStream streams the logs for a specified container.
// The 'since' parameter specifies the timestamp from which to start streaming logs.
// The 'follow' parameter indicates whether to continue streaming logs as they are produced.
// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
func (c *Client) GetOutputStream(
	ctx context.Context,
	containerID string,
	since string,
	follow bool,
) (io.ReadCloser, error) {
	log.Infow("docker_get_output_stream_started", "containerID", containerID)

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Since:      since,
	}

	logReader, err := c.client.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		log.Errorw("docker_get_output_stream_failure", "error", err)
		return nil, errors.Wrap(err, "failed to get container logs")
	}

	log.Infow("docker_get_output_stream_success", "containerID", containerID)
	return logReader, nil
}

// FindContainer searches for a container by label and value, returning its ID if found.
func (c *Client) FindContainer(ctx context.Context, label string, value string) (string, error) {
	log.Infow("docker_find_container_started", "label", label, "value", value)

	containers, err := c.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.Errorw("docker_find_container_failure", "error", err)
		return "", err
	}

	for _, cont := range containers {
		if cont.Labels[label] == value {
			log.Infow("docker_find_container_success", "containerID", cont.ID)
			return cont.ID, nil
		}
	}

	err = fmt.Errorf("unable to find container for %s=%s", label, value)
	log.Warnw("docker_find_container_failure", "error", err)
	return "", err
}

// HasImage checks if an image exists locally
func (c *Client) HasImage(ctx context.Context, imageName string) bool {
	// If imageName does not contain a tag, we need to append ":latest" to the image name
	if !strings.Contains(imageName, ":") {
		imageName = fmt.Sprintf("%s:latest", imageName)
	}

	_, _, err := c.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false
		}
		log.Warnf("Failed to inspect image: %v", err)
		return false
	}

	return true
}

// GetImage returns detailed information about a Docker image.
func (c *Client) GetImage(ctx context.Context, imageName string) (image.Summary, error) {
	images, err := c.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return image.Summary{}, err
	}

	// If imageName does not contain a tag, we need to append ":latest" to the image name.
	if !strings.Contains(imageName, ":") {
		imageName = fmt.Sprintf("%s:latest", imageName)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return image, nil
			}
		}
	}

	return image.Summary{}, fmt.Errorf("unable to find image %s", imageName)
}

// PullImage pulls a Docker image from a registry.
func (c *Client) PullImage(ctx context.Context, imageName string) (string, error) {
	log.Infow("docker_pull_image_started",
		"labels", string(observability.LabelDeployment),
		"image", imageName)

	out, err := c.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		log.Errorw("docker_pull_image_failure",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return "", err
	}

	defer out.Close()
	d := json.NewDecoder(io.TeeReader(out, os.Stdout))

	var message jsonmessage.JSONMessage
	var digest string
	for {
		if err := d.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			log.Errorw("docker_pull_image_failure",
				"labels", string(observability.LabelDeployment),
				"error", err)
			return "", err
		}
		if message.Aux != nil {
			continue
		}
		if message.Error != nil {
			log.Errorw("docker_pull_image_failure",
				"labels", string(observability.LabelDeployment),
				"error", message.Error.Message)
			return "", errors.New(message.Error.Message)
		}
		if strings.HasPrefix(message.Status, "Digest") {
			digest = strings.TrimPrefix(message.Status, "Digest: ")
		}
	}

	log.Infow("docker_pull_image_success",
		"labels", string(observability.LabelDeployment),
		"digest", digest)
	return digest, nil
}

// Exec executes a command inside a running container.
// Returns (exit code, stdout, stderr, and an error if the operation fails)
func (c *Client) Exec(ctx context.Context, containerID string, cmd []string) (int, string, string, error) {
	log.Infow("docker_container_exec_started", "container ID", containerID)

	idresp, err := c.client.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		log.Errorw("docker_container_exec_failure", "error", err)
		return 1, "", "", err
	}

	hijconn, err := c.client.ContainerExecAttach(ctx, idresp.ID, container.ExecAttachOptions{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		log.Errorw("docker_container_exec_failure", "error", err)
		return 1, "", "", err
	}

	defer hijconn.Close()

	var stdout, stderr bytes.Buffer
	n, err := stdcopy.StdCopy(&stdout, &stderr, hijconn.Reader)
	if err != nil {
		log.Errorw("docker_container_exec_failure", "error", err)
		return 1, "", "", err
	}

	execInspect, err := c.client.ContainerExecInspect(ctx, idresp.ID)
	if err != nil {
		log.Errorw("docker_container_exec_failure", "error", err)
		return 1, "", "", err
	}

	log.Debugw("docker_container_exec_inspect",
		"exitCode", execInspect.ExitCode,
		"bytesCopied", n,
		"containerID", containerID)
	return execInspect.ExitCode, stdout.String(), stderr.String(), nil
}

// CopyToContainer copies content to a container's file system.
// dstPath is the path in the container where the content will be copied.
func (c *Client) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error {
	log.Infow("docker_copy_to_container_started",
		"containerID", containerID,
		"dstPath", dstPath)

	err := c.client.CopyToContainer(ctx, containerID, dstPath, content, options)
	if err != nil {
		log.Errorw("docker_copy_to_container_failure",
			"containerID", containerID,
			"dstPath", dstPath,
			"error", err)
		return err
	}

	log.Infow("docker_copy_to_container_success",
		"containerID", containerID,
		"dstPath", dstPath)
	return nil
}
