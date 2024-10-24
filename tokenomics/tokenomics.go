// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build ignore

package tokenomics

import (
	"contract"
	"jobs"
	"orchestrator"
	"payments"
	"proofs"
)

// Contract defines the methods for contract operations
type contract interface {
	NewContract() Contract
	InitiateContractClosure(n1 dms.NodeID, n2 dms.NodeID, bid orchestrator.Bid)
	InitiateContractSettlement(n1 dms.NodeID, n2 dms.NodeID, contractID int, verificationResult orchestrator.JobVerificationResult)
}

// Proof defines the methods for proof-based operations
type proofs interface {
	InitiateContractApproval() error
	CreateContractProof() (string, error)
	SaveProof(contractID, proof string) error
	VerifyProof(contractID, proof string) (bool, error)
}

// Payment defines the operations for managing payments and settlements
type payments interface {
	Deposit(contractID int, payment Payment, pricing PricingMethod) error
	SettleContract(contractID int, verificationResult jobs.JobVerificationResult) error
}
