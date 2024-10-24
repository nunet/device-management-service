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

func NewFirecrackerClient() (*Client, error) {
	return &Client{}, nil
}

// IsInstalled checks if Firecracker is installed on the host.
func (c *Client) IsInstalled(ctx context.Context) bool {
	// Check if the Firecracker binary is installed.
	// This implementation sends a version request to the Firecracker binary.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := firecracker.VMCommandBuilder{}.WithArgs([]string{"--version"}).Build(ctx)

	version, err := cmd.Output()
	if err != nil || !cmd.ProcessState.Success() {
		return false
	}

	return string(version) != ""
}

// CreateVM creates a new Firecracker VM with the specified configuration.
func (c *Client) CreateVM(
	ctx context.Context,
	cfg firecracker.Config,
) (*firecracker.Machine, error) {
	cmd := firecracker.VMCommandBuilder{}.
		WithSocketPath(cfg.SocketPath).
		Build(ctx)
	machineOpts := []firecracker.Opt{
		firecracker.WithProcessRunner(cmd),
	}

	m, err := firecracker.NewMachine(ctx, cfg, machineOpts...)
	return m, err
}

// StartVM starts the Firecracker VM.
func (c *Client) StartVM(ctx context.Context, m *firecracker.Machine) error {
	return m.Start(ctx)
}

// StopVM stops the Firecracker VM.
func (c *Client) StopVM(_ context.Context, m *firecracker.Machine) error {
	return m.StopVMM()
}

// ShutdownVM shuts down the Firecracker VM.
func (c *Client) ShutdownVM(ctx context.Context, m *firecracker.Machine) error {
	return m.Shutdown(ctx)
}

// DestroyVM destroys the Firecracker VM.
func (c *Client) DestroyVM(
	ctx context.Context,
	m *firecracker.Machine,
	timeout time.Duration,
) error {
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
			return fmt.Errorf("failed to kill process: %v", err)
		}
	}

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
	if _, err := os.Stat(socketPath); err != nil {
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
		return nil, fmt.Errorf("failed to create machine with socket %s: %v", socketPath, err)
	}

	// Check if the VM is running by getting its instance info.
	info, err := machine.DescribeInstanceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance info for socket %s: %v", socketPath, err)
	}

	if *info.State != fcmodels.InstanceInfoStateRunning {
		return nil, fmt.Errorf(
			"VM with socket %s is not running, current state: %s",
			socketPath,
			*info.State,
		)
	}

	return machine, nil
}
