// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store/payment"
)

// PaymentProcessor defines the interface for processing payment items.
// This interface allows for different implementations and easy testing.
type PaymentProcessor interface {
	// ProcessPaymentItems processes a batch of payment items for a contract.
	// This method:
	// - Generates unique IDs for items that don't have one
	// - Saves each payment to the payment store
	// - Forwards transaction requests to the service provider
	// - Continues processing even if individual items fail
	// Returns an error only if there's a critical failure that prevents processing.
	ProcessPaymentItems(
		contract *contracts.Contract,
		items []*contracts.PaymentItem,
		baseUniqueID string,
	) error
}

// paymentProcessorImpl implements PaymentProcessor
type paymentProcessorImpl struct {
	paymentStore    *payment.Store
	network         network.Network
	actor           actor.Handle
	invokeBehaviour func(destination actor.Handle, behavior string, req interface{}, timeout time.Duration) (actor.Envelope, error)
}

// NewPaymentProcessor creates a new payment processor implementation
func NewPaymentProcessor(
	paymentStore *payment.Store,
	network network.Network,
	actor actor.Handle,
	invokeBehaviour func(destination actor.Handle, behavior string, req interface{}, timeout time.Duration) (actor.Envelope, error),
) PaymentProcessor {
	if paymentStore == nil {
		panic("payment store cannot be nil")
	}
	if network == nil {
		panic("network cannot be nil")
	}
	if actor.ID.String() == "" {
		panic("actor handle cannot be empty")
	}
	if invokeBehaviour == nil {
		panic("invokeBehaviour function cannot be nil")
	}

	return &paymentProcessorImpl{
		paymentStore:    paymentStore,
		network:         network,
		actor:           actor,
		invokeBehaviour: invokeBehaviour,
	}
}

// ProcessPaymentItems implements PaymentProcessor.ProcessPaymentItems
func (pp *paymentProcessorImpl) ProcessPaymentItems(
	contract *contracts.Contract,
	items []*contracts.PaymentItem,
	baseUniqueID string,
) error {
	if contract == nil {
		return fmt.Errorf("contract cannot be nil")
	}
	if len(items) == 0 {
		return nil // No items to process
	}

	// Process each item
	for _, item := range items {
		// Set unique ID if not set
		if item.UniqueID == "" {
			if item.DeploymentID != "" {
				item.UniqueID = fmt.Sprintf("%s-%s", baseUniqueID, item.DeploymentID)
			} else {
				item.UniqueID = baseUniqueID
			}
		}

		// Save payment
		if err := pp.savePayment(contract, item); err != nil {
			log.Errorf("failed to save payment for item %s: %v", item.UniqueID, err)
			continue // Continue with other items
		}

		// Forward transaction
		if err := pp.forwardTransaction(contract, item); err != nil {
			log.Errorf("failed to forward transaction for item %s: %v", item.UniqueID, err)
			// Continue - payment is saved
		}
	}

	return nil
}

// savePayment saves a payment to the payment store
func (pp *paymentProcessorImpl) savePayment(contract *contracts.Contract, item *contracts.PaymentItem) error {
	return pp.paymentStore.Insert(payment.Payment{
		UniqueID: item.UniqueID,
		Contract: *contract,
		Usages:   item.Usages,
		Amount:   item.Amount,
		Paid:     false,
	})
}

// forwardTransaction forwards a transaction request to the service provider
func (pp *paymentProcessorImpl) forwardTransaction(
	contract *contracts.Contract,
	item *contracts.PaymentItem,
) error {
	txReq := contracts.TransactionForServiceProviderRequest{
		PaymentValidatorDID: contract.PaymentValidatorDID.URI,
		UniqueID:            item.UniqueID,
		ContractDID:         contract.ContractDID,
		ToAddress:           contract.PaymentDetails.Addresses,
		Amount:              item.Amount,
	}

	destination, err := actor.HandleFromDID(contract.ContractParticipants.Requestor.URI)
	if err != nil {
		return fmt.Errorf("failed to get requestor handle: %w", err)
	}

	go func() {
		reply, err := pp.invokeBehaviour(destination, behaviors.ContractTransactionBehavior, txReq, invokeMessageTimeout)
		if reply.Message == nil || err != nil {
			log.Errorf("failed to forward transaction %s: %v", item.UniqueID, err)
		}
	}()

	return nil
}
