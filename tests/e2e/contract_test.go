// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
)

// DeployWithContractTest runs the tests that deploy with contracts
func DeployWithContractTest(suite *TestSuite) {
	suite.Run("dms with contracts", func() {
		requester := suite.nodes[0]
		solutionEnabler := suite.nodes[1]
		provider := suite.nodes[2]

		// offboard this machine to not accept any bid request
		solutionEnabler.client.offboard(suite.T(), solutionEnabler.userContext, solutionEnabler.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		err = replacePlaceholders(destinationFile, solutionEnabler.dmsDID, provider.dmsDID, requester.dmsDID)
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password, solutionEnabler.dmsDID)
		fmt.Println(cmdOut, err)

		// sleep until actor starts
		time.Sleep(5 * time.Second)
		// suite.Require().Equal("status", cmdOut)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		pubKeyActorStr, err := getPublicKey(cmdOut)
		suite.Require().NoError(err)

		pubKeyBytes, err := hex.DecodeString(pubKeyActorStr)
		suite.Require().NoError(err)

		pubKeyActor, err := crypto.BytesToPublicKey(pubKeyBytes)
		suite.Require().NoError(err)

		destinationSolutionEnabler, err := actor.HandleFromPublicKeyWithInboxAddress(pubKeyActor, contractDID, solutionEnabler.peerID)

		suite.Require().NoError(err)
		address, err := json.Marshal(destinationSolutionEnabler)
		suite.Require().NoError(err)

		// check the list and see the contract that is not approved yet localy
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(3 * time.Second)
		// check again the list of contracts to see if approved
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(7 * time.Second)
		cmdOut, err = requester.client.contractStatus(suite.T(), contractDID, requester.dmsContext, requester.password, string(address))
		suite.Require().NoError(err)

		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		// deploy
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + solutionEnabler.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"))
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment with contract did not reach Running status")
		time.Sleep(2 * time.Second)
	})
}

func getContractID(input string) (string, error) {
	pattern := `"contract_did"\s*:\s*"([^"]+)"`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("contract_did not found in the input string")
	}

	return match[1], nil
}

func extractContractState(input string) (string, error) {
	pattern := `"current_state"\s*:\s*"([^"]+)"`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("contract_did not found in the input string")
	}
	return match[1], nil
}

func getPublicKey(input string) (string, error) {
	pattern := `"pub_key"\s*:\s*"([^"]+)"`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("pub_key not found in the input string")
	}

	return match[1], nil
}

func replacePlaceholders(filePath, seDID, providerDID, requesterDID string) error {
	if filePath == "" {
		return fmt.Errorf("filePath is empty")
	}
	if seDID == "" || providerDID == "" || requesterDID == "" {
		return fmt.Errorf("one or more DIDs are empty")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	updatedContent := strings.ReplaceAll(string(content), "{{solutionEnablerDID}}", seDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{providerDID}}", providerDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{requesterDID}}", requesterDID)

	if err := os.WriteFile(filePath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}
