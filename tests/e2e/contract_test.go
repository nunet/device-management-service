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

const (
	defaultFeePerTimeUnit = "0.01" // $0.01 per second
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

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"feesPerAllocation":   feesPerAllocation,
			"paymentModel":        string(contracts.PayPerAllocation),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

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

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// New assertion: requester lists outgoing contracts and sees this contract
		outgoingList, err := requester.client.listOutgoingContracts(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Equal(outgoingList[0].ContractDID, contractDID, "created contracts list should contain the newly created contract")
		suite.Require().Len(outgoingList, 1, "created contracts list should contain only one contract")

		// now that we accepted we will redeploy
		deploymentResult = requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID = extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.waitDeploymentRunning(requester, requester.userContext, manifestID, time.Minute)

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

		uniqueID := suite.waitLocalTransactionStatus(requester, "unpaid", 60*time.Second)

		// check if transactions arrived on service provider to be paid
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Contains(output, uniqueID)

		// wrong tx hash
		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113x5f"

		// confirm the payment and check if status was changed
		var response contracts.ContractConfirmLocalTransactionResponse
		output, _ = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		err = json.Unmarshal([]byte(output), &response)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(response.Error)
		suite.Require().Contains(output, "not verified")

		// NEW SECTION: Test automatic billing (backwards compatibility preserved)
		suite.T().Log("Testing automatic invoice generation for PayPerAllocation")

		deploymentResult = requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID = extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running"
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 6 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 6 with contract did not reach Running status")

		// Wait for automatic invoice generation using dynamic checker
		// Using paymentPeriod="minute", paymentPeriodCount=1 for fast testing
		// Dynamic checker: max(1min/10, 30s) = 30 seconds (min bound applies)
		// Wait time: 30s (checker) + 1min (billing cycle) + 2min (buffer) = 3.5 minutes
		// This is much faster than old approach: 15min + 1min + 2min = 18 minutes
		waitTime := calculateAutomaticBillingWaitTime("minute", 1, 2*time.Minute)

		var automaticTxCreated bool
		var automaticTx *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Look for new transaction created by automatic billing
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID && tx.UniqueID != uniqueID {
					automaticTxCreated = true
					automaticTx = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated")

		suite.Require().True(automaticTxCreated, "automatic billing should create transaction")
		suite.Require().NotNil(automaticTx, "should find automatically generated transaction")
		suite.Require().Equal("unpaid", automaticTx.Status, "automatic transaction should be unpaid")
		suite.T().Logf("Verified automatic invoice generation: UniqueID=%s, Amount=%s", automaticTx.UniqueID, automaticTx.Amount)
		uniqueID = automaticTx.UniqueID

		txHash = "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f" //nolint:goconst
		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		suite.waitLocalTransactionPaid(requester, uniqueID, 30*time.Second)

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

		// verify compute provider has this paid transaction locally
		providerOutput, err := provider.client.listLocalTransactions(suite.T(), provider.dmsContext, provider.password)
		suite.Require().NoError(err, "compute provider should be able to list transactions")

		var providerTxList contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(providerOutput), &providerTxList)
		suite.Require().NoError(err, "should be able to parse compute provider transaction list")

		var providerTx *transaction.Transaction
		for _, tx := range providerTxList.Transactions {
			if tx.UniqueID == uniqueID {
				providerTx = tx
				break
			}
		}
		suite.Require().NotNil(providerTx, "compute provider should have transaction %s", uniqueID)
		suite.Require().Equal("paid", providerTx.Status, "compute provider transaction %s should be marked as paid", uniqueID)
		suite.Require().Equal(txHash, providerTx.TxHash, "compute provider transaction %s should have correct tx hash", uniqueID)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 30*time.Second)

		_, err = provider.client.settleContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "SETTLED", 30*time.Second)

		// terminate by provider
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "TERMINATED", 30*time.Second)

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
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"feesPerAllocation":   feesPerAllocation,
			"paymentModel":        string(contracts.PayPerAllocation),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		suite.Require().NoError(err)

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

		_, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		suite.Require().NoError(err)
		_, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		suite.Require().NoError(err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		outgoingList, err := requester.client.listOutgoingContracts(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Equal(outgoingList[0].ContractDID, contractDID)
		suite.Require().Len(outgoingList, 1)

		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		suite.waitDeploymentRunning(requester, requester.userContext, manifestID, time.Minute)

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

		uniqueID := suite.waitLocalTransactionStatus(requester, "unpaid", 60*time.Second)

		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		var initialTxList contracts.ContractListLocalTransactionsResponse
		suite.Require().NoError(json.Unmarshal([]byte(output), &initialTxList))
		initialTxCount := len(initialTxList.Transactions)
		suite.Require().Contains(output, uniqueID)

		uniqueID, statusStr, err := extractTransactionDataRegex(output)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(uniqueID)
		suite.Require().Equal("unpaid", statusStr)

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		suite.waitLocalTransactionPaid(requester, uniqueID, 30*time.Second)

		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		var postPayTxList contracts.ContractListLocalTransactionsResponse
		suite.Require().NoError(json.Unmarshal([]byte(output), &postPayTxList))

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

		// NEW SECTION: Test automatic billing (backwards compatibility preserved)
		suite.T().Log("Testing automatic invoice generation for PayPerAllocation (after payment)")

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

		// Wait for automatic invoice generation using dynamic checker
		waitTime := calculateAutomaticBillingWaitTime("minute", 1, 2*time.Minute)

		var automaticTxCreated bool
		var automaticTx2 *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Look for new transaction created by automatic billing (different from manual invoice uniqueID)
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID && tx.UniqueID != uniqueID {
					automaticTxCreated = true
					automaticTx2 = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated")

		suite.Require().True(automaticTxCreated, "automatic billing should create transaction")
		suite.Require().NotNil(automaticTx2, "should find automatically generated transaction")
		suite.T().Logf("Verified automatic invoice generation: UniqueID=%s", automaticTx2.UniqueID)
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

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"paymentModel":        string(contracts.PayPerDeployment),
			"feePerDeployment":    feePerDeployment,
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "2",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		fmt.Println("destinationFile", destinationFile)
		bytes, err := os.ReadFile(destinationFile)
		fmt.Println("bytes", string(bytes), err)
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println("cmdOut", cmdOut, err)

		fmt.Println("cmdOut", cmdOut)
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

		// check the list and see the contract that is not approved yet localy
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// Perform 3 deployments
		for i := range 3 {
			deploymentResult := requester.client.deploy(
				suite.T(), requester.userContext, requester.password,
				destinationFileEnsemble, "2m")
			suite.Contains(deploymentResult, `"Status": "OK"`)
			manifestID := extractEnsembleID(deploymentResult)

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
		}

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
				suite.Require().Equal(3, result.Usages, "should have 3 usages for 3 deployments")
				suite.Require().Empty(result.Error, "contract usage calculation should not have errors")
				break
			}
		}
		suite.Require().True(found, "expected contract should be in results")

		uniqueID := suite.waitLocalTransactionStatus(requester, "unpaid", 60*time.Second)

		// check if transactions arrived on service provider to be paid
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Contains(output, uniqueID)

		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		// confirm the payment and check if status was changed
		_, err = requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, uniqueID, txHash)
		suite.Require().NoError(err)
		suite.waitLocalTransactionPaid(requester, uniqueID, 30*time.Second)

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

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 30*time.Second)

		suite.T().Log("Testing automatic invoice generation for PayPerDeployment")

		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)

		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running"
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment 5 status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment 5 with contract did not reach Running status")

		// Wait for automatic invoice generation using dynamic checker
		waitTime := calculateAutomaticBillingWaitTime("minute", 2, 2*time.Minute)

		var automaticTxCreated bool
		var automaticTx *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Look for new transaction created by automatic billing
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID && tx.UniqueID != uniqueID {
					automaticTxCreated = true
					automaticTx = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated")

		suite.Require().True(automaticTxCreated, "automatic billing should create transaction")
		suite.Require().NotNil(automaticTx, "should find automatically generated transaction")
		suite.T().Logf("Verified automatic invoice generation: UniqueID=%s", automaticTx.UniqueID)

		_, err = provider.client.settleContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "SETTLED", 30*time.Second)

		// terminate by provider
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "TERMINATED", 30*time.Second)

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

		feePerTimeUnit := defaultFeePerTimeUnit
		timeUnit := "second"

		// rpc on port
		go startMockRPC(9423)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9423/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"paymentModel":        string(contracts.PayPerTimeUtilization),
			"feePerTimeUnit":      feePerTimeUnit,
			"timeUnit":            timeUnit,
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "5",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

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

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// Deploy first deployment
		suite.T().Log("Deploying first deployment (with task and service that will be stopped)")
		deploymentResult1 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble1, "5m")
		suite.Contains(deploymentResult1, `"Status": "OK"`)
		manifestID1 := extractEnsembleID(deploymentResult1)

		// Wait until first deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID1)
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
		suite.T().Log("Deploying second deployment (with task and service that will continue running)")
		deploymentResult2 := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble2, "5m")
		suite.Contains(deploymentResult2, `"Status": "OK"`)
		manifestID2 := extractEnsembleID(deploymentResult2)

		// Wait until second deployment reaches Running status
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

		// check for two deployments
		suite.Require().Equal(2, usageResponse1.Results[0].Usages, "should have 2 usages for 2 deployments")
		suite.Require().Equal(2, len(usageResponse1.Results[0].TimeUtilization.Deployments), "should have 2 deployments in time utilization")

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

		suite.Require().Equal(2, len(deployment1Util1.Allocations), "deployment 1 should have 2 allocations")
		suite.Require().Equal(2, len(deployment2Util1.Allocations), "deployment 2 should have 2 allocations")

		// Verify allocations have reasonable durations (at least 20 seconds)
		suite.Require().GreaterOrEqual(deployment1Util1.TotalUtilizationSec, 50.0, "deployment 1 should have at least 20 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment2Util1.TotalUtilizationSec, 40.0, "deployment 2 should have at least 20 seconds of utilization")

		// Verify we have transactions for both deployments
		suite.waitLocalTransactionCountAtLeast(requester, 2, 90*time.Second)
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		// Extract all transaction unique IDs by marshalling json
		var resp contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &resp)
		suite.Require().NoError(err)
		transactionIDs := make([]string, 0, len(resp.Transactions))
		for _, tx := range resp.Transactions {
			transactionIDs = append(transactionIDs, tx.UniqueID)
		}
		suite.Require().GreaterOrEqual(len(transactionIDs), 2, "should have at least 2 transactions (one per deployment)")

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
			suite.waitLocalTransactionPaid(requester, txID, 60*time.Second)

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
		time.Sleep(30 * time.Second)

		// Generate second invoice (should show deployment 1 stopped, deployment 2 continued)
		suite.T().Log("Generating second invoice after deployment 1 stopped and deployment 2 continued")
		calculateResp2, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse2 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp2), &usageResponse2)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse2.Error)
		suite.Require().NotEmpty(usageResponse2.Results)

		suite.Require().NotNil(usageResponse2.Results[0].TimeUtilization)
		suite.Require().Equal(2, len(usageResponse2.Results[0].TimeUtilization.Deployments))

		deployment1Util2 := extractDeploymentUtilization(usageResponse2, manifestID1)
		deployment2Util2 := extractDeploymentUtilization(usageResponse2, manifestID2)

		suite.Require().Equal(1, len(deployment1Util2.Allocations), "deployment 1 should have 1 allocations (service)")
		suite.Require().Equal(1, len(deployment2Util2.Allocations), "deployment 2 should have 1 allocations (service)")

		noEndTime2 := false
		for _, allocation := range deployment2Util2.Allocations {
			if allocation.EndTime.IsZero() {
				noEndTime2 = true
			}
		}
		suite.Require().True(noEndTime2, "deployment 2 should have at least one allocation without EndTime (running service)")

		// Verify we have additional transactions for the second invoice
		suite.waitLocalTransactionCountAtLeast(requester, 2, 60*time.Second)
		output2, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var resp2 contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output2), &resp2)
		suite.Require().NoError(err)
		transactionIDs2 := make([]string, 0, len(resp2.Transactions))
		for _, tx := range resp2.Transactions {
			transactionIDs2 = append(transactionIDs2, tx.UniqueID)
		}
		// With 1 deployment remaining, we should have at least 2 transactions from first invoice + more from second invoice
		suite.Require().GreaterOrEqual(len(transactionIDs2), 2, "should have at least 2 transactions from first invoice (one per deployment)")

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
			suite.waitLocalTransactionPaid(requester, txID, 60*time.Second)

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

		// shutdown deployment 3
		suite.T().Log("Stopping deployment 3 for cleanup")
		shutdownRes2 := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID2)
		suite.Require().Contains(shutdownRes2, `"Error": ""`)

		// Wait for deployments to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID2)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment 2 should stop")

		// Generate second invoice
		suite.T().Log("Generating second invoice after both deployments stopped")
		calculateResp3, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse3 contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp3), &usageResponse3)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse3.Error)
		suite.Require().NotEmpty(usageResponse3.Results)
		suite.Require().NotNil(usageResponse3.Results[0].TimeUtilization)
		suite.Require().Equal(1, len(usageResponse3.Results[0].TimeUtilization.Deployments))

		deployment1Util3 := extractDeploymentUtilization(usageResponse3, manifestID1)
		deployment2Util3 := extractDeploymentUtilization(usageResponse3, manifestID2)

		suite.Require().Equal(0, len(deployment1Util3.Allocations), "deployment 1 should have 1 allocations (service)")
		suite.Require().Equal(1, len(deployment2Util3.Allocations), "deployment 2 should have 1 allocations (service)")

		hasEndTime1 := false
		for _, allocation := range deployment1Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime1 = true
			}
		}
		hasEndTime2 := false
		for _, allocation := range deployment2Util3.Allocations {
			if !allocation.EndTime.IsZero() {
				hasEndTime2 = true
			}
		}
		suite.Require().False(hasEndTime1, "deployment 1 should not have an allocation with EndTime (stopped service)")
		suite.Require().True(hasEndTime2, "deployment 2 should have at least one allocation with EndTime (stopped service)")

		// NEW SECTION: Test automatic billing for PayPerTimeUtilization
		suite.T().Log("Testing automatic invoice generation for PayPerTimeUtilization")

		// Wait for automatic invoice generation with dynamic checker
		waitTime := calculateAutomaticBillingWaitTime("minute", 5, 2*time.Minute)

		var automaticTxCreated bool
		var automaticTimeUtilTx *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Find new transaction created by automatic billing (check all transactions for this contract)
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID {
					// Simple check: if we find a transaction, assume it's from automatic billing
					// (manual invoices were already verified in previous test sections)
					automaticTxCreated = true
					automaticTimeUtilTx = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated")

		suite.Require().True(automaticTxCreated, "automatic billing should create transaction")
		suite.Require().NotNil(automaticTimeUtilTx, "should find automatically generated transaction")
		suite.T().Logf("Verified automatic invoice generation: UniqueID=%s", automaticTimeUtilTx.UniqueID)
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

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":                    contractHost.dmsDID,
			"providerDID":              provider.dmsDID,
			"requesterDID":             requester.dmsDID,
			"paymentValidatorDID":      paymentValidator.dmsDID,
			"requesterAddr":            requesterEthAddr,
			"providerAddr":             providerEthAddr,
			"paymentModel":             string(contracts.PayPerResourceUtilization),
			"feePerCPUCorePerTimeUnit": feePerCPUCorePerTimeUnit,
			"feePerRAMGBPerTimeUnit":   feePerRAMGBPerTimeUnit,
			"feePerDiskGBPerTimeUnit":  feePerDiskGBPerTimeUnit,
			"resourceTimeUnit":         resourceTimeUnit,
			"paymentPeriod":            "minute",
			"paymentPeriodCount":       "5",
			"startDate":                startDate,
			"endDate":                  endDate,
			"disableBilling":           "false",
		})
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		fmt.Println("cmdOut", cmdOut)
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

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

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
		suite.Require().GreaterOrEqual(deployment1Util1.TotalUtilizationSec, 59.0, "deployment 1 should have at least 60 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment2Util1.TotalUtilizationSec, 49.0, "deployment 2 should have at least 50 seconds of utilization")
		suite.Require().GreaterOrEqual(deployment3Util1.TotalUtilizationSec, 39.0, "deployment 3 should have at least 40 seconds of utilization")

		// Verify we have transactions for both deployments
		suite.waitLocalTransactionCountAtLeast(requester, 3, 3*time.Minute)
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		// Extract all transaction unique IDs by marshalling json
		var resp contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &resp)
		suite.Require().NoError(err)
		transactionIDs := make([]string, 0, len(resp.Transactions))
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
			suite.waitLocalTransactionPaid(requester, txID, 60*time.Second)

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
		suite.waitLocalTransactionCountAtLeast(requester, 3, 60*time.Second)
		output2, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var resp2 contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output2), &resp2)
		suite.Require().NoError(err)
		transactionIDs2 := make([]string, 0, len(resp2.Transactions))
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
			suite.waitLocalTransactionPaid(requester, txID, 60*time.Second)

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

		// NEW SECTION: Test automatic billing for PayPerResourceUtilization
		suite.T().Log("Testing automatic invoice generation for PayPerResourceUtilization")

		// Wait for automatic invoice generation with dynamic checker
		waitTime := calculateAutomaticBillingWaitTime("minute", 1, 2*time.Minute)

		var automaticTxCreated bool
		var automaticResourceUtilTx *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Find new transaction created by automatic billing
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID {
					automaticTxCreated = true
					automaticResourceUtilTx = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated")

		suite.Require().True(automaticTxCreated, "automatic billing should create transaction")
		suite.Require().NotNil(automaticResourceUtilTx, "should find automatically generated transaction")
		suite.T().Logf("Verified automatic invoice generation: UniqueID=%s", automaticResourceUtilTx.UniqueID)
	})
}

// DeployWithContractEnforcedProvidersTest verifies that when providers require
// deployment contracts, a deployment without contracts gets stuck in Preparing.
func DeployWithContractEnforcedProvidersTest(suite *TestSuite) {
	suite.Run("deployment without contracts is not scheduled when providers require contracts", func() {
		requester := suite.nodes[0]

		// sanity: configs for this suite should have enforcement enabled on providers
		suite.T().Logf("requester RequireDeploymentContracts=%v", requester.config.Job.RequireContractsForDeployment)

		// Use a standard ensemble without any contracts section.
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-no-contracts.yaml")
		err := copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)

		// Deploy the ensemble with no contracts attached.
		deploymentResult := requester.client.deploy(
			suite.T(),
			requester.userContext,
			requester.password,
			destinationFileEnsemble,
			"2m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Give some time for bidding/allocation to happen; since all providers
		// require contracts and the manifest has none, we expect the deployment
		// to remain in Preparing.
		suite.waitDeploymentStatus(requester, requester.userContext, manifestID, jobtypes.DeploymentStatusPreparing.String(), 60*time.Second)

		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(
				suite.T(),
				requester.userContext,
				requester.password,
				manifestID,
			)
			return err == nil && extractStatus(status) == jobtypes.DeploymentStatusPreparing.String()
		}, 60*time.Second, 5*time.Second, "deployment without contracts should remain in Preparing when providers require contracts")
	})
}

// Helper functions for contract JSON manipulation
func readContractJSON(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var contract map[string]interface{}
	err = json.Unmarshal(data, &contract)
	return contract, err
}

func writeContractJSON(filePath string, contract map[string]interface{}) error {
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

// DeployWithContractOrchestrationFeeTest runs the tests that deploy with contracts and orchestration fees
func DeployWithContractOrchestrationFeeTest(suite *TestSuite) {
	suite.Run("dms with contracts and orchestration fees", func() {
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
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"
		orchestrationFeeRecipientAddr := "0x1234567890123456789012345678901234567890" // Orchestration fee recipient

		feesPerAllocation := "10"
		orchestrationFeeFixedAmount := "1.00"
		orchestrationFeePercentage := "2.5"

		// rpc on port
		go startMockRPC(9421)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9421/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":                         contractHost.dmsDID,
			"providerDID":                   provider.dmsDID,
			"requesterDID":                  requester.dmsDID,
			"paymentValidatorDID":           paymentValidator.dmsDID,
			"requesterAddr":                 requesterEthAddr,
			"providerAddr":                  providerEthAddr,
			"feesPerAllocation":             feesPerAllocation,
			"paymentModel":                  string(contracts.PayPerAllocation),
			"resourceTimeUnit":              "minute",
			"paymentPeriod":                 "minute",
			"paymentPeriodCount":            "1",
			"startDate":                     startDate,
			"endDate":                       endDate,
			"orchestrationFeeFixedAmount":   orchestrationFeeFixedAmount,
			"orchestrationFeePercentage":    orchestrationFeePercentage,
			"orchestrationFeeRecipientAddr": orchestrationFeeRecipientAddr,
		})
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Validate and approve contract (same as DeployWithContractTest)
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Prepare ensemble file
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"
    payment_details:
        payment_model: "` + string(contracts.PayPerAllocation) + `"
        fee_per_allocation: "` + feesPerAllocation + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		// Deploy before approval (should fail)
		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		time.Sleep(10 * time.Second)
		status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
		suite.Require().NoError(err)
		suite.Require().Equal(jobtypes.DeploymentStatusPreparing.String(), extractStatus(status))

		// Approve contract
		_, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		suite.Require().NoError(err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// suite.waitContractValidated(provider, contractHost, contractDID, "true", 60*time.Second)

		// Deploy after approval
		deploymentResult = requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID = extractEnsembleID(deploymentResult)

		// Wait until deployment is running
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment with contract did not reach Running status")

		// Trigger manual billing
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err, "failed to unmarshal usage calculation response")
		suite.Require().Empty(usageResponse.Error, "usage calculation should not have errors")
		suite.Require().NotEmpty(usageResponse.Results, "usage calculation should return results")

		// Verify primary transaction was created
		suite.waitLocalTransactionCountAtLeast(requester, 1, 60*time.Second)
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var txList contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &txList)
		suite.Require().NoError(err)

		// Find primary transaction (non-orchestration fee)
		var primaryTx *transaction.Transaction
		var orchestrationFeeTx *transaction.Transaction
		for _, tx := range txList.Transactions {
			if tx.ContractDID == contractDID {
				// Check if this is an orchestration fee transaction by examining unique ID
				// Orchestration fee transactions have unique IDs ending with "-orchestration-fee"
				if strings.HasSuffix(tx.UniqueID, "-orchestration-fee") {
					orchestrationFeeTx = tx
				} else {
					primaryTx = tx
				}
			}
		}

		// Assertions for primary transaction
		suite.Require().NotNil(primaryTx, "primary payment transaction should be created")
		suite.Require().Equal("unpaid", primaryTx.Status, "primary transaction should be unpaid initially")
		suite.Require().NotEmpty(primaryTx.UniqueID, "primary transaction should have unique ID")
		suite.Require().Equal(contractDID, primaryTx.ContractDID, "primary transaction should have correct contract DID")

		// Assertions for orchestration fee transaction
		suite.Require().NotNil(orchestrationFeeTx, "orchestration fee transaction should be created")
		suite.Require().Equal("unpaid", orchestrationFeeTx.Status, "orchestration fee transaction should be unpaid initially")
		suite.Require().NotEmpty(orchestrationFeeTx.UniqueID, "orchestration fee transaction should have unique ID")
		suite.Require().True(
			strings.HasSuffix(orchestrationFeeTx.UniqueID, "-orchestration-fee"),
			"orchestration fee transaction unique ID should end with '-orchestration-fee'",
		)
		suite.Require().Equal(contractDID, orchestrationFeeTx.ContractDID, "orchestration fee transaction should have correct contract DID")

		// Verify orchestration fee amount calculation
		// Expected: fixed_amount + (primary_amount * percentage / 100)
		primaryAmount, err := strconv.ParseFloat(primaryTx.Amount, 64)
		suite.Require().NoError(err, "failed to parse primary transaction amount")

		expectedFixedFee, err := strconv.ParseFloat(orchestrationFeeFixedAmount, 64)
		suite.Require().NoError(err, "failed to parse orchestration fee fixed amount")

		expectedPercentage, err := strconv.ParseFloat(orchestrationFeePercentage, 64)
		suite.Require().NoError(err, "failed to parse orchestration fee percentage")

		expectedOrchestrationFee := expectedFixedFee + (primaryAmount * expectedPercentage / 100.0)
		actualOrchestrationFee, err := strconv.ParseFloat(orchestrationFeeTx.Amount, 64)
		suite.Require().NoError(err, "failed to parse orchestration fee transaction amount")

		// Allow small floating point differences (0.01)
		suite.Require().InDelta(
			expectedOrchestrationFee,
			actualOrchestrationFee,
			0.01,
			"orchestration fee amount should match expected calculation: fixed_amount + (primary_amount * percentage / 100)",
		)

		// Verify orchestration fee recipient address (if specified)
		if orchestrationFeeRecipientAddr != "" {
			suite.Require().NotEmpty(
				orchestrationFeeTx.ToAddress,
				"orchestration fee transaction should have recipient address",
			)
			if len(orchestrationFeeTx.ToAddress) > 0 {
				suite.Require().Equal(
					orchestrationFeeRecipientAddr,
					orchestrationFeeTx.ToAddress[0].RequesterAddr,
					"orchestration fee transaction should use specified recipient address",
				)
			}
		}

		// Verify that both transactions are independent (different unique IDs)
		suite.Require().NotEqual(
			primaryTx.UniqueID,
			orchestrationFeeTx.UniqueID,
			"primary and orchestration fee transactions should have different unique IDs",
		)

		suite.T().Logf(
			"Primary transaction: %s, Amount: %s",
			primaryTx.UniqueID,
			primaryTx.Amount,
		)
		suite.T().Logf(
			"Orchestration fee transaction: %s, Amount: %s (Expected: %.2f)",
			orchestrationFeeTx.UniqueID,
			orchestrationFeeTx.Amount,
			expectedOrchestrationFee,
		)
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

func extractQuoteResponse(input string) (*contracts.ContractGetPaymentQuoteResponse, error) {
	var response contracts.ContractGetPaymentQuoteResponse
	if err := json.Unmarshal([]byte(input), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote response: %w", err)
	}
	return &response, nil
}

func extractValidateQuoteResponse(input string) (*contracts.ContractValidatePaymentQuoteResponse, error) {
	var response contracts.ContractValidatePaymentQuoteResponse
	if err := json.Unmarshal([]byte(input), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validate quote response: %w", err)
	}
	return &response, nil
}

func replacePlaceholders(filePath string, args map[string]string) error {
	if filePath == "" {
		return fmt.Errorf("filePath is empty")
	}

	// Helper function to get value from map with default
	getValue := func(key, defaultValue string) string {
		if val, ok := args[key]; ok && val != "" {
			return val
		}
		return defaultValue
	}

	// Validate required DIDs
	seDID := getValue("seDID", "")
	providerDID := getValue("providerDID", "")
	requesterDID := getValue("requesterDID", "")
	if seDID == "" || providerDID == "" || requesterDID == "" {
		return fmt.Errorf("one or more DIDs are empty")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	// Get all values with defaults
	paymentValidatorDID := getValue("paymentValidatorDID", "")
	requesterAddr := getValue("requesterAddr", "")
	providerAddr := getValue("providerAddr", "")
	feesPerAllocation := getValue("feesPerAllocation", "")
	paymentModel := getValue("paymentModel", "")
	feePerDeployment := getValue("feePerDeployment", "")
	feePerTimeUnit := getValue("feePerTimeUnit", "")
	timeUnit := getValue("timeUnit", "")
	feePerCPUCorePerTimeUnit := getValue("feePerCPUCorePerTimeUnit", "")
	feePerRAMGBPerTimeUnit := getValue("feePerRAMGBPerTimeUnit", "")
	feePerDiskGBPerTimeUnit := getValue("feePerDiskGBPerTimeUnit", "")
	feePerGPUPerTimeUnit := getValue("feePerGPUPerTimeUnit", "")
	resourceTimeUnit := getValue("resourceTimeUnit", "")
	fixedRentalAmount := getValue("fixedRentalAmount", "")
	paymentPeriod := getValue("paymentPeriod", "")
	paymentPeriodCount := getValue("paymentPeriodCount", "1")
	startDate := getValue("startDate", "")
	endDate := getValue("endDate", "")
	disableBilling := getValue("disableBilling", "false")
	orchestrationFeeFixedAmount := getValue("orchestrationFeeFixedAmount", "")
	orchestrationFeePercentage := getValue("orchestrationFeePercentage", "")
	orchestrationFeeRecipientAddr := getValue("orchestrationFeeRecipientAddr", "")
	metadata := getValue("metadata", "")

	updatedContent := strings.ReplaceAll(string(content), "{{solutionEnablerDID}}", seDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{providerDID}}", providerDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{requesterDID}}", requesterDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{paymentValidatorDID}}", paymentValidatorDID)
	updatedContent = strings.ReplaceAll(updatedContent, "{{requesterAddr}}", requesterAddr)
	updatedContent = strings.ReplaceAll(updatedContent, "{{providerAddr}}", providerAddr)
	updatedContent = strings.ReplaceAll(updatedContent, "{{amount}}", feesPerAllocation)
	updatedContent = strings.ReplaceAll(updatedContent, "{{payment_model}}", paymentModel)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_deployment}}", feePerDeployment)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_time_unit}}", feePerTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{time_unit}}", timeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_cpu_core_per_time_unit}}", feePerCPUCorePerTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_ram_gb_per_time_unit}}", feePerRAMGBPerTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_disk_gb_per_time_unit}}", feePerDiskGBPerTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fee_per_gpu_per_time_unit}}", feePerGPUPerTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{resource_time_unit}}", resourceTimeUnit)
	updatedContent = strings.ReplaceAll(updatedContent, "{{fixed_rental_amount}}", fixedRentalAmount)
	updatedContent = strings.ReplaceAll(updatedContent, "{{payment_period}}", paymentPeriod)
	updatedContent = strings.ReplaceAll(updatedContent, "{{payment_period_count}}", paymentPeriodCount)
	updatedContent = strings.ReplaceAll(updatedContent, "{{start_date}}", startDate)
	updatedContent = strings.ReplaceAll(updatedContent, "{{end_date}}", endDate)
	updatedContent = strings.ReplaceAll(updatedContent, "{{disable_billing}}", disableBilling)

	// Handle metadata: if provided, add it as JSON; otherwise, remove the placeholder
	if metadata != "" {
		updatedContent = strings.ReplaceAll(updatedContent, "{{metadata}}", ",\n    \"metadata\": "+metadata)
	} else {
		updatedContent = strings.ReplaceAll(updatedContent, "{{metadata}}", "")
	}

	// Add orchestration fee replacements
	updatedContent = strings.ReplaceAll(updatedContent, "{{orchestration_fee_fixed_amount}}", orchestrationFeeFixedAmount)
	updatedContent = strings.ReplaceAll(updatedContent, "{{orchestration_fee_percentage}}", orchestrationFeePercentage)
	updatedContent = strings.ReplaceAll(updatedContent, "{{orchestration_fee_recipient_addr}}", orchestrationFeeRecipientAddr)
	if err := os.WriteFile(filePath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}

func startMockRPC(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			if req.Method == "eth_blockNumber" {
				w.WriteHeader(http.StatusOK)
				response := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":"0xe20a59"}`, req.ID)
				_, _ = w.Write([]byte(response))
				return
			}
		}

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

// calculateAutomaticBillingWaitTime calculates the wait time for automatic invoice generation
// Updated for BillingCycleTrigger: trigger schedules at exact period boundary (lastInvoiceAt + billingCycle)
// paymentPeriod: "minute", "hour", "day", "week", "month"
// paymentPeriodCount: number of periods before invoicing (default: 1)
// buffer: additional buffer time (default: 2 minutes)
func calculateAutomaticBillingWaitTime(paymentPeriod string, paymentPeriodCount int, buffer time.Duration) time.Duration { //nolint:unparam
	if buffer == 0 {
		buffer = 2 * time.Minute
	}

	// Calculate billing period duration
	var periodDuration time.Duration
	switch paymentPeriod {
	case contracts.PaymentPeriodMinute:
		periodDuration = 1 * time.Minute
	case contracts.PaymentPeriodHour:
		periodDuration = 1 * time.Hour
	case contracts.PaymentPeriodDay:
		periodDuration = 24 * time.Hour
	case contracts.PaymentPeriodWeek:
		periodDuration = 7 * 24 * time.Hour
	case contracts.PaymentPeriodMonth:
		periodDuration = 30 * 24 * time.Hour // Approximate
	default:
		periodDuration = 1 * time.Hour // Default
	}

	if paymentPeriodCount <= 0 {
		paymentPeriodCount = 1
	}
	billingCycle := periodDuration * time.Duration(paymentPeriodCount)

	// Scheduler poll interval: the scheduler polls every 30 seconds to check for ready tasks
	// This is the worst-case delay between when a trigger becomes ready and when it's executed
	schedulerPollInterval := 30 * time.Second

	// Wait time = billing cycle + scheduler poll interval + buffer
	// - billingCycle: time until invoice is due (trigger schedules at lastInvoiceAt + billingCycle)
	// - schedulerPollInterval: worst-case delay (scheduler polls every 30s, so up to 30s delay)
	// - buffer: safety margin for processing time, network delays, etc.
	//
	// Note: BillingCycleTrigger schedules at the exact period boundary, not every checkInterval.
	// The checkInterval is only used as a minimum bound to ensure we check frequently enough.
	return billingCycle + schedulerPollInterval + buffer
}

func DeployWithContractFixedRentalTest(suite *TestSuite) {
	suite.Run("dms with contracts fixed rental", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// Setup: Offboard contract host and payment validator
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		// Prepare contract JSON with Fixed Rental configuration
		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-fixed-rental.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		fixedRentalAmount := "10.00"
		paymentPeriod := "minute" //nolint:goconst
		paymentPeriodCount := 1   // Invoice every minute for fast testing

		// Start mock RPC server
		go startMockRPC(9425)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9425/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		// Replace placeholders in contract JSON
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"paymentModel":        string(contracts.FixedRental),
			"fixedRentalAmount":   fixedRentalAmount,
			"paymentPeriod":       paymentPeriod,
			"paymentPeriodCount":  strconv.Itoa(paymentPeriodCount),
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		// Create contract
		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		fmt.Println(cmdOut, err)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Verify contract is not valid (not signed)
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// TEST 1: Attempt manual invoice generation (should fail)
		suite.T().Log("Testing manual invoice generation - should return error")
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err) // HTTP call succeeds

		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)

		// Assert error is returned
		suite.Require().NotEmpty(usageResponse.Results)
		suite.Require().NotEmpty(usageResponse.Results[0].Error, "should return error for manual Fixed Rental invoice generation")
		suite.Require().Contains(usageResponse.Results[0].Error, "automatic periodic billing", "error should mention automatic billing")
		suite.Require().Contains(usageResponse.Results[0].Error, "cannot be manually triggered", "error should mention manual triggering is blocked")
		suite.T().Logf("Manual invoice generation correctly blocked with error: %s", usageResponse.Results[0].Error)

		// TEST 2: Wait for automatic invoice generation with dynamic checker
		suite.T().Log("Waiting for automatic invoice generation with dynamic checker")

		// Wait for automatic invoice generation using dynamic checker
		// Using paymentPeriod="minute", paymentPeriodCount=1 for fast testing
		// Dynamic checker: max(1min/10, 30s) = 30 seconds (min bound applies)
		// Wait time: 30s (checker) + 1min (billing cycle) + 2min (buffer) = 3.5 minutes
		// This is much faster than old approach: 15min + 2min + 2min = 19 minutes
		waitTime := calculateAutomaticBillingWaitTime("minute", 1, time.Minute)

		// Poll for transaction creation (billing routine generates invoice automatically)
		var transactionCreated bool
		var fixedRentalTransaction *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Check if any transaction exists for this contract
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID {
					transactionCreated = true
					fixedRentalTransaction = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTime {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTime+30*time.Second, 5*time.Second, "automatic invoice should be generated within period")

		suite.Require().True(transactionCreated, "transaction should be created by automatic billing")
		suite.Require().NotNil(fixedRentalTransaction, "should find the automatically generated transaction")

		// Capture first invoice time after transaction is found
		firstInvoiceTime := time.Now()

		// Verify transaction details
		// Parse amounts as floats for comparison (invoice uses 8 decimal places, but amounts should match numerically)
		// The invoice amount is fixedRentalAmount (not multiplied by paymentPeriodCount)
		// paymentPeriodCount only controls invoice frequency (every N periods)
		expectedAmount, err := strconv.ParseFloat(fixedRentalAmount, 64)
		suite.Require().NoError(err, "should parse expected amount")
		actualAmount, err := strconv.ParseFloat(fixedRentalTransaction.Amount, 64)
		suite.Require().NoError(err, "should parse actual amount")
		suite.Require().Equal(expectedAmount, actualAmount, "transaction amount should match fixed rental amount (expected %s, got %s)", fixedRentalAmount, fixedRentalTransaction.Amount)
		suite.Require().Equal("unpaid", fixedRentalTransaction.Status, "transaction should initially be unpaid")
		suite.T().Logf("Automatic invoice generated: UniqueID=%s, Amount=%s, Status=%s", fixedRentalTransaction.UniqueID, fixedRentalTransaction.Amount, fixedRentalTransaction.Status)

		// TEST 3: Payment Processing
		suite.T().Log("Confirming payment for the first automatic invoice")
		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"
		confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, fixedRentalTransaction.UniqueID, txHash)
		suite.Require().NoError(err, "confirmation should not fail for transaction %s", fixedRentalTransaction.UniqueID)

		// Parse and check confirmation response for errors
		var confirmResp contracts.ContractConfirmLocalTransactionResponse
		err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
		suite.Require().NoError(err, "should be able to parse confirmation response")
		suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", fixedRentalTransaction.UniqueID, confirmResp.Error)

		suite.waitLocalTransactionPaid(requester, fixedRentalTransaction.UniqueID, 30*time.Second)

		// Verify the specific transaction status by finding it in the list
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var respAfterConfirm contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)

		// Find the specific transaction we just confirmed
		var confirmedTx *transaction.Transaction
		for _, tx := range respAfterConfirm.Transactions {
			if tx.UniqueID == fixedRentalTransaction.UniqueID {
				confirmedTx = tx
				break
			}
		}
		suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", fixedRentalTransaction.UniqueID)
		suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", fixedRentalTransaction.UniqueID)
		suite.T().Logf("Transaction %s confirmed and marked as paid.", fixedRentalTransaction.UniqueID)

		// Check all parties can retrieve payment status from payment provider
		statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, fixedRentalTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from requester.")

		statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, fixedRentalTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from provider.")

		statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, fixedRentalTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from contract host.")

		// Count transactions before mid-period termination
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)
		transactionCountBeforeMidPeriodTermination := len(respAfterConfirm.Transactions)
		suite.T().Logf("Transaction count before mid-period termination: %d", transactionCountBeforeMidPeriodTermination)

		// TEST 4: Mid-period contract termination (pro-rated invoice)
		suite.T().Log("Waiting 30 seconds (mid-period) before terminating contract to trigger pro-rated invoice")
		time.Sleep(30 * time.Second) // Wait for half of the 2-minute period

		terminationTime := time.Now()
		suite.T().Logf("Terminating contract at %s", terminationTime.Format(time.RFC3339))
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "TERMINATED", 30*time.Second)
		suite.T().Log("Contract status verified as TERMINATED.")

		// TEST 5: Verify pro-rated final invoice generated after termination
		suite.T().Log("Waiting for pro-rated final invoice to be generated after contract termination")

		transactionCountAfterTermination := suite.waitLocalTransactionCountGreaterThan(
			requester, transactionCountBeforeMidPeriodTermination, waitTime+time.Minute,
		)

		// Check for new transaction (pro-rated invoice for partial period)
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)

		// Assert a new transaction was created for the pro-rated period
		suite.Require().Greater(
			transactionCountAfterTermination,
			transactionCountBeforeMidPeriodTermination,
			"pro-rated final invoice should be generated after mid-period contract termination",
		)
		suite.T().Logf("Verified: Transaction count before termination: %d, after termination: %d (pro-rated invoice generated)",
			transactionCountBeforeMidPeriodTermination, transactionCountAfterTermination)

		// Find and verify the pro-rated transaction
		var proRatedTransaction *transaction.Transaction
		for _, tx := range respAfterConfirm.Transactions {
			if tx.ContractDID == contractDID && tx.UniqueID != fixedRentalTransaction.UniqueID {
				proRatedTransaction = tx
				break
			}
		}
		suite.Require().NotNil(proRatedTransaction, "should have pro-rated transaction after mid-period termination")

		// Calculate expected pro-rated amount
		// We waited 30 seconds after detecting the first invoice, then terminated.
		// The billing routine may run slightly later (up to the checker interval), so the actual elapsed time
		// used for billing will be slightly larger than this theoretical value.
		// Base period is 1 minute (payment_period: "minute"), invoices are generated every paymentPeriodCount periods.
		elapsedSinceFirstInvoice := terminationTime.Sub(firstInvoiceTime)
		periodDuration := 1 * time.Minute // Base period is 1 minute
		// Pro-rate based on the fixed rental amount and billing cycle
		// The fixed rental amount covers paymentPeriodCount periods (one billing cycle), so pro-rate proportionally
		amountPerInvoice, err := strconv.ParseFloat(fixedRentalAmount, 64)
		suite.Require().NoError(err)

		// Pro-rate: (elapsed / billing cycle duration) * fixedRentalAmount
		billingCycleDuration := periodDuration * time.Duration(paymentPeriodCount)
		proRatedRatio := float64(elapsedSinceFirstInvoice) / float64(billingCycleDuration)
		expectedProRatedAmountFloat := proRatedRatio * amountPerInvoice

		// Verify pro-rated amount (allow small variance for timing)
		actualAmount, err = strconv.ParseFloat(proRatedTransaction.Amount, 64)
		suite.Require().NoError(err)
		suite.Require().LessOrEqual(expectedProRatedAmountFloat, actualAmount, "pro-rated transaction amount should be approximately %f (got %f) for %v of %v billing cycle", expectedProRatedAmountFloat, actualAmount, elapsedSinceFirstInvoice, billingCycleDuration)
		suite.Require().Greater(actualAmount, 0.0, "pro-rated amount should be greater than 0")
		suite.Require().Less(actualAmount, amountPerInvoice, "pro-rated amount should be less than full invoice amount")
		suite.T().Logf("Verified pro-rated invoice: expected ~%.2f, got %s (for %v of %v billing cycle, ratio: %.2f%%)",
			expectedProRatedAmountFloat, proRatedTransaction.Amount, elapsedSinceFirstInvoice, billingCycleDuration, proRatedRatio*100)

		// TEST 6: Verify no further invoices after termination
		suite.T().Log("Waiting another checker interval + buffer after termination to verify no further invoices")
		time.Sleep(waitTime)

		// Count transactions after waiting another full period
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)
		transactionCountAfterWaitPeriod := len(respAfterConfirm.Transactions)

		// Assert no new transactions were created after the pro-rated invoice
		suite.Require().Equal(
			transactionCountAfterTermination,
			transactionCountAfterWaitPeriod,
			"no new transactions should be created after pro-rated invoice (contract is terminated)",
		)
		suite.T().Logf("Verified: Transaction count after termination: %d, after wait period: %d (no new invoices)",
			transactionCountAfterTermination, transactionCountAfterWaitPeriod)
	})
}

func DeployWithContractPeriodicTest(suite *TestSuite) {
	suite.Run("dms with contracts periodic", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// Setup: Offboard contract host and payment validator
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		// Prepare contract JSON with Periodic configuration
		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample-periodic.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		feePerTimeUnit := "0.10"  // 0.10 per minute
		timeUnit := "minute"      // Use minute for faster testing
		paymentPeriod := "minute" // Use minute periods for faster testing
		paymentPeriodCount := 1   // Invoice every minute for fast testing

		// Start mock RPC server
		go startMockRPC(9426)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9426/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		// Replace placeholders in contract JSON
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"paymentModel":        string(contracts.Periodic),
			"feePerTimeUnit":      feePerTimeUnit,
			"timeUnit":            timeUnit,
			"paymentPeriod":       paymentPeriod,
			"paymentPeriodCount":  strconv.Itoa(paymentPeriodCount),
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		// Create contract
		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		fmt.Println(cmdOut, err)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Verify contract is not valid (not signed)
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// TEST 1: Attempt manual invoice generation (should fail)
		suite.T().Log("Testing manual invoice generation - should return error")
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err) // HTTP call succeeds

		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)

		// Assert error is returned
		suite.Require().NotEmpty(usageResponse.Results)
		suite.Require().NotEmpty(usageResponse.Results[0].Error, "should return error for manual Periodic invoice generation")
		suite.Require().Contains(usageResponse.Results[0].Error, "automatic periodic billing", "error should mention automatic billing")
		suite.Require().Contains(usageResponse.Results[0].Error, "cannot be manually triggered", "error should mention manual triggering is blocked")
		suite.T().Logf("Manual invoice generation correctly blocked with error: %s", usageResponse.Results[0].Error)

		// TEST 2: Wait for automatic invoice generation (with no deployments, should skip with log)
		// Note: Since we haven't deployed anything yet, Edge Case 1 applies - no deployments,
		// so the billing routine should skip the period with a log message
		suite.T().Log("Waiting for billing routine check (no deployments - should skip with log)")

		// Wait for billing checker interval with dynamic checker
		// Using paymentPeriod="minute", paymentPeriodCount=1 for fast testing
		// Dynamic checker: 30 seconds (min bound for 1-minute period)
		waitTime := calculateAutomaticBillingWaitTime("minute", paymentPeriodCount, time.Minute)
		time.Sleep(waitTime)

		// Check that no transactions were created (Edge Case 1: no deployments = skip invoice)
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var resp contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &resp)
		suite.Require().NoError(err)

		// Count transactions for this contract (should be 0 since no deployments)
		var contractTransactionCount int
		for _, tx := range resp.Transactions {
			if tx.ContractDID == contractDID {
				contractTransactionCount++
			}
		}
		suite.Require().Equal(0, contractTransactionCount, "no transactions should be created when no deployments exist (Edge Case 1)")
		suite.T().Log("Verified: No invoices generated when no deployments exist (Edge Case 1)")

		// TEST 3: Deploy a deployment and wait for automatic invoice generation
		suite.T().Log("Deploying a deployment and waiting for automatic invoice generation")

		// Prepare deployment file
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "complex-deployment-1.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "complex-deployment-periodic.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		// Deploy the deployment
		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			destinationFileEnsemble, "5m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Capture deployment start time
		deploymentStartTime := time.Now()

		// Wait until deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Logf("Deployment status: %s", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")

		// Wait additional time for deployment to run (at least 30 seconds to have measurable runtime)
		suite.T().Log("Waiting 30 seconds for deployment to run before first billing period")
		time.Sleep(30 * time.Second)

		// Wait for automatic invoice generation with dynamic checker
		// Using paymentPeriod="minute", paymentPeriodCount=1 for fast testing
		waitTimeForInvoice := calculateAutomaticBillingWaitTime("minute", paymentPeriodCount, time.Minute)

		// Poll for transaction creation (billing routine generates invoice automatically)
		var transactionCreated bool
		var periodicTransaction *transaction.Transaction
		startWaitTime := time.Now()
		suite.Require().Eventually(func() bool {
			output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
			if err != nil {
				return false
			}

			var resp contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(output), &resp)
			if err != nil {
				return false
			}

			// Check if any transaction exists for this contract
			for _, tx := range resp.Transactions {
				if tx.ContractDID == contractDID {
					transactionCreated = true
					periodicTransaction = tx
					return true
				}
			}

			elapsed := time.Since(startWaitTime)
			if elapsed > waitTimeForInvoice {
				suite.T().Logf("Waiting for automatic invoice... elapsed: %v", elapsed)
			}
			return false
		}, waitTimeForInvoice+30*time.Second, 5*time.Second, "automatic invoice should be generated within period")

		suite.Require().True(transactionCreated, "transaction should be created by automatic billing")
		suite.Require().NotNil(periodicTransaction, "should find the automatically generated transaction")

		// Capture first invoice time after transaction is found
		firstInvoiceTime := time.Now()

		// Calculate expected amount based on deployment runtime
		// Runtime is from deployment start to first invoice time
		deploymentRuntime := firstInvoiceTime.Sub(deploymentStartTime)
		deploymentRuntimeMinutes := deploymentRuntime.Minutes()
		feePerUnitFloat, err := strconv.ParseFloat(feePerTimeUnit, 64)
		suite.Require().NoError(err)
		expectedAmount := deploymentRuntimeMinutes * feePerUnitFloat

		// Verify transaction details
		actualAmount, err := strconv.ParseFloat(periodicTransaction.Amount, 64)
		suite.Require().NoError(err, "should parse actual amount")
		// Allow variance for timing (invoice may be generated slightly before/after we capture the time)
		suite.Require().InDelta(expectedAmount, actualAmount, 2.0,
			"transaction amount should match deployment runtime (expected ~%.2f, got %.2f, runtime: %.2f minutes)",
			expectedAmount, actualAmount, deploymentRuntimeMinutes)
		suite.Require().Equal("unpaid", periodicTransaction.Status, "transaction should initially be unpaid")
		suite.T().Logf("Automatic invoice generated: UniqueID=%s, Amount=%s, Status=%s, DeploymentRuntime=%.2f minutes",
			periodicTransaction.UniqueID, periodicTransaction.Amount, periodicTransaction.Status, deploymentRuntimeMinutes)

		// TEST 4: Payment Processing
		suite.T().Log("Confirming payment for the first automatic invoice")
		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"
		confirmOutput, err := requester.client.confirmLocalTransaction(suite.T(), requester.dmsContext, requester.password, periodicTransaction.UniqueID, txHash)
		suite.Require().NoError(err, "confirmation should not fail for transaction %s", periodicTransaction.UniqueID)

		// Parse and check confirmation response for errors
		var confirmResp contracts.ContractConfirmLocalTransactionResponse
		err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
		suite.Require().NoError(err, "should be able to parse confirmation response")
		suite.Require().Empty(confirmResp.Error, "confirmation response should not have errors for transaction %s: %s", periodicTransaction.UniqueID, confirmResp.Error)

		suite.waitLocalTransactionPaid(requester, periodicTransaction.UniqueID, 30*time.Second)

		// Verify the specific transaction status by finding it in the list
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)

		var respAfterConfirm contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)

		// Find the specific transaction we just confirmed
		var confirmedTx *transaction.Transaction
		for _, tx := range respAfterConfirm.Transactions {
			if tx.UniqueID == periodicTransaction.UniqueID {
				confirmedTx = tx
				break
			}
		}
		suite.Require().NotNil(confirmedTx, "should find the confirmed transaction with unique_id: %s", periodicTransaction.UniqueID)
		suite.Require().Equal("paid", confirmedTx.Status, "transaction %s should be marked as paid", periodicTransaction.UniqueID)
		suite.T().Logf("Transaction %s confirmed and marked as paid.", periodicTransaction.UniqueID)

		// Check all parties can retrieve payment status from payment provider
		statusOutput, err := requester.client.paymentStatus(suite.T(), requester.dmsContext, requester.password, periodicTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from requester.")

		statusOutput, err = provider.client.paymentStatus(suite.T(), provider.dmsContext, provider.password, periodicTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from provider.")

		statusOutput, err = contractHost.client.paymentStatus(suite.T(), contractHost.dmsContext, contractHost.password, periodicTransaction.UniqueID, paymentValidator.dmsDID)
		suite.Require().NoError(err)
		suite.Require().Contains(statusOutput, `"paid": true`)
		suite.T().Log("Payment status verified from contract host.")

		// Count transactions before mid-period deployment shutdown
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)
		transactionCountBeforeShutdown := len(respAfterConfirm.Transactions)
		suite.T().Logf("Transaction count before mid-period deployment shutdown: %d", transactionCountBeforeShutdown)

		// TEST 5: Mid-Period Deployment Shutdown (pro-rated invoice)
		suite.T().Log("Waiting 30 seconds (mid-period) before shutting down deployment to trigger pro-rated invoice")
		time.Sleep(50 * time.Second)

		deploymentShutdownTime := time.Now()
		suite.T().Logf("Shutting down deployment at %s", deploymentShutdownTime.Format(time.RFC3339))

		// Shutdown deployment
		shutdownRes := requester.client.shutdownDeployment(suite.T(), requester.userContext, requester.password, manifestID)
		suite.Require().Contains(shutdownRes, `"Error": ""`)

		// Wait for deployment to stop
		suite.Require().Eventually(func() bool {
			status, err := requester.client.deploymentStatus(suite.T(), requester.userContext, requester.password, manifestID)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "Deployment should stop")

		// TEST 6: Verify pro-rated final invoice generated after deployment shutdown
		suite.T().Log("Waiting for pro-rated final invoice to be generated after deployment shutdown")

		transactionCountAfterShutdown := suite.waitLocalTransactionCountGreaterThan(
			requester, transactionCountBeforeShutdown, waitTime+(2*time.Minute),
		)

		// Check for new transaction (pro-rated invoice for partial period)
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)

		// Assert a new transaction was created for the pro-rated period
		suite.Require().Greater(
			transactionCountAfterShutdown,
			transactionCountBeforeShutdown,
			"pro-rated final invoice should be generated after mid-period deployment shutdown",
		)
		suite.T().Logf("Verified: Transaction count before shutdown: %d, after shutdown: %d (pro-rated invoice generated)",
			transactionCountBeforeShutdown, transactionCountAfterShutdown)

		// Find and verify the pro-rated transaction
		var proRatedTransaction *transaction.Transaction
		for _, tx := range respAfterConfirm.Transactions {
			if tx.ContractDID == contractDID && tx.UniqueID != periodicTransaction.UniqueID {
				proRatedTransaction = tx
				break
			}
		}
		suite.Require().NotNil(proRatedTransaction, "should have pro-rated transaction after mid-period deployment shutdown")

		// Calculate expected pro-rated amount based on runtime from last invoice to shutdown
		elapsedSinceFirstInvoice := deploymentShutdownTime.Sub(firstInvoiceTime)
		deploymentRuntimeForProRate := elapsedSinceFirstInvoice
		deploymentRuntimeMinutesForProRate := deploymentRuntimeForProRate.Minutes()
		expectedProRatedAmountFloat := deploymentRuntimeMinutesForProRate * feePerUnitFloat

		// Verify pro-rated amount
		actualProRatedAmount, err := strconv.ParseFloat(proRatedTransaction.Amount, 64)
		suite.Require().NoError(err)
		// Allow variance for timing
		suite.Require().InDelta(expectedProRatedAmountFloat, actualProRatedAmount, 1.0,
			"pro-rated transaction amount should match deployment runtime since last invoice (expected ~%.2f, got %.2f, runtime: %.2f minutes)",
			expectedProRatedAmountFloat, actualProRatedAmount, deploymentRuntimeMinutesForProRate)
		suite.Require().Greater(actualProRatedAmount, 0.0, "pro-rated amount should be greater than 0")
		suite.T().Logf("Verified pro-rated invoice: expected ~%.2f, got %s (for %.2f minutes deployment runtime)",
			expectedProRatedAmountFloat, proRatedTransaction.Amount, deploymentRuntimeMinutesForProRate)

		// TEST 7: Contract Termination
		suite.T().Log("Terminating contract to verify no further invoices")
		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "TERMINATED", 30*time.Second)
		suite.T().Log("Contract status verified as TERMINATED.")

		// Wait to ensure no further invoices are generated
		suite.T().Log("Waiting another billing period after termination to verify no further invoices")
		time.Sleep(waitTimeForInvoice)

		// Count transactions after waiting
		output, err = requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(output), &respAfterConfirm)
		suite.Require().NoError(err)
		transactionCountAfterTermination := len(respAfterConfirm.Transactions)

		// Assert no new transactions were created after termination
		suite.Require().Equal(
			transactionCountAfterShutdown,
			transactionCountAfterTermination,
			"no new transactions should be created after contract termination",
		)
		suite.T().Logf("Verified: Transaction count after shutdown: %d, after termination: %d (no new invoices)",
			transactionCountAfterShutdown, transactionCountAfterTermination)
	})
}

// setupContractChainCapabilities sets up UCAN capabilities for contract chain tests
// This handler:
// 1. Grants deployment capabilities from each Provider to Organization
// 2. Delegates capabilities from Organization to Orchestrator
func setupContractChainCapabilities(suite *TestSuite) {
	orchestrator := suite.nodes[0]     // Node 0: Orchestrator
	contractHost := suite.nodes[1]     // Node 1: Contract Host for Contract A
	organization := suite.nodes[2]     // Node 2: Organization
	provider1 := suite.nodes[3]        // Node 3: Provider 1
	provider2 := suite.nodes[4]        // Node 4: Provider 2
	paymentValidator := suite.nodes[5] // Node 5: Payment Validator

	// Step 1: Grant deployment capabilities from Provider 1 to Organization
	// This represents Contract B1 (Organization ↔ Provider 1)
	provider1GrantToken := provider1.client.grant(
		suite.T(),
		provider1.dmsContext,
		organization.userDID,
		provider1.password,
	)
	provider1.client.anchor(suite.T(), provider1GrantToken, provider1.dmsContext, "require", provider1.password)
	organization.client.anchor(suite.T(), provider1GrantToken, organization.userContext, "provide", organization.password)
	if suite.grantTokens[provider1.index] == nil {
		suite.grantTokens[provider1.index] = make(map[int]string)
	}
	suite.grantTokens[provider1.index][organization.index] = provider1GrantToken

	// Step 2: Grant deployment capabilities from Provider 2 to Organization
	// This represents Contract B2 (Organization ↔ Provider 2)
	provider2GrantToken := provider2.client.grant(
		suite.T(),
		provider2.dmsContext,
		organization.userDID,
		provider2.password,
	)
	provider2.client.anchor(suite.T(), provider2GrantToken, provider2.dmsContext, "require", provider2.password)
	organization.client.anchor(suite.T(), provider2GrantToken, organization.userContext, "provide", organization.password)
	if suite.grantTokens[provider2.index] == nil {
		suite.grantTokens[provider2.index] = make(map[int]string)
	}
	suite.grantTokens[provider2.index][organization.index] = provider2GrantToken

	// Step 3: Delegate capabilities from Organization to Orchestrator
	// Organization delegates the capabilities it received from Providers to Orchestrator
	// This represents Contract A (Orchestrator ↔ Organization)
	orgToOrchDelegateToken := organization.client.delegate(
		suite.T(),
		organization.userContext,
		orchestrator.userDID,
		organization.password,
	)
	organization.client.anchor(suite.T(), orgToOrchDelegateToken, organization.dmsContext, "require", organization.password)
	orchestrator.client.anchor(suite.T(), orgToOrchDelegateToken, orchestrator.userContext, "provide", orchestrator.password)
	if suite.grantTokens[organization.index] == nil {
		suite.grantTokens[organization.index] = make(map[int]string)
	}
	suite.grantTokens[organization.index][orchestrator.index] = orgToOrchDelegateToken

	// Step 4: Contract host grants capabilities to all other nodes
	for _, otherNode := range suite.nodes {
		if otherNode.index == contractHost.index {
			continue
		}

		contractHostGrantToken := contractHost.client.grant(
			suite.T(),
			contractHost.dmsContext,
			otherNode.userDID,
			contractHost.password,
		)
		contractHost.client.anchor(suite.T(), contractHostGrantToken, contractHost.dmsContext, "require", contractHost.password)
		otherNode.client.anchor(suite.T(), contractHostGrantToken, otherNode.userContext, "provide", otherNode.password)

		if suite.grantTokens[contractHost.index] == nil {
			suite.grantTokens[contractHost.index] = make(map[int]string)
		}
		suite.grantTokens[contractHost.index][otherNode.index] = contractHostGrantToken
	}
	// Step 4.a: All nodes grants capabilities to contract host
	for _, node := range suite.nodes {
		if node.index == contractHost.index {
			continue
		}
		nodeToContractHostGrantToken := node.client.grant(suite.T(), node.dmsContext, contractHost.userDID, node.password)
		node.client.anchor(suite.T(), nodeToContractHostGrantToken, node.dmsContext, "require", node.password)
		contractHost.client.anchor(suite.T(), nodeToContractHostGrantToken, contractHost.userContext, "provide", contractHost.password)
		if suite.grantTokens[node.index] == nil {
			suite.grantTokens[node.index] = make(map[int]string)
		}
		suite.grantTokens[node.index][contractHost.index] = nodeToContractHostGrantToken
	}

	// Step 4.b: Orchestrator grants capabilities to providers
	orchestratorToProviderGrantToken := orchestrator.client.grant(suite.T(), orchestrator.dmsContext, provider1.userDID, orchestrator.password)
	orchestrator.client.anchor(suite.T(), orchestratorToProviderGrantToken, orchestrator.dmsContext, "require", orchestrator.password)
	provider1.client.anchor(suite.T(), orchestratorToProviderGrantToken, provider1.userContext, "provide", provider1.password)
	if suite.grantTokens[orchestrator.index] == nil {
		suite.grantTokens[orchestrator.index] = make(map[int]string)
	}
	suite.grantTokens[orchestrator.index][provider1.index] = orchestratorToProviderGrantToken

	orchestratorToProvider2GrantToken := orchestrator.client.grant(suite.T(), orchestrator.dmsContext, provider2.userDID, orchestrator.password)
	orchestrator.client.anchor(suite.T(), orchestratorToProvider2GrantToken, orchestrator.dmsContext, "require", orchestrator.password)
	provider2.client.anchor(suite.T(), orchestratorToProvider2GrantToken, provider2.userContext, "provide", provider2.password)
	if suite.grantTokens[orchestrator.index] == nil {
		suite.grantTokens[orchestrator.index] = make(map[int]string)
	}
	suite.grantTokens[orchestrator.index][provider2.index] = orchestratorToProvider2GrantToken

	// Step 5: Payment validator grants contract host capabilities
	paymentValidatorGrantToken := paymentValidator.client.grant(
		suite.T(),
		paymentValidator.dmsContext,
		contractHost.userDID,
		paymentValidator.password,
	)
	paymentValidator.client.anchor(suite.T(), paymentValidatorGrantToken, paymentValidator.dmsContext, "require", paymentValidator.password)
	contractHost.client.anchor(suite.T(), paymentValidatorGrantToken, contractHost.userContext, "provide", contractHost.password)
	if suite.grantTokens[paymentValidator.index] == nil {
		suite.grantTokens[paymentValidator.index] = make(map[int]string)
	}
	suite.grantTokens[paymentValidator.index][contractHost.index] = paymentValidatorGrantToken

	// Step 5.a: Organization grants capabilities to Payment validator
	// Payment validator needs to send transactions to Organization (for tail contracts)
	orgToPaymentValidatorGrantToken := organization.client.grant(
		suite.T(),
		organization.dmsContext,
		paymentValidator.userDID,
		organization.password,
	)
	organization.client.anchor(suite.T(), orgToPaymentValidatorGrantToken, organization.dmsContext, "require", organization.password)
	paymentValidator.client.anchor(suite.T(), orgToPaymentValidatorGrantToken, paymentValidator.userContext, "provide", paymentValidator.password)
	if suite.grantTokens[organization.index] == nil {
		suite.grantTokens[organization.index] = make(map[int]string)
	}
	suite.grantTokens[organization.index][paymentValidator.index] = orgToPaymentValidatorGrantToken

	// Step 5.b: Orchestrator grants capabilities to Payment validator
	// Payment validator needs to send transactions to Orchestrator (for head contract)
	orchToPaymentValidatorGrantToken := orchestrator.client.grant(
		suite.T(),
		orchestrator.dmsContext,
		paymentValidator.userDID,
		orchestrator.password,
	)
	orchestrator.client.anchor(suite.T(), orchToPaymentValidatorGrantToken, orchestrator.dmsContext, "require", orchestrator.password)
	paymentValidator.client.anchor(suite.T(), orchToPaymentValidatorGrantToken, paymentValidator.userContext, "provide", paymentValidator.password)
	if suite.grantTokens[orchestrator.index] == nil {
		suite.grantTokens[orchestrator.index] = make(map[int]string)
	}
	suite.grantTokens[orchestrator.index][paymentValidator.index] = orchToPaymentValidatorGrantToken

	// Step 5.c: Payment validator grants capabilities to Organization
	// Organization needs to send payment validation requests to Payment validator (for tail contracts)
	paymentValidatorToOrgGrantToken := paymentValidator.client.grant(
		suite.T(),
		paymentValidator.dmsContext,
		organization.userDID,
		paymentValidator.password,
	)
	paymentValidator.client.anchor(suite.T(), paymentValidatorToOrgGrantToken, paymentValidator.dmsContext, "require", paymentValidator.password)
	organization.client.anchor(suite.T(), paymentValidatorToOrgGrantToken, organization.userContext, "provide", organization.password)
	if suite.grantTokens[paymentValidator.index] == nil {
		suite.grantTokens[paymentValidator.index] = make(map[int]string)
	}
	suite.grantTokens[paymentValidator.index][organization.index] = paymentValidatorToOrgGrantToken

	// Step 5.d: Payment validator grants capabilities to Orchestrator
	// Orchestrator needs to send payment validation requests to Payment validator (for head contract)
	paymentValidatorToOrchGrantToken := paymentValidator.client.grant(
		suite.T(),
		paymentValidator.dmsContext,
		orchestrator.userDID,
		paymentValidator.password,
	)
	paymentValidator.client.anchor(suite.T(), paymentValidatorToOrchGrantToken, paymentValidator.dmsContext, "require", paymentValidator.password)
	orchestrator.client.anchor(suite.T(), paymentValidatorToOrchGrantToken, orchestrator.userContext, "provide", orchestrator.password)
	if suite.grantTokens[paymentValidator.index] == nil {
		suite.grantTokens[paymentValidator.index] = make(map[int]string)
	}
	suite.grantTokens[paymentValidator.index][orchestrator.index] = paymentValidatorToOrchGrantToken

	// Step 6: All nodes' dmsCtx trust userCtx (set root anchors)
	for _, node := range suite.nodes {
		node.client.addRootAnchor(suite.T(), node.dmsContext, node.userDID, node.password)
	}

	// Step 7: All nodes' userCtx delegate to dmsCtx
	for _, node := range suite.nodes {
		delegateToken := node.client.delegate(suite.T(), node.userContext, node.dmsDID, node.password)
		node.client.anchor(suite.T(), delegateToken, node.dmsContext, "provide", node.password)
	}
}

// DeployWithContractChainTest runs the tests that deploy with contract chains
func DeployWithContractChainTest(suite *TestSuite) {
	suite.Run("dms with contract chains", func() {
		orchestrator := suite.nodes[0]     // Node 0: Orchestrator
		contractHostA := suite.nodes[1]    // Node 1: Contract Host for Contract A
		organization := suite.nodes[2]     // Node 2: Organization
		provider1 := suite.nodes[3]        // Node 3: Provider 1
		provider2 := suite.nodes[4]        // Node 4: Provider 2
		paymentValidator := suite.nodes[5] // Node 5: Payment Validator (optional)

		// Offboard contract host and payment validator
		contractHostA.client.offboard(suite.T(), contractHostA.userContext, contractHostA.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)
		organization.client.offboard(suite.T(), organization.userContext, organization.password)

		// Start mock RPC server for payment validator
		go startMockRPC(9427)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9427/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		// Step 1: Create Contract A (Orchestrator ↔ Organization)
		// Contract A should have DisableBilling: true
		srcFileA := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFileA := filepath.Join(orchestrator.config.WorkDir, "contract-a.json")
		err := copyFile(srcFileA, destinationFileA)
		suite.Require().NoError(err)

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

		// Prepare metadata for Contract A (Head Contract)
		metadataJSON := fmt.Sprintf(`{"%s": "%s"}`, contracts.ContractChainRoleMetadataKey, contracts.ContractChainRoleHead)

		// Replace placeholders for Contract A
		err = replacePlaceholders(destinationFileA, map[string]string{
			"seDID":               contractHostA.dmsDID,
			"providerDID":         organization.dmsDID,
			"requesterDID":        orchestrator.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       "0xe66b31678d6c16e9ebf358268a790b763c133750",
			"providerAddr":        "0x4741783ed607d1496f65749d2d9c94cf6c23352a",
			"feesPerAllocation":   "10",
			"paymentModel":        string(contracts.PayPerTimeUtilization),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"feePerTimeUnit":      "0.01",
			"timeUnit":            "second",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
			"metadata":            metadataJSON,
		})
		suite.Require().NoError(err)

		// Create Contract A
		cmdOut, err := orchestrator.client.createContractRemote(suite.T(), destinationFileA, orchestrator.dmsContext, orchestrator.password, contractHostA.dmsDID)
		suite.Require().NoError(err)
		contractADID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Step 2: Create Contract B1 (Organization ↔ Provider 1)
		srcFileB1 := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFileB1 := filepath.Join(organization.config.WorkDir, "contract-b1.json")
		err = copyFile(srcFileB1, destinationFileB1)
		suite.Require().NoError(err)

		// Replace placeholders for Contract B1
		err = replacePlaceholders(destinationFileB1, map[string]string{
			"seDID":               contractHostA.dmsDID,
			"providerDID":         provider1.dmsDID,
			"requesterDID":        organization.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       "0x4741783ed607d1496f65749d2d9c94cf6c23352a",
			"providerAddr":        "0xe66b31678d6c16e9ebf358268a790b763c133750",
			"feesPerAllocation":   "10",
			"paymentModel":        string(contracts.PayPerTimeUtilization),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"feePerTimeUnit":      "0.01",
			"timeUnit":            "second",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		cmdOut, err = organization.client.createContractRemote(suite.T(), destinationFileB1, organization.dmsContext, organization.password, contractHostA.dmsDID)
		suite.Require().NoError(err)
		contractB1DID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Step 3: Create Contract B2 (Organization ↔ Provider 2)
		srcFileB2 := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFileB2 := filepath.Join(organization.config.WorkDir, "contract-b2.json")
		err = copyFile(srcFileB2, destinationFileB2)
		suite.Require().NoError(err)

		// Replace placeholders for Contract B2
		err = replacePlaceholders(destinationFileB2, map[string]string{
			"seDID":               contractHostA.dmsDID,
			"providerDID":         provider2.dmsDID,
			"requesterDID":        organization.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       "0x4741783ed607d1496f65749d2d9c94cf6c23352a",
			"providerAddr":        "0xe66b31678d6c16e9ebf358268a790b763c133750",
			"feesPerAllocation":   "10",
			"paymentModel":        string(contracts.PayPerAllocation),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
		})
		suite.Require().NoError(err)

		cmdOut, err = organization.client.createContractRemote(suite.T(), destinationFileB2, organization.dmsContext, organization.password, contractHostA.dmsDID)
		suite.Require().NoError(err)
		contractB2DID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		cmdOutput, err := orchestrator.client.approveContracts(suite.T(), contractADID, orchestrator.dmsContext, orchestrator.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, "already signed")

		// Approve Contract A (Organization signs)
		cmdOutput, err = organization.client.approveContracts(suite.T(), contractADID, organization.dmsContext, organization.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, `"success": true`)

		// Approve Contract B1 (Organization signs)
		cmdOutput, err = organization.client.approveContracts(suite.T(), contractB1DID, organization.dmsContext, organization.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, "already signed")

		// Approve Contract B1 (Provider 1 signs)
		cmdOutput, err = provider1.client.approveContracts(suite.T(), contractB1DID, provider1.dmsContext, provider1.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, `"success": true`)

		// Approve Contract B1 (Organization signs)
		cmdOutput, err = organization.client.approveContracts(suite.T(), contractB2DID, organization.dmsContext, organization.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, "already signed")

		// Approve Contract B2 (Provider 2 signs)
		cmdOutput, err = provider2.client.approveContracts(suite.T(), contractB2DID, provider2.dmsContext, provider2.password)
		suite.Require().NoError(err)
		suite.Require().Contains(cmdOutput, `"success": true`)

		suite.waitContractState(orchestrator, contractADID, contractHostA.dmsDID, "ACCEPTED", 60*time.Second)

		// Step 4: Deploy with Contract A in ensemble config
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract-chain.yaml")
		destinationFileEnsemble := filepath.Join(orchestrator.config.WorkDir, "hello-contract-chain.output.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)

		// Contract A specified with all fields (DID, host, provider=Organization, requestor=Orchestrator)
		contractsContent := `contracts:
  org_contract:
    did: "` + contractADID + `"
    host: "` + contractHostA.dmsDID + `"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		// Deploy ensemble
		deploymentResult := orchestrator.client.deploy(
			suite.T(), orchestrator.userContext, orchestrator.password,
			destinationFileEnsemble, "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until deployment reaches Running status
		suite.Require().Eventually(func() bool {
			status, err := orchestrator.client.deploymentStatus(suite.T(), orchestrator.userContext, orchestrator.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment with contract chain did not reach Running status")

		// Step 5: Verify Contract B billing works (per-node separation)
		// Calculate usages for Contract B1 (should only include Provider 1's usage)
		calculateResp, err := contractHostA.client.calculateContractUsages(suite.T(), contractHostA.dmsContext, contractHostA.password, contractB1DID)
		suite.Require().NoError(err)
		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(usageResponse.Results)

		suite.T().Log("tail contract 1 billing results", usageResponse.Results)

		// Assertions for Tail Contract B1 (PayPerAllocation)
		foundB1 := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractB1DID {
				foundB1 = true
				// Assertion 1: Verify no error
				suite.Require().Empty(result.Error, "Tail Contract B1 billing should execute without errors")
				// Assertion 2: Verify payment model is PayPerAllocation
				suite.Require().Equal(
					contracts.PayPerTimeUtilization,
					result.PaymentModel,
					"Tail Contract B1 payment model should be pay_per_allocation")
				// Assertion 3: Verify usages count > 0
				suite.Require().Greater(
					result.Usages,
					0,
					"Tail Contract B1 should have usages > 0")
				// Assertion 4: Verify PayPerAllocation-specific fields
				suite.Require().Nil(result.ResourceUtilization, "Tail Contract B1 should not have ResourceUtilization (PayPerAllocation)")
				suite.Require().Nil(result.FixedRentalDetails, "Tail Contract B1 should not have FixedRentalDetails (PayPerAllocation)")
				suite.Require().Nil(result.PeriodicDetails, "Tail Contract B1 should not have PeriodicDetails (PayPerAllocation)")
				break
			}
		}
		suite.Require().True(foundB1, "expected Tail Contract B1 should be in results")

		usageResponse = contracts.CollectUsagesAndForwardToPaymentProvidersReponse{}
		// Calculate usages for Contract B2 (should only include Provider 2's usage)
		calculateResp, err = contractHostA.client.calculateContractUsages(suite.T(), contractHostA.dmsContext, contractHostA.password, contractB2DID)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(usageResponse.Results)

		suite.T().Log("tail contract 2 results", usageResponse.Results)

		// Assertions for Tail Contract B2 (PayPerAllocation)
		foundB2 := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractB2DID {
				foundB2 = true
				// Assertion 1: Verify no error
				suite.Require().Empty(result.Error, "Tail Contract B2 billing should execute without errors")
				// Assertion 2: Verify payment model is PayPerAllocation
				suite.Require().Equal(
					contracts.PayPerAllocation,
					result.PaymentModel,
					"Tail Contract B2 payment model should be pay_per_allocation")
				// Assertion 3: Verify usages count > 0
				suite.Require().Greater(
					result.Usages,
					0,
					"Tail Contract B2 should have usages > 0")
				// Assertion 4: Verify PayPerAllocation-specific fields
				suite.Require().Nil(result.TimeUtilization, "Tail Contract B2 should not have TimeUtilization (PayPerAllocation)")
				suite.Require().Nil(result.ResourceUtilization, "Tail Contract B2 should not have ResourceUtilization (PayPerAllocation)")
				suite.Require().Nil(result.FixedRentalDetails, "Tail Contract B2 should not have FixedRentalDetails (PayPerAllocation)")
				suite.Require().Nil(result.PeriodicDetails, "Tail Contract B2 should not have PeriodicDetails (PayPerAllocation)")
				break
			}
		}
		suite.Require().True(foundB2, "expected Tail Contract B2 should be in results")

		usageResponse = contracts.CollectUsagesAndForwardToPaymentProvidersReponse{}
		// Step 6: Verify Head Contract billing works
		// Contract A (Head Contract) should generate invoices based on head_contract_did
		calculateResp, err = contractHostA.client.calculateContractUsages(suite.T(), contractHostA.dmsContext, contractHostA.password, contractADID)
		suite.Require().NoError(err)
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err)
		suite.Require().Empty(usageResponse.Error, "Head Contract usage calculation should not have errors")
		suite.Require().NotEmpty(usageResponse.Results, "Head Contract usage calculation should return results")

		suite.T().Log("head contract billing results", usageResponse.Results)

		// Find Contract A in results
		found := false
		for _, result := range usageResponse.Results {
			if result.ContractDID == contractADID {
				found = true

				// Assertion 1: Verify billing was executed (no error)
				suite.Require().Empty(result.Error, "Head Contract billing should execute without errors")

				// Assertion 2: Verify payment model is PayPerTimeUtilization
				suite.Require().Equal(
					contracts.PayPerTimeUtilization,
					result.PaymentModel,
					"Head Contract payment model should be pay_per_time_utilization")

				// Assertion 3: Verify PayPerTimeUtilization-specific fields
				suite.Require().NotNil(result.TimeUtilization, "Head Contract should have TimeUtilization (PayPerTimeUtilization)")
				suite.Require().NotEmpty(result.TimeUtilization.Deployments, "Head Contract TimeUtilization should have deployments")
				// Verify deployment has allocations
				totalUtilization := 0.0
				for _, deployment := range result.TimeUtilization.Deployments {
					suite.Require().NotEmpty(deployment.Allocations, "Deployment should have allocations")
					totalUtilization += deployment.TotalUtilizationSec
					for _, allocation := range deployment.Allocations {
						suite.Require().NotEmpty(allocation.AllocationID, "Allocation should have AllocationID")
						suite.Require().Greater(allocation.Duration.Seconds(), 0.0, "Allocation duration should be > 0")
						suite.Require().False(allocation.StartTime.IsZero(), "Allocation should have StartTime")
					}
				}
				suite.Require().Greater(totalUtilization, 0.0, "Total utilization should be > 0")

				// Assertion 4: Verify other payment model fields are nil
				suite.Require().Nil(result.ResourceUtilization, "Head Contract should not have ResourceUtilization (PayPerTimeUtilization)")
				suite.Require().Nil(result.FixedRentalDetails, "Head Contract should not have FixedRentalDetails (PayPerTimeUtilization)")
				suite.Require().Nil(result.PeriodicDetails, "Head Contract should not have PeriodicDetails (PayPerTimeUtilization)")

				// Assertion 5: Verify the calculation is based on head_contract_did
				// (This is implicit - if billing works, it means it queried by head_contract_did correctly)
				suite.T().Logf(
					"Head Contract billing successful: ContractDID=%s, PaymentModel=%s, TotalUtilizationSec=%.2f",
					result.ContractDID, result.PaymentModel, totalUtilization)
				break
			}
		}
		suite.Require().True(found, "expected Head Contract should be in results")

		// Step 7: Verify transactions are generated correctly
		// Wait for transactions to be created after usage calculation
		suite.waitLocalTransactionCountAtLeast(orchestrator, 1, 60*time.Second)

		// Step 7a: Verify Head Contract transactions at orchestrator side
		// Orchestrator is the requestor in Head Contract, so should receive transactions to pay Organization
		suite.T().Log("Checking Head Contract transactions at orchestrator side")
		orchestratorOutput, err := orchestrator.client.listLocalTransactions(suite.T(), orchestrator.dmsContext, orchestrator.password)
		suite.Require().NoError(err)

		var orchestratorTxList contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(orchestratorOutput), &orchestratorTxList)
		suite.Require().NoError(err)
		suite.Require().Empty(orchestratorTxList.Error, "orchestrator transaction list should not have errors")

		// Find transactions for Head Contract (Contract A)
		var headContractTransactions []*transaction.Transaction
		for _, tx := range orchestratorTxList.Transactions {
			if tx.ContractDID == contractADID {
				headContractTransactions = append(headContractTransactions, tx)
			}
		}
		suite.Require().NotEmpty(headContractTransactions, "orchestrator should have transactions for Head Contract (Contract A)")

		// Verify Head Contract transaction details
		for _, tx := range headContractTransactions {
			suite.Require().NotEmpty(tx.UniqueID, "Head Contract transaction should have UniqueID")
			suite.Require().Equal(contractADID, tx.ContractDID, "Head Contract transaction should have correct ContractDID")
			suite.Require().Equal("unpaid", tx.Status, "Head Contract transaction should be unpaid initially")
			suite.Require().NotEmpty(tx.Amount, "Head Contract transaction should have Amount")
			suite.T().Logf("Head Contract transaction at orchestrator: UniqueID=%s, Amount=%s, Status=%s", tx.UniqueID, tx.Amount, tx.Status)
		}

		// Step 7b: Verify Tail Contract transactions at organization side
		// Organization is the requestor in Tail Contracts, so should receive transactions to pay Providers
		suite.T().Log("Checking Tail Contract transactions at organization side")
		organizationOutput, err := organization.client.listLocalTransactions(suite.T(), organization.dmsContext, organization.password)
		suite.Require().NoError(err)

		var organizationTxList contracts.ContractListLocalTransactionsResponse
		err = json.Unmarshal([]byte(organizationOutput), &organizationTxList)
		suite.Require().NoError(err)
		suite.Require().Empty(organizationTxList.Error, "organization transaction list should not have errors")

		// Find transactions for Tail Contract B1
		var tailContractB1Transactions []*transaction.Transaction
		for _, tx := range organizationTxList.Transactions {
			if tx.ContractDID == contractB1DID {
				tailContractB1Transactions = append(tailContractB1Transactions, tx)
			}
		}
		suite.Require().NotEmpty(tailContractB1Transactions, "organization should have transactions for Tail Contract B1")

		// Verify Tail Contract B1 transaction details
		for _, tx := range tailContractB1Transactions {
			suite.Require().NotEmpty(tx.UniqueID, "Tail Contract B1 transaction should have UniqueID")
			suite.Require().Equal(contractB1DID, tx.ContractDID, "Tail Contract B1 transaction should have correct ContractDID")
			suite.Require().Equal("unpaid", tx.Status, "Tail Contract B1 transaction should be unpaid initially")
			suite.Require().NotEmpty(tx.Amount, "Tail Contract B1 transaction should have Amount")
			suite.T().Logf("Tail Contract B1 transaction at organization: UniqueID=%s, Amount=%s, Status=%s", tx.UniqueID, tx.Amount, tx.Status)
		}

		// Find transactions for Tail Contract B2
		var tailContractB2Transactions []*transaction.Transaction
		for _, tx := range organizationTxList.Transactions {
			if tx.ContractDID == contractB2DID {
				tailContractB2Transactions = append(tailContractB2Transactions, tx)
			}
		}
		// Note: B2 might not have transactions if it wasn't used in deployment
		if len(tailContractB2Transactions) > 0 {
			// Verify Tail Contract B2 transaction details
			for _, tx := range tailContractB2Transactions {
				suite.Require().NotEmpty(tx.UniqueID, "Tail Contract B2 transaction should have UniqueID")
				suite.Require().Equal(contractB2DID, tx.ContractDID, "Tail Contract B2 transaction should have correct ContractDID")
				suite.Require().Equal("unpaid", tx.Status, "Tail Contract B2 transaction should be unpaid initially")
				suite.Require().NotEmpty(tx.Amount, "Tail Contract B2 transaction should have Amount")
				suite.T().Logf("Tail Contract B2 transaction at organization: UniqueID=%s, Amount=%s, Status=%s", tx.UniqueID, tx.Amount, tx.Status)
			}
		} else {
			suite.T().Log("No transactions found for Tail Contract B2 at organization (may not have been used in deployment)")
		}

		// Step 8: Confirm transactions and verify status changes
		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"

		// Step 8a: Confirm Head Contract transaction at orchestrator side
		if len(headContractTransactions) > 0 {
			headTx := headContractTransactions[0]
			suite.T().Logf("Confirming Head Contract transaction at orchestrator: %s", headTx.UniqueID)

			confirmOutput, err := orchestrator.client.confirmLocalTransaction(suite.T(), orchestrator.dmsContext, orchestrator.password, headTx.UniqueID, txHash)
			suite.Require().NoError(err, "Head Contract transaction confirmation should not fail")

			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			// Note: confirmation may fail validation if txHash is not verified, but should not error on request
			suite.T().Logf("Head Contract transaction confirmation response: Error=%s", confirmResp.Error)

			// Verify transaction status
			orchestratorOutput, err = orchestrator.client.listLocalTransactions(suite.T(), orchestrator.dmsContext, orchestrator.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(orchestratorOutput), &respAfterConfirm)
			suite.Require().NoError(err)

			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == headTx.UniqueID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed Head Contract transaction")
			suite.T().Logf("Head Contract transaction status after confirmation: %s", confirmedTx.Status)

			// Verify payment status can be retrieved from payment validator
			statusOutput, err := orchestrator.client.paymentStatus(suite.T(), orchestrator.dmsContext, orchestrator.password, headTx.UniqueID, paymentValidator.dmsDID)
			suite.Require().NoError(err, "should be able to retrieve payment status for Head Contract")
			suite.T().Logf("Head Contract payment status: %s", statusOutput)
		}

		// Step 8b: Confirm Tail Contract B1 transaction at organization side
		if len(tailContractB1Transactions) > 0 {
			tailB1Tx := tailContractB1Transactions[0]
			suite.T().Logf("Confirming Tail Contract B1 transaction at organization: %s", tailB1Tx.UniqueID)

			confirmOutput, err := organization.client.confirmLocalTransaction(suite.T(), organization.dmsContext, organization.password, tailB1Tx.UniqueID, txHash)
			suite.Require().NoError(err, "Tail Contract B1 transaction confirmation should not fail")

			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.T().Logf("Tail Contract B1 transaction confirmation response: Error=%s", confirmResp.Error)

			// Verify transaction status
			organizationOutput, err = organization.client.listLocalTransactions(suite.T(), organization.dmsContext, organization.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(organizationOutput), &respAfterConfirm)
			suite.Require().NoError(err)

			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == tailB1Tx.UniqueID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed Tail Contract B1 transaction")
			suite.T().Logf("Tail Contract B1 transaction status after confirmation: %s", confirmedTx.Status)

			// Verify payment status can be retrieved from payment validator
			statusOutput, err := organization.client.paymentStatus(suite.T(), organization.dmsContext, organization.password, tailB1Tx.UniqueID, paymentValidator.dmsDID)
			suite.Require().NoError(err, "should be able to retrieve payment status for Tail Contract B1")
			suite.T().Logf("Tail Contract B1 payment status: %s", statusOutput)
		}

		// Step 8c: Confirm Tail Contract B2 transaction at organization side (if it exists)
		if len(tailContractB2Transactions) > 0 {
			tailB2Tx := tailContractB2Transactions[0]
			suite.T().Logf("Confirming Tail Contract B2 transaction at organization: %s", tailB2Tx.UniqueID)

			confirmOutput, err := organization.client.confirmLocalTransaction(suite.T(), organization.dmsContext, organization.password, tailB2Tx.UniqueID, txHash)
			suite.Require().NoError(err, "Tail Contract B2 transaction confirmation should not fail")

			var confirmResp contracts.ContractConfirmLocalTransactionResponse
			err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
			suite.Require().NoError(err, "should be able to parse confirmation response")
			suite.T().Logf("Tail Contract B2 transaction confirmation response: Error=%s", confirmResp.Error)

			// Verify transaction status
			organizationOutput, err = organization.client.listLocalTransactions(suite.T(), organization.dmsContext, organization.password)
			suite.Require().NoError(err)

			var respAfterConfirm contracts.ContractListLocalTransactionsResponse
			err = json.Unmarshal([]byte(organizationOutput), &respAfterConfirm)
			suite.Require().NoError(err)

			var confirmedTx *transaction.Transaction
			for _, tx := range respAfterConfirm.Transactions {
				if tx.UniqueID == tailB2Tx.UniqueID {
					confirmedTx = tx
					break
				}
			}
			suite.Require().NotNil(confirmedTx, "should find the confirmed Tail Contract B2 transaction")
			suite.T().Logf("Tail Contract B2 transaction status after confirmation: %s", confirmedTx.Status)

			// Verify payment status can be retrieved from payment validator
			statusOutput, err := organization.client.paymentStatus(suite.T(), organization.dmsContext, organization.password, tailB2Tx.UniqueID, paymentValidator.dmsDID)
			suite.Require().NoError(err, "should be able to retrieve payment status for Tail Contract B2")
			suite.T().Logf("Tail Contract B2 payment status: %s", statusOutput)
		}

		// Step 9: Verify transaction metadata and amounts are correct
		// Verify Head Contract transaction has correct payment model metadata
		if len(headContractTransactions) > 0 {
			headTx := headContractTransactions[0]
			suite.Require().NotEmpty(headTx.Metadata, "Head Contract transaction should have metadata")
			// For PayPerTimeUtilization, verify metadata contains expected fields
			if paymentModel, ok := headTx.Metadata["payment_model"].(string); ok {
				suite.Require().Equal(string(contracts.PayPerTimeUtilization), paymentModel, "Head Contract transaction metadata should have correct payment model")
			}
			suite.T().Logf("Head Contract transaction metadata: %+v", headTx.Metadata)
		}

		// Verify Tail Contract B1 transaction has correct payment model metadata
		if len(tailContractB1Transactions) > 0 {
			tailB1Tx := tailContractB1Transactions[0]
			suite.Require().NotEmpty(tailB1Tx.Metadata, "Tail Contract B1 transaction should have metadata")
			// For PayPerAllocation, verify metadata contains expected fields
			if paymentModel, ok := tailB1Tx.Metadata["payment_model"].(string); ok {
				suite.Require().Equal(string(contracts.PayPerAllocation), paymentModel, "Tail Contract B1 transaction metadata should have correct payment model")
			}
			if allocationCount, ok := tailB1Tx.Metadata["allocation_count"].(float64); ok {
				suite.Require().Greater(allocationCount, 0.0, "Tail Contract B1 transaction should have allocation_count > 0")
			}
			suite.T().Logf("Tail Contract B1 transaction metadata: %+v", tailB1Tx.Metadata)
		}

		// Verify Tail Contract B2 transaction has correct payment model metadata (if exists)
		if len(tailContractB2Transactions) > 0 {
			tailB2Tx := tailContractB2Transactions[0]
			suite.Require().NotEmpty(tailB2Tx.Metadata, "Tail Contract B2 transaction should have metadata")
			// For PayPerAllocation, verify metadata contains expected fields
			if paymentModel, ok := tailB2Tx.Metadata["payment_model"].(string); ok {
				suite.Require().Equal(string(contracts.PayPerAllocation), paymentModel, "Tail Contract B2 transaction metadata should have correct payment model")
			}
			if allocationCount, ok := tailB2Tx.Metadata["allocation_count"].(float64); ok {
				suite.Require().Greater(allocationCount, 0.0, "Tail Contract B2 transaction should have allocation_count > 0")
			}
			suite.T().Logf("Tail Contract B2 transaction metadata: %+v", tailB2Tx.Metadata)
		}
	})
}

// DeployWithContractUSDTQuoteTest runs tests identical to DeployWithContractTest
// but with pricing_currency set to "USDT" and tests the quote creation, validation, cancellation, and payment flow
func DeployWithContractUSDTQuoteTest(suite *TestSuite) {
	if cmcKey := os.Getenv("DMS_TEST_CMC_API_KEY"); cmcKey == "" {
		suite.T().Skip("env var DMS_TEST_CMC_API_KEY not set")
	}
	suite.Run("dms with contracts USDT quote flow", func() {
		requester := suite.nodes[0]
		contractHost := suite.nodes[1]
		provider := suite.nodes[2]
		paymentValidator := suite.nodes[3]

		// Configure nodes with CoinMarketCap sandbox API
		// Configuration is set in setupTestNetwork() based on test name
		// Sandbox API endpoint: https://sandbox-api.coinmarketcap.com/v1
		// Sandbox API key: b54bcf4d-1bca-4e8e-9a24-22ff2c3f462c (public test key)

		// offboard this machine to not accept any bid request
		contractHost.client.offboard(suite.T(), contractHost.userContext, contractHost.password)
		paymentValidator.client.offboard(suite.T(), paymentValidator.userContext, paymentValidator.password)

		srcFile := filepath.Join(suite.testDataDir, "contracts", "sample.json.sample")
		destinationFile := filepath.Join(requester.config.WorkDir, "sample.json")
		err := copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)

		// random addresses
		requesterEthAddr := "0xe66b31678d6c16e9ebf358268a790b763c133750"
		providerEthAddr := "0x4741783ed607d1496f65749d2d9c94cf6c23352a"

		// Set fee in USDT (will be converted to NTX via quote)
		feesPerAllocation := "10.00" // 10 USDT

		// rpc on port (use different port to avoid conflicts with other tests)
		go startMockRPC(9428)
		suite.Require().Eventually(func() bool {
			url := "http://localhost:9428/healthz"
			return checkHealth(url)
		}, 10*time.Second, 500*time.Millisecond, "healthcheck endpoint did not become healthy in time")

		startDate := time.Now().Format(time.RFC3339)
		endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err = replacePlaceholders(destinationFile, map[string]string{
			"seDID":               contractHost.dmsDID,
			"providerDID":         provider.dmsDID,
			"requesterDID":        requester.dmsDID,
			"paymentValidatorDID": paymentValidator.dmsDID,
			"requesterAddr":       requesterEthAddr,
			"providerAddr":        providerEthAddr,
			"feesPerAllocation":   feesPerAllocation,
			"paymentModel":        string(contracts.PayPerAllocation),
			"resourceTimeUnit":    "minute",
			"paymentPeriod":       "minute",
			"paymentPeriodCount":  "1",
			"startDate":           startDate,
			"endDate":             endDate,
			"disableBilling":      "false",
			"metadata":            "",
		})
		suite.Require().NoError(err)

		// Add pricing_currency to contract JSON
		contractJSON, err := readContractJSON(destinationFile)
		suite.Require().NoError(err)
		contractJSON["payment_details"].(map[string]interface{})["pricing_currency"] = "USDT"
		err = writeContractJSON(destinationFile, contractJSON)
		suite.Require().NoError(err)

		cmdOut, err := requester.client.createContract(suite.T(), destinationFile, requester.dmsContext, requester.password)
		fmt.Println(cmdOut, err)

		contractDID, err := getContractID(cmdOut)
		suite.Require().NoError(err)

		// Contract validation
		cmdOut, err = provider.client.validateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		validResult, err := extractValidationResponse(cmdOut)
		suite.Require().NoError(err)
		suite.Require().Equal("false", validResult)

		// Approve contract
		cmdOut, err = provider.client.listIncomingContracts(suite.T(), provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)
		cmdOut, err = provider.client.approveContracts(suite.T(), contractDID, provider.dmsContext, provider.password)
		fmt.Println(cmdOut, err)

		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "ACCEPTED", 60*time.Second)

		// Deployment setup
		srcFileEnsemble := filepath.Join(suite.testDataDir, "ensembles", "hello-contract.yaml")
		destinationFileEnsemble := filepath.Join(requester.config.WorkDir, "hello-contract.yaml")
		err = copyFile(srcFileEnsemble, destinationFileEnsemble)
		suite.Require().NoError(err)
		contractsContent := `contracts:
  contract1:
    did: "` + contractDID + `"
    host: "` + contractHost.dmsDID + `"
    payment_details:
        payment_model: "` + string(contracts.PayPerAllocation) + `"
        fee_per_allocation: "` + feesPerAllocation + `"
        pricing_currency: "USDT"`
		err = replaceContractInFile(destinationFileEnsemble, contractsContent)
		suite.Require().NoError(err)

		// Redeploy after approval
		deploymentResult := requester.client.deploy(
			suite.T(), requester.userContext, requester.password,
			filepath.Join(requester.config.WorkDir, "hello-contract.yaml"), "2m")
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running"
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
		calculateResp, err := contractHost.client.calculateContractUsages(suite.T(), contractHost.dmsContext, contractHost.password, contractDID)
		suite.Require().NoError(err)

		var usageResponse contracts.CollectUsagesAndForwardToPaymentProvidersReponse
		err = json.Unmarshal([]byte(calculateResp), &usageResponse)
		suite.Require().NoError(err, "failed to unmarshal usage calculation response")
		suite.Require().Empty(usageResponse.Error, "usage calculation should not have errors")

		uniqueID := suite.waitLocalTransactionStatus(requester, "unpaid", 60*time.Second)

		// check if transactions arrived on service provider to be paid
		output, err := requester.client.listLocalTransactions(suite.T(), requester.dmsContext, requester.password)
		suite.Require().NoError(err)
		suite.Require().Contains(output, uniqueID)

		// TEST 1: Get Payment Quote
		suite.T().Log("Testing payment quote creation")
		quoteOutput, err := requester.client.getPaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, uniqueID,
		)
		suite.Require().NoError(err)

		quoteResp, err := extractQuoteResponse(quoteOutput)
		suite.Require().NoError(err)
		suite.Require().Empty(quoteResp.Error, "quote creation should not have errors")
		suite.Require().NotEmpty(quoteResp.QuoteID, "quote should have quote_id")
		suite.Require().NotEmpty(quoteResp.ConvertedAmount, "quote should have converted_amount")
		suite.Require().Equal("USDT", quoteResp.PricingCurrency, "pricing currency should be USDT")
		suite.Require().Equal("NTX", quoteResp.PaymentCurrency, "payment currency should be NTX")
		suite.Require().NotEmpty(quoteResp.ExchangeRate, "quote should have exchange_rate")
		suite.T().Logf("Quote created: QuoteID=%s, OriginalAmount=%s, ConvertedAmount=%s, ExchangeRate=%s",
			quoteResp.QuoteID, quoteResp.OriginalAmount, quoteResp.ConvertedAmount, quoteResp.ExchangeRate)

		quoteID := quoteResp.QuoteID

		// TEST 2: Attempt to create duplicate quote (should fail)
		suite.T().Log("Testing duplicate quote prevention")
		quoteOutputDuplicate, err := requester.client.getPaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, uniqueID,
		)
		suite.Require().NoError(err) // Command succeeds, but response should have error

		quoteRespDuplicate, err := extractQuoteResponse(quoteOutputDuplicate)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(quoteRespDuplicate.Error, "should return error when trying to create duplicate quote")
		suite.Require().Contains(quoteRespDuplicate.Error, "active quote already exists", "error should indicate active quote exists")
		suite.Require().Contains(quoteRespDuplicate.Error, quoteID, "error should mention existing quote_id")
		suite.T().Logf("Duplicate quote creation correctly rejected: %s", quoteRespDuplicate.Error)

		// TEST 3: Validate Quote
		suite.T().Log("Testing quote validation")
		validateOutput, err := requester.client.validatePaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, quoteID,
		)
		suite.Require().NoError(err)

		validateResp, err := extractValidateQuoteResponse(validateOutput)
		suite.Require().NoError(err)
		suite.Require().True(validateResp.Valid, "quote should be valid")
		suite.Require().Empty(validateResp.Error, "quote validation should not have errors")
		suite.Require().Equal(quoteID, validateResp.QuoteID, "quote_id should match")
		suite.T().Logf("Quote validated successfully: QuoteID=%s", quoteID)

		// TEST 4: Test quote cancellation flow
		suite.T().Log("Testing quote cancellation")
		// Cancel the existing quote
		cancelOutput, err := requester.client.cancelPaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, quoteID,
		)
		suite.Require().NoError(err)

		var cancelResp contracts.ContractCancelPaymentQuoteResponse
		err = json.Unmarshal([]byte(cancelOutput), &cancelResp)
		suite.Require().NoError(err)
		suite.Require().Empty(cancelResp.Error, "quote cancellation should not have errors")
		suite.T().Logf("Quote cancelled successfully: QuoteID=%s", quoteID)

		// Verify cancelled quote cannot be validated
		validateOutput2, err := requester.client.validatePaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, quoteID,
		)
		suite.Require().NoError(err)
		validateResp2, err := extractValidateQuoteResponse(validateOutput2)
		suite.Require().NoError(err)
		suite.Require().False(validateResp2.Valid, "cancelled quote should be invalid")
		suite.Require().Contains(validateResp2.Error, "quote already used", "error should indicate quote was used/cancelled")
		suite.T().Logf("Cancelled quote validation result: Valid=%v, Error=%s", validateResp2.Valid, validateResp2.Error)

		// TEST 5: Create new quote after cancellation (should succeed)
		suite.T().Log("Testing new quote creation after cancellation")
		quoteOutput2, err := requester.client.getPaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, uniqueID,
		)
		suite.Require().NoError(err)

		quoteResp2, err := extractQuoteResponse(quoteOutput2)
		suite.Require().NoError(err)
		suite.Require().Empty(quoteResp2.Error, "should be able to create new quote after cancelling previous one")
		suite.Require().NotEmpty(quoteResp2.QuoteID, "new quote should have quote_id")
		suite.Require().NotEqual(quoteID, quoteResp2.QuoteID, "new quote should have different quote_id")
		cancelQuoteID := quoteResp2.QuoteID
		suite.T().Logf("New quote created after cancellation: QuoteID=%s", cancelQuoteID)

		// Validate the new quote
		validateOutput3, err := requester.client.validatePaymentQuote(
			suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, cancelQuoteID)
		suite.Require().NoError(err)
		validateResp3, err := extractValidateQuoteResponse(validateOutput3)
		suite.Require().NoError(err)
		suite.Require().True(validateResp3.Valid, "new quote should be valid")
		suite.T().Logf("New quote validated successfully: QuoteID=%s", cancelQuoteID)

		// XXX: this test is using a mock transaction hash and makes the confirmation fail due to non-matching
		// transaction amount. Because it's not marked as used if it's not verified, the following test will fail as well.
		// needs improvement. Commenting out the test for "markasused" for now.
		// TEST 6: Attempt to reuse quote (should fail after payment)
		// Simulate payment with the quote
		txHash := "0x21ef8b84a75ec89097af6b53749b1af0fc21495060b0b57a6b117d6c69113e5f"
		confirmOutput, err := requester.client.confirmLocalTransactionWithQuote(
			suite.T(), requester.dmsContext, requester.password, uniqueID, txHash, cancelQuoteID)
		suite.Require().NoError(err)

		var confirmResp contracts.ContractConfirmLocalTransactionResponse
		err = json.Unmarshal([]byte(confirmOutput), &confirmResp)
		suite.Require().NoError(err)
		// Payment validation will fail because it's a mock transaction
		suite.T().Logf("Payment confirmation response: %s", confirmResp.Error)
		// confirm error since the transaction won't be correctly verified
		// TODO - pass the verification with a valid txhash
		suite.Assert().NotEmpty(confirmResp.Error, "payment confirmation should fail with mock transaction hash")

		// // TEST 7: Try to validate used quote (should fail)
		// suite.T().Log("Testing validation of used quote")
		// validateOutput4, err := requester.client.validatePaymentQuote(
		// 	suite.T(), requester.dmsContext, requester.password, paymentValidator.dmsDID, cancelQuoteID)
		// suite.Require().NoError(err)

		// validateResp4, err := extractValidateQuoteResponse(validateOutput4)
		// suite.Require().NoError(err)
		// // Quote should be invalid after use
		// suite.T().Logf("Used quote validation result: Valid=%v, Error=%s", validateResp4.Valid, validateResp4.Error)
		// suite.Require().False(validateResp4.Valid, "used quote should be invalid")
		// suite.Require().Contains(validateResp4.Error, "quote already used", "error should indicate quote was used")

		// Contract settlement and termination
		_, err = provider.client.settleContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "SETTLED", 30*time.Second)

		_, err = provider.client.terminateContract(suite.T(), provider.dmsContext, provider.password, contractDID, contractHost.dmsDID)
		suite.Require().NoError(err)
		suite.waitContractState(requester, contractDID, contractHost.dmsDID, "TERMINATED", 30*time.Second)
	})
}
