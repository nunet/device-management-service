package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	labelExecutorName = "nunet-executor"
	labelJobID        = "nunet-jobID"
	labelExecutionID  = "nunet-executionID"

	outputStreamCheckTickTime = 100 * time.Millisecond
	outputStreamCheckTimeout  = 5 * time.Second
)

// Executor manages the lifecycle of Docker containers for execution requests.
type Executor struct {
	ID string

	handlers utils.SyncMap[string, *executionHandler] // Maps execution IDs to their handlers.
	client   *Client                                  // Docker client for container management.
}

// NewExecutor initializes a new Executor instance with a Docker client.
func NewExecutor(ctx context.Context, id string) (*Executor, error) {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, err
	}

	if !dockerClient.IsInstalled(ctx) {
		return nil, fmt.Errorf("docker is not installed")
	}

	return &Executor{
		ID:     id,
		client: dockerClient,
	}, nil
}

// Start begins the execution of a request by starting a Docker container.
func (e *Executor) Start(ctx context.Context, request *types.ExecutionRequest) error {
	zlog.Sugar().
		Infof("Starting execution for job %s, execution %s", request.JobID, request.ExecutionID)

	// It's possible that this is being called due to a restart. We should check if the
	// container is already running.
	containerID, err := e.FindRunningContainer(ctx, request.JobID, request.ExecutionID)
	if err != nil {
		// Unable to find a running container for this execution, we will instead check for a handler, and
		// failing that will create a new container.
		if handler, ok := e.handlers.Get(request.ExecutionID); ok {
			if handler.active() {
				return fmt.Errorf("execution is already started")
			} else {
				return fmt.Errorf("execution is already completed")
			}
		}

		// Create a new handler for the execution.
		containerID, err = e.newDockerExecutionContainer(ctx, request)
		if err != nil {
			return fmt.Errorf("failed to create new container: %w", err)
		}
	}

	handler := &executionHandler{
		client:      e.client,
		ID:          e.ID,
		executionID: request.ExecutionID,
		containerID: containerID,
		resultsDir:  request.ResultsDir,
		waitCh:      make(chan bool),
		activeCh:    make(chan bool),
		running:     &atomic.Bool{},
		TTYEnabled:  true,
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)

	// run the container.
	go handler.run(ctx)
	return nil
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
	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *types.ExecutionResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- fmt.Errorf("execution (%s) not found", executionID)
		return resultCh, errCh
	}

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
	zlog.Sugar().Infof("executionID %s waiting for execution", handler.executionID)
	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handler.waitCh:
		if handler.result != nil {
			zlog.Sugar().
				Infof("executionID %s received results from execution", handler.executionID)
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

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *Executor) Run(
	ctx context.Context,
	request *types.ExecutionRequest,
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
func (e *Executor) Cleanup(ctx context.Context) error {
	err := e.client.RemoveObjectsWithLabel(ctx, labelExecutorName, e.ID)
	if err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}
	zlog.Info("Cleaned up all Docker resources")
	return nil
}

// newDockerExecutionContainer is an internal method called by Start to set up a new Docker container
// for the job execution. It configures the container based on the provided ExecutionRequest.
// This includes decoding engine specifications, setting up environment variables, mounts and resource
// constraints. It then creates the container but does not start it.
// The method returns a container.CreateResponse and an error if any part of the setup fails.
func (e *Executor) newDockerExecutionContainer(
	ctx context.Context,
	params *types.ExecutionRequest,
) (string, error) {
	dockerArgs, err := DecodeSpec(params.EngineSpec)
	if err != nil {
		return "", fmt.Errorf("failed to decode docker engine spec: %w", err)
	}

	// TODO: Move this code block ( L263-272) to the allocator in future
	// Select the GPU with the highest available free VRAM and choose the GPU vendor for container's host config
	gpus, err := resources.ManagerInstance.SystemSpecs().GetGPUs()
	maxFreeVRAMGpu, err := types.GPUList(gpus).GetGPUWithHighestFreeVRAM()
	if err != nil {
		return "", fmt.Errorf("failed to get GPU with highest free VRAM: %w", err)
	}
	// Essential for multi-vendor GPU nodes. For example,
	// if a machine has an 8 GB NVIDIA and a 16 GB Intel GPU, the latter should be used first.
	// Even for machines with a single GPU, this is important as integrated GPUs would also be commonly detected.
	chosenGPUVendor := maxFreeVRAMGpu.Vendor

	containerConfig := container.Config{
		Image:      dockerArgs.Image,
		Tty:        true, // Needs to be true for applications such as Jupyter or Gradio to work correctly. See issue #459 for details.
		Env:        dockerArgs.Environment,
		Entrypoint: dockerArgs.Entrypoint,
		Cmd:        dockerArgs.Cmd,
		Labels:     e.containerLabels(params.JobID, params.ExecutionID),
		WorkingDir: dockerArgs.WorkingDirectory,
	}

	mounts, err := makeContainerMounts(params.Inputs, params.Outputs, params.ResultsDir)
	if err != nil {
		return "", fmt.Errorf("failed to create container mounts: %w", err)
	}

	zlog.Sugar().Infof("Adding %d GPUs to request", len(params.Resources.GPUs))
	hostConfig := configureHostConfig(chosenGPUVendor, params, mounts)

	if _, err = e.client.PullImage(ctx, dockerArgs.Image); err != nil {
		return "", fmt.Errorf("failed to pull docker image: %w", err)
	}

	executionContainer, err := e.client.CreateContainer(
		ctx,
		&containerConfig,
		&hostConfig,
		nil,
		nil,
		labelExecutionValue(e.ID, params.JobID, params.ExecutionID),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return executionContainer, nil
}

// configureHostConfig sets up the host configuration for the container based on the
// GPU vendor and resources requested by the execution. It supports both GPU and CPU configurations.
func configureHostConfig(vendor types.GPUVendor, params *types.ExecutionRequest, mounts []mount.Mount) container.HostConfig {
	var hostConfig container.HostConfig
	switch vendor {
	case types.GPUVendorNvidia:
		deviceIDs := make([]string, len(params.Resources.GPUs))
		for i, gpu := range params.Resources.GPUs {
			deviceIDs[i] = fmt.Sprint(gpu.Index)
		}
		hostConfig = container.HostConfig{
			Mounts: mounts,
			Resources: container.Resources{
				NanoCPUs: int64(params.Resources.CPU.ClockSpeedHz),
				CPUCount: int64(params.Resources.CPU.Cores),
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
			Mounts: mounts,
			Binds: []string{
				"/dev/kfd:/dev/kfd",
				"/dev/dri:/dev/dri",
			},
			Resources: container.Resources{
				NanoCPUs: int64(params.Resources.CPU.ClockSpeedHz),
				CPUCount: int64(params.Resources.CPU.Cores),
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
			Mounts: mounts,
			Binds: []string{
				"/dev/dri:/dev/dri",
			},
			Resources: container.Resources{
				NanoCPUs: int64(params.Resources.CPU.ClockSpeedHz),
				CPUCount: int64(params.Resources.CPU.Cores),
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
		hostConfig = container.HostConfig{
			Mounts: mounts,
			Resources: container.Resources{
				NanoCPUs: int64(params.Resources.CPU.ClockSpeedHz),
				CPUCount: int64(params.Resources.CPU.Cores),
			},
		}
	}
	return hostConfig
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
	var mounts []mount.Mount
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

		if err := os.MkdirAll(resultsDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create results directory: %w", err)
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
