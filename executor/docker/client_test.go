package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/executor/docker"
)

var (
	defaultImage = "alpine"
	defaultCmd   = []string{"echo", "hello world"}
	defaultName  = "nunet-test-container"
)

// ClientTestSuite is the test suite for the Docker client.
type ClientTestSuite struct {
	suite.Suite
	client *docker.Client
}

// SetupTest sets up the test suite by initializing a new Docker client.
func (s *ClientTestSuite) SetupTest() {
	c, err := docker.NewDockerClient()
	require.NoError(s.T(), err)
	s.client = c
}

// TestClientTestSuite runs the test suite for the Docker client.
func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

// createTestContainer is a helper method to create a container for testing.
func (s *ClientTestSuite) createTestContainer(name, image string, cmd []string) string {
	config := &container.Config{
		Image: image,
		Cmd:   cmd,
	}
	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}
	platform := &v1.Platform{}

	id, err := s.client.CreateContainer(
		context.Background(),
		config,
		hostConfig,
		networkingConfig,
		platform,
		name,
	)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() {
		_ = s.client.StopContainer(context.Background(), id, docker.DestroyTimeout)
		_ = s.client.RemoveContainer(context.Background(), id)
	})
	return id
}

// TestIsInstalled tests the IsInstalled method of the Docker client.
func (s *ClientTestSuite) TestIsInstalled() {
	assert.True(s.T(), s.client.IsInstalled(context.Background()))
}

// TestCreateContainer tests the CreateContainer method of the Docker client.
func (s *ClientTestSuite) TestCreateContainer() {
	id := s.createTestContainer(defaultName, defaultImage, defaultCmd)
	assert.NotEmpty(s.T(), id)
}

// TestInspectContainer tests the InspectContainer method of the Docker client.
func (s *ClientTestSuite) TestInspectContainer() {
	id := s.createTestContainer(defaultName, defaultImage, defaultCmd)
	assert.NotEmpty(s.T(), id)

	container, err := s.client.InspectContainer(context.Background(), id)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), id, container.ID)
}

// TestStartContainer tests the StartContainer method of the Docker client.
func (s *ClientTestSuite) TestStartContainer() {
	id := s.createTestContainer(defaultName, defaultImage, defaultCmd)
	assert.NotEmpty(s.T(), id)

	err := s.client.StartContainer(context.Background(), id)
	require.NoError(s.T(), err)
}

// TestStopContainer tests the StopContainer method of the Docker client.
func (s *ClientTestSuite) TestStopContainer() {
	id := s.createTestContainer(defaultName, defaultImage, defaultCmd)
	assert.NotEmpty(s.T(), id)

	err := s.client.StartContainer(context.Background(), id)
	require.NoError(s.T(), err)

	err = s.client.StopContainer(context.Background(), id, time.Second*5)
	require.NoError(s.T(), err)
}

// TestRemoveContainer tests the RemoveContainer method of the Docker client.
func (s *ClientTestSuite) TestRemoveContainer() {
	id := s.createTestContainer(defaultName, defaultImage, defaultCmd)
	assert.NotEmpty(s.T(), id)

	err := s.client.RemoveContainer(context.Background(), id)
	require.NoError(s.T(), err)
}
