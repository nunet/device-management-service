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
	"os"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	dmsnode "gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

// TODO: Deprecate Context. It can be simply represented by a string inside Node
// Context represents a named context within a node
// TODO pass godog.TestingT
type Context struct {
	Name     string
	DID      string
	instance *Instance
}

func (c *Context) Grant(did string) (token string, err error) {
	expire := time.Now().AddDate(1, 0, 0).Format(time.DateOnly)
	return c.instance.RunDMSCmd(fmt.Sprintf("nunet cap grant --context %s --cap /dms/deployment --cap /public --cap /broadcast --cap /dms/tokenomics --topic /nunet --expiry %s %s",
		c.Name, expire, did))
}

func (c *Context) Delegate(did string) (token string, err error) {
	expire := time.Now().AddDate(1, 0, 0).Format(time.DateOnly)
	return c.instance.RunDMSCmd(fmt.Sprintf("nunet cap delegate --context %s --cap /dms/deployment --cap /public --cap /broadcast --cap /dms/tokenomics --topic /nunet --expiry %s %s",
		c.Name, expire, did))
}

func (c *Context) Anchor(kind, arg string) error {
	_, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet cap anchor --context %s --%s '%s'",
		c.Name, kind, arg))
	return err
}

func (c *Context) Revoke(token string) (revokedToken string, err error) {
	return c.instance.RunDMSCmd(fmt.Sprintf("nunet cap revoke --context %s '%s'",
		c.Name, token))
}

func (c *Context) SetConfig(key, value string) error {
	_, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet config set %s %s", key, value))
	return err
}

func (c *Context) GetConfig(key string) (string, error) {
	return c.instance.RunDMSCmd(fmt.Sprintf("nunet config get %s", key))
}

func (c *Context) Run(t godog.TestingT) error {
	frSec := os.Getenv(observability.EnvFlightrecSec)
	if frSec != "" {
		t.Log("starting DMS with " + frSec + "sec flight recorder")
	}

	// TODO use api.(InstanceExecPost).Environment
	return c.instance.RunDMSCmdBackground(fmt.Sprintf(
		// env
		"GOLOG_LOG_LEVEL=debug "+
			"%s=%s "+
			// cmd
			"nunet run -c %s > dms-logs.txt 2>&1",
		observability.EnvFlightrecSec, frSec, c.Name))
}

// Stop stops a running DMS instance by issuing SIGTERM, which should cause it to exit cleanly.
func (c *Context) Stop() error {
	_, err := c.instance.RunCMD([]string{"pkill", "-SIGINT", "-f", "nunet"})
	return err
}

func (c *Context) PeerAddr() (*dmsnode.PeerAddrInfoResponse, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/peers/self", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call self behavior: %w", err)
	}
	var resp dmsnode.PeerAddrInfoResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	return &resp, nil
}

func (c *Context) Connect(target string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/peers/connect --address %s", c.Name, target))
	if err != nil {
		return fmt.Errorf("failed to call connect behavior: %w", err)
	}
	var resp dmsnode.PeerConnectResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to connect: %s", resp.Error)
	}
	return nil
}

func (c *Context) Onboard() error {
	ram, cores, disk, err := c.instance.OnboardingResources()
	if err != nil {
		return fmt.Errorf("failed to get onboarding resources: %w", err)
	}
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/onboarding/onboard -N -R %.f -C %.f -D %.f", c.Name, ram, cores, disk))
	if err != nil {
		return fmt.Errorf("failed to call onboard behavior: %w", err)
	}
	// TODO: see a better way to remove this from output....
	trimmed := strings.Replace(out, "Skipping GPU selection.", "", 1)

	var resp dmsnode.OnboardResponse
	if err = json.Unmarshal([]byte(trimmed), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to onboard: %s", resp.Error)
	}
	return nil
}

func (c *Context) Deploy(ensemble string) (string, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet -c %s deploy -f %s -t 2m", c.Name, ensemble))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment new behavior: %s", out)
	}
	var resp dmsnode.NewDeploymentResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to deploy: %s", resp.Error)
	}
	return resp.EnsembleID, nil
}

func (c *Context) EnsembleStatus(id string) (string, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/status --id %s", c.Name, id))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment status behavior: %s", out)
	}
	var resp dmsnode.DeploymentStatusResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to get ensemble status: %s", resp.Error)
	}
	return resp.Status, nil
}

func (c *Context) LogsFromAllocation(ensembleID, allocName string) (string, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/logs --id %s --allocation %s", c.Name, ensembleID, allocName))
	if err != nil {
		return "", fmt.Errorf("failed to call deployment logs behavior: %s", out)
	}
	var resp dmsnode.DeploymentLogsResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("failed to get logs from deployment: %s", resp.Error)
	}
	return resp.LogsWrittenTo, nil
}

func (c *Context) Manifest(ensembleID string) (*jobtypes.EnsembleManifest, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/manifest --id %s", c.Name, ensembleID))
	if err != nil {
		return nil, fmt.Errorf("failed to call deployment manifest behavior: %s", out)
	}
	var resp dmsnode.DeploymentManifestResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get ensemble manifest: %s", resp.Error)
	}
	return &resp.Manifest, nil
}

func (c *Context) AllocationList() ([]jobs.AllocationInfo, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet -c %s get allocations", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call allocation list manifest behavior: %s", out)
	}
	var resp dmsnode.AllocationsListResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get list of allocations: %s", resp.Error)
	}
	return resp.Allocations, nil
}

func (c *Context) UpdateEnsemble(id, path string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/update -i %s -f %s -t 15m", c.Name, id, path))
	if err != nil {
		return fmt.Errorf("failed to call deployment update behavior: %w", err)
	}
	var resp dmsnode.UpdateDeploymentResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to update deployment: %s", resp.Error)
	}
	return nil
}

func (c *Context) StopEnsemble(id string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/shutdown --id %s", c.Name, id))
	if err != nil {
		return fmt.Errorf("failed to call deployment shutdown behavior: %s", out)
	}
	var resp dmsnode.DeploymentShutdownResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to shutdown deployment: %s", resp.Error)
	}
	return nil
}

func (c *Context) CreateContract(contractFile string) (contracts.CreateContractResponseBehaviour, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/create --contract-file %s --timeout 1m", c.Name, contractFile))
	if err != nil {
		return contracts.CreateContractResponseBehaviour{}, fmt.Errorf("failed to call contract create behavior: %w", err)
	}
	var resp contracts.CreateContractResponseBehaviour
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return contracts.CreateContractResponseBehaviour{}, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return contracts.CreateContractResponseBehaviour{}, fmt.Errorf("failed to create contract: %s", resp.Error)
	}
	return resp, nil
}

func (c *Context) ContractStatus(contractDID, hostDID string) (*contracts.Contract, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/state --contract-did %s --contract-host-did %s --timeout 1m", c.Name, contractDID, hostDID))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract state behavior: %w", err)
	}
	var resp contracts.ContractStatusResponseBehaviour
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get contract status: %s", resp.Error)
	}
	return &resp.Contract, nil
}

func (c *Context) ListContracts() ([]*contracts.Contract, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/list --timeout 1m", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract list behavior: %w", err)
	}
	var resp contracts.ContractListIncomingResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get list of contracts: %s", resp.Error)
	}
	return resp.Contracts, nil
}

func (c *Context) ListIncomingContracts() ([]*contracts.Contract, error) {
	return c.listContractsByRole(string(contracts.ContractRoleProvider))
}

func (c *Context) ListOutgoingContracts() ([]*contracts.Contract, error) {
	return c.listContractsByRole(string(contracts.ContractRoleRequestor))
}

func (c *Context) listContractsByRole(role string) ([]*contracts.Contract, error) {
	direction := "incoming"
	switch role {
	case string(contracts.ContractRoleProvider):
		direction = "incoming"
	case string(contracts.ContractRoleRequestor):
		direction = "outgoing"
	}
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet contracts --context %s list %s --timeout 5s", c.Name, direction))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract list_incoming behavior: %s", out)
	}
	var resp contracts.ContractListIncomingResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to get list of incoming/outgoing contracts: %s", resp.Error)
	}
	return resp.Contracts, nil
}

func (c *Context) ApproveContract(contractDID string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/approve_local --contract-did %s --timeout 1m", c.Name, contractDID))
	if err != nil {
		return fmt.Errorf("failed to call contract approve_local behavior: %w", err)
	}
	var resp contracts.ContractApproveLocalResponseBehaviour
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to approve contract: %s", resp.Error)
	}
	if !resp.Success {
		return fmt.Errorf("failed to approve contract")
	}
	return nil
}

func (c *Context) CompleteContract(contractDID, hostDID string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/complete --contract-did %s --contract-host-did %s --timeout 1m", c.Name, contractDID, hostDID))
	if err != nil {
		return fmt.Errorf("failed to call contract complete behavior: %w", err)
	}
	var resp contracts.ContractCompletionResponseBehaviour
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to complete contract: %s", resp.Error)
	}
	return nil
}

func (c *Context) TerminateContract(contractDID, hostDID string) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/tokenomics/contract/terminate --contract-did %s --contract-host-did %s --timeout 1m", c.Name, contractDID, hostDID))
	if err != nil {
		return fmt.Errorf("failed to call contract terminate behavior: %w", err)
	}
	var resp contracts.ContractTerminationResponseBehaviour
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to terminate contract: %s", resp.Error)
	}
	return nil
}

func (c *Context) DeploymentList() (map[string]string, error) {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/list", c.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to call deployment list behavior: %w", err)
	}
	var resp dmsnode.DeploymentListResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	return resp.Deployments, nil
}

func (c *Context) PruneDeployments() error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/prune --all", c.Name))
	if err != nil {
		return fmt.Errorf("failed to call deployment prune behavior: %w", err)
	}

	var resp dmsnode.DeploymentPruneResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("failed to prune deployments: %s", resp.Error)
	}

	return nil
}

func (c *Context) PruneDeploymentsBefore(before time.Time) error {
	out, err := c.instance.RunDMSCmd(fmt.Sprintf("nunet actor cmd -c %s /dms/node/deployment/prune --before '%s'", c.Name, before.Format(time.RFC3339)))
	if err != nil {
		return fmt.Errorf("failed to call deployment prune behavior: %w", err)
	}

	var resp dmsnode.DeploymentPruneResponse
	if err = json.Unmarshal([]byte(out), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("failed to prune deployments: %s", resp.Error)
	}

	return nil
}

func (c *Context) Hello(receiver *dmsnode.PeerAddrInfoResponse) ([]string, error) {
	// if receiver is nil, broadcast hello
	var responses []dmsnode.HelloResponse

	if receiver == nil {
		out, err := c.instance.RunDMSCmd(
			fmt.Sprintf("nunet actor cmd -c %s /broadcast/hello --timeout 5s", c.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to call /broadcast/hello behavior: %w", err)
		}
		if err = json.Unmarshal([]byte(out), &responses); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
		}
	} else {
		var resp dmsnode.HelloResponse
		out, err := c.instance.RunDMSCmd(
			fmt.Sprintf("nunet actor cmd -c %s /public/hello --dest %s --timeout 5s", c.Name, receiver.ID))
		if err != nil {
			// note: direct invocation will error when greeter has no caps
			// but currently not possible to tell if no caps or other error
			return nil, fmt.Errorf("failed to call /public/hello behavior: %s", out)
		}
		if err = json.Unmarshal([]byte(out), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
		}
		responses = append(responses, resp)
	}
	if len(responses) == 0 {
		return nil, fmt.Errorf("no hello responses received")
	}

	dids := make([]string, 0, len(responses))
	for _, r := range responses {
		dids = append(dids, r.DID.String())
	}

	return dids, nil
}
