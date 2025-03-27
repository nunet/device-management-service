// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build integration || !unit

package itest

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestIntegration is the entry point for the integration tests.
//
// We need to ensure the following while adding more tests
// 1. Tests should run in parallel
// 2. portIndex should be unique for each test suite
// 3. Runner function should be defined in the respective test file and must be passed to the test suite
func TestIntegration(t *testing.T) {
	t.Parallel()
	setupGlusterfsServer(t)
	t.Cleanup(func() {
		if err := deleteGlusterContainer(); err != nil {
			t.Logf("failed to delete gluster container: %v", err)
		}
	})

	t.Run("BasicTests", func(t *testing.T) {
		t.Parallel()

		basicTests := &TestSuite{
			numNodes:      3,
			Name:          "basic_tests",
			restPortIndex: 8090,
			p2pPortIndex:  10689,
			runner:        BasicTests,
		}
		suite.Run(t, basicTests)
	})

	t.Run("DeploymentTests", func(t *testing.T) {
		t.Parallel()

		deploymentTests := &TestSuite{
			numNodes:      3,
			Name:          "deployment_tests",
			restPortIndex: 8093,
			p2pPortIndex:  10692,
			runner:        DeploymentTest,
		}
		suite.Run(t, deploymentTests)
	})

	t.Run("DeploymentWithVolumesTests", func(t *testing.T) {
		t.Parallel()
		t.Skip("not implemented")

		deploymentWithVolumesTests := &TestSuite{
			numNodes:      3,
			Name:          "deployment_with_volumes_tests",
			restPortIndex: 8096,
			p2pPortIndex:  10695,
			runner:        DeployWithVolumeTest,
		}
		suite.Run(t, deploymentWithVolumesTests)
	})
}
