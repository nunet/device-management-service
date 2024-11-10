// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/observability"
)

var (
	defaultImage  = "python:alpine"
	transientCmd  = []string{"echo", "hello world"}
	persistentCmd = []string{"sh", "-c", "while true; do date; sleep 1; done"}
)

// ClientTestSuite is the test suite for the Docker client.
type ClientTestSuite struct {
	suite.Suite
	client *docker.Client
}

// SetupTest sets up the test suite by initializing a new Docker client.
func (s *ClientTestSuite) SetupTest() {
	// Set observability to no-op mode for this test
	observability.SetNoOpMode(true)

	c, err := docker.NewDockerClient()
	s.NoError(err)
	s.client = c
}

func ensureDockerSetup(t *testing.T) {
	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	errMsg := "Docker is not installed or running. Skipping Docker client tests"

	c, err := docker.NewDockerClient()

	if err != nil || !c.IsInstalled(context.Background()) {
		if isPipeline {
			t.Fatal(errMsg)
		} else {
			t.Skip(errMsg)
		}
	}
}

// TestClientTestSuite runs the test suite for the Docker client.
func TestClientTestSuite(t *testing.T) {
	ensureDockerSetup(t)
	suite.Run(t, new(ClientTestSuite))
}

// createTestContainer is a helper method to create a container for testing.
func (s *ClientTestSuite) createTestContainer(image string, cmd []string) string {
	config := &container.Config{
		Image: image,
		Cmd:   cmd,
	}
	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}
	platform := &v1.Platform{}
	pullImage := true

	_, err := s.client.GetImage(context.Background(), image)
	if err == nil {
		pullImage = false
	}

	id, err := s.client.CreateContainer(
		context.Background(),
		config,
		hostConfig,
		networkingConfig,
		platform,
		fmt.Sprintf("nunet_test_container-%s", uuid.New()),
		pullImage,
	)

	s.NoError(err)

	s.T().Cleanup(func() {
		timeout := int(docker.DestroyTimeout)
		options := container.StopOptions{Timeout: &timeout}
		_ = s.client.StopContainer(context.Background(), id, options)
		_ = s.client.RemoveContainer(context.Background(), id)
	})
	return id
}

// TestIsInstalled tests the IsInstalled method of the Docker client.
func (s *ClientTestSuite) TestIsInstalled() {
	s.True(s.client.IsInstalled(context.Background()))
}

// TestCreateContainer tests the CreateContainer method of the Docker client.
func (s *ClientTestSuite) TestCreateContainer() {
	id := s.createTestContainer(defaultImage, transientCmd)
	s.NotEmpty(id)
}

// TestInspectContainer tests the InspectContainer method of the Docker client.
func (s *ClientTestSuite) TestInspectContainer() {
	id := s.createTestContainer(defaultImage, transientCmd)
	s.Require().NotEmpty(id)

	container, err := s.client.InspectContainer(context.Background(), id)
	s.NoError(err)
	s.Equal(id, container.ID)
}

// TestContainerLifecycle tests the lifecycle methods of the Docker client.
func (s *ClientTestSuite) TestContainerLifecycle() {
	id := s.createTestContainer(defaultImage, persistentCmd)
	s.Require().NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.Require().NoError(err)

	err = s.client.PauseContainer(context.Background(), id)
	s.Require().NoError(err)

	err = s.client.ResumeContainer(context.Background(), id)
	s.Require().NoError(err)

	stopTime := 0
	err = s.client.StopContainer(context.Background(), id, container.StopOptions{Timeout: &stopTime})
	s.Require().NoError(err)

	err = s.client.RemoveContainer(context.Background(), id)
	s.Require().NoError(err)
}

// TestFollowLogs tests the FollowLogs method of the Docker client.
func (s *ClientTestSuite) TestFollowLogs() {
	// Create a container that outputs some logs
	id := s.createTestContainer(defaultImage, []string{"sh", "-c", "echo stdout message && echo stderr message >&2"})
	s.NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stdout, stderr, err := s.client.FollowLogs(ctx, id)
	s.NoError(err)

	// Use a WaitGroup to ensure we've read all the output
	var wg sync.WaitGroup
	wg.Add(2)

	var stdoutContent, stderrContent []byte
	var stdoutErr, stderrErr error

	go func() {
		defer wg.Done()
		stdoutContent, stdoutErr = io.ReadAll(stdout)
	}()

	go func() {
		defer wg.Done()
		stderrContent, stderrErr = io.ReadAll(stderr)
	}()

	// Wait for both goroutines to finish
	wg.Wait()

	s.NoError(stdoutErr)
	s.NoError(stderrErr)

	// Check the content of stdout and stderr
	s.Contains(string(stdoutContent), "stdout message")
	s.Contains(string(stderrContent), "stderr message")
}
