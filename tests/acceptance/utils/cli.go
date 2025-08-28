// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
)

// Context represents a named context within a node
type Context struct {
	Name string
	DID  string
	node *Node
}

func (c *Context) Grant(did string) (token string, err error) {
	return c.node.RunDMSCmd(fmt.Sprintf("nunet cap grant --context %s --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-30 %s",
		c.Name, did))
}

func (c *Context) Delegate(did string) (token string, err error) {
	return c.node.RunDMSCmd(fmt.Sprintf("nunet cap delegate --context %s --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-30 %s",
		c.Name, did))
}

func (c *Context) Anchor(kind, arg string) error {
	_, err := c.node.RunDMSCmd(fmt.Sprintf("nunet cap anchor --context %s --%s '%s'",
		c.Name, kind, arg))
	return err
}

func (c *Context) Run() error {
	return c.node.RunDMSCmdBackground(fmt.Sprintf("GOLOG_LOG_LEVEL=debug nunet run -c %s > dms-logs.txt 2>&1", c.Name))
}

func (c *Context) PeerAddr() (*node.PeerAddrInfoResponse, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/peers/self", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call self behavior: %w", err)
	}
	var resp node.PeerAddrInfoResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	return &resp, nil
}

func (c *Context) Connect(target string) error {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/peers/connect --address %s", c.Name, target))
	if err != nil {
		return fmt.Errorf("failed to call connect behavior: %w", err)
	}
	var resp node.PeerConnectResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to connect: %s", resp.Error)
	}
	return nil
}

func (c *Context) Onboard() error {
	ram, cores, disk, err := c.node.GetOnboardingResources()
	if err != nil {
		return fmt.Errorf("failed to get onboarding resources: %w", err)
	}
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/onboarding/onboard -N -R %.f -C %.f -D %.f", c.Name, ram, cores, disk))
	if err != nil {
		return fmt.Errorf("failed to call onboard behavior: %w", err)
	}
	// TODO: see a better way to remove this from output....
	trimmed := strings.Replace(out, "Skipping GPU selection.", "", 1)

	var resp node.OnboardResponse
	if err = json.Unmarshal([]byte(trimmed), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to onboard: %s", resp.Error)
	}
	return nil
}

func (c *Context) Deploy(ensemble string) (string, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/new -f %s -t 2m", c.Name, ensemble))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment new behavior: %s", out)
	}
	var resp node.NewDeploymentResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to deploy: %s", resp.Error)
	}
	return resp.EnsembleID, nil
}

func (c *Context) EnsembleStatus(id string) (string, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/status --id %s", c.Name, id))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment status behavior: %s", out)
	}
	var resp node.DeploymentStatusResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to get ensemble status: %s", resp.Error)
	}
	return resp.Status, nil
}

func (c *Context) LogsFromAllocation(ensembleID, allocName string) (string, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/logs --id %s --allocation %s", c.Name, ensembleID, allocName))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment logs behavior: %s", out)
	}
	var resp node.DeploymentLogsResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to get logs from deployment: %s", resp.Error)
	}
	return resp.LogsWrittenTo, nil
}

func (c *Context) Manifest(ensembleID string) (*jobtypes.EnsembleManifest, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/manifest --id %s", c.Name, ensembleID))
	if err != nil {
		return nil, fmt.Errorf("failed to call deployment manifest behavior: %s", out)
	}
	var resp node.DeploymentManifestResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get ensemble manifest: %s", resp.Error)
	}
	return &resp.Manifest, nil
}

func (c *Context) AllocationList() ([]jobs.AllocationInfo, error) {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/allocations/list", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call allocation list manifest behavior: %s", out)
	}
	var resp node.AllocationsListResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get list of allocations: %s", resp.Error)
	}
	return resp.Allocations, nil
}

func (c *Context) UpdateEnsemble(id, path string) error {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/update -i %s -f %s -t 2m", c.Name, id, path))
	if err != nil {
		return fmt.Errorf("failed to call deployment update behavior: %w", err)
	}
	var resp node.UpdateDeploymentResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to update deployment: %s", resp.Error)
	}
	return nil
}

func (c *Context) StopEnsemble(id string) error {
	out, err := c.node.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/shutdown --id %s", c.Name, id))
	if err != nil {
		return fmt.Errorf("failed to call deployment shutdown behavior: %s", out)
	}
	var resp node.DeploymentShutdownResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to shutdown deployment: %s", resp.Error)
	}
	return nil
}

// JoinOrg allows a user to join an existing organization
func (c *Context) JoinOrg(dmsCtx *Context, orgDID, grantFromOrg string) error {
	err := c.Anchor("provide", grantFromOrg)
	if err != nil {
		return fmt.Errorf("could not anchor cap: %w", err)
	}

	grantToken, err := c.Grant(orgDID)
	if err != nil {
		return fmt.Errorf("failed to grant: %w", err)
	}

	err = dmsCtx.Anchor("require", grantToken)
	if err != nil {
		return err
	}

	delegateToken, err := c.Delegate(dmsCtx.DID)
	if err != nil {
		return fmt.Errorf("failed to delegate: %w", err)
	}

	err = dmsCtx.Anchor("provide", delegateToken)
	if err != nil {
		return err
	}

	return nil
}
