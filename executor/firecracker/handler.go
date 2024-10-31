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
	"fmt"
	"sync/atomic"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	fcmodels "github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	"gitlab.com/nunet/device-management-service/types"
)

const (
	vmDestroyTimeout = time.Second * 10
)

// executionHandler is a struct that holds the necessary information to manage the execution of a firecracker VM.
type executionHandler struct {
	//
	// provided by the executor
	ID     string
	client *Client

	// meta data about the task
	JobID       string
	executionID string
	machine     *firecracker.Machine
	resultsDir  string

	// synchronization
	activeCh chan bool    // Blocks until the VM starts running.
	waitCh   chan bool    // Blocks until execution completes or fails.
	running  *atomic.Bool // Indicates if the VM is currently running.

	// result of the execution
	result *types.ExecutionResult
}

// active returns true if the firecracker VM is running.
func (h *executionHandler) active() bool {
	return h.running.Load()
}

// run starts the firecracker VM and waits for it to finish.
func (h *executionHandler) run(ctx context.Context) {
	h.running.Store(true)

	defer func() {
		if err := h.destroy(vmDestroyTimeout); err != nil {
			log.Warnf("failed to destroy VM: %v", err)
		}
		h.running.Store(false)
		close(h.waitCh)
	}()

	// Start the VM
	log.Infow("firecracker_execution_starting", "executionID", h.executionID)
	if err := h.client.StartVM(ctx, h.machine); err != nil {
		h.result = types.NewFailedExecutionResult(fmt.Errorf("failed to start VM: %v", err))
		log.Errorw("firecracker_vm_start_failure", "error", err, "executionID", h.executionID)
		return
	}

	close(h.activeCh) // Indicate that the VM has started.

	err := h.machine.Wait(ctx)
	if err != nil {
		if ctx.Err() != nil {
			h.result = types.NewFailedExecutionResult(fmt.Errorf("context closed while waiting on VM: %v", err))
			log.Errorw("firecracker_execution_context_closed", "error", err, "executionID", h.executionID)
			return
		}
		h.result = types.NewFailedExecutionResult(fmt.Errorf("failed to wait on VM: %v", err))
		log.Errorw("firecracker_vm_wait_failure", "error", err, "executionID", h.executionID)
		return
	}

	h.result = types.NewExecutionResult(types.ExecutionStatusCodeSuccess)
	log.Infow("firecracker_execution_success", "executionID", h.executionID)
}

// kill stops the firecracker VM.
func (h *executionHandler) kill(ctx context.Context) error {
	log.Infow("firecracker_kill_vm", "executionID", h.executionID)
	return h.client.ShutdownVM(ctx, h.machine)
}

// destroy stops the firecracker VM and removes its resources.
func (h *executionHandler) destroy(timeout time.Duration) error {
	log.Infow("firecracker_destroy_vm", "executionID", h.executionID)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return h.client.DestroyVM(ctx, h.machine, timeout)
}

// pause pauses the firecracker VM.
func (h *executionHandler) pause(ctx context.Context) error {
	return h.client.PauseVM(ctx, h.machine)
}

// resume resumes the firecracker VM.
func (h *executionHandler) resume(ctx context.Context) error {
	return h.client.ResumeVM(ctx, h.machine)
}

// status returns the result of the execution.
func (h *executionHandler) status(ctx context.Context) (types.ExecutionStatus, error) {
	if !h.active() {
		if h.result != nil {
			if h.result.ExitCode == types.ExecutionStatusCodeSuccess {
				return types.ExecutionStatusSuccess, nil
			}
			return types.ExecutionStatusFailed, fmt.Errorf("VM exited: %v", h.result.ErrorMsg)
		}
		return types.ExecutionStatusPending, nil
	}

	info, err := h.machine.DescribeInstanceInfo(ctx)
	if err != nil {
		return types.ExecutionStatusFailed, fmt.Errorf("failed to get VM status: %v", err)
	}
	switch *info.State {
	case fcmodels.InstanceInfoStateNotStarted:
		return types.ExecutionStatusPending, nil
	case fcmodels.InstanceInfoStateRunning:
		return types.ExecutionStatusRunning, nil
	case fcmodels.InstanceInfoStatePaused:
		return types.ExecutionStatusPaused, nil
	default:
		return types.ExecutionStatusFailed, fmt.Errorf("unknown VM state: %s", *info.State)
	}
}
