// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package firecracker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	fcmodels "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"go.uber.org/multierr"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	socketDir             = "/tmp"
	customInitPrefixPath  = "/tmp/custom-init"
	initScriptsPrefixPath = "/tmp/init_scripts"

	DefaultCPUCount int64 = 1
	DefaultMemSize  int64 = 50 * 1024

	statusWaitTickTime = 100 * time.Millisecond
	statusWaitTimeout  = 10 * time.Second

	defaultKernelArgs = "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules systemd.unified_cgroup_hierarchy=0 systemd.journald.forward_to_console systemd.log_color=false systemd.unit=firecracker.target"
)

var ErrNotInstalled = errors.New("firecracker is not installed")

// Executor manages the lifecycle of Firecracker VMs for execution requests.
type Executor struct {
	ID string

	handlers utils.SyncMap[string, *executionHandler] // Maps execution IDs to their handlers.
	client   *Client                                  // Firecracker client for VM management.
}

// NewExecutor initializes a new executor for Firecracker VMs.
func NewExecutor(ctx context.Context, id string) (*Executor, error) {
	log.Infow("firecracker_executor_init_started", "executorID", id)

	firecrackerClient, err := NewFirecrackerClient()
	if err != nil {
		log.Errorw("firecracker_executor_init_failure", "error", err)
		return nil, err
	}

	if !firecrackerClient.IsInstalled(ctx) {
		log.Errorw("firecracker_executor_not_installed", "executorID", id)
		return nil, ErrNotInstalled
	}

	fe := &Executor{
		ID:     id,
		client: firecrackerClient,
	}

	log.Infow("firecracker_executor_init_success", "executorID", id)
	return fe, nil
}

// List the current executions.
func (e *Executor) List() []types.ExecutionListItem {
	executions := make([]types.ExecutionListItem, 0)

	e.handlers.Range(func(key, value any) bool {
		strKey := key.(string)
		val := value.(*executionHandler)

		executions = append(executions, types.ExecutionListItem{
			ExecutionID: strKey,
			Running:     val.running.Load(),
		})

		return true
	})

	log.Infow("firecracker_executor_list", "executionCount", len(executions))
	return executions
}

// start begins the execution of a request by starting a new Firecracker VM.
func (e *Executor) Start(ctx context.Context, request *types.ExecutionRequest) error {
	log.Infow("firecracker_start_execution", "jobID", request.JobID, "executionID", request.ExecutionID)

	// It's possible that this is being called due to a restart. We should check if the
	// VM is already running.
	machine, err := e.FindRunningVM(ctx, request.JobID, request.ExecutionID)
	if err != nil {
		// Unable to find a running VM for this execution, we will instead check for a handler, and
		// failing that will create a new VM.
		if handler, ok := e.handlers.Get(request.ExecutionID); ok {
			if handler.active() {
				return fmt.Errorf("execution is already started")
			}
			return fmt.Errorf("execution is already completed")
		}

		// Create a new handler for the execution.
		machine, err = e.newFirecrackerExecutionVM(ctx, request)
		if err != nil {
			log.Errorw("firecracker_create_vm_failure", "error", err)
			return fmt.Errorf("failed to create new firecracker VM: %w", err)
		}
	}

	handler := &executionHandler{
		client:      e.client,
		ID:          e.ID,
		executionID: request.ExecutionID,
		machine:     machine,
		resultsDir:  request.ResultsDir,
		waitCh:      make(chan bool),
		activeCh:    make(chan bool),
		running:     &atomic.Bool{},
	}
	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	// run the VM.
	go handler.run(ctx)

	log.Infow("firecracker_start_execution_success", "jobID", request.JobID, "executionID", request.ExecutionID)
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
	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *types.ExecutionResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- fmt.Errorf("execution (%s) not found", executionID)
		log.Errorw("firecracker_wait_execution_not_found", "executionID", executionID)
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
	log.Infow("firecracker_wait_execution", "executionID", handler.executionID)

	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handler.waitCh:
		if handler.result != nil {
			log.Infow("firecracker_wait_execution_result_received", "executionID", handler.executionID)
			out <- handler.result
		} else {
			log.Errorw("firecracker_wait_execution_result_nil", "executionID", handler.executionID)
			errCh <- fmt.Errorf("execution (%s) result is nil", handler.executionID)
		}
	}
}

// Cancel tries to cancel a specific execution by its executionID.
// It returns an error if the execution is not found.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		log.Errorw("firecracker_cancel_execution_not_found", "executionID", executionID)
		return fmt.Errorf("failed to cancel execution (%s). execution not found", executionID)
	}
	return handler.kill(ctx)
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

// GetLogStream is not implemented for Firecracker.
// It is defined to satisfy the Executor interface.
// This method will return an error if called.
func (e *Executor) GetLogStream(_ context.Context, _ types.LogStreamRequest) (io.ReadCloser, error) {
	log.Error("firecracker_get_log_stream_not_implemented")
	return nil, fmt.Errorf("GetLogStream is not implemented for Firecracker")
}

// GetStatus returns the status of the execution identified by the executionID.
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
			if s >= status {
				return nil
			}
		case <-time.After(waitTimeout):
			return fmt.Errorf("execution (%s) did not reach status %s", executionID, status)
		}
	}
}

// Cleanup removes all resources associated with the executor.
// This includes stopping and removing all running VMs and deleting their socket paths.
func (e *Executor) Cleanup() error {
	log.Infow("firecracker_cleanup_started", "executorID", e.ID)

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(e.handlers.Keys()))
	e.handlers.Iter(func(_ string, handler *executionHandler) bool {
		wg.Add(1)
		go func(handler *executionHandler, wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			errCh <- handler.destroy(vmDestroyTimeout)
		}(handler, &wg, errCh)
		return true
	})
	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs error
	for err := range errCh {
		errs = multierr.Append(errs, err)
	}

	log.Infow("firecracker_cleanup_complete", "executorID", e.ID)
	return errs
}

// newFirecrackerExecutionVM is an internal method called by Start to set up a new Firecracker VM
// for the job execution. It configures the VM based on the provided ExecutionRequest.
// This includes decoding engine specifications, setting up mounts and resource constraints.
// It then creates the VM but does not start it. The method returns a firecracker.Machine instance
// and an error if any part of the setup fails.
func (e *Executor) newFirecrackerExecutionVM(
	ctx context.Context,
	params *types.ExecutionRequest,
) (*firecracker.Machine, error) {
	log.Infow("firecracker_create_vm_started", "executionID", params.ExecutionID)

	fcArgs, err := DecodeSpec(params.EngineSpec)
	if err != nil {
		log.Errorw("firecracker_create_vm_decode_spec_failure", "error", err)
		return nil, fmt.Errorf("failed to decode firecracker engine spec: %w", err)
	}

	customInitPath := fmt.Sprintf("%s-%s", customInitPrefixPath, params.ExecutionID)
	initScriptsPath := fmt.Sprintf("%s-%s", initScriptsPrefixPath, params.ExecutionID)

	err = prepareCustomInit(params.ProvisionScripts, customInitPath, initScriptsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare custom init: %w", err)
	}

	fcArgs.KernelArgs = strings.Join([]string{defaultKernelArgs, fcArgs.KernelArgs}, " ")
	fcArgs.KernelArgs = strings.Join([]string{fcArgs.KernelArgs, fmt.Sprintf("init=%s", customInitPrefixPath)}, " ")

	fcConfig := firecracker.Config{
		VMID:            params.ExecutionID,
		SocketPath:      e.generateSocketPath(params.JobID, params.ExecutionID),
		KernelImagePath: fcArgs.KernelImage,
		InitrdPath:      fcArgs.Initrd,
		KernelArgs:      fcArgs.KernelArgs,
		LogLevel:        "Error",
		MachineCfg: fcmodels.MachineConfiguration{
			VcpuCount:  firecracker.Int64(int64(params.Resources.CPU.Cores)),
			MemSizeMib: firecracker.Int64(int64(params.Resources.RAM.Size)),
		},
	}

	mounts, err := makeVMMounts(
		fcArgs.RootFileSystem,
		params.Inputs,
		params.Outputs,
		params.ResultsDir,
		customInitPath,
		initScriptsPath,
	)
	if err != nil {
		log.Errorw("firecracker_create_vm_mounts_failure", "error", err)
		return nil, fmt.Errorf("failed to create VM mounts: %w", err)
	}
	fcConfig.Drives = mounts

	machine, err := e.client.CreateVM(ctx, fcConfig)
	if err != nil {
		log.Errorw("firecracker_create_vm_failure", "error", err)
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	log.Infow("firecracker_create_vm_success", "executionID", params.ExecutionID)
	return machine, nil
}

// makeVMMounts creates the mounts for the VM based on the input and output volumes
// provided in the execution request. It also creates the results directory if it
// does not exist. The function returns a list of mounts and an error if any part of the
// process fails.
func makeVMMounts(
	rootFileSystem string,
	inputs []*types.StorageVolumeExecutor,
	outputs []*types.StorageVolumeExecutor,
	resultsDir, customInitPath, initScriptsPath string,
) ([]fcmodels.Drive, error) {
	var drives []fcmodels.Drive
	drivesBuilder := firecracker.NewDrivesBuilder(rootFileSystem)
	for _, input := range inputs {
		drivesBuilder.AddDrive(input.Source, input.ReadOnly)
	}

	drivesBuilder.AddDrive(customInitPath, true)
	drivesBuilder.AddDrive(initScriptsPath, true)

	for _, output := range outputs {
		if output.Source == "" {
			return drives, fmt.Errorf("output source is empty")
		}

		if resultsDir == "" {
			return drives, fmt.Errorf("results directory is empty")
		}

		if err := os.MkdirAll(resultsDir, os.ModePerm); err != nil {
			return drives, fmt.Errorf("failed to create results directory: %w", err)
		}

		drivesBuilder.AddDrive(output.Source, false)
	}
	drives = drivesBuilder.Build()
	return drives, nil
}

// FindRunningVM finds the VM that is running the execution with the given ID.
// It returns the Mchine instance if found, or an error if the VM is not found.
func (e *Executor) FindRunningVM(
	ctx context.Context,
	jobID string,
	executionID string,
) (*firecracker.Machine, error) {
	return e.client.FindVM(ctx, e.generateSocketPath(jobID, executionID))
}

// generateSocketPath generates a socket path based on the job identifiers.
func (e *Executor) generateSocketPath(jobID string, executionID string) string {
	return fmt.Sprintf("%s/%s_%s_%s.sock", socketDir, e.ID, jobID, executionID)
}
