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
	"strconv"
	"strings"
	"time"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
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
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750" //nolint:goconst
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"  //nolint:goconst
		// contractAmount := "1034.007244"

		feesPerAllocation := "10"

		// rpc on port
		go startMockRPC(9421)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9421/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, feesPerAllocation, string(contracts.PayPerAllocation), "", "", "", "", "", "", "", "")
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

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f" //nolint:goconst

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

// DeployWithContractCollectAfterPayTest is identical to DeployWithContractTest
// but triggers usage collection immediately after payment to surface potential
// double-billing issues.
func DeployWithContractCollectAfterPayTest(suite *TestSuite) {
	suite.Run("dms with contracts collect after pay", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-collect-after-pay.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"
		feesPerAllocation := "10"

		go startMockRPC(9425)
		suite.Require().Eventually(func() bool {
			return checkHealth("http://localhost:9425/healthz")
		}, 10*time.Second, 500*time.Millisecond)

		err = replacePlaceholders(
			destinationFile,
			contractHost.dmsDID,
			provider.dmsDID,
			requester.dmsDID,
			paymentValidator.dmsDID,
			requesterEthAddr,
			providerEthAddr,
			feesPerAllocation,
			string(contracts.PayPerAllocation),
			"", "", "", "", "", "", "", "")
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		time.Sleep(5 * time.Second)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract-collect-after-pay.yaml")
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
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		time.Sleep(10 * time.Second)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		_, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		suite.Require().NoError(err)
		_, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		suite.Require().NoError(err)

		time.Sleep(7 * time.Second)
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		outgoingList, err := requester.client.listOutgoingContracts(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Equal(outgoingList[0].ContractDID, contractDID)
		suite.Require().Len(outgoingList, 1)

		time.Sleep(6 * time.Second)

		deploymentResult = requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID = extractEnsembleID(deploymentResult)

		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second)

		time.Sleep(10 * time.Second)

		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse.Error)
		suite.Require().NotEmpty(usageResponse.Results)

		found := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractDID {
				found = true
				suite.Require().Equal(contracts.PayPerAllocation, result.PaymentModel)
				suite.Require().Empty(result.Error)
				break
			}
		}
		suite.Require().True(found)

		time.Sleep(10 * time.Second)

		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		var initialTxList contracts.ContractListLocalTransactionsResponse
		suite.Require().NoError(json.Unmarshal([]byte(output), &initialTxList))
		initialTxCount := len(initialTxList.Transactions)

		uniqueID, statusStr, err := extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(uniqueID)
		suite.Require().Equal("unpaid", statusStr)

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		time.Sleep(2 * time.Second)

		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		var postPayTxList contracts.ContractListLocalTransactionsResponse
		suite.Require().NoError(json.Unmarshal([]byte(output), &postPayTxList))
		uniqueID, statusStr, err = extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().Equal("paid", statusStr)

		// Immediately collect usages again to surface double-billing
		calculateResp, err = contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse.Error)

		// After re-collection, ensure no new unpaid transactions were added
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		var afterRecollectTxList contracts.ContractListLocalTransactionsResponse
		suite.Require().NoError(json.Unmarshal([]byte(output), &afterRecollectTxList))
		suite.Require().LessOrEqual(len(afterRecollectTxList.Transactions), initialTxCount, "no additional transactions should be created after recollection")
		for _, tx := range afterRecollectTxList.Transactions {
			suite.Require().NotEqual("unpaid", tx.Status, "no unpaid transactions should appear after re-collection")
		}

		statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)

		statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, uniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
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

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, "", string(contracts.PayPerDeployment), feePerDeployment, "", "", "", "", "", "", "")
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

// DeployWithContractPayPerTimeUtilizationTest runs comprehensive tests for pay_per_time_utilization payment model
func DeployWithContractPayPerTimeUtilizationTest(suite *TestSuite) {
	suite.Run("dms with contracts pay per time utilization", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// offboard this machine to not accept any bid request
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-pay-per-time-utilization.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		// random addresses
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		feePerTimeUnit := "0.01" // $0.01 per second
		timeUnit := "second"

		// rpc on port
		go startMockRPC(9423)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9423/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, "", string(contracts.PayPerTimeUtilization), "", feePerTimeUnit, timeUnit, "", "", "", "", "")
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		// sleep until actor starts
		time.Sleep(5 * time.Second)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Contract should not be valid at this point because its not signed by all parties
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Prepare ensemble files for deployments using test data
		// Deployment 1: Will be shut down later (service will be stopped)
		srcFileEnsemble1 := filepath.Join(suite.testDataDir, "ensembles", "complex-deployment-1.yaml")
		destinationFileEnsemble1 := filepath.Join(requester.config.WorkDir, "complex-deployment-1.yaml")
		err = copyFile(srcFileEnsemble1, destinationFileEnsemble1)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble1, contractsContent)
		suite.Require().NoError(err)

		// Deployment 2: Will continue running after first invoice
		srcFileEnsemble2 := filepath.Join(suite.testDataDir, "ensembles", "complex-deployment-2.yaml")
		destinationFileEnsemble2 := filepath.Join(requester.config.WorkDir, "complex-deployment-2.yaml")
		err = copyFile(srcFileEnsemble2, destinationFileEnsemble2)
		suite.Require().NoError(err)
		err = replaceContractInFile(destinationFileEnsemble2, contractsContent)
		suite.Require().NoError(err)

		suite.T().Log("Deploying first deployment before approving the contract")
		// Do a deployment before approving the contract, it should fail
		deploymentResult1 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble1, "5m")
		suite.Contains(deploymentResult1, `"Status": "OK"`)

		// Deployment should not go through, check after 10 seconds
		time.Sleep(10 * time.Second)
		manifestID1 := extractEnsembleID(deploymentResult1)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID1)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(7 * time.Second)
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		// Wait 6 seconds for payment to be validated before we deploy
		time.Sleep(time.Second * 6)

		// Deploy first deployment
		suite.T().Log("Deploying second deployment (with task and service that will be stopped)")
		deploymentResult2 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble1, "5m")
		suite.Contains(deploymentResult2, `"Status": "OK"`)
		manifestID2 := extractEnsembleID(deploymentResult2)

		// Wait until first deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID2)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 2 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 did not reach Running status")

		// Wait longer to ensure all allocations (task + service) have started and StartAllocationEvent is pushed
		suite.T().Log("Waiting 10 seconds for all allocations to start and StartAllocationEvent to be pushed")
		time.Sleep(10 * time.Second)

		// Deploy second deployment
		suite.T().Log("Deploying third deployment (with task and service that will continue running)")
		deploymentResult3 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble2, "5m")
		suite.Contains(deploymentResult3, `"Status": "OK"`)
		manifestID3 := extractEnsembleID(deploymentResult3)

		// Wait until second deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID3)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 2 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 did not reach Running status")

		// Wait longer to ensure all allocations (task + service) have started and StartAllocationEvent is pushed
		suite.T().Log("Waiting 10 seconds for deployment 2 allocations to start and StartAllocationEvent to be pushed")
		time.Sleep(10 * time.Second)

		// Wait for allocations to run for 30 seconds before first invoice
		suite.T().Log("Waiting 30 seconds for allocations to run before first invoice generation")
		time.Sleep(30 * time.Second)

		// Generate first invoice (should capture both deployments)
		suite.T().Log("Generating first invoice")
		calculateResp1, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse1 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp1), &usageResponse1)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse1.Error)
		suite.Require().NotEmpty(usageResponse1.Results)
		suite.Require().Equal(contracts.PayPerTimeUtilization, usageResponse1.Results[0].PaymentModel)
		suite.Require().NotNil(usageResponse1.Results[0].TimeUtilization)
		// Account for 3 deployments: the first deployment before approval (may succeed after approval),
		// plus deployment 1 and deployment 2 after approval
		suite.Require().Equal(3, usageResponse1.Results[0].Usages, "should have 3 usages for 3 deployments")
		suite.Require().Equal(3, len(usageResponse1.Results[0].TimeUtilization.Deployments), "should have 3 deployments in time utilization")

		// Store deployment utilization details from first invoice
		extractDeploymentUtilization := func(usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse, deploymentID string) contracts.DeploymentTimeUtilization {
			for _, deployment := range usageResponse.Results[0].TimeUtilization.Deployments {
				if deployment.DeploymentID == deploymentID {
					return deployment
				}
			}
			return contracts.DeploymentTimeUtilization{}
		}
		deployment1Util1 := extractDeploymentUtilization(usageResponse1, manifestID1)
		deployment2Util1 := extractDeploymentUtilization(usageResponse1, manifestID2)
		deployment3Util1 := extractDeploymentUtilization(usageResponse1, manifestID3)

		suite.Require().Equal(2, len(deployment1Util1.Allocations), "deployment 1 should have 2 allocations")
		suite.Require().Equal(2, len(deployment2Util1.Allocations), "deployment 2 should have 2 allocations")
		suite.Require().Equal(2, len(deployment3Util1.Allocations), "deployment 3 should have 2 allocations")

		// Verify allocations have reasonable durations (at least 20 seconds)
		suite.Require().GreaterOrEqual(deployment1Util1.TotalUtilizationSec, 60.0, "deployment 1 should have at least 20 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment2Util1.TotalUtilizationSec, 50.0, "deployment 2 should have at least 20 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment3Util1.TotalUtilizationSec, 40.0, "deployment 3 should have at least 20 seconds of utilization")

		// Verify we have transactions for both deployments
		time.Sleep(10 * time.Second) // Wait for transactions to be created
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		// Extract all transaction unique IDs by marshalling json
		var transactionIDs []string
		var resp contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &resp)
		suite.Require().NoError(err)
		for _, tx := range resp.Transactions {
			transactionIDs = append(transactionIDs, tx.UniqueID)
		}
		suite.Require().GreaterOrEqual(len(transactionIDs), 3, "should have at least 3 transactions (one per deployment)")

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"
		for _, txID := range transactionIDs {
			confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, txID, txHash)
			suite.Require().NoError(err, "confirmation should not fail for transaction %s", txID)

			// Parse and check confirmation response for errors
			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", txID, confirmResp.Error)

			// Wait for transaction status to be updated
			time.Sleep(10 * time.Second)

			// Verify the specific transaction status by finding it in the list
			output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &respAfterConfirm)
			suite.Require().NoError(err)

			// Find the specific transaction we just confirmed
			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == txID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", txID)
			suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", txID)

			// check all parties can retrieve payment status from payment provider
			// requester
			statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// provider
			statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// contract host
			statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)
		}

		// Now stop the service allocation in deployment 1 (shutdown deployment 1)
		suite.T().Log("Stopping deployment 1 (will stop service allocation)")
		_ = requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID1)

		// Wait for deployment 1 to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID1)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 1 should stop")

		// Wait additional 30 seconds while deployment 2 continues running
		suite.T().Log("Waiting 30 seconds while deployment 2 continues running")
		time.Sleep(1 * time.Minute)

		// Generate second invoice (should show deployment 1 stopped, deployment 2 continued)
		suite.T().Log("Generating second invoice after deployment 1 stopped and deployment 2 and 3 continued")
		calculateResp2, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse2 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp2), &usageResponse2)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse2.Error)
		suite.Require().NotEmpty(usageResponse2.Results)

		suite.Require().NotNil(usageResponse2.Results[0].TimeUtilization)
		suite.Require().Equal(3, len(usageResponse2.Results[0].TimeUtilization.Deployments))

		deployment2Util2 := extractDeploymentUtilization(usageResponse2, manifestID2)
		deployment3Util2 := extractDeploymentUtilization(usageResponse2, manifestID3)

		suite.Require().Equal(1, len(deployment2Util2.Allocations), "deployment 2 should have at least 1 allocations (service)")
		suite.Require().Equal(1, len(deployment3Util2.Allocations), "deployment 2 should have 1 allocations (service)")

		noEndTime1 := false
		for _, allocation := range deployment2Util2.Allocations {
			if allocation.EndTime.IsZero() {
				noEndTime1 = true
			}
		}
		noEndTime2 := false
		for _, allocation := range deployment3Util2.Allocations {
			if allocation.EndTime.IsZero() {
				noEndTime2 = true
			}
		}
		suite.Require().True(noEndTime1, "deployment 2 should have at least one allocation without EndTime (running service)")
		suite.Require().True(noEndTime2, "deployment 3 should have at least one allocation without EndTime (running service)")

		// Verify we have additional transactions for the second invoice
		time.Sleep(10 * time.Second)
		output2, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var transactionIDs2 []string
		var resp2 contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output2), &resp2)
		suite.Require().NoError(err)
		for _, tx := range resp2.Transactions {
			transactionIDs2 = append(transactionIDs2, tx.UniqueID)
		}
		// With 2 deployments remaining, we should have at least 2 transactions from first invoice + more from second invoice
		// may be 3 due to the time the first deployment runs for before shutdown
		suite.Require().GreaterOrEqual(len(transactionIDs2), 2, "should have at least 3 transactions from first invoice (one per deployment)")

		// Verify transactions can be paid
		for _, txID := range transactionIDs2 {
			confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, txID, txHash)
			suite.Require().NoError(err, "confirmation should not fail for transaction %s", txID)

			// Parse and check confirmation response for errors
			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", txID, confirmResp.Error)

			// Wait for transaction status to be updated
			time.Sleep(10 * time.Second)

			// Verify the specific transaction status by finding it in the list
			output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &respAfterConfirm)
			suite.Require().NoError(err)

			// Find the specific transaction we just confirmed
			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == txID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", txID)
			suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", txID)

			// check all parties can retrieve payment status from payment provider
			// requester
			statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// provider
			statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// contract host
			statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)
		}

		// shutdown deployment 2 and 3
		suite.T().Log("Stopping deployment 2 and 3 for cleanup")
		shutdownRes2 := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID2)
		suite.Require().Contains(shutdownRes2, `"Error": ""`)
		shutdownRes3 := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID3)
		suite.Require().Contains(shutdownRes3, `"Error": ""`)

		// Wait for deployments to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID2)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 should stop")
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID3)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 3 should stop")

		// Generate second invoice (should show deployment 1 stopped, deployment 2 continued)
		suite.T().Log("Generating second invoice after deployment 1 stopped and deployment 2 and 3 continued")
		calculateResp3, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse3 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp3), &usageResponse3)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse3.Error)
		suite.Require().NotEmpty(usageResponse3.Results)
		suite.Require().NotNil(usageResponse3.Results[0].TimeUtilization)
		suite.Require().Equal(2, len(usageResponse3.Results[0].TimeUtilization.Deployments))

		deployment2Util3 := extractDeploymentUtilization(usageResponse3, manifestID2)
		deployment3Util3 := extractDeploymentUtilization(usageResponse3, manifestID3)

		suite.Require().Equal(1, len(deployment2Util3.Allocations), "deployment 2 should have 1 allocations (service)")
		suite.Require().Equal(1, len(deployment3Util3.Allocations), "deployment 3 should have 1 allocations (service)")

		hasEndTime1 := false
		for _, allocation := range deployment2Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime1 = true
			}
		}
		hasEndTime2 := false
		for _, allocation := range deployment3Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime2 = true
			}
		}
		suite.Require().True(hasEndTime1, "deployment 2 should have at least one allocation with EndTime (stopped service)")
		suite.Require().True(hasEndTime2, "deployment 3 should have at least one allocation with EndTime (stopped service)")
	})
}

func DeployWithContractPayPerResourceUtilizationTest(suite *TestSuite) {
	suite.Run("dms with contracts pay per resource utilization", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// offboard this machine to not accept any bid request
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-pay-per-resource-utilization.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		// random addresses
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		feePerCPUCorePerTimeUnit := "0.10" // $0.10 per core per hour
		feePerRAMGBPerTimeUnit := "0.05"   // $0.05 per GB per hour
		feePerDiskGBPerTimeUnit := "0.01"  // $0.01 per GB per hour
		resourceTimeUnit := "hour"

		// rpc on port
		go startMockRPC(9424)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9424/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		err = replacePlaceholders(destinationFile, contractHost.dmsDID, provider.dmsDID, requester.dmsDID, paymentValidator.dmsDID, requesterEthAddr, providerEthAddr, "", string(contracts.PayPerResourceUtilization), "", "", "", feePerCPUCorePerTimeUnit, feePerRAMGBPerTimeUnit, feePerDiskGBPerTimeUnit, "", resourceTimeUnit)
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		// sleep until actor starts
		time.Sleep(5 * time.Second)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Contract should not be valid at this point because its not signed by all parties
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Prepare ensemble files for deployments using test data
		// Deployment 1: Will be shut down later (service will be stopped)
		srcFileEnsemble1 := filepath.Join(suite.testDataDir, "ensembles", "complex-deployment-1.yaml")
		destinationFileEnsemble1 := filepath.Join(requester.config.WorkDir, "complex-deployment-1.yaml")
		err = copyFile(srcFileEnsemble1, destinationFileEnsemble1)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble1, contractsContent)
		suite.Require().NoError(err)

		// Deployment 2: Will continue running after first invoice
		srcFileEnsemble2 := filepath.Join(suite.testDataDir, "ensembles", "complex-deployment-2.yaml")
		destinationFileEnsemble2 := filepath.Join(requester.config.WorkDir, "complex-deployment-2.yaml")
		err = copyFile(srcFileEnsemble2, destinationFileEnsemble2)
		suite.Require().NoError(err)
		err = replaceContractInFile(destinationFileEnsemble2, contractsContent)
		suite.Require().NoError(err)

		suite.T().Log("Deploying first deployment before approving the contract")
		// Do a deployment before approving the contract, it should fail
		deploymentResult1 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble1, "5m")
		suite.Contains(deploymentResult1, `"Status": "OK"`)

		// Deployment should not go through, check after 10 seconds
		time.Sleep(10 * time.Second)
		manifestID1 := extractEnsembleID(deploymentResult1)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID1)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		time.Sleep(7 * time.Second)
		cmdOut, err = requester.client.contractStatus(suite.T(), requester.dmsContext, requester.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		contractState, err := extractContractState(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("ACCEPTED", contractState)

		// Wait 6 seconds for payment to be validated before we deploy
		time.Sleep(time.Second * 6)

		// Deploy first deployment
		suite.T().Log("Deploying second deployment (with task and service that will be stopped)")
		deploymentResult2 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble1, "5m")
		suite.Contains(deploymentResult2, `"Status": "OK"`)
		manifestID2 := extractEnsembleID(deploymentResult2)

		// Wait until first deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID2)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 1 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 1 did not reach Running status")

		// Wait longer to ensure all allocations (task + service) have started and StartAllocationEvent is pushed
		suite.T().Log("Waiting 10 seconds for all allocations to start and StartAllocationEvent to be pushed")
		time.Sleep(10 * time.Second)

		// Deploy second deployment
		suite.T().Log("Deploying third deployment (with task and service that will continue running)")
		deploymentResult3 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble2, "5m")
		suite.Contains(deploymentResult3, `"Status": "OK"`)
		manifestID3 := extractEnsembleID(deploymentResult3)

		// Wait until second deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID3)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 2 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 did not reach Running status")

		// Wait longer to ensure all allocations (task + service) have started and StartAllocationEvent is pushed
		suite.T().Log("Waiting 10 seconds for deployment 2 allocations to start and StartAllocationEvent to be pushed")
		time.Sleep(10 * time.Second)

		// Wait for allocations to run for 30 seconds before first invoice
		suite.T().Log("Waiting 30 seconds for allocations to run before first invoice generation")
		time.Sleep(30 * time.Second)

		// Generate first invoice (should capture both deployments)
		suite.T().Log("Generating first invoice")
		calculateResp1, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse1 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp1), &usageResponse1)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse1.Error)
		suite.Require().NotEmpty(usageResponse1.Results)
		suite.Require().Equal(contracts.PayPerResourceUtilization, usageResponse1.Results[0].PaymentModel)
		suite.Require().NotNil(usageResponse1.Results[0].ResourceUtilization)
		// Account for 3 deployments: the first deployment before approval (may succeed after approval),
		// plus deployment 1 and deployment 2 after approval
		suite.Require().Equal(3, usageResponse1.Results[0].Usages, "should have 3 usages for 3 deployments")
		suite.Require().Equal(3, len(usageResponse1.Results[0].ResourceUtilization.Deployments), "should have 3 deployments in resource utilization")

		// Store deployment resource utilization details from first invoice
		extractDeploymentResourceUtilization := func(usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse, deploymentID string) contracts.DeploymentResourceUtilization {
			for _, deployment := range usageResponse.Results[0].ResourceUtilization.Deployments {
				if deployment.DeploymentID == deploymentID {
					return deployment
				}
			}
			return contracts.DeploymentResourceUtilization{}
		}
		deployment1Util1 := extractDeploymentResourceUtilization(usageResponse1, manifestID1)
		deployment2Util1 := extractDeploymentResourceUtilization(usageResponse1, manifestID2)
		deployment3Util1 := extractDeploymentResourceUtilization(usageResponse1, manifestID3)

		suite.Require().Equal(2, len(deployment1Util1.Allocations), "deployment 1 should have 2 allocations")
		suite.Require().Equal(2, len(deployment2Util1.Allocations), "deployment 2 should have 2 allocations")
		suite.Require().Equal(2, len(deployment3Util1.Allocations), "deployment 3 should have 2 allocations")

		// Verify each deployment has allocations with resources
		for _, deployment := range usageResponse1.Results[0].ResourceUtilization.Deployments {
			suite.Require().Greater(len(deployment.Allocations), 0, "deployment should have allocations")

			// Verify each allocation has resources
			// Note: RAM.Size and Disk.Size are in bytes
			for _, alloc := range deployment.Allocations {
				suite.Require().Greater(alloc.Resources.CPU.Cores, float32(0), "allocation should have CPU cores")
				suite.Require().Greater(alloc.Resources.RAM.Size, uint64(0), "allocation should have RAM")
				suite.Require().Greater(alloc.Resources.Disk.Size, uint64(0), "allocation should have Disk")
				suite.Require().Greater(alloc.Duration.Seconds(), 0.0, "allocation should have duration")
				suite.Require().False(alloc.StartTime.IsZero(), "allocation should have start time")
			}

			suite.Require().Greater(deployment.TotalUtilizationSec, 0.9, "deployment should have total utilization")
		}

		// Verify allocations have reasonable durations (at least 20 seconds)
		suite.Require().GreaterOrEqual(deployment1Util1.TotalUtilizationSec, 60.0, "deployment 1 should have at least 60 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment2Util1.TotalUtilizationSec, 50.0, "deployment 2 should have at least 50 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment3Util1.TotalUtilizationSec, 40.0, "deployment 3 should have at least 40 seconds of utilization")

		// Verify we have transactions for both deployments
		time.Sleep(2 * 60 * time.Second) // Wait for transactions to be created
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		// Extract all transaction unique IDs by marshalling json
		var transactionIDs []string
		var resp contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &resp)
		suite.Require().NoError(err)
		for _, tx := range resp.Transactions {
			transactionIDs = append(transactionIDs, tx.UniqueID)
		}
		suite.Require().GreaterOrEqual(len(transactionIDs), 3, "should have at least 3 transactions (one per deployment)")

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"
		for _, txID := range transactionIDs {
			confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, txID, txHash)
			suite.Require().NoError(err, "confirmation should not fail for transaction %s", txID)

			// Parse and check confirmation response for errors
			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", txID, confirmResp.Error)

			// Wait for transaction status to be updated
			time.Sleep(10 * time.Second)

			// Verify the specific transaction status by finding it in the list
			output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &respAfterConfirm)
			suite.Require().NoError(err)

			// Find the specific transaction we just confirmed
			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == txID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", txID)
			suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", txID)

			// Verify transaction amounts reflect resource-based pricing
			// Amount should be calculated as: (CPU × CPU_price × time) + (RAM × RAM_price × time) + (Disk × Disk_price × time)
			suite.Require().NotEmpty(confirmedTx.Amount, "transaction should have amount")

			// Parse amount and verify it's positive
			amount, err := strconv.ParseFloat(confirmedTx.Amount, 64)
			suite.Require().NoError(err)
			suite.Require().Greater(amount, 0.0, "amount should be greater than 0")

			// check all parties can retrieve payment status from payment provider
			// requester
			statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// provider
			statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// contract host
			statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)
		}

		// Now stop the service allocation in deployment 1 (shutdown deployment 1)
		suite.T().Log("Stopping deployment 1 (will stop service allocation)")
		_ = requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID1)

		// Wait for deployment 1 to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID1)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 1 should stop")

		// Wait additional 30 seconds while deployment 2 continues running
		suite.T().Log("Waiting 60 seconds while deployment 2 continues running")
		time.Sleep(1 * time.Minute)

		// Generate second invoice (should show deployment 1 stopped, deployment 2 continued)
		suite.T().Log("Generating second invoice after deployment 1 stopped and deployment 2 and 3 continued")
		calculateResp2, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse2 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp2), &usageResponse2)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse2.Error)
		suite.Require().NotEmpty(usageResponse2.Results)

		suite.Require().NotNil(usageResponse2.Results[0].ResourceUtilization)
		suite.Require().Equal(3, len(usageResponse2.Results[0].ResourceUtilization.Deployments))

		// Find second deployment (the one that continued running)
		var secondDeploymentUtil contracts.DeploymentResourceUtilization
		var thirdDeploymentUtil contracts.DeploymentResourceUtilization
		for _, deployment := range usageResponse2.Results[0].ResourceUtilization.Deployments {
			if deployment.DeploymentID == manifestID2 {
				secondDeploymentUtil = deployment
			}
			if deployment.DeploymentID == manifestID3 {
				thirdDeploymentUtil = deployment
			}
		}

		suite.Require().NotEmpty(secondDeploymentUtil.DeploymentID, "should find second deployment")
		suite.Require().NotEmpty(thirdDeploymentUtil.DeploymentID, "should find third deployment")
		suite.Require().Greater(len(secondDeploymentUtil.Allocations), 0, "second deployment should have allocations")
		suite.Require().Greater(len(thirdDeploymentUtil.Allocations), 0, "third deployment should have allocations")
		suite.Require().Greater(secondDeploymentUtil.TotalUtilizationSec, 0.0, "second deployment should have utilization")
		suite.Require().Greater(thirdDeploymentUtil.TotalUtilizationSec, 0.0, "third deployment should have utilization")

		// Verify resources are still present
		// Note: RAM.Size and Disk.Size are in bytes
		for _, alloc := range secondDeploymentUtil.Allocations {
			suite.Require().Greater(alloc.Resources.CPU.Cores, float32(0), "allocation should have CPU cores")
			suite.Require().Greater(alloc.Resources.RAM.Size, uint64(0), "allocation should have RAM")
			suite.Require().Greater(alloc.Resources.Disk.Size, uint64(0), "allocation should have Disk")
		}
		for _, alloc := range thirdDeploymentUtil.Allocations {
			suite.Require().Greater(alloc.Resources.CPU.Cores, float32(0), "allocation should have CPU cores")
			suite.Require().Greater(alloc.Resources.RAM.Size, uint64(0), "allocation should have RAM")
			suite.Require().Greater(alloc.Resources.Disk.Size, uint64(0), "allocation should have Disk")
		}

		// Verify we have additional transactions for the second invoice
		time.Sleep(10 * time.Second)
		output2, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var transactionIDs2 []string
		var resp2 contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output2), &resp2)
		suite.Require().NoError(err)
		for _, tx := range resp2.Transactions {
			transactionIDs2 = append(transactionIDs2, tx.UniqueID)
		}
		// With 3 deployments, we should have at least 3 transactions from first invoice + more from second invoice
		suite.Require().GreaterOrEqual(len(transactionIDs2), 3, "should have at least 3 transactions from first invoice (one per deployment)")

		// Clean up: Stop deployment 2 and 3
		suite.T().Log("Stopping deployment 2 and 3 for cleanup")
		shutdownRes2 := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID2)
		suite.Require().Contains(shutdownRes2, `"Error": ""`)
		shutdownRes3 := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID3)
		suite.Require().Contains(shutdownRes3, `"Error": ""`)

		// Wait for deployments to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID2)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 should stop")
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID3)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 3 should stop")

		// Verify transactions can be paid
		for _, txID := range transactionIDs2 {
			confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, txID, txHash)
			suite.Require().NoError(err, "confirmation should not fail for transaction %s", txID)

			// Parse and check confirmation response for errors
			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", txID, confirmResp.Error)

			// Wait for transaction status to be updated
			time.Sleep(10 * time.Second)

			// Verify the specific transaction status by finding it in the list
			output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &respAfterConfirm)
			suite.Require().NoError(err)

			// Find the specific transaction we just confirmed
			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == txID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", txID)
			suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", txID)

			// check all parties can retrieve payment status from payment provider
			// requester
			statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// provider
			statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)

			// contract host
			statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, txID, paymentValidator.dmsDID)
			suite.Require().NoError(err)
			suite.Require().Contains(statusOutput, `"paid": true`)
		}

		// Generate second invoice (should show deployment 1 stopped, deployment 2 continued)
		suite.T().Log("Generating second invoice after deployment 1 stopped and deployment 2 and 3 continued")
		calculateResp3, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse3 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp3), &usageResponse3)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse3.Error)
		suite.Require().NotEmpty(usageResponse3.Results)
		suite.Require().NotNil(usageResponse3.Results[0].ResourceUtilization)
		suite.Require().Equal(2, len(usageResponse3.Results[0].ResourceUtilization.Deployments))

		deployment2Util3 := extractDeploymentResourceUtilization(usageResponse3, manifestID2)
		deployment3Util3 := extractDeploymentResourceUtilization(usageResponse3, manifestID3)

		suite.Require().Equal(1, len(deployment2Util3.Allocations), "deployment 2 should have 1 allocations (service)")
		suite.Require().Equal(1, len(deployment3Util3.Allocations), "deployment 3 should have 1 allocations (service)")

		hasEndTime1 := false
		for _, allocation := range deployment2Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime1 = true
			}
		}
		hasEndTime2 := false
		for _, allocation := range deployment3Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime2 = true
			}
		}
		suite.Require().True(hasEndTime1, "deployment 2 should have at least one allocation with EndTime (stopped service)")
		suite.Require().True(hasEndTime2, "deployment 3 should have at least one allocation with EndTime (stopped service)")
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

func replacePlaceholders(filePath, seDID, providerDID, requesterDID, paymentValidatorDID, requesterAddr, providerAddr, feesPerAllocation, paymentModel, feePerDeployment, feePerTimeUnit, timeUnit, feePerCPUCorePerTimeUnit, feePerRAMGBPerTimeUnit, feePerDiskGBPerTimeUnit, feePerGPUPerTimeUnit, resourceTimeUnit string) error { //nolint:unparam
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
	if feePerTimeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_time_unit}}", feePerTimeUnit)
	}
	if timeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{time_unit}}", timeUnit)
	}
	if feePerCPUCorePerTimeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_cpu_core_per_time_unit}}", feePerCPUCorePerTimeUnit)
	}
	if feePerRAMGBPerTimeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_ram_gb_per_time_unit}}", feePerRAMGBPerTimeUnit)
	}
	if feePerDiskGBPerTimeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_disk_gb_per_time_unit}}", feePerDiskGBPerTimeUnit)
	}
	// Always replace GPU fee placeholder (it's optional, so replace with empty string if not provided)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_gpu_per_time_unit}}", feePerGPUPerTimeUnit)
	if resourceTimeUnit != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{resource_time_unit}}", resourceTimeUnit)
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
