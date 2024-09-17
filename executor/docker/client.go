package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Client wraps the Docker client to provide high-level operations on Docker containers and networks.
type Client struct {
	client *client.Client // Embed the Docker client.
}

// NewDockerClient initializes a new Docker client with environment variables and API version negotiation.
func NewDockerClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{client: c}, nil
}

// IsInstalled checks if Docker is installed and reachable by pinging the Docker daemon.
func (c *Client) IsInstalled(ctx context.Context) bool {
	_, err := c.client.Ping(ctx)
	return err == nil
}

// CreateContainer creates a new Docker container with the specified configuration.
func (c *Client) CreateContainer(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *v1.Platform,
	name string,
) (string, error) {
	_, err := c.PullImage(ctx, config.Image)
	if err != nil {
		return "", err
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
		return "", err
	}
	return resp.ID, nil
}

// InspectContainer returns detailed information about a Docker container.
func (c *Client) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return c.client.ContainerInspect(ctx, id)
}

// FollowLogs tails the logs of a specified container, returning separate readers for stdout and stderr.
func (c *Client) FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error) {
	cont, err := c.InspectContainer(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get container")
	}

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	logsReader, err := c.client.ContainerLogs(ctx, cont.ID, logOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get container logs")
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	go func() {
		stdoutBuffer := bufio.NewWriter(stdoutWriter)
		stderrBuffer := bufio.NewWriter(stderrWriter)
		defer func() {
			logsReader.Close()
			stdoutBuffer.Flush()
			stdoutWriter.Close()
			stderrBuffer.Flush()
			stderrWriter.Close()
		}()

		_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
		if err != nil && !errors.Is(err, context.Canceled) {
			zlog.Sugar().Warnf("context closed while getting logs: %v\n", err)
		}
	}()

	return stdoutReader, stderrReader, nil
}

// StartContainer starts a specified Docker container.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
}

// WaitContainer waits for a container to stop, returning channels for the result and errors.
func (c *Client) WaitContainer(
	ctx context.Context,
	containerID string,
) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return c.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
}

// StopContainer stops a running Docker container with a specified timeout.
func (c *Client) StopContainer(
	ctx context.Context,
	containerID string,
	timeout time.Duration,
) error {
	return c.client.ContainerStop(ctx, containerID, &timeout)
}

// RemoveContainer removes a Docker container, optionally forcing removal and removing associated volumes.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerRemove(
		ctx,
		containerID,
		types.ContainerRemoveOptions{RemoveVolumes: true, Force: true},
	)
}

// removeContainers removes all containers matching the specified filters.
func (c *Client) removeContainers(ctx context.Context, filterz filters.Args) error {
	containers, err := c.client.ContainerList(
		ctx,
		types.ContainerListOptions{All: true, Filters: filterz},
	)
	if err != nil {
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
	return errs
}

// removeNetworks removes all networks matching the specified filters.
func (c *Client) removeNetworks(ctx context.Context, filterz filters.Args) error {
	networks, err := c.client.NetworkList(ctx, types.NetworkListOptions{Filters: filterz})
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(networks))
	for _, network := range networks {
		wg.Add(1)
		go func(network types.NetworkResource, wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			errCh <- c.client.NetworkRemove(ctx, network.ID)
		}(network, &wg, errCh)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs error
	for err := range errCh {
		errs = multierr.Append(errs, err)
	}
	return errs
}

// RemoveObjectsWithLabel removes all Docker containers and networks with a specific label.
func (c *Client) RemoveObjectsWithLabel(ctx context.Context, label string, value string) error {
	filterz := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", label, value)),
	)

	containerErr := c.removeContainers(ctx, filterz)
	networkErr := c.removeNetworks(ctx, filterz)
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
	cont, err := c.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container")
	}

	if !cont.State.Running {
		return nil, fmt.Errorf("cannot get logs for a container that is not running")
	}

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Since:      since,
	}

	logReader, err := c.client.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container logs")
	}

	return logReader, nil
}

// FindContainer searches for a container by label and value, returning its ID if found.
func (c *Client) FindContainer(ctx context.Context, label string, value string) (string, error) {
	containers, err := c.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", err
	}

	for _, container := range containers {
		if container.Labels[label] == value {
			return container.ID, nil
		}
	}

	return "", fmt.Errorf("unable to find container for %s=%s", label, value)
}

// PullImage pulls a Docker image from a registry.
func (c *Client) PullImage(ctx context.Context, imageName string) (string, error) {
	out, err := c.client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		zlog.Sugar().Errorf("unable to pull image: %v", err)
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
			zlog.Sugar().Errorf("unable pull image: %v", err)
			return "", err
		}
		if message.Aux != nil {
			continue
		}
		if message.Error != nil {
			zlog.Sugar().Errorf("unable pull image: %v", message.Error.Message)
			return "", errors.New(message.Error.Message)
		}
		if strings.HasPrefix(message.Status, "Digest") {
			digest = strings.TrimPrefix(message.Status, "Digest: ")
		}
	}

	return digest, nil
}
