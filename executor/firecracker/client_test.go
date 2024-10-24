// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package firecracker_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	firecrackerSdk "github.com/firecracker-microvm/firecracker-go-sdk"
	firecrackerModels "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/firecracker"
)

const (
	defaultSocketPath = "/tmp/firecracker.sock"
	rootDrivePath     = "testdata/rootfs.ext4"
	kernelImagePath   = "testdata/vmlinux.bin"
)

// ClientTestSuite is the test suite for the Firecracker client.
type ClientTestSuite struct {
	suite.Suite
	client *firecracker.Client
}

// SetupTest sets up the test suite by initializing a new Firecracker client.
func (s *ClientTestSuite) SetupTest() {
	c, err := firecracker.NewFirecrackerClient()
	require.NoError(s.T(), err)
	s.client = c
}

func ensureFirecrackerSetup(t *testing.T) {
	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	errMsg := "Firecracker is not installed or running. Skipping Firecracker tests."

	c, err := firecracker.NewFirecrackerClient()

	if err != nil || !c.IsInstalled(context.Background()) {
		if isPipeline {
			t.Fatal(errMsg)
		} else {
			t.Skip(errMsg)
		}
	}

	// Check if test data exists.
	testDataErrMsg := "%s not found (%s). Run \"make testdata\" before running tests."
	if _, err := os.Stat(rootDrivePath); os.IsNotExist(err) {
		t.Fatalf(testDataErrMsg, "root drive image", rootDrivePath)
	}
	if _, err := os.Stat(kernelImagePath); os.IsNotExist(err) {
		t.Fatalf(testDataErrMsg, "kernel image", kernelImagePath)
	}
}

// TestClientTestSuite runs the test suite for the Firecracker client.
func TestClientTestSuite(t *testing.T) {
	ensureFirecrackerSetup(t)
	suite.Run(t, new(ClientTestSuite))
}

// createTestVM is a helper method to create a VM for testing.
func (s *ClientTestSuite) createTestVM(socketPath string) *firecrackerSdk.Machine {
	cfg := firecrackerSdk.Config{
		SocketPath:      socketPath,
		KernelImagePath: kernelImagePath,
		Drives: []firecrackerModels.Drive{
			{
				DriveID:      firecrackerSdk.String("1"),
				PathOnHost:   firecrackerSdk.String(rootDrivePath),
				IsRootDevice: firecrackerSdk.Bool(true),
				IsReadOnly:   firecrackerSdk.Bool(true),
			},
		},
		MachineCfg: firecrackerModels.MachineConfiguration{
			VcpuCount:  firecrackerSdk.Int64(1),
			MemSizeMib: firecrackerSdk.Int64(1024),
		},
	}
	m, err := s.client.CreateVM(context.Background(), cfg)
	require.NoError(s.T(), err)

	s.T().Cleanup(func() {
		_ = s.client.DestroyVM(context.Background(), m, 10*time.Second)
	})

	return m
}

// TestIsInstalled tests the IsInstalled method of the Firecracker client.
func (s *ClientTestSuite) TestIsInstalled() {
	assert.True(s.T(), s.client.IsInstalled(context.Background()))
}

// TestFindVM tests the FindVM method of the Firecracker client.
func (s *ClientTestSuite) TestFindVM() {
	m := s.createTestVM(defaultSocketPath)
	require.NotNil(s.T(), m)

	err := s.client.StartVM(context.Background(), m)
	require.NoError(s.T(), err)

	vm, err := s.client.FindVM(context.Background(), m.Cfg.SocketPath)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), vm)
}

// TestVMLifecycle tests the lifecycle of a Firecracker VM.
func (s *ClientTestSuite) TestVMLifecycle() {
	m := s.createTestVM(defaultSocketPath)
	require.NotNil(s.T(), m)

	err := s.client.StartVM(context.Background(), m)
	require.NoError(s.T(), err)

	err = s.client.PauseVM(context.Background(), m)
	require.NoError(s.T(), err)

	err = s.client.ResumeVM(context.Background(), m)
	require.NoError(s.T(), err)

	err = s.client.ShutdownVM(context.Background(), m)
	require.NoError(s.T(), err)
}
