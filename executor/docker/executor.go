// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

var ErrNotInstalled = errors.New("docker is not installed")

const (
	nanoCPUsPerCore = 1e9

	labelExecutorName = "nunet-executor"
	labelJobID        = "nunet-jobID"
	labelExecutionID  = "nunet-executionID"

	outputStreamCheckTickTime = 100 * time.Millisecond
	outputStreamCheckTimeout  = 5 * time.Second

	statusWaitTickTime = 100 * time.Millisecond
	statusWaitTimeout  = 10 * time.Second

	initScriptsBaseDir = "/tmp/nunet/init-scripts-"

	enableTTY = false
)

// Executor manages the lifecycle of Docker containers for execution requests.
type Executor struct {
	ID string

	handlers utils.SyncMap[string, *executionHandler] // Maps execution IDs to their handlers.
	client   *Client                                  // Docker client for container management.

	fs afero.Afero
}

var _ types.Executor = (*Executor)(nil)

// NewExecutor initializes a new Executor instance with a Docker client.
func NewExecutor(ctx context.Context, fs afero.Afero, id string) (*Executor, error) {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, err
	}

	if !dockerClient.IsInstalled(ctx) {
		return nil, ErrNotInstalled
	}

	return &Executor{
		ID:     id,
		fs:     fs,
		client: dockerClient,
	}, nil
}

func (e *Executor) GetID() string {
	return e.ID
}

// Start begins the execution of a request by starting a Docker container.
func (e *Executor) Start(ctx context.Context, request *types.ExecutionRequest) error {
	endTrace := observability.StartTrace(ctx, "docker_executor_start_duration")
	defer endTrace()

	// Log starting execution
	log.Infow("docker_executor_start_begin", "jobID", request.JobID, "executionID", request.ExecutionID)

	// It's possible that this is being called due to a restart. We should check if the
	// container is already running.
	containerID, err := e.FindRunningContainer(ctx, request.JobID, request.ExecutionID)
	if err != nil {
		// Unable to find a running container for this execution, we will instead check for a handler, and
		// failing that will create a new container.
		if handler, ok := e.handlers.Get(request.ExecutionID); ok {
			if handler.active() {
				log.Errorw("docker_executor_start_failure", "executionID", request.ExecutionID, "error", "execution already started")
				return fmt.Errorf("execution is already started")
			}
			log.Errorw("docker_executor_start_failure", "executionID", request.ExecutionID, "error", "execution completed")
			return fmt.Errorf("execution is already completed")
		}

		// Create a new handler for the execution.
		containerID, err = e.newDockerExecutionContainer(ctx, request, enableTTY)
		if err != nil {
			log.Errorw("docker_executor_start_failure", "executionID", request.ExecutionID, "error", err)
			return fmt.Errorf("failed to create new container: %w", err)
		}
	}

	handler := &executionHandler{
		client:              e.client,
		ID:                  e.ID,
		fs:                  e.fs,
		executionID:         request.ExecutionID,
		containerID:         containerID,
		resultsDir:          request.ResultsDir,
		persistLogsDuration: request.PersistLogsDuration,
		waitCh:              make(chan bool),
		activeCh:            make(chan bool),
		running:             &atomic.Bool{},
		TTYEnabled:          enableTTY,
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)

	// run the container.
	go handler.run(ctx)
	return nil
}

// Pause pauses the container
func (e *Executor) Pause(
	ctx context.Context,
	executionID string,
) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.pause(ctx)
}

// Resume resumes the container
func (e *Executor) Resume(
	ctx context.Context,
	executionID string,
) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.resume(ctx)
}

// Wait initiates a wait for the completion of a specific execution using its
// executionID. The function returns two channels: one for the result and another
// for any potential error. If the executionID is not found, an error is immediately
// sent to the error channel. Otherwise, an internal goroutine (doWait) is spawned
// to handle the asynchronous waiting. Callers should use the two returned channels
// to wait for the result of the execution or an error. This can be due to issues
// either beginning the wait or in getting the response. This approach allows the
// caller to synchronize Wait with calls to Start, waiting for the execution to complete.
func (e *Executor) Wait(
	ctx context.Context,
	executionID string,
) (<-chan *types.ExecutionResult, <-chan error) {
	endTrace := observability.StartTrace(ctx, "docker_executor_wait_duration")
	defer endTrace()

	log.Infow("docker_executor_wait_begin", "executionID", executionID)

	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *types.ExecutionResult, 1)
	errCh := make(chan error, 1)

	if !found {
		log.Errorw("docker_executor_wait_failure", "executionID", executionID, "error", "execution not found")
		errCh <- fmt.Errorf("execution (%s) not found", executionID)
		return resultCh, errCh
	}

	log.Infow("docker_executor_wait_success", "executionID", executionID)
	go e.doWait(ctx, resultCh, errCh, handler)
	return resultCh, errCh
}

// doWait is a helper function that actively waits for an execution to finish. It
// listens on the executionHandler's wait channel for completion signals. Once the
// signal is received, the result is sent to the provided output channel. If there's
// a cancellation request (context is done) before completion, an error is relayed to
// the error channel. If the execution result is nil, an error suggests a potential
// flaw in the executor logic.
func (e *Executor) doWait(
	ctx context.Context,
	out chan *types.ExecutionResult,
	errCh chan error,
	handler *executionHandler,
) {
	log.Infof("executionID %s waiting for execution", handler.executionID)
	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handler.waitCh:
		if handler.result != nil {
			log.Debugf("executionID %s received results from execution ", handler.executionID)
			out <- handler.result
		} else {
			errCh <- fmt.Errorf("execution (%s) result is nil", handler.executionID)
		}
	}
}

// Cancel tries to cancel a specific execution by its executionID.
// It returns an error if the execution is not found.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("failed to cancel execution (%s). execution not found", executionID)
	}
	return handler.kill(ctx)
}

// Remove removes a container identified by its executionID.
// It returns an error if the execution is not found.
func (e *Executor) Remove(executionID string, timeout time.Duration) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.destroy(timeout)
}

// GetLogStream provides a stream of output logs for a specific execution.
// Parameters 'withHistory' and 'follow' control whether to include past logs
// and whether to keep the stream open for new logs, respectively.
// It returns an error if the execution is not found.
func (e *Executor) GetLogStream(
	ctx context.Context,
	request types.LogStreamRequest,
) (io.ReadCloser, error) {
	// It's possible we've recorded the execution as running, but have not yet added the handler to
	// the handler map because we're still waiting for the container to start. We will try and wait
	// for a few seconds to see if the handler is added to the map.
	chHandler := make(chan *executionHandler)
	chExit := make(chan struct{})

	go func(ch chan *executionHandler, exit chan struct{}) {
		// Check the handlers every 100ms and send it down the
		// channel if we find it. If we don't find it after 5 seconds
		// then we'll be told on the exit channel
		ticker := time.NewTicker(outputStreamCheckTickTime)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h, found := e.handlers.Get(request.ExecutionID)
				if found {
					ch <- h
					return
				}
			case <-exit:
				ticker.Stop()
				return
			}
		}
	}(chHandler, chExit)

	// Either we'll find a handler for the execution (which might have finished starting)
	// or we'll timeout and return an error.
	select {
	case handler := <-chHandler:
		return handler.outputStream(ctx, request)
	case <-time.After(outputStreamCheckTimeout):
		chExit <- struct{}{}
	}

	return nil, fmt.Errorf("execution (%s) not found", request.ExecutionID)
}

// List returns a slice of ExecutionListItem containing information about current executions.
// This implementation currently returns an empty list and should be updated in the future.
func (e *Executor) List() []types.ExecutionListItem {
	// TODO: list dms containers
	return nil
}

// GetStatus returns the status of the execution identified by its executionID.
// It returns the status of the execution and an error if the execution is not found or status is unknown.
func (e *Executor) GetStatus(ctx context.Context, executionID string) (types.ExecutionStatus, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return "", fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.status(ctx)
}

// WaitForStatus waits for the execution to reach a specific status.
// It returns an error if the execution is not found or the status is unknown.
func (e *Executor) WaitForStatus(
	ctx context.Context,
	executionID string,
	status types.ExecutionStatus,
	timeout *time.Duration,
) error {
	waitTimeout := statusWaitTimeout
	if timeout != nil {
		waitTimeout = *timeout
	}
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}
	ticker := time.NewTicker(statusWaitTickTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s, err := handler.status(ctx)
			if err != nil {
				return err
			}
			if s == status {
				return nil
			}
		case <-time.After(waitTimeout):
			return fmt.Errorf("execution (%s) did not reach status %s", executionID, status)
		}
	}
}

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *Executor) Run(
	ctx context.Context, request *types.ExecutionRequest,
) (*types.ExecutionResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}
	resCh, errCh := e.Wait(ctx, request.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resCh:
		return out, nil
	case err := <-errCh:
		return nil, err
	}
}

// Cleanup removes all Docker resources associated with the executor.
// This includes removing containers including networks and volumes with the executor's label.
// It also removes all temporary directories created for init scripts.
func (e *Executor) Cleanup(ctx context.Context) error {
	endTrace := observability.StartTrace(ctx, "docker_executor_cleanup_duration")
	defer endTrace()

	log.Infow("docker_executor_cleanup_begin", "executorID", e.ID)

	err := e.client.RemoveObjectsWithLabel(ctx, labelExecutorName, e.ID)
	if err != nil {
		log.Errorw("docker_executor_cleanup_failure", "executorID", e.ID, "error", err)
		return fmt.Errorf("failed to remove containers: %w", err)
	}

	log.Infow("docker_executor_cleanup_success", "executorID", e.ID)

	// Remove all provision scripts used for mounting
	pattern := initScriptsBaseDir + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find init script directories: %w", err)
	}

	for _, dir := range matches {
		if err := os.RemoveAll(dir); err != nil {
			log.Warnf("Failed to remove init script directory %s: %v", dir, err)
		} else {
			log.Infof("Removed init script directory: %s", dir)
		}
	}

	return nil
}

// Exec executes a command in the container with the given containerID.
// Returns the exit code, stdout, stderr and an error if the execution fails.
func (e *Executor) Exec(ctx context.Context, executionID string, command []string) (int, string, string, error) {
	h, found := e.handlers.Get(executionID)
	if !found {
		return 0, "", "", fmt.Errorf("failed to get execution handler for execution=%q", executionID)
	}

	return e.client.Exec(ctx, h.containerID, command)
}

// newDockerExecutionContainer is an internal method called by Start to set up a new Docker container
// for the job execution. It configures the container based on the provided ExecutionRequest.
// This includes decoding engine specifications, setting up environment variables, mounts and resource
// constraints. It then creates the container but does not start it.
// The method returns a container.CreateResponse and an error if any part of the setup fails.
func (e *Executor) newDockerExecutionContainer(
	ctx context.Context,
	params *types.ExecutionRequest,
	tty bool,
) (string, error) {
	dockerArgs, err := DecodeSpec(params.EngineSpec)
	if err != nil {
		return "", fmt.Errorf("failed to decode docker engine spec: %w", err)
	}

	containerConfig := container.Config{
		Image:      dockerArgs.Image,
		Env:        dockerArgs.Environment,
		Entrypoint: dockerArgs.Entrypoint,
		Cmd:        dockerArgs.Cmd,
		Labels:     e.containerLabels(params.JobID, params.ExecutionID),
		WorkingDir: dockerArgs.WorkingDirectory,
		// Needs to be true for applications such as Jupyter or Gradio to work correctly. See issue #459 for details.
		Tty: tty,
	}

	mounts, err := makeContainerMounts(params.Inputs, params.Outputs, params.ResultsDir)
	if err != nil {
		return "", fmt.Errorf("failed to create container mounts: %w", err)
	}

	initScriptsDir, err := prepareInitScripts(params.ProvisionScripts, params.ExecutionID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare init scripts: %w", err)
	}

	if initScriptsDir != "" {
		oldEntryPoint := containerConfig.Entrypoint
		oldCmd := containerConfig.Cmd

		// Execute init scripts first
		containerConfig.Entrypoint = []string{"/bin/sh", "-c"}
		containerConfig.Cmd = []string{
			fmt.Sprintf("%s/run_provision_scripts.sh && %s %s",
				initScriptsDir,
				strings.Join(oldEntryPoint, " "),
				strings.Join(oldCmd, " ")),
		}

		// Add a mount for the init scripts
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: initScriptsDir,
			Target: initScriptsDir,
		})
	}

	log.Infof("Adding %d GPUs to request", len(params.Resources.GPUs))
	hostConfig, err := configureHostConfig(params, dockerArgs, mounts)
	if err != nil {
		return "", fmt.Errorf("failed to configure host config: %w", err)
	}

	hasImage := e.client.HasImage(ctx, dockerArgs.Image)

	executionContainer, err := e.client.CreateContainer(
		ctx,
		&containerConfig,
		&hostConfig,
		nil,
		nil,
		labelExecutionValue(e.ID, params.JobID, params.ExecutionID),
		!hasImage, // only pull if we don't have the image
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return executionContainer, nil
}

// prepareInitScripts creates a shell script that will run all init scripts
func prepareInitScripts(scripts map[string][]byte, id string) (string, error) {
	if len(scripts) == 0 {
		return "", nil
	}

	tempDir := initScriptsBaseDir + id
	err := os.MkdirAll(tempDir, 0o700)
	if err != nil {
		return "", fmt.Errorf("failed to create init scripts base directory: %w", err)
	}

	scriptNames := make([]string, 0, len(scripts))
	for name, content := range scripts {
		filename := filepath.Join(tempDir, name)
		if err := os.WriteFile(filename, content, 0o700); err != nil {
			return "", fmt.Errorf("failed to write init script %s: %w", name, err)
		}
		scriptNames = append(scriptNames, filename)
	}

	// Create a wrapper script to execute all init scripts
	wrapperContent := "#!/bin/sh\n\n"
	for _, script := range scriptNames {
		wrapperContent += fmt.Sprintf("echo 'Executing %s'\n", filepath.Base(script))
		wrapperContent += fmt.Sprintf("%s\n", script)
	}

	wrapperPath := filepath.Join(tempDir, "run_provision_scripts.sh")
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0o700); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return tempDir, nil
}

// configureHostConfig sets up the host configuration for the container based on the
// GPU vendor and resources requested by the execution. It supports both GPU and CPU configurations.
func configureHostConfig(params *types.ExecutionRequest, dockerArgs EngineSpec, mounts []mount.Mount) (container.HostConfig, error) {
	var hostConfig container.HostConfig

	// set the config for the first selected gpu
	// TODO: support multiple GPUs
	if len(params.Resources.GPUs) >= 1 {
		switch params.Resources.GPUs[0].Vendor {
		case types.GPUVendorNvidia:
			deviceIDs := make([]string, len(params.Resources.GPUs))
			for i, gpu := range params.Resources.GPUs {
				deviceIDs[i] = fmt.Sprint(gpu.Index)
			}
			hostConfig = container.HostConfig{
				Resources: container.Resources{
					DeviceRequests: []container.DeviceRequest{
						{
							DeviceIDs:    deviceIDs,
							Capabilities: [][]string{{"gpu"}},
						},
					},
				},
			}
		case types.GPUVendorAMDATI:
			hostConfig = container.HostConfig{
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
		// Updated the device handling for Intel GPUs.
		// Previously, specific device paths were determined using PCI addresses and symlinks.
		// Now, the approach has been simplified by directly binding the entire /dev/dri directory.
		// This change exposes all Intel GPUs to the container, which may be preferable for
		// environments with multiple Intel GPUs. It reduces complexity as granular control
		// is not required if all GPUs need to be accessible.
		case types.GPUVendorIntel:
			hostConfig = container.HostConfig{
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
		}
	}

	// Set the cpu config
	hostConfig.Resources.NanoCPUs = int64(params.Resources.CPU.Cores * nanoCPUsPerCore)
	hostConfig.Resources.CPUCount = int64(params.Resources.CPU.Cores)

	// setup mounts
	hostConfig.Mounts = mounts

	// configure port binding
	portMaps := make(map[nat.Port][]nat.PortBinding)
	for _, toBind := range params.PortsToBind {
		natPort, err := nat.NewPort("tcp", strconv.Itoa(toBind.ExecutorPort))
		if err != nil {
			return hostConfig, fmt.Errorf("failed to create port: %w", err)
		}

		if _, ok := portMaps[natPort]; ok {
			portMaps[natPort] = append(portMaps[natPort], nat.PortBinding{
				HostIP:   toBind.IP,
				HostPort: fmt.Sprintf("%d", toBind.HostPort),
			})
			continue
		}

		portMaps[natPort] = []nat.PortBinding{
			{
				HostIP:   toBind.IP,
				HostPort: fmt.Sprintf("%d", toBind.HostPort),
			},
		}
	}

	hostConfig.PortBindings = portMaps
	hostConfig.Privileged = dockerArgs.Privileged

	// Configure DNS settings
	hostConfig.DNS = []string{params.GatewayIP, "1.1.1.1"}
	hostConfig.DNSSearch = []string{"internal"}
	hostConfig.DNSOptions = []string{
		"ndots:1", // reduce DNS lookups by setting ndots lower
		"timeout:2",
		"attempts:1",
	}

	return hostConfig, nil
}

// makeContainerMounts creates the mounts for the container based on the input and output
// volumes provided in the execution request. It also creates the results directory if it
// does not exist. The function returns a list of mounts and an error if any part of the
// process fails.
func makeContainerMounts(
	inputs []*types.StorageVolumeExecutor,
	outputs []*types.StorageVolumeExecutor,
	resultsDir string,
) ([]mount.Mount, error) {
	// the actual mounts we will give to the container
	// these are paths for both input and output data
	mounts := make([]mount.Mount, 0)
	for _, input := range inputs {
		if input.Type != types.StorageVolumeTypeBind {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   input.Source,
				Target:   input.Target,
				ReadOnly: input.ReadOnly,
			})
		} else {
			return nil, fmt.Errorf("unsupported storage volume type: %s", input.Type)
		}
	}

	for _, output := range outputs {
		if output.Source == "" {
			return nil, fmt.Errorf("output source is empty")
		}

		if resultsDir == "" {
			return nil, fmt.Errorf("results directory is empty")
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: output.Source,
			Target: output.Target,
			// this is an output volume so can be written to
			ReadOnly: false,
		})
	}

	return mounts, nil
}

// containerLabels returns the labels to be applied to the container for the given job and execution.
func (e *Executor) containerLabels(jobID string, executionID string) map[string]string {
	return map[string]string{
		labelExecutorName: e.ID,
		labelJobID:        labelJobValue(e.ID, jobID),
		labelExecutionID:  labelExecutionValue(e.ID, jobID, executionID),
	}
}

// labelJobValue returns the value for the job label.
func labelJobValue(executorID string, jobID string) string {
	return fmt.Sprintf("%s_%s", executorID, jobID)
}

// labelExecutionValue returns the value for the execution label.
func labelExecutionValue(executorID string, jobID string, executionID string) string {
	return fmt.Sprintf("%s_%s_%s", executorID, jobID, executionID)
}

// FindRunningContainer finds the container that is running the execution
// with the given ID. It returns the container ID if found, or an error if
// the container is not found.
func (e *Executor) FindRunningContainer(
	ctx context.Context,
	jobID string,
	executionID string,
) (string, error) {
	labelValue := labelExecutionValue(e.ID, jobID, executionID)
	return e.client.FindContainer(ctx, labelExecutionID, labelValue)
}
