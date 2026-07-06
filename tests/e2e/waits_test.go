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
	"time"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

const (
	defaultPollInterval = 1000 * time.Millisecond
)

func (s *TestSuite) waitDeploymentStatus(
	node *mockNode, context, ensembleID, wantStatus string, timeout time.Duration,
) {
	s.Require().Eventually(func() bool {
		status, err := node.client.deploymentStatus(s.T(), context, node.password, ensembleID)
		if err != nil {
			s.T().Logf("deployment status error: %v", err)
			return false
		}
		got := extractStatus(status)
		s.T().Logf("deployment status: %s (want %s)", got, wantStatus)
		return got == wantStatus
	}, timeout, defaultPollInterval, "deployment %s did not reach status %s", ensembleID, wantStatus)
}

func (s *TestSuite) waitDeploymentCompleted(node *mockNode, context, ensembleID string, timeout time.Duration) {
	s.waitDeploymentStatus(node, context, ensembleID, jobtypes.DeploymentStatusCompleted.String(), timeout)
}

func (s *TestSuite) waitDeploymentRunning(node *mockNode, context, ensembleID string, timeout time.Duration) {
	s.waitDeploymentStatus(node, context, ensembleID, jobtypes.DeploymentStatusRunning.String(), timeout)
}

func (s *TestSuite) waitContractState(
	node *mockNode, contractDID, hostDID, wantState string, timeout time.Duration,
) {
	s.Require().Eventually(func() bool {
		cmdOut, err := node.client.contractStatus(s.T(), node.dmsContext, node.password, contractDID, hostDID)
		if err != nil {
			s.T().Logf("contract status error: %v", err)
			return false
		}
		state, err := extractContractState(cmdOut)
		if err != nil {
			s.T().Logf("extract contract state error: %v", err)
			return false
		}
		s.T().Logf("contract state: %s (want %s)", state, wantState)
		return state == wantState
	}, timeout, defaultPollInterval, "contract %s did not reach state %s", contractDID, wantState)
}

func (s *TestSuite) waitLocalTransactionStatus(node *mockNode, wantStatus string, timeout time.Duration) string { //nolint
	var uniqueID string
	s.Require().Eventually(func() bool {
		output, err := node.client.listLocalTransactions(s.T(), node.dmsContext, node.password)
		if err != nil {
			s.T().Logf("list local transactions error: %v", err)
			return false
		}
		id, status, err := extractTransactionDataRegex(output)
		if err != nil {
			return false
		}
		if status != wantStatus {
			return false
		}
		uniqueID = id
		return true
	}, timeout, defaultPollInterval, "local transaction with status %s was not created", wantStatus)
	return uniqueID
}

func (s *TestSuite) waitLocalTransactionCountGreaterThan(node *mockNode, count int, timeout time.Duration) int {
	var txCount int
	s.Require().Eventually(func() bool {
		output, err := node.client.listLocalTransactions(s.T(), node.dmsContext, node.password)
		if err != nil {
			s.T().Logf("list local transactions error: %v", err)
			return false
		}
		var resp contracts.ContractListLocalTransactionsResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			s.T().Logf("unmarshal local transactions error: %v", err)
			return false
		}
		txCount = len(resp.Transactions)
		return txCount > count
	}, timeout, defaultPollInterval, "expected more than %d local transactions", count)
	return txCount
}

func (s *TestSuite) waitLocalTransactionCountAtLeast(node *mockNode, minCount int, timeout time.Duration) int { //nolint
	var txCount int
	s.Require().Eventually(func() bool {
		output, err := node.client.listLocalTransactions(s.T(), node.dmsContext, node.password)
		if err != nil {
			s.T().Logf("list local transactions error: %v", err)
			return false
		}
		var resp contracts.ContractListLocalTransactionsResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			s.T().Logf("unmarshal local transactions error: %v", err)
			return false
		}
		txCount = len(resp.Transactions)
		return txCount >= minCount
	}, timeout, defaultPollInterval, "expected at least %d local transactions", minCount)
	return txCount
}

func (s *TestSuite) waitLocalTransactionPaid(node *mockNode, txID string, timeout time.Duration) {
	s.Require().Eventually(func() bool {
		output, err := node.client.listLocalTransactions(s.T(), node.dmsContext, node.password)
		if err != nil {
			return false
		}
		var resp contracts.ContractListLocalTransactionsResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			return false
		}
		for _, tx := range resp.Transactions {
			if tx.UniqueID == txID && tx.Status == "paid" {
				return true
			}
		}
		return false
	}, timeout, defaultPollInterval, "transaction %s was not marked paid", txID)
}
