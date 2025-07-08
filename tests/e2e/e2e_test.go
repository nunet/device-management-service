// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build e2e || !unit

package itest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/network/utils"
)

// TestE2E is the entry point for the e2e tests.
//
// We need to ensure the following while adding more tests
// 1. Tests should run in parallel
// 2. portIndex should be unique for each test suite
// 3. Runner function should be defined in the respective test file and must be passed to the test suite
func TestE2E(t *testing.T) {
	t.Parallel()
	var (
		testSuites         = 5
		totalPortsRequired = 2 * testSuites
	)

	ports, err := utils.GetMultipleAvailablePorts(totalPortsRequired)
	require.NoError(t, err)
	require.Len(t, ports, totalPortsRequired)

	t.Run("BasicTests", func(t *testing.T) {
		t.Parallel()
		basicTests := &TestSuite{
			numNodes:      3,
			Name:          "basic_tests",
			restPortIndex: ports[0],
			p2pPortIndex:  ports[1],
			runner:        BasicTests,
		}
		suite.Run(t, basicTests)
	})

	t.Run("DeploymentTests", func(t *testing.T) {
		t.Parallel()
		deploymentTests := &TestSuite{
			numNodes:      3,
			Name:          "deployment_tests",
			restPortIndex: ports[2],
			p2pPortIndex:  ports[3],
			runner:        DeploymentTest,
		}
		suite.Run(t, deploymentTests)
	})

	t.Run("DeploymentWithVolumesTests", func(t *testing.T) {
		t.Parallel()
		t.Skip("not implemented")

		err := supportsGluster()
		if err != nil {
			t.Skipf("glusterfs not supported, skipping gluster tests: %v", err)
		} else {
			setupGlusterfsServer(t)
		}

		deploymentWithVolumesTests := &TestSuite{
			numNodes:      3,
			Name:          "deployment_with_volumes_tests",
			restPortIndex: ports[4],
			p2pPortIndex:  ports[5],
			runner:        DeployWithVolumeTest,
		}
		suite.Run(t, deploymentWithVolumesTests)
	})

	t.Run("DeploymentUpdates", func(t *testing.T) {
		deploymentUpdates := &TestSuite{
			numNodes:      3,
			Name:          "deployment_updates",
			restPortIndex: ports[6],
			p2pPortIndex:  ports[7],
			runner:        DeploymentUpdates,
		}
		suite.Run(t, deploymentUpdates)
	})

	t.Run("DeploymentFullAssertion", func(t *testing.T) {
		deploymentFullAssertion := &TestSuite{
			numNodes:      4,
			Name:          "deployment_assert_subnet",
			restPortIndex: ports[8],
			p2pPortIndex:  ports[9],
			runner:        DeploymentFullAssertion,
		}
		suite.Run(t, deploymentFullAssertion)
	})
}
