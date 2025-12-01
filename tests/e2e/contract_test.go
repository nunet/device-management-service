// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

// DeployWithContractTest runs the tests that deploy with contracts
func DeployWithContractTest(suite *TestSuite) {
	suite.Run("dms with contracts", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// offboard this machine to not accept any bid request
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		// random addresses
		// we will setup a mock http server to response with the following addresses
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"
		// contractAmount := "1034.007244"

		feesPerAllocation := "10"

		// rpc on port
		go startMockRPC(9421)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9421/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, feesPerAllocation, string(contracts.PayPerAllocation), "")
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		// sleep until actor starts
		time.Sleep(5 * time.Second)
		// suite.Require().Equal("status", cmdOut)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// contract should not be valid at this point because its not signed
		// by all parties
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// do a deployment before approving the contract, it should fail
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// deployment should not go through, lets check after 10 seconds
		// the status should ne Preparing
		time.Sleep(10 * time.Second)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		// check the list and see the contract that is not approved yet localy
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(7 * time.Second)
		// contractStatus can be changed and no more needed to contract the address here
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		// New assertion: requester lists outgoing contracts and sees this contract
		outgoingList, err := requester.client.listOutgoingContracts(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Equal(outgoingList[0].ContractDID, contractDID, "created contracts list should contain the newly created contract")
		suite.Require().Len(outgoingList, 1, "created contracts list should contain only one contract")

		// wait 6 seconds for payment to be validated
		// before we deploy
		time.Sleep(time.Second * 6)

		// now that we accepted we will redeploy

		deploymentResult = requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID = extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment with contract did not reach Running status")

		time.Sleep(10 * time.Second)

		// contract host generates usages and sends them to payment provider
		// Optionally specify contract DID to process only this contract
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		// Verify the response structure
		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err, "failed to unmarshal usage calculation response")
		suite.Require().Empty(usageResponse.Error, "usage calculation should not have errors")
		suite.Require().NotEmpty(usageResponse.Results, "usage calculation should return results")

		// Verify the result contains the expected contract
		found := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractDID {
				found = true
				suite.Require().Equal(contracts.PayPerAllocation, result.PaymentModel, "payment model should be pay_per_allocation")
				suite.Require().Empty(result.Error, "contract usage calculation should not have errors")
				break
			}
		}
		suite.Require().True(found, "expected contract should be in results")

		time.Sleep(10 * time.Second)

		// check if transactions arrived on service provider to be paid
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		uniqueID, status, err := extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(uniqueID)
		suite.Require().Equal("unpaid", status)

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		// confirm the payment and check if status was changed
		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		time.Sleep(2 * time.Second)
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		uniqueID, status, err = extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().Equal("paid", status)
		suite.Require().NotEmpty(uniqueID)

		// check all parties can retrieve payment status from payment provider
		// requester
		statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// provider
		statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// contract host
		statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// check the status of the contract actor
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		time.Sleep(3 * time.Second)

		_, err = provider.client.settleContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("SETTLED", contractState)

		time.Sleep(3 * time.Second)

		// terminate by provider
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		time.Sleep(3 * time.Second)

		// check the status of the contract actor
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("TERMINATED", contractState)

		// validate contract
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err = extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)
	})
}

// DeployWithContractPayPerDeploymentTest runs tests that deploy with contracts using pay_per_deployment payment model
func DeployWithContractPayPerDeploymentTest(suite *TestSuite) {
	suite.Run("dms with contracts pay per deployment", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// offboard this machine to not accept any bid request
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-pay-per-deployment.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		// random addresses
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		feePerDeployment := "15"

		// rpc on port
		go startMockRPC(9422)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9422/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, "", string(contracts.PayPerDeployment), feePerDeployment)
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		// sleep until actor starts
		time.Sleep(5 * time.Second)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// contract should not be valid at this point because its not signed
		// by all parties
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Prepare ensemble file for deployments
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract-pay-per-deployment.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		// do a deployment before approving the contract, it should fail
		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)

		// deployment should not go through, lets check after 10 seconds
		// the status should be Preparing
		time.Sleep(10 * time.Second)
		manifestID := extractEnsembleID(deploymentResult)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		// check the list and see the contract that is not approved yet localy
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(7 * time.Second)
		// contractStatus can be changed and no more needed to contract the address here
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		// wait 6 seconds for payment to be validated before we deploy
		time.Sleep(time.Second * 6)

		// Perform 3 deployments
		for i := 0; i < 3; i++ {
			deploymentResult = requester.client.deploy(
				suite.T(), requester.userContext, requester.password,
				destinationFileEnsemble, "2m")
			suite.Contains(deploymentResult, `"Status": "OK"`)
			manifestID = extractEnsembleID(deploymentResult)

			// Wait until the deployment status is "Running"
			suite.Require().Eventually(func() bool {
				status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
				if err != nil {
					suite.T().Logf("Error getting deployment status: %v", err)
					return false
				}
				suite.T().Logf("Deployment %d status: %s", i+1, extractStatus(status))
				return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
			}, 60*time.Second, 5*time.Second, fmt.Sprintf("Deployment %d with contract did not reach Running status", i+1))

			// Small delay between deployments
			if i < 2 {
				time.Sleep(5 * time.Second)
			}
		}

		time.Sleep(10 * time.Second)

		// contract host generates usages and sends them to payment provider
		// Specify contract DID to process only this contract
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		// Verify the response structure
		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err, "failed to unmarshal usage calculation response")
		suite.Require().Empty(usageResponse.Error, "usage calculation should not have errors")
		suite.Require().NotEmpty(usageResponse.Results, "usage calculation should return results")

		// Verify the result contains the expected contract with 3 usages
		found := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractDID {
				found = true
				suite.Require().Equal(contracts.PayPerDeployment, result.PaymentModel, "payment model should be pay_per_deployment")
				suite.Require().Equal(4, result.Usages, "should have 3 usages for 3 deployments")
				suite.Require().Empty(result.Error, "contract usage calculation should not have errors")
				break
			}
		}
		suite.Require().True(found, "expected contract should be in results")

		time.Sleep(10 * time.Second)

		// check if transactions arrived on service provider to be paid
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		uniqueID, status, err := extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(uniqueID)
		suite.Require().Equal("unpaid", status)

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		// confirm the payment and check if status was changed
		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		time.Sleep(2 * time.Second)
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		uniqueID, status, err = extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().Equal("paid", status)
		suite.Require().NotEmpty(uniqueID)

		// check all parties can retrieve payment status from payment provider
		// requester
		statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// provider
		statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// contract host
		statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		// check the status of the contract actor
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		time.Sleep(3 * time.Second)

		_, err = provider.client.settleContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("SETTLED", contractState)

		time.Sleep(3 * time.Second)

		// terminate by provider
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		time.Sleep(3 * time.Second)

		// check the status of the contract actor
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)

		contractState, err = extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("TERMINATED", contractState)

		// validate contract
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err = extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)
	})
}

func getContractID(input string) (string, error) {
	var response contracts.CreateContractResponse
	if err := json.Unmarshal([]byte(input), &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal contract create response: %w", err)
	}
	if response.Error != "" {
		return "", fmt.Errorf("error from solution enabler: %s", response.Error)
	}
	return response.ContractDID, nil
}

func extractContractState(input string) (string, error) {
	pattern := `"current_state"\s*:\s*"([^"]+)"`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("state not found in the input string")
	}
	return match[1], nil
}

func extractValidationResponse(input string) (string, error) {
	pattern := `"valid"\s*:\s*(true|false)`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("valid not found in the input string")
	}
	return match[1], nil
}

func replacePlaceholders(filePath, seDID, providerDID, requesterDID, paymentValidatorDID, requesterAddr, providerAddr, feesPerAllocation, paymentModel, feePerDeployment string) error {
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
	updatedContent = strings.ReplaceAll(updatedContent, "{{paymentValidatorDID}}", paymentValidatorDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{requesterAddr}}", requesterAddr)
	updatedContent = strings.ReplaceAll(updatedContent, "{{providerAddr}}", providerAddr)
	updatedContent = strings.ReplaceAll(updatedContent, "{{amount}}", feesPerAllocation)
	updatedContent = strings.ReplaceAll(updatedContent, "{{payment_model}}", paymentModel)
	if feePerDeployment != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_deployment}}", feePerDeployment)
	}

	if err := os.WriteFile(filePath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}

func startMockRPC(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonPayload))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	addr := fmt.Sprintf(":%d", port)
	_ = http.ListenAndServe(addr, mux)
}

const jsonPayload = `{
  "jsonrpc": "2.0",
  "result": [
	{
		"removed": false,
		"logIndex": "0x69",
		"transactionIndex": "0x21",
		"transactionHash": "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f",
		"blockHash": "0xe37f93752f8182da7ebae9aba41795eec824cb502662e27849f54bb88022cce9",
		"blockNumber": "0xe20a59",
		"address": "0xf0d33beda4d734c72684b5f9abbebf715d0a7935",
		"data": "0x000000000000000000000000000000000000000000000000000000003da1b2cc",
		"topics": [
		"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		"0x000000000000000000000000e66b31678d6c16e9ebf358268a790b763c133750",
		"0x0000000000000000000000004741783ed607d1496f65749d2d9c94cf6c23352a"
		]
	}
	],
  "id": 1
}`

func checkHealth(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	return string(body) == "ok"
}

func extractTransactionDataRegex(input string) (string, string, error) {
	re := regexp.MustCompile(`"unique_id"\s*:\s*"([^"]+)"[\s\S]*?"status"\s*:\s*"([^"]+)"`)

	match := re.FindStringSubmatch(input)
	if match == nil {
		return "", "", fmt.Errorf("no match found")
	}

	uniqueID := match[1]
	status := match[2]

	return uniqueID, status, nil
}
