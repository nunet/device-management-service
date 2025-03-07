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
	"context"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/node"
	// TODO: required from updated orchestrator package for BidRequest
	// "gitlab.com/nunet/device-management-service/dms/orchestrator"
)

// Contract represents the contract details between nodes
type Contract struct {
	ContractID     int
	JobID          int         // Example default: 1001
	PaymentDetails Payment     // Zero value: zero value of payments.Payment struct
	Signatures     []byte      // Changed to []byte to hold binary signature data or encrypted
	Settled        bool        // Example default: false
	Verification   jobs.Status // Zero value: zero value of jobs.Status
	ContractProof  string      // Example default: "Pending"
}

// TODO: BidRequest is a placeholder for bestBidSelected that will be used to invoke contract closure and
// will be fetched from orchestrator package
type BidRequest struct {
	JobID       int
	Requestor   string
	Provider    string
	PriceBid    string
	PaymentType PaymentType
	PaymentMode PaymentMode
	PricingMeta PricingMetadata
}

// NewContract creates a new contract and returns a pointer to it with specific initial values
func NewContract() *Contract {
	return &Contract{}
}

// initiateContractClosure initiates the contract closure between two nodes
func initiateContractClosure(_ context.Context, n1 *node.Node, n2 *node.Node, bid BidRequest) error {
	contract := NewContract()
	contract.JobID = bid.JobID

	// Create a new Payment instance based on the bid details
	contract.PaymentDetails = Payment{
		Requestor:   bid.Requestor,
		Provider:    bid.Provider,
		Currency:    bid.PriceBid,
		Timestamp:   time.Now(),
		PaymentType: bid.PaymentType,
		PaymentMode: PaymentMode(bid.PaymentMode),
		PricingMeta: bid.PricingMeta,
	}

	// Sign and notarize the contract
	// contract.signContract()
	// contract.askForSignature()

	// Save the contract in the respective nodes' contract lists and the central database
	if err := saveContractToNodesAndDB(n1, n2, contract); err != nil {
		return fmt.Errorf("failed to save contract to nodes and database: %w", err) // TODO: will be removed and updated via telemetry.
	}

	// TODO:
	// contract.notarize()
	return nil
}

// saveContractToNodesAndDB saves the contract to both nodes and the database
func saveContractToNodesAndDB(_ *node.Node, _ *node.Node, contract *Contract) error {
	return nil
}

// initiateContractSettlement initiates the settlement process for a contract
func initiateContractSettlement(ctx context.Context, n1 *node.Node, n2 *node.Node, contractID int, status jobs.Status) error {
	// Retrieve the contract from the database
	contract, err := GetContractFromDatabase(contractID)
	if err != nil {
		return fmt.Errorf("error retrieving contract: %w", err)
	}

	// Update the verification result
	contract.Verification = status

	// Process the contract settlement
	if err := processContractSettlement(contract, status); err != nil {
		return fmt.Errorf("error during contract settlement: %w", err) // TODO: will be removed and updated via telemetry.
	}

	// Notify both nodes about the settlement
	if err := notifyContractSettlement(n1, contract); err != nil {
		return fmt.Errorf("error notifying node 1: %w", err) // TODO: will be removed and updated via telemetry.
	}

	if err := notifyContractSettlement(n2, contract); err != nil {
		return fmt.Errorf("error notifying node 2: %w", err) // TODO: will be removed and updated via telemetry.
	}

	// Update the contract in the central database
	if err := updateContractInDatabase(contract); err != nil {
		return fmt.Errorf("error updating contract in database: %w", err) // TODO: will be removed and updated via telemetry.
	}

	return nil
}

// processContractSettlement processes the contract settlement based on the pricing method and verification result
func processContractSettlement(contract *Contract, verificationResult jobs.Status) error {
	if verificationResult.Status == "Running" {
		// Calculate the payment amount based on the pricing method
		paymentAmount, err := calculatePaymentAmount(contract.PaymentDetails.PricingMeta)
		if err != nil {
			return fmt.Errorf("error calculating payment amount: %w", err) // TODO: will be removed and updated via telemetry.
		}

		// Process the payment based on the payment type
		if err := processPayment(contract.PaymentDetails.PaymentType, paymentAmount); err != nil {
			return fmt.Errorf("error processing payment: %w", err) // TODO: will be removed and updated via telemetry.
		}
	} else if verificationResult.Status == "Stopped" {
		if err := handleFailedJob(contract); err != nil {
			return fmt.Errorf("error handling failed job: %w", err) // TODO: will be removed and updated via telemetry.
		}

		// TODO implementation of contract marked as settled
		// contract.markAsSettled()
	}
	return nil
}

// handleFailedJob handles the case where the job has stopped, including any necessary refunds
func handleFailedJob(contract *Contract) error {
	if contract.PaymentDetails.PaymentType == EscrowPaymentType { // Changed from string literal to constant
		return refundEscrowPayment(contract.ContractID)
	}
	return nil
}

func notifyContractSettlement(_ *node.Node, contract *Contract) error {
	// Example notification to node
	return nil
}

func updateContractInDatabase(_ *Contract) error {
	// TODO: Implement the function logic
	return nil
}

// TODO: getContractFromDatabase retrieves a contract from the database
func GetContractFromDatabase(contractID int) (*Contract, error) {
	// Logic to retrieve contract from the database using the contractID
	return &Contract{}, nil
}
