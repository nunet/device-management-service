package firecracker

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	fcModels "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"go.uber.org/multierr"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	socketDir = "/tmp"

	DefaultCpuCount int64 = 1
	DefaultMemSize  int64 = 50
)

// Executor manages the lifecycle of Firecracker VMs for execution requests.
type Executor struct {
	ID string

	handlers utils.SyncMap[string, *executionHandler] // Maps execution IDs to their handlers.
	client   *Client                                  // Firecracker client for VM management.
}

// NewExecutor initializes a new executor for Firecracker VMs.
func NewExecutor(
	_ context.Context,
	id string,
) (*Executor, error) {
	firecrackerClient, err := NewFirecrackerClient()
	if err != nil {
		return nil, err
	}
	fe := &Executor{
		ID:     id,
		client: firecrackerClient,
	}

	return fe, nil
}

// IsInstalled checks if Firecracker is installed on the host.
func (e *Executor) IsInstalled(ctx context.Context) bool {
	return e.client.IsInstalled()
}

// start begins the execution of a request by starting a new Firecracker VM.
func (e *Executor) Start(ctx context.Context, request *models.ExecutionRequest) error {
	zlog.Sugar().
		Infof("Starting execution for job %s, execution %s", request.JobID, request.ExecutionID)

	// It's possible that this is being called due to a restart. We should check if the
	// VM is already running.
	machine, err := e.FindRunningVM(ctx, request.JobID, request.ExecutionID)
	if err != nil {
		// Unable to find a running VM for this execution, we will instead check for a handler, and
		// failing that will create a new VM.
		if handler, ok := e.handlers.Get(request.ExecutionID); ok {
			if handler.active() {
				return fmt.Errorf("execution is already started")
			} else {
				return fmt.Errorf("execution is already completed")
			}
		}

		// Create a new handler for the execution.
		machine, err = e.newFirecrackerExecutionVM(ctx, request)
		if err != nil {
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
) (<-chan *models.ExecutionResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *models.ExecutionResult, 1)
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
	out chan *models.ExecutionResult,
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
				Infof("executionID %s recieved results from execution", handler.executionID)
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

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *Executor) Run(
	ctx context.Context,
	request *models.ExecutionRequest,
) (*models.ExecutionResult, error) {
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
func (e *Executor) GetLogStream(
	ctx context.Context,
	request models.LogStreamRequest,
) (io.ReadCloser, error) {
	return nil, fmt.Errorf("GetLogStream is not implemented for Firecracker")
}

// Cleanup removes all resources associated with the executor.
// This includes stopping and removing all running VMs and deleting their socket paths.
func (e *Executor) Cleanup(ctx context.Context) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(e.handlers.Keys()))
	e.handlers.Iter(func(_ string, handler *executionHandler) bool {
		wg.Add(1)
		go func(handler *executionHandler, wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			errCh <- handler.destroy(time.Second * 10)
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
	zlog.Info("Cleaned up all firecracker resources")
	return errs
}

// newFirecrackerExecutionVM is an internal method called by Start to set up a new Firecracker VM
// for the job execution. It configures the VM based on the provided ExecutionRequest.
// This includes decoding engine specifications, setting up mounts and resource constraints.
// It then creates the VM but does not start it. The method returns a firecracker.Machine instance
// and an error if any part of the setup fails.
func (e *Executor) newFirecrackerExecutionVM(
	ctx context.Context,
	params *models.ExecutionRequest,
) (*firecracker.Machine, error) {
	fcArgs, err := DecodeSpec(params.EngineSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to decode firecracker engine spec: %w", err)
	}

	fcConfig := firecracker.Config{
		VMID:            params.ExecutionID,
		SocketPath:      e.generateSocketPath(params.JobID, params.ExecutionID),
		KernelImagePath: fcArgs.KernelImage,
		InitrdPath:      fcArgs.Initrd,
		KernelArgs:      fcArgs.KernelArgs,
		MachineCfg: fcModels.MachineConfiguration{
			VcpuCount:  firecracker.Int64(int64(params.Resources.CPU)),
			MemSizeMib: firecracker.Int64(int64(params.Resources.Memory)),
		},
	}

	mounts, err := makeVMMounts(
		fcArgs.RootFileSystem,
		params.Inputs,
		params.Outputs,
		params.ResultsDir,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM mounts: %w", err)
	}
	fcConfig.Drives = mounts

	machine, err := e.client.CreateVM(ctx, fcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	// e.client.VMPassMMDs(ctx, machine, fcArgs.MMDSMessage)
	return machine, nil
}

// makeVMMounts creates the mounts for the VM based on the input and output volumes
// provided in the execution request. It also creates the results directory if it
// does not exist. The function returns a list of mounts and an error if any part of the
// process fails.
func makeVMMounts(
	rootFileSystem string,
	inputs []*models.StorageVolumeExecutor,
	outputs []*models.StorageVolumeExecutor,
	resultsDir string,
) ([]fcModels.Drive, error) {
	var drives []fcModels.Drive
	drivesBuilder := firecracker.NewDrivesBuilder(rootFileSystem)
	for _, input := range inputs {
		drivesBuilder.AddDrive(input.Source, input.ReadOnly)
	}

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
