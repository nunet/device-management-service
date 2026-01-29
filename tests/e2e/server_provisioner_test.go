// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// DeployWithOnDemandProvisioner
func DeployWithOnDemandProvisioner(suite *TestSuite) {
	suite.Run("dms with provisioner", func() {
		defer saveLogAndDeleteVM()
		gateway := suite.nodes[0]
		orchestrator := suite.nodes[1]

		// offboard this machine to not accept any bid request
		gateway.client.offboard(suite.T(), gateway.userContext, gateway.password)
		orchestrator.client.offboard(suite.T(), orchestrator.userContext, orchestrator.password)

		deploymentResult := orchestrator.client.deploy(
			suite.T(), orchestrator.userContext, orchestrator.password,
			filepath.Join(suite.testDataDir, "ensembles", "hello.yaml"),
			"5m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		suite.Require().Eventually(func() bool {
			status, err := orchestrator.client.deploymentStatus(suite.T(), orchestrator.userContext, orchestrator.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 5*time.Minute, 5*time.Second, "Hello-world deployment did not reach Running status")
	})
}

func saveLogAndDeleteVM() {
	_, _ = runCommand(context.Background(), "incus", "file", "pull", "VM1/root/logfile.log", "/tmp/logfile.log")

	// incus stop <vm-name>
	_, _ = runCommand(context.Background(), "incus", "stop", "VM1", "--force")

	// incus delete <vm-name>
	_, _ = runCommand(context.Background(), "incus", "delete", "VM1", "--force")
}

//nolint:unparam
func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command %q failed: %w; stderr: %s", name, err, stderr.String())
	}

	return out.String(), nil
}
