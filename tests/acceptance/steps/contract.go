// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

// Contract registers all step definitions for contract feature
func Contract(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, scenario *godog.Scenario, _ error) (context.Context, error) {
		scenarioName := strings.ReplaceAll(scenario.Name, " ", "_")
		if err := hooks.SaveLogs(ctx, scenarioName); err != nil {
			return ctx, err
		}
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.Step(`^the following nodes$`, theFollowingNodes)
	ctx.Step(`^"([^"]*)" requests a contract with "([^"]*)" through "([^"]*)"$`, requestsAContractWithThrough)
	ctx.Step(`^"([^"]*)" accepts the contract$`, acceptsTheContract)
	ctx.Step(`^"([^"]*)" does not accept the contract$`, doesNotAcceptTheContract)
	ctx.Step(`^a contract is created between "([^"]*)" and "([^"]*)" through "([^"]*)"$`, aContractIsCreatedBetweenAndThrough)
	ctx.Step(`^contract listings for "([^"]*)" and "([^"]*)" should reflect the contract$`, contractListingsShouldReflectContract)
	ctx.Step(`^"([^"]*)" creates an unrelated contract with "([^"]*)" through "([^"]*)"$`, createsUnrelatedContractWithThrough)
	ctx.Step(`^"([^"]*)" should not see unrelated contracts$`, shouldNotSeeUnrelatedContracts)
	ctx.Step(`^"([^"]*)" should see contracts for "([^"]*)" and "([^"]*)"$`, contractHostShouldSeeContractsFor)
	ctx.Step(`^"([^"]*)" deploys a task on "([^"]*)" with their contract$`, deploysATaskOnWithTheirContract)
	ctx.Step(`^"([^"]*)" deployment is (\w+)$`, deploymentIs)
	ctx.Step(`^"([^"]*)" deployment should (\w+)$`, deploymentIs)
	ctx.Step(`^"([^"]*)" contract should be active$`, contractShouldBeActive)
	ctx.Step(`^"([^"]*)" has an active contract with "([^"]*)" created through "([^"]*)"$`, hasAnActiveContractWithCreatedThrough)
	ctx.Step(`^"([^"]*)" marks contract as completed$`, marksContractAsCompleted)
	ctx.Step(`^"([^"]*)" requests to terminate the contract$`, requestsToTerminateTheContract)
	ctx.Step(`^"([^"]*)" should see contract as (\w+)$`, shouldSeeContractAs)
}

func requestsAContractWithThrough(ctx context.Context, spName, cpName, chName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	_, chDmsCtx := utils.NodeWithDMS(nodes, chName)
	assert.NotNil(t, chDmsCtx)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	assert.NoError(t, err)

	chInfo, err := chDmsCtx.PeerAddr()
	assert.NoError(t, err)

	chAddr, err := utils.MultiaddrFromCLI(chInfo)
	assert.NoError(t, err)

	err = spDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	err = spDmsCtx.Connect(chAddr)
	assert.NoError(t, err)

	err = chDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	contractFile := "/contracts/sample_contract.json"
	contractPath := utils.FindTestdata(contractFile)

	rawContract, err := os.ReadFile(contractPath)
	assert.NoError(t, err)

	var req contracts.CreateContractRequest
	err = json.Unmarshal(rawContract, &req)
	assert.NoError(t, err)

	req.SolutionEnablerDID.URI = chDmsCtx.DID
	req.ContractParticipants.Provider.URI = cpDmsCtx.DID
	req.ContractParticipants.Requestor.URI = spDmsCtx.DID
	req.Duration.StartDate = time.Now()
	req.Duration.EndDate = time.Now().AddDate(0, 0, 1)

	contractContent, err := json.Marshal(req)
	assert.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "contract-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(contractContent)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	remotePath := filepath.Join("/tmp", filepath.Base(tmpFile.Name()))
	err = sp.UploadFile(tmpFile.Name(), remotePath, 0o755)
	assert.NoError(t, err)

	contract, err := spDmsCtx.CreateContract(remotePath)
	assert.NoError(t, err)
	assert.NotEmpty(t, contract)

	contractData := utils.ContractData{
		DID:          contract.ContractDID,
		HostDID:      chDmsCtx.DID,
		ProviderDID:  cpDmsCtx.DID,
		RequestorDID: spDmsCtx.DID,
	}

	tc = tc.WithContract(contractData)

	return tc.Unwrap(), nil
}

func acceptsTheContract(ctx context.Context, cpName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		incomingContracts, err := cpDmsCtx.ListIncomingContracts()
		if err != nil {
			return false
		}
		for _, c := range incomingContracts {
			if c.ContractDID == contract.DID {
				return true
			}
		}
		return false
	}, 10*time.Second, 500*time.Millisecond, "contract not found in compute provider's incoming list")

	err = cpDmsCtx.ApproveContract(contract.DID)
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		incomingContracts, err := cpDmsCtx.ListIncomingContracts()
		if err != nil {
			return false
		}
		for _, c := range incomingContracts {
			if c.CurrentState == contracts.ContractAccepted {
				return true
			}
		}
		return false
	}, 10*time.Second, 500*time.Millisecond, "contract not accepted")

	return nil
}

func doesNotAcceptTheContract(ctx context.Context, cpName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	var selected *contracts.Contract
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		incomingContracts, err := cpDmsCtx.ListIncomingContracts()
		assert.NoError(c, err)

		var found bool
		for _, c := range incomingContracts {
			if c.ContractDID == contract.DID {
				found = true
				selected = c
			}
		}
		assert.True(c, found)
	}, 10*time.Second, 500*time.Millisecond, "contract not found in provider's incoming list")

	assert.NotEmpty(t, selected)
	assert.Equal(t, contracts.ContractDraft, selected.CurrentState)

	return nil
}

func aContractIsCreatedBetweenAndThrough(ctx context.Context, spName, cpName, _ string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		status, err := spDmsCtx.ContractStatus(contract.DID, contract.HostDID)
		assert.NoError(c, err)
		assert.Equal(c, contracts.ContractAccepted, status.CurrentState)
		assert.Equal(c, spDmsCtx.DID, status.ContractParticipants.Requestor.String())
		assert.Equal(c, cpDmsCtx.DID, status.ContractParticipants.Provider.String())
		assert.Len(c, status.Signatures, 2)
		sigs := make([]actor.DID, 0, len(status.Signatures))
		for _, sig := range status.Signatures {
			sigs = append(sigs, sig.DID)
		}
		assert.Contains(c, sigs, status.ContractParticipants.Requestor)
		assert.Contains(c, sigs, status.ContractParticipants.Provider)
	}, 10*time.Second, 500*time.Millisecond)
	require.Eventually(t, func() bool {
		outgoingContracts, err := spDmsCtx.ListOutgoingContracts()
		if err != nil {
			return false
		}
		for _, c := range outgoingContracts {
			if c.ContractDID == contract.DID {
				return true
			}
		}
		return false
	}, 10*time.Second, 500*time.Millisecond, "contract not found in service provider's outgoing list")

	return nil
}

func contractListingsShouldReflectContract(ctx context.Context, spName, cpName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		outgoing, err := spDmsCtx.ListOutgoingContracts()
		if err != nil {
			return false
		}
		return containsContract(outgoing, contract.DID)
	}, 10*time.Second, 500*time.Millisecond, "expected contract not present in service provider outgoing list")

	require.Eventually(t, func() bool {
		incoming, err := spDmsCtx.ListIncomingContracts()
		if err != nil {
			return false
		}
		return len(incoming) == 0
	}, 10*time.Second, 500*time.Millisecond, "service provider incoming list should not contain the contract")

	require.Eventually(t, func() bool {
		incoming, err := cpDmsCtx.ListIncomingContracts()
		if err != nil {
			return false
		}
		return containsContract(incoming, contract.DID)
	}, 10*time.Second, 500*time.Millisecond, "expected contract not present in compute provider incoming list")

	require.Eventually(t, func() bool {
		outgoing, err := cpDmsCtx.ListOutgoingContracts()
		if err != nil {
			return false
		}
		return len(outgoing) == 0
	}, 10*time.Second, 500*time.Millisecond, "compute provider outgoing list should not contain the contract")

	return nil
}

func createsUnrelatedContractWithThrough(ctx context.Context, spName, cpName, chName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	_, chDmsCtx := utils.NodeWithDMS(nodes, chName)
	assert.NotNil(t, chDmsCtx)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	assert.NoError(t, err)

	chInfo, err := chDmsCtx.PeerAddr()
	assert.NoError(t, err)

	chAddr, err := utils.MultiaddrFromCLI(chInfo)
	assert.NoError(t, err)

	err = spDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	err = spDmsCtx.Connect(chAddr)
	assert.NoError(t, err)

	err = chDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	contractFile := "/contracts/sample_contract.json"
	contractPath := utils.FindTestdata(contractFile)

	rawContract, err := os.ReadFile(contractPath)
	assert.NoError(t, err)

	var req contracts.CreateContractRequest
	err = json.Unmarshal(rawContract, &req)
	assert.NoError(t, err)

	req.SolutionEnablerDID.URI = chDmsCtx.DID
	req.ContractParticipants.Provider.URI = cpDmsCtx.DID
	req.ContractParticipants.Requestor.URI = spDmsCtx.DID
	req.Duration.StartDate = time.Now()
	req.Duration.EndDate = time.Now().AddDate(0, 0, 1)

	contractContent, err := json.Marshal(req)
	assert.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "contract-unrelated-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(contractContent)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	remotePath := filepath.Join("/tmp", filepath.Base(tmpFile.Name()))
	err = sp.UploadFile(tmpFile.Name(), remotePath, 0o755)
	assert.NoError(t, err)

	resp, err := spDmsCtx.CreateContract(remotePath)
	assert.NoError(t, err)

	tc = tc.WithExtraContract(utils.ContractData{
		DID:          resp.ContractDID,
		HostDID:      chDmsCtx.DID,
		ProviderDID:  cpDmsCtx.DID,
		RequestorDID: spDmsCtx.DID,
	})

	return tc.Unwrap(), nil
}

func shouldNotSeeUnrelatedContracts(ctx context.Context, cpName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		incoming, err := cpDmsCtx.ListIncomingContracts()
		if err != nil {
			return false
		}
		if len(incoming) != 1 {
			return false
		}
		return incoming[0].ContractDID == contract.DID
	}, 10*time.Second, 500*time.Millisecond, "compute provider incoming list contains unrelated contracts")

	require.Eventually(t, func() bool {
		outgoing, err := cpDmsCtx.ListOutgoingContracts()
		if err != nil {
			return false
		}
		return len(outgoing) == 0
	}, 10*time.Second, 500*time.Millisecond, "compute provider outgoing list should be empty")

	return nil
}

func containsContract(contractsList []*contracts.Contract, contractID string) bool {
	for _, c := range contractsList {
		if c.ContractDID == contractID {
			return true
		}
	}
	return false
}

func contractHostShouldSeeContractsFor(ctx context.Context, chName, cp1Name, cp2Name string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, chDmsCtx := utils.NodeWithDMS(nodes, chName)
	assert.NotNil(t, chDmsCtx)

	_, cp1DmsCtx := utils.NodeWithDMS(nodes, cp1Name)
	assert.NotNil(t, cp1DmsCtx)

	_, cp2DmsCtx := utils.NodeWithDMS(nodes, cp2Name)
	assert.NotNil(t, cp2DmsCtx)

	contract1, err := tc.Contract()
	assert.NoError(t, err)
	assert.NotEmpty(t, contract1)

	contract2 := tc.ExtraContracts()
	assert.NotEmpty(t, contract2)
	assert.Len(t, contract2, 1)

	allContracts := make([]*contracts.Contract, 0)
	require.Eventually(t, func() bool {
		all, err := chDmsCtx.ListContracts()
		if err != nil {
			return false
		}
		allContracts = append(allContracts, all...)
		return containsContract(all, contract1.DID) && containsContract(all, contract2[0].DID) && len(all) == 2
	}, 10*time.Second, 500*time.Millisecond, "contract host missing initial contract")

	for _, c := range allContracts {
		if c.ContractDID == contract1.DID {
			assert.Equal(t, contract1.HostDID, c.SolutionEnablerDID.String())
			assert.Equal(t, contract1.ProviderDID, c.ContractParticipants.Provider.String())
			assert.Equal(t, contract1.RequestorDID, c.ContractParticipants.Requestor.String())
		}
		if c.ContractDID == contract2[0].DID {
			assert.Equal(t, contract2[0].HostDID, c.SolutionEnablerDID.String())
			assert.Equal(t, contract2[0].ProviderDID, c.ContractParticipants.Provider.String())
			assert.Equal(t, contract2[0].RequestorDID, c.ContractParticipants.Requestor.String())
		}
	}
	return nil
}

func deploysATaskOnWithTheirContract(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	ensembleName := "docker_hello.yaml"
	ensemblePath := utils.FindTestdata(fmt.Sprintf("/ensembles/%s", ensembleName))
	remotePath := filepath.Join("/tmp", ensembleName)

	err = sp.UploadFile(ensemblePath, remotePath, 0o755)
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".contracts.contract1.did = \"%s\"", contract.DID), remotePath})
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".contracts.contract1.host = \"%s\"", contract.HostDID), remotePath})
	assert.NoError(t, err)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.peer = \"%s\"", cpInfo.ID), remotePath})
	assert.NoError(t, err)

	ensembleID, err := spDmsCtx.Deploy(remotePath)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	tc = tc.WithEnsembleID(ensembleID)
	tc = tc.WithEnsembleFile(remotePath)

	return tc.Unwrap(), nil
}

func contractShouldBeActive(ctx context.Context, spName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	spDID, err := did.FromString(spDmsCtx.DID)
	assert.NoError(t, err)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		status, err := spDmsCtx.ContractStatus(contract.DID, contract.HostDID)
		assert.NoError(c, err)
		assert.True(c, status.ContractParticipants.Requestor.Equal(spDID))
		assert.Equal(c, contracts.ContractAccepted, status.CurrentState) // active = accepted status
	}, 10*time.Second, 500*time.Millisecond)

	return nil
}

func hasAnActiveContractWithCreatedThrough(ctx context.Context, spName, cpName, chName string) (context.Context, error) {
	t := godog.T(ctx)

	var err error
	ctx, err = requestsAContractWithThrough(ctx, spName, cpName, chName)
	assert.NoError(t, err)

	err = acceptsTheContract(ctx, cpName)
	assert.NoError(t, err)

	err = contractShouldBeActive(ctx, spName)
	assert.NoError(t, err)

	return ctx, nil
}

func marksContractAsCompleted(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	err = spDmsCtx.CompleteContract(contract.DID, contract.HostDID)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func requestsToTerminateTheContract(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	err = spDmsCtx.TerminateContract(contract.DID, contract.HostDID)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func shouldSeeContractAs(ctx context.Context, cpName string, status string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	cpDID, err := did.FromString(cpDmsCtx.DID)
	assert.NoError(t, err)

	contract, err := tc.Contract()
	assert.NoError(t, err)

	var expectedState contracts.ContractState
	switch status {
	case "completed":
		expectedState = contracts.ContractCompleted
	case "terminated":
		expectedState = contracts.ContractTerminated
	default:
		return fmt.Errorf("unknown contract status: %s", status)
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		contractStatus, err := cpDmsCtx.ContractStatus(contract.DID, contract.HostDID)
		assert.NoError(c, err)
		assert.True(c, contractStatus.ContractParticipants.Provider.Equal(cpDID))
		assert.Equal(c, expectedState, contractStatus.CurrentState)
	}, 10*time.Second, 500*time.Millisecond)

	return nil
}
