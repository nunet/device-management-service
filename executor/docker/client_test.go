// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package docker_test

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
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
func (s *ClientTestSuite) createTestContainer(imageName string, cmd []string) string {
	config := &container.Config{
		Image: imageName,
		Cmd:   cmd,
	}
	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}
	platform := &v1.Platform{}
	pullImage := true

	_, err := s.client.GetImage(context.Background(), imageName)
	if err == nil {
		pullImage = false
	}

	id, err := s.client.CreateContainer(
		context.Background(),
		config,
		hostConfig,
		networkingConfig,
		image.PullOptions{},
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
// It tests if logs are uploaded continuously.
func (s *ClientTestSuite) TestFollowLogs() {
	wordBeforeSleep := "one"
	wordAfterSleep := "two"

	// Create command that prints first word, sleeps, then prints second word
	id := s.createTestContainer(defaultImage, []string{
		"sh", "-c",
		fmt.Sprintf("echo '%s'; sleep %d; echo '%s'", wordBeforeSleep, baseSleep, wordAfterSleep),
	})
	s.NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.NoError(err)

	stdout, stderr, err := s.client.FollowLogs(context.Background(), id)
	s.NoError(err)

	// Create buffers for stdout and stderr
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	// Copy the output streams to our buffers
	go func() {
		_, err := io.Copy(stdoutBuf, stdout)
		s.NoError(err)
	}()

	go func() {
		_, err := io.Copy(stderrBuf, stderr)
		s.NoError(err)
	}()

	// Check for first message
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		stdoutStr := stdoutBuf.String()
		assert.Contains(c, stdoutStr, wordBeforeSleep, "First message should appear when starting container")
		assert.NotContains(c, stdoutStr, wordAfterSleep, "Second message should not appear yet")
	}, baseTimeout-1, 100*time.Millisecond)

	// Check for second message
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		stdoutStr := stdoutBuf.String()
		assert.Contains(c, stdoutStr, wordBeforeSleep, "First message should appear when starting container")
		assert.Contains(c, stdoutStr, wordAfterSleep, "Second message should appear after ~2 seconds")
	}, baseTimeout, 100*time.Millisecond)
}

// TestPullImage tests the PullImage method of the Docker client.
func (s *ClientTestSuite) TestPullImage() {
	testImage := "alpine:latest"

	digest, err := s.client.PullImage(context.Background(), testImage, image.PullOptions{})
	s.NoError(err)
	s.NotEmpty(digest)

	image, err := s.client.GetImage(context.Background(), testImage)
	s.NoError(err)
	s.Contains(image.RepoTags, testImage)
}

// TestGetOutputStream tests the GetOutputStream method of the Docker client.
func (s *ClientTestSuite) TestGetOutputStream() {
	const msg = "test output"
	id := s.createTestContainer(defaultImage, []string{"echo", msg})
	s.Require().NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.Require().NoError(err)

	waitCh, errCh := s.client.WaitContainer(context.Background(), id)
	select {
	case <-waitCh:
	case err := <-errCh:
		s.Require().NoError(err)
	case <-time.After(baseTimeout + 3):
		s.Fail("Container did not finish in time")
	}

	stream, err := s.client.GetOutputStream(context.Background(), id, "", false)
	s.NoError(err)
	defer stream.Close()

	output, err := io.ReadAll(stream)
	s.NoError(err)
	s.Contains(string(output), msg)
}

// TestWaitContainer tests the WaitContainer method of the Docker client.
func (s *ClientTestSuite) TestWaitContainer() {
	id := s.createTestContainer(defaultImage, []string{"sleep", fmt.Sprintf("%d", baseSleep-1)})
	s.Require().NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.Require().NoError(err)

	waitCh, errCh := s.client.WaitContainer(context.Background(), id)

	select {
	case result := <-waitCh:
		s.Equal(int64(0), result.StatusCode)
	case err := <-errCh:
		s.Require().NoError(err)
	case <-time.After(baseTimeout * 5):
		s.Fail("Container did not finish in time")
	}
}

// TestFindContainer tests the FindContainer method of the Docker client.
func (s *ClientTestSuite) TestFindContainer() {
	testLabel := "nunet-test-label"
	testValue := uuid.New().String()

	config := &container.Config{
		Image: defaultImage,
		Cmd:   transientCmd,
		Labels: map[string]string{
			testLabel: testValue,
		},
	}

	id, err := s.client.CreateContainer(
		context.Background(),
		config,
		&container.HostConfig{},
		&network.NetworkingConfig{},
		image.PullOptions{},
		&v1.Platform{},
		fmt.Sprintf("nunet_test_container-%s", uuid.New()),
		false,
	)
	s.NoError(err)
	s.T().Cleanup(func() {
		_ = s.client.RemoveContainer(context.Background(), id)
	})

	foundID, err := s.client.FindContainer(context.Background(), testLabel, testValue)
	s.NoError(err)
	s.Equal(id, foundID)

	// Test non-existent container
	_, err = s.client.FindContainer(context.Background(), "non-existent", "value")
	s.Error(err)
}

// TestRemoveObjectsWithLabel tests the RemoveObjectsWithLabel method of the Docker client.
func (s *ClientTestSuite) TestRemoveObjectsWithLabel() {
	testLabel := "nunet-test-cleanup"
	testValue := uuid.New().String()

	// Create multiple containers with the same label
	for i := range 3 {
		config := &container.Config{
			Image: defaultImage,
			Cmd:   transientCmd,
			Labels: map[string]string{
				testLabel: testValue,
			},
		}

		_, err := s.client.CreateContainer(
			context.Background(),
			config,
			&container.HostConfig{},
			&network.NetworkingConfig{},
			image.PullOptions{},
			&v1.Platform{},
			fmt.Sprintf("nunet_test_container-%s-%d", uuid.New(), i),
			false,
		)
		s.NoError(err)
	}

	// Remove all containers with the label
	err := s.client.RemoveObjectsWithLabel(context.Background(), testLabel, testValue)
	s.NoError(err)

	// Verify containers are removed
	_, err = s.client.FindContainer(context.Background(), testLabel, testValue)
	s.Error(err)
}

// TestExec tests the Exec method of the Docker client.
func (s *ClientTestSuite) TestExec() {
	id := s.createTestContainer(defaultImage, persistentCmd)
	s.Require().NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.Require().NoError(err)

	// Execute a simple command
	const msg = "exec test"
	exitCode, stdout, stderr, err := s.client.Exec(context.Background(), id, []string{"echo", msg})
	s.NoError(err)
	s.Equal(0, exitCode)
	s.Contains(stdout, msg)
	s.Empty(stderr)

	// Execute a command that fails
	exitCode, _, _, err = s.client.Exec(context.Background(), id, []string{"sh", "-c", "exit 1"})
	s.NoError(err)
	s.Equal(1, exitCode)
}

// TestCopyToContainer tests the CopyToContainer method of the Docker client.
func (s *ClientTestSuite) TestCopyToContainer() {
	id := s.createTestContainer(defaultImage, persistentCmd)
	s.Require().NotEmpty(id)

	err := s.client.StartContainer(context.Background(), id)
	s.Require().NoError(err)

	// Create test content
	testContent := "test file content"
	testFileName := "test.txt"

	// Create TAR archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add file to TAR archive
	header := &tar.Header{
		Name: testFileName,
		Mode: 0o644,
		Size: int64(len(testContent)),
	}

	err = tw.WriteHeader(header)
	s.Require().NoError(err)

	_, err = tw.Write([]byte(testContent))
	s.Require().NoError(err)

	err = tw.Close()
	s.Require().NoError(err)

	// Copy TAR archive to container
	err = s.client.CopyToContainer(context.Background(), id, "/tmp", &buf, container.CopyToContainerOptions{})
	s.NoError(err)

	// Verify file was copied
	exitCode, stdout, _, err := s.client.Exec(context.Background(), id, []string{"cat", "/tmp/" + testFileName})
	s.NoError(err)
	s.Equal(0, exitCode)
	s.Equal(testContent, stdout)
}
