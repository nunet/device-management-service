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
	"os"
	"syscall"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	fcmodels "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

const pidCheckTickTime = 100 * time.Millisecond

// Client wraps the Firecracker SDK to provide high-level operations on Firecracker VMs.
type Client struct{}

// NewFirecrackerClient initializes a new Firecracker client.
func NewFirecrackerClient() (*Client, error) {
	log.Infow("firecracker_client_init_started")
	client := &Client{}
	log.Infow("firecracker_client_init_success")
	return client, nil
}

// IsInstalled checks if Firecracker is installed on the host.
func (c *Client) IsInstalled(ctx context.Context) bool {
	log.Infow("firecracker_client_is_installed_check_started")

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := firecracker.VMCommandBuilder{}.WithArgs([]string{"--version"}).Build(ctx)

	version, err := cmd.Output()
	if err != nil || !cmd.ProcessState.Success() {
		log.Errorw("firecracker_client_is_installed_failure", "error", err)
		return false
	}

	isInstalled := string(version) != ""
	if isInstalled {
		log.Infow("firecracker_client_is_installed_success")
	} else {
		log.Errorw("firecracker_client_is_installed_failure", "error", "version check failed")
	}
	return isInstalled
}

// CreateVM creates a new Firecracker VM with the specified configuration.
func (c *Client) CreateVM(
	ctx context.Context,
	cfg firecracker.Config,
) (*firecracker.Machine, error) {
	log.Infow("firecracker_create_vm_started", "socketPath", cfg.SocketPath)
	cmd := firecracker.VMCommandBuilder{}.
		WithSocketPath(cfg.SocketPath).
		Build(ctx)
	machineOpts := []firecracker.Opt{
		firecracker.WithProcessRunner(cmd),
	}

	m, err := firecracker.NewMachine(ctx, cfg, machineOpts...)
	if err != nil {
		log.Errorw("firecracker_create_vm_failure", "error", err)
		return nil, err
	}

	log.Infow("firecracker_create_vm_success", "socketPath", cfg.SocketPath)
	return m, nil
}

// StartVM starts the Firecracker VM.
func (c *Client) StartVM(ctx context.Context, m *firecracker.Machine) error {
	log.Infow("firecracker_start_vm_started")
	err := m.Start(ctx)
	if err != nil {
		log.Errorw("firecracker_start_vm_failure", "error", err)
		return err
	}
	log.Infow("firecracker_start_vm_success")
	return nil
}

// StopVM stops the Firecracker VM.
func (c *Client) StopVM(_ context.Context, m *firecracker.Machine) error {
	return m.StopVMM()
}

// ShutdownVM shuts down the Firecracker VM.
func (c *Client) ShutdownVM(ctx context.Context, m *firecracker.Machine) error {
	log.Infow("firecracker_shutdown_vm_started")
	err := m.Shutdown(ctx)
	if err != nil {
		log.Errorw("firecracker_shutdown_vm_failure", "error", err)
		return err
	}
	log.Infow("firecracker_shutdown_vm_success")
	return nil
}

// DestroyVM destroys the Firecracker VM.
func (c *Client) DestroyVM(
	ctx context.Context,
	m *firecracker.Machine,
	timeout time.Duration,
) error {
	log.Infow("firecracker_destroy_vm_started")

	defer os.Remove(m.Cfg.SocketPath)

	// Get the PID of the Firecracker process and shut down the VM.
	// If the process is still running after the timeout, kill it.

	// If the process is not running, return early.
	pid, _ := m.PID()
	if pid <= 0 {
		return nil
	}

	err := c.ShutdownVM(ctx, m)
	if err != nil {
		return err
	}

	pid, _ = m.PID()
	if pid <= 0 {
		log.Infow("firecracker_destroy_vm_no_pid")
		return nil
	}

	// This checks if the process is still running every pidCheckTickTime.
	// If the process is still running after the timeout it will set done to false.
	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(pidCheckTickTime)
		defer ticker.Stop()
		to := time.NewTimer(timeout)
		defer to.Stop()
		for {
			select {
			case <-to.C:
				done <- false
				return
			case <-ticker.C:
				if pid, _ = m.PID(); pid <= 0 {
					done <- true
					return
				}
			}
		}
	}()

	// Wait for the check to finish.
	killed := <-done
	if !killed {
		// The shutdown request timed out, kill the process with SIGKILL.
		err := syscall.Kill(pid, syscall.SIGKILL)
		if err != nil {
			log.Errorw("firecracker_destroy_vm_kill_failure", "error", err)
			return fmt.Errorf("failed to kill process: %v", err)
		}
		log.Infow("firecracker_destroy_vm_kill_success")
	}
	log.Infow("firecracker_destroy_vm_success")
	return nil
}

// PauseVM pauses the Firecracker VM.
func (c *Client) PauseVM(ctx context.Context, m *firecracker.Machine) error {
	return m.PauseVM(ctx)
}

// ResumeVM resumes the Firecracker VM.
func (c *Client) ResumeVM(ctx context.Context, m *firecracker.Machine) error {
	return m.ResumeVM(ctx)
}

// FindVM finds a Firecracker VM by its socket path.
// This implementation checks if the VM is running by sending a request to the Firecracker API.
func (c *Client) FindVM(ctx context.Context, socketPath string) (*firecracker.Machine, error) {
	// Check if the socket file exists.

	log.Infow("firecracker_find_vm_started", "socketPath", socketPath)

	if _, err := os.Stat(socketPath); err != nil {
		log.Errorw("firecracker_find_vm_failure", "error", err)
		return nil, fmt.Errorf("VM with socket path %v not found", socketPath)
	}

	// Create a new Firecracker machine instance.
	cmd := firecracker.VMCommandBuilder{}.WithSocketPath(socketPath).Build(ctx)
	machine, err := firecracker.NewMachine(
		ctx,
		firecracker.Config{SocketPath: socketPath},
		firecracker.WithProcessRunner(cmd),
	)
	if err != nil {
		log.Errorw("firecracker_find_vm_failure", "error", err)
		return nil, fmt.Errorf("failed to create machine with socket %s: %v", socketPath, err)
	}

	// Check if the VM is running by getting its instance info.
	info, err := machine.DescribeInstanceInfo(ctx)
	if err != nil {
		log.Errorw("firecracker_find_vm_failure", "error", err)
		return nil, fmt.Errorf("failed to get instance info for socket %s: %v", socketPath, err)
	}

	if *info.State != fcmodels.InstanceInfoStateRunning {
		return nil, fmt.Errorf(
			"VM with socket %s is not running, current state: %s",
			socketPath,
			*info.State,
		)
	}

	log.Infow("firecracker_find_vm_success", "socketPath", socketPath)
	return machine, nil
}
