// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tokenomics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts/processors"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	contractstore "gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

const (
	contractActorCapsLifespan = 7 * 24 * time.Hour
	contractPeriodicChecker   = 5 * time.Minute
)

// Sentinel errors for fixed rental and periodic invoice calculation
var (
	ErrFullPeriodElapsed = errors.New("full billing period has elapsed, use regular billing instead of pro-rated")
	ErrPeriodNotElapsed  = errors.New("billing period has not elapsed yet, no invoice needed")
	ErrNoDeployments     = errors.New("no deployments active during billing period, skipping invoice")
)

type ContractActor struct {
	*actor.BasicActor
	ContractDID        did.DID
	SolutionEnablerDID did.DID
	forwardInvoice     func(contracts.ContractUsageRequest) error // Function to forward invoice using solution enabler's actor
	ctx                context.Context
	cancel             context.CancelFunc

	contractStore *contractstore.Store
	participants  contracts.ContractParticipants

	PaymentProviderDID did.DID

	usageStore *usage.Store
}

func NewContractActor(
	solutionEnabler actor.Handle,
	paymentValidator did.DID,
	net network.Network,
	participants contracts.ContractParticipants,
	privKey crypto.PrivKey, pubKey crypto.PubKey,
	contractStore *contractstore.Store,
	usageStore *usage.Store,
	forwardInvoice func(contracts.ContractUsageRequest) error, // Function to forward invoice using solution enabler's actor
) (*ContractActor, error) {
	provider, err := did.ProviderFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	ctx := did.NewTrustContext()
	ctx.AddProvider(provider)

	contractKeyDID := did.FromPublicKey(pubKey)
	contractCap, err := ucan.NewCapabilityContext(ctx, contractKeyDID, []did.DID{solutionEnabler.DID}, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	if err != nil {
		return nil, fmt.Errorf("contract actor capability context: %w", err)
	}

	newSecurityContext, err := actor.NewBasicSecurityContext(pubKey, privKey, contractCap)
	if err != nil {
		return nil, err
	}

	self := actor.Handle{
		ID:  newSecurityContext.ID(),
		DID: newSecurityContext.DID(),
		Address: actor.Address{
			HostID:       net.GetHostID().String(),
			InboxAddress: contractKeyDID.URI,
		},
	}

	actor, err := actor.New(self, net, newSecurityContext, actor.NewRateLimiter(actor.DefaultRateLimiterConfig()), actor.BasicActorParams{}, self)
	if err != nil {
		return nil, fmt.Errorf("new contract actor: %w", err)
	}
	ctxActor, cancel := context.WithCancel(context.Background())

	contractActor := ContractActor{
		BasicActor:         actor,
		ContractDID:        contractKeyDID,
		SolutionEnablerDID: solutionEnabler.DID,
		forwardInvoice:     forwardInvoice,
		ctx:                ctxActor,
		cancel:             cancel,
		contractStore:      contractStore,
		participants:       participants,
		usageStore:         usageStore,
		PaymentProviderDID: paymentValidator,
	}

	if err := contractActor.setupBehaviorsAndCapabilities(); err != nil {
		return nil, fmt.Errorf("setting up behaviors and capabilities: %w", err)
	}

	if err := contractActor.SetupParticipantsCapabilities(participants); err != nil {
		return nil, fmt.Errorf("failed to setup participant capabilities: %w", err)
	}

	if err := contractActor.setupPaymentValidatorBehaviorAndCapabilities(paymentValidator); err != nil {
		return nil, fmt.Errorf("failed to setup payment validator capabilities: %w", err)
	}

	return &contractActor, nil
}

func (c *ContractActor) Start() error {
	err := c.BasicActor.Start()
	if err != nil {
		return fmt.Errorf("failed to start contract actor: %w", err)
	}

	go func() {
		ticker := time.NewTicker(contractPeriodicChecker)
		defer ticker.Stop()

		for range ticker.C {
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			// Refresh contract from store to get latest state
			contract, err := c.contractStore.GetContract(c.ContractDID.URI)
			if err != nil {
				log.Errorw("contract not found while checking its status",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"error", err)
				continue
			}

			if time.Now().After(contract.Duration.EndDate) {
				// if we reach duration then we mark it as completed
				contract.CurrentState = contracts.ContractCompleted
				if err := c.contractStore.Upsert(contract); err != nil {
					log.Errorw("failed to update contract with status completed",
						"labels", string(observability.LabelContract),
						"contract_did", c.ContractDID.URI)
				}
				log.Infow("contract has reached its end date",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"end_date", contract.Duration.EndDate)
				return // Contract completed, stop checking
			}
		}
	}()

	// REMOVED: Old per-actor goroutine billing routine
	// The billing is now handled by the centralized scheduler
	// Registration happens via RegisterBilling() method called by Node

	log.Infow("contract actor started",
		"labels", string(observability.LabelContract),
		"contract_did", c.ContractDID.URI,
	)

	return nil
}

func (c *ContractActor) setupBehaviorsAndCapabilities() error {
	contractBehaviors := c.getContractBehaviors()
	for behavior, handler := range contractBehaviors {
		if err := c.BasicActor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}

	// Grant all capabilities to contract creator
	err := c.Security().Grant(
		c.SolutionEnablerDID,
		c.ContractDID,
		[]ucan.Capability{behaviors.TokenomicNamespace, behaviors.ContractStatusBehavior, behaviors.ContractProposeBehavior}, contractActorCapsLifespan)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities: %w", err)
	}

	return nil
}

func (c *ContractActor) setupPaymentValidatorBehaviorAndCapabilities(paymentValidatorDID did.DID) error {
	err := c.Security().Grant(
		paymentValidatorDID,
		c.ContractDID,
		[]ucan.Capability{behaviors.ContractPaymentValidateBehavior},
		contractActorCapsLifespan,
	)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities to Provider: %w", err)
	}

	return nil
}

// Setup contract participants capabilities
func (c *ContractActor) SetupParticipantsCapabilities(participants contracts.ContractParticipants) error {
	// Grant capabilities to the primary party
	err := c.Security().Grant(
		participants.Provider,
		c.ContractDID,
		[]ucan.Capability{
			behaviors.ContractEventsBehavior,
			behaviors.ContractTerminationBehavior,
			behaviors.ContractCompleteBehavior,
			behaviors.ContractStatusBehavior,
			behaviors.ContractSettleBehavior,
			behaviors.ContractValidationBehavior,
			behaviors.ContractSignBehavior,
			behaviors.ContractPaymentValidateBehavior,
		},
		contractActorCapsLifespan,
	)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities to Provider: %w", err)
	}

	// Grant capabilities to the secondary party
	err = c.Security().Grant(
		participants.Requestor,
		c.ContractDID,
		[]ucan.Capability{
			behaviors.ContractEventsBehavior,
			behaviors.ContractTerminationBehavior,
			behaviors.ContractCompleteBehavior,
			behaviors.ContractStatusBehavior,
			behaviors.ContractSettleBehavior,
			behaviors.ContractValidationBehavior,
			behaviors.ContractSignBehavior,
			behaviors.ContractPaymentValidateBehavior,
		},
		contractActorCapsLifespan,
	)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities to Requestor: %w", err)
	}

	return nil
}

// getContractBehaviors returns a map of behavior names to handler functions
func (c *ContractActor) getContractBehaviors() map[string]struct {
	fn   func(actor.Envelope)
	opts []actor.BehaviorOption
} {
	contractBehaviors := map[string]struct {
		fn   func(actor.Envelope)
		opts []actor.BehaviorOption
	}{
		behaviors.ContractTerminationBehavior: {
			fn: c.handleContractTermination,
		},
		behaviors.ContractCompleteBehavior: {
			fn: c.handleCompleteContract,
		},
		behaviors.ContractSettleBehavior: {
			fn: c.handleSettleContract,
		},
		behaviors.ContractStatusBehavior: {
			fn: c.handleContractState,
		},
		behaviors.ContractValidationBehavior: {
			fn: c.handleContractValidation,
		},
		behaviors.ContractSignBehavior: {
			fn: c.handleContractSignByParticipants,
		},
		behaviors.ContractPaymentValidateBehavior: {
			fn: c.handlePaymentValidate,
		},
		behaviors.ContractEventsBehavior: {
			fn: c.handleContractEvents,
		},
	}

	return contractBehaviors
}

func (c *ContractActor) handleContractEvents(msg actor.Envelope) {
	defer msg.Discard()
	resp := contracts.ContractEventResponse{}

	var req contracts.ContractEventRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Errorf("handleContractEvents: failed to unmarshal ContractEventRequest: %v", err)
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}
	log.Infof("handleContractEvents: received contract event request")

	// Extract event type, provider DID, and Head Contract DID
	var eventType events.EventType
	var providerDID string
	var headContractDID string
	var eventMap map[string]interface{}
	if err := json.Unmarshal(req.Payload, &eventMap); err == nil {
		if typeStr, ok := eventMap["type"].(string); ok {
			eventType = events.EventType(typeStr)
		}
		// Extract provider DID from event payload if available
		if provDID, ok := eventMap["provider_did"].(string); ok {
			providerDID = provDID
		}
		// Extract Head Contract DID from event payload if available
		if hcDid, ok := eventMap["head_contract_did"].(string); ok {
			headContractDID = hcDid
		}
		log.Infof("handleContractEvents: extracted eventType=%s, providerDID=%s, headContractDID=%s from payload", eventType, providerDID, headContractDID)
	} else {
		log.Infof("handleContractEvents: could not unmarshal event payload as map: %v", err)
	}

	// If not in payload, use message sender (for backwards compatibility)
	if providerDID == "" {
		providerDID = msg.From.DID.String()
		log.Infof("handleContractEvents: providerDID not found in payload, using sender DID: %s", providerDID)
	}

	// Store with event_type, provider_did, and head_contract_did
	err := c.usageStore.AddUsageEvent(usage.Usage{
		ContractDID:     c.ContractDID.URI, // Tail Contract DID (this contract)
		HeadContractDID: headContractDID,   // Head Contract DID (if part of chain)
		ProviderDID:     providerDID,
		EventType:       eventType, // Optional - extracted from JSON if empty
		Data:            req.Payload,
	})
	if err != nil {
		log.Errorf("handleContractEvents: failed to add usage event: %v", err)
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	log.Infof("handleContractEvents: usage event stored successfully for contract %s", c.ContractDID.URI)
	c.sendReply(msg, resp)
}

func (c *ContractActor) handlePaymentValidate(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.PaymentValidateResponse{}
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	contract.Paid = true
	err = c.contractStore.Upsert(contract)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	c.sendReply(msg, resp)
}

func (c *ContractActor) handleContractSignByParticipants(msg actor.Envelope) {
	defer msg.Discard()
	resp := contracts.ContractSignResponse{}

	var req contracts.ContractSignRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	// check if its already signed
	signed := false
	for _, v := range contract.Signatures {
		if v.DID == msg.From.DID {
			signed = true
			break
		}
	}

	if signed {
		resp.Error = "contract already signed"
		c.sendReply(msg, resp)
		return
	}

	if contract.ContractParticipants.Provider == msg.From.DID || contract.ContractParticipants.Requestor == msg.From.DID {
		contract.Signatures = append(contract.Signatures, contracts.Signature{
			DID:        msg.From.DID,
			Signatures: req.Signature,
		})

		// if both sigs available mark the contract as accepted
		// no need additional checks since the capabilities define
		// who can access this contract so in this case only participants can sign
		if len(contract.Signatures) == 2 {
			contract.CurrentState = contracts.ContractAccepted
			contract.Transitions = []contracts.StateTransition{
				{
					FromState: contracts.ContractDraft,
					ToState:   contracts.ContractAccepted,
					Event:     contracts.EventAccepted,
					Timestamp: time.Now(),
				},
			}
		}

		err := c.contractStore.Upsert(contract)
		if err != nil {
			resp.Error = err.Error()
			c.sendReply(msg, resp)
			return
		}

		resp.Contract = *contract
		c.sendReply(msg, resp)
		return
	}

	resp.Error = "not allowed to sign this contract"
	c.sendReply(msg, resp)
}

func (c *ContractActor) handleContractState(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractStatusResponse{}
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	resp.Contract = *contract
	c.sendReply(msg, resp)
}

// sendReply sends a reply to the given message envelope with the provided payload
func (c *ContractActor) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(c.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		return
	}

	_ = c.Send(reply)
}

func (c *ContractActor) handleContractTermination(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractTerminationResponse{}
	savedContract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	if !savedContract.TerminationOption.Allowed {
		resp.Error = "contract is not allowed to be terminated"
		c.sendReply(msg, resp)
		return
	}

	var lastTransition contracts.ContractState
	if len(savedContract.Transitions) > 0 {
		lastTransition = savedContract.Transitions[len(savedContract.Transitions)-1].ToState
	}

	savedContract.CurrentState = contracts.ContractTerminated
	savedContract.Transitions = append(savedContract.Transitions, contracts.StateTransition{
		FromState:   lastTransition,
		ToState:     contracts.ContractTerminated,
		Timestamp:   time.Now(),
		Event:       contracts.EventTerminate,
		InitiatedBy: msg.From.DID,
	})

	err = c.contractStore.Upsert(savedContract)
	if err != nil {
		resp.Error = err.Error()
	}

	c.sendReply(msg, resp)
}

func (c *ContractActor) handleCompleteContract(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractCompletionResponse{}
	savedContract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	var lastTransition contracts.ContractState
	if len(savedContract.Transitions) > 0 {
		lastTransition = savedContract.Transitions[len(savedContract.Transitions)-1].ToState
	}

	savedContract.CurrentState = contracts.ContractCompleted
	savedContract.Transitions = append(savedContract.Transitions, contracts.StateTransition{
		FromState:   lastTransition,
		ToState:     contracts.ContractCompleted,
		Timestamp:   time.Now(),
		Event:       contracts.EventComplete,
		InitiatedBy: msg.From.DID,
	})

	err = c.contractStore.Upsert(savedContract)
	if err != nil {
		resp.Error = err.Error()
	}

	c.sendReply(msg, resp)
}

func (c *ContractActor) handleSettleContract(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractSettleResponse{}
	savedContract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	var lastTransition contracts.ContractState
	if len(savedContract.Transitions) > 0 {
		lastTransition = savedContract.Transitions[len(savedContract.Transitions)-1].ToState
	}

	savedContract.CurrentState = contracts.ContractSettled
	savedContract.Transitions = append(savedContract.Transitions, contracts.StateTransition{
		FromState:   lastTransition,
		ToState:     contracts.ContractSettled,
		Timestamp:   time.Now(),
		Event:       contracts.EventSettle,
		InitiatedBy: msg.From.DID,
	})

	err = c.contractStore.Upsert(savedContract)
	if err != nil {
		resp.Error = err.Error()
	}

	c.sendReply(msg, resp)
}

func (c *ContractActor) handleContractValidation(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractValidateResponse{}
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}
	resp.CurrentStatus = string(contract.CurrentState)

	if contract.CurrentState == contracts.ContractAccepted || contract.CurrentState == contracts.ContractActive {
		resp.Valid = true
	}

	c.sendReply(msg, resp)
}

func (c *ContractActor) sendFixedRentalInvoice(
	contract *contracts.Contract,
	fixedRentalUsage *contracts.FixedRentalUsage,
	now time.Time,
) {
	// Create ContractUsageRequest
	req := contracts.ContractUsageRequest{
		UniqueID:           uuid.NewString(),
		Contract:           *contract,
		Usages:             fixedRentalUsage.PeriodsInvoiced,
		FixedRentalDetails: fixedRentalUsage,
	}

	// Forward invoice using solution enabler's actor (contract host node actor)
	// This ensures the message is sent from the solution enabler's actor handle, which has the required capabilities
	if c.forwardInvoice == nil {
		log.Errorw("forwardInvoice function not set for contract actor",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	if err := c.forwardInvoice(req); err != nil {
		log.Errorw("failed to forward fixed rental invoice to payment validator",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return
	}

	// Update last processed timestamp after successful send
	// Note: If payment validator processing fails, we'll catch it on next cycle
	err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, now)
	if err != nil {
		log.Errorw("failed to save last processed timestamp after fixed rental invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
	} else {
		log.Infow("fixed rental invoice generated and sent successfully",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"periods_invoiced", fixedRentalUsage.PeriodsInvoiced,
			"amount", fixedRentalUsage.Amount)
	}
}

// checkAndGenerateInvoice is the unified invoice checking and generation logic
// Returns true if the billing routine should stop (contract terminated/completed), false otherwise.
func (c *ContractActor) checkAndGenerateInvoice() bool {
	// Get current contract state
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get contract for automatic billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	// Skip all billing if explicitly disabled (Contract A)
	// Organization contracts (Contract A) will be billed manually by the organization
	// outside the contract system
	if contract.DisableBilling {
		log.Debugw("skipping billing (disabled by contract flag)",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return true // Stop billing routine
	}

	// Get processor for this payment model
	processor, err := contracts.GetPaymentModelProcessor(contract.PaymentDetails.PaymentModel)
	if err != nil {
		log.Errorw("failed to get payment model processor",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return true // Stop routine if processor not found
	}

	// Defensive check: Only process contracts that support automatic billing
	if !processor.SupportsAutomaticBilling() {
		return true // Stop routine if payment model doesn't support automatic billing
	}

	// Check if contract is terminated - generate final invoice
	if contract.CurrentState == contracts.ContractTerminated {
		return c.handleTerminatedContractInvoice(contract, processor)
	}

	// Check if contract is completed or expired
	if contract.CurrentState == contracts.ContractCompleted ||
		time.Now().After(contract.Duration.EndDate) {
		return true // Stop billing routine
	}

	// Get last invoice timestamp
	lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get last processed timestamp for automatic billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	now := time.Now()

	// Initialize lastInvoiceAt if zero (first invoice)
	// For contracts that start mid-period, first invoice should cover partial period
	// by using contract start date as baseline.
	// For automatic-only models (FixedRental, Periodic), we save the initialization
	// to prevent infinite re-initialization loop. For models that support manual billing,
	// we don't save to avoid interfering with manual collection which should start from zero.
	unixEpoch := time.Unix(0, 0)
	if lastInvoiceAt.IsZero() || lastInvoiceAt.Equal(unixEpoch) {
		// For first invoice calculation, use contract start date to ensure partial period coverage
		// This ensures no usage is lost if contract starts mid-period
		lastInvoiceAt = contract.Duration.StartDate
		// Save initialization timestamp for all automatic billing models
		// This prevents infinite re-initialization loop.
		// Manual collection can still work correctly - it will use this saved timestamp
		// when querying, which is correct because it represents the last time automatic
		// billing processed usage. Manual collection queries from lastProcessedAt to now,
		// so if it's the first manual collection, it will still capture all usage from
		// contract start date to now.
		if err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, lastInvoiceAt); err != nil {
			log.Errorw("failed to save initial timestamp for automatic billing",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
		}
		// Return false to continue checking - not enough time has passed yet for first invoice
		return false
	}

	// Use processor to check and generate invoice
	usageData, err := processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, processors.ErrPeriodNotElapsed) {
			// Not enough time has passed yet, check again next time
			return false // Continue checking
		}
		if errors.Is(err, processors.ErrNoDeployments) {
			// No deployments during period - skip invoice and update timestamp
			// This is specific to Periodic model
			if contract.PaymentDetails.PaymentModel == contracts.Periodic {
				// Update lastInvoiceAt to skip this period
				periodDuration, err := parsePaymentPeriod(contract.PaymentDetails.PaymentPeriod)
				if err == nil {
					paymentPeriodCount := contract.PaymentDetails.PaymentPeriodCount
					if paymentPeriodCount <= 0 {
						paymentPeriodCount = 1
					}
					billingCycleDuration := periodDuration * time.Duration(paymentPeriodCount)
					nextPeriodStart := lastInvoiceAt.Add(billingCycleDuration)
					if err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, nextPeriodStart); err != nil {
						log.Errorw("failed to update last processed timestamp after skipping period",
							"labels", string(observability.LabelContract),
							"contract_did", c.ContractDID.URI,
							"error", err)
					}
				}
			}
			return false // Continue checking for next period
		}
		// Other error occurred - log but continue checking (might be transient error)
		log.Errorw("failed to check and generate invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	if usageData != nil {
		// Invoice should be generated - send it
		c.sendInvoice(contract, usageData, now)
		return false // Continue billing routine
	}

	return false
}

// CheckAndGenerateInvoice is the public method for checking and generating invoices
// This is called by the centralized billing scheduler
// Returns an error if the contract is terminated or completed, signaling the scheduler to unregister
func (c *ContractActor) CheckAndGenerateInvoice() error {
	shouldStop := c.checkAndGenerateInvoice()
	if shouldStop {
		// Contract terminated/completed - scheduler will detect this and unregister
		return fmt.Errorf("contract terminated or completed")
	}
	return nil
}

// RegisterBilling registers this contract actor with the billing scheduler
// This method is called by the Node after creating the actor
func (c *ContractActor) RegisterBilling(scheduler *ContractBillingScheduler) error {
	return scheduler.RegisterContract(c.ContractDID)
}

// UnregisterBilling unregisters this contract from billing
func (c *ContractActor) UnregisterBilling(scheduler *ContractBillingScheduler) {
	scheduler.UnregisterContract(c.ContractDID)
}

// handleTerminatedContractInvoice handles final invoice generation for terminated contracts
func (c *ContractActor) handleTerminatedContractInvoice(contract *contracts.Contract, processor contracts.PaymentModelProcessor) bool {
	lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get last processed timestamp for terminated contract final invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return true // Stop billing routine
	}

	if lastInvoiceAt.IsZero() {
		// No previous invoice, nothing to invoice
		return true // Stop billing routine
	}

	// Get termination timestamp from contract transitions
	// This ensures we pro-rate based on actual termination time, not when billing runs
	var terminationTime time.Time
	if contract.Transitions != nil {
		for i := len(contract.Transitions) - 1; i >= 0; i-- {
			if contract.Transitions[i].ToState == contracts.ContractTerminated {
				terminationTime = contract.Transitions[i].Timestamp
				break
			}
		}
	}

	// Use termination time if available, otherwise use current time as fallback
	endTime := terminationTime
	if terminationTime.IsZero() {
		endTime = time.Now()
	}

	elapsed := endTime.Sub(lastInvoiceAt)
	if elapsed <= 0 {
		// No elapsed time, nothing to invoice
		return true // Stop billing routine
	}

	// Try to generate regular invoice first
	usageData, err := processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, endTime)
	if err != nil {
		// If period not elapsed, try pro-rated invoice for periodic and fixed rental contracts
		if errors.Is(err, processors.ErrPeriodNotElapsed) {
			var proRatedUsageData *contracts.UsageData
			var proRateErr error
			var paymentModel string

			switch contract.PaymentDetails.PaymentModel {
			case contracts.Periodic:
				// Pro-rate based on actual deployment time within the partial period
				if periodicProcessor, ok := processor.(*processors.PeriodicProcessor); ok {
					proRatedUsageData, proRateErr = periodicProcessor.GenerateProRatedInvoice(contract, lastInvoiceAt, endTime)
					paymentModel = "periodic"
				}
			case contracts.FixedRental:
				// Pro-rate based on elapsed time ratio to billing cycle
				if fixedRentalProcessor, ok := processor.(*processors.FixedRentalProcessor); ok {
					proRatedUsageData, proRateErr = fixedRentalProcessor.GenerateProRatedInvoice(contract, lastInvoiceAt, endTime)
					paymentModel = "fixed_rental"
				}
			}

			if proRateErr != nil {
				// Pro-rating also failed
				log.Warnw("could not generate pro-rated invoice for terminated contract",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"payment_model", paymentModel,
					"error", proRateErr)
				return true // Stop billing routine
			}

			if proRatedUsageData != nil {
				// Pro-rated invoice generated - send it
				var amount string
				switch contract.PaymentDetails.PaymentModel {
				case contracts.Periodic:
					amount = proRatedUsageData.Data.(*contracts.PeriodicUsage).Amount
				case contracts.FixedRental:
					amount = proRatedUsageData.Data.(*contracts.FixedRentalUsage).Amount
				}
				log.Infow("generated pro-rated invoice for terminated contract",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"payment_model", paymentModel,
					"amount", amount,
					"termination_time", terminationTime,
					"elapsed_since_last_invoice", elapsed)
				c.sendInvoice(contract, proRatedUsageData, endTime)
				return true // Stop billing routine after final invoice
			}
		}
		// For other errors or payment models, just log and stop
		log.Warnw("could not generate final invoice for terminated contract",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return true // Stop billing routine
	}

	if usageData != nil {
		// Final invoice generated - send it
		c.sendInvoice(contract, usageData, endTime)
	}

	return true // Stop billing routine after final invoice
}

// sendInvoice sends invoice using the appropriate format based on payment model
func (c *ContractActor) sendInvoice(contract *contracts.Contract, usageData *contracts.UsageData, now time.Time) {
	switch contract.PaymentDetails.PaymentModel {
	case contracts.FixedRental:
		c.sendFixedRentalInvoiceFromUsageData(contract, usageData, now)
	case contracts.Periodic:
		c.sendPeriodicInvoiceFromUsageData(contract, usageData, now)
	case contracts.PayPerAllocation, contracts.PayPerDeployment, contracts.PayPerTimeUtilization, contracts.PayPerResourceUtilization:
		// For these models, convert UsageData to ContractUsageRequest and forward via forwardInvoice
		c.sendGenericInvoiceFromUsageData(contract, usageData, now)
	default:
		log.Errorw("unsupported payment model for automatic billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"payment_model", contract.PaymentDetails.PaymentModel)
	}
}

// sendGenericInvoiceFromUsageData sends invoice for PayPerAllocation, PayPerDeployment, PayPerTimeUtilization, PayPerResourceUtilization
// by converting UsageData to ContractUsageRequest and forwarding via forwardInvoice
func (c *ContractActor) sendGenericInvoiceFromUsageData(contract *contracts.Contract, usageData *contracts.UsageData, now time.Time) {
	req := contracts.ContractUsageRequest{
		UniqueID: uuid.NewString(),
		Contract: *contract,
	}

	// Convert UsageData to ContractUsageRequest format
	switch usageData.PaymentModel {
	case contracts.PayPerAllocation:
		usageCount, ok := usageData.Data.(int)
		if !ok {
			log.Errorw("invalid usage data type for pay_per_allocation",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}
		req.Usages = usageCount

	case contracts.PayPerDeployment:
		usageCount, ok := usageData.Data.(int)
		if !ok {
			log.Errorw("invalid usage data type for pay_per_deployment",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}
		req.Usages = usageCount

	case contracts.PayPerTimeUtilization:
		timeUtil, ok := usageData.Data.(*contracts.TimeUtilizationUsage)
		if !ok {
			log.Errorw("invalid usage data type for pay_per_time_utilization",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}
		req.TimeUtilization = timeUtil
		req.Usages = len(timeUtil.Deployments) // For backward compatibility

	case contracts.PayPerResourceUtilization:
		resourceUtil, ok := usageData.Data.(*contracts.ResourceUtilizationUsage)
		if !ok {
			log.Errorw("invalid usage data type for pay_per_resource_utilization",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}
		req.ResourceUtilization = resourceUtil
		req.Usages = len(resourceUtil.Deployments) // For backward compatibility

	default:
		log.Errorw("unsupported payment model for generic invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"payment_model", usageData.PaymentModel)
		return
	}

	// Forward invoice using solution enabler's actor (contract host node actor)
	if c.forwardInvoice == nil {
		log.Errorw("forwardInvoice function not set for contract actor",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	if err := c.forwardInvoice(req); err != nil {
		log.Errorw("failed to forward invoice to payment validator",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"payment_model", usageData.PaymentModel,
			"error", err)
		return
	}

	// Update last processed timestamp after successful send
	err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, now)
	if err != nil {
		log.Errorw("failed to save last processed timestamp after invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
	} else {
		log.Infow("invoice generated and sent successfully",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"payment_model", usageData.PaymentModel,
			"usages", req.Usages)
	}
}

// sendFixedRentalInvoiceFromUsageData sends fixed rental invoice from UsageData
func (c *ContractActor) sendFixedRentalInvoiceFromUsageData(contract *contracts.Contract, usageData *contracts.UsageData, now time.Time) {
	fixedRentalUsage, ok := usageData.Data.(*contracts.FixedRentalUsage)
	if !ok {
		log.Errorw("invalid usage data type for fixed rental",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	// Use existing sendFixedRentalInvoice function
	c.sendFixedRentalInvoice(contract, fixedRentalUsage, now)
}

// sendPeriodicInvoiceFromUsageData sends periodic invoice from UsageData
func (c *ContractActor) sendPeriodicInvoiceFromUsageData(contract *contracts.Contract, usageData *contracts.UsageData, now time.Time) {
	periodicUsage, ok := usageData.Data.(*contracts.PeriodicUsage)
	if !ok {
		log.Errorw("invalid usage data type for periodic",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	// Use existing sendPeriodicInvoice function
	c.sendPeriodicInvoice(contract, periodicUsage, now)
}

// sendPeriodicInvoice sends periodic invoice(s) - one per deployment (Edge Case 5)
func (c *ContractActor) sendPeriodicInvoice(
	contract *contracts.Contract,
	periodicUsage *contracts.PeriodicUsage,
	now time.Time,
) {
	pd := contract.PaymentDetails

	// Parse fee per time unit
	feePerUnit, err := strconv.ParseFloat(pd.FeePerTimeUnit, 64)
	if err != nil {
		log.Errorw("failed to parse fee_per_time_unit for periodic invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return
	}

	// Edge Case 5: Generate one invoice per deployment for the period
	for _, deployment := range periodicUsage.Deployments {
		// Calculate amount for this deployment
		var timeInUnit float64
		deploymentTimeSec := deployment.TotalUtilizationSec
		switch pd.TimeUnit {
		case contracts.TimeUnitSecond:
			timeInUnit = deploymentTimeSec
		case contracts.TimeUnitMinute:
			timeInUnit = deploymentTimeSec / 60.0
		case contracts.TimeUnitHour:
			timeInUnit = deploymentTimeSec / 3600.0
		default:
			log.Errorw("unsupported time_unit for periodic invoice",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"time_unit", pd.TimeUnit)
			continue // Skip this deployment
		}

		deploymentAmount := feePerUnit * timeInUnit

		// Edge Case 4: Use deployment stop time as period end
		// The deployment stop time is already accounted for in the runtime calculation
		// via CalculateDeploymentTimeUtilizationByContract, so we use periodicUsage.PeriodEnd
		// which is set correctly based on whether deployment stopped during the period
		periodEnd := periodicUsage.PeriodEnd

		// Generate unique ID for this deployment's invoice
		// Note: UUID will be generated in PeriodicProcessor.CalculatePayment,
		// this is just for the request identifier
		uniqueID := uuid.NewString()

		// Create PeriodicUsage for this single deployment
		deploymentPeriodicUsage := &contracts.PeriodicUsage{
			PeriodStart:     periodicUsage.PeriodStart,
			PeriodEnd:       periodEnd, // Use deployment stop time if applicable
			LastInvoiceAt:   periodicUsage.LastInvoiceAt,
			Deployments:     []contracts.DeploymentTimeUtilization{deployment}, // Single deployment
			TotalTimeSec:    deploymentTimeSec,
			Amount:          fmt.Sprintf("%.8f", deploymentAmount),
			PeriodsInvoiced: periodicUsage.PeriodsInvoiced,
		}

		// Create ContractUsageRequest for this deployment
		req := contracts.ContractUsageRequest{
			UniqueID:        uniqueID,
			Contract:        *contract,
			PeriodicDetails: deploymentPeriodicUsage,
		}

		log.Infow("generating periodic invoice for deployment",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"deployment_id", deployment.DeploymentID,
			"period_start", periodicUsage.PeriodStart,
			"period_end", periodEnd,
			"amount", deploymentPeriodicUsage.Amount,
			"runtime_sec", deploymentTimeSec)

		// Forward invoice using solution enabler's actor (contract host node actor)
		// This ensures the message is sent from the solution enabler's actor handle, which has the required capabilities
		if c.forwardInvoice == nil {
			log.Errorw("forwardInvoice function not set for contract actor",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			continue // Continue with other deployments
		}

		// Send invoice using forwardInvoice function
		if err := c.forwardInvoice(req); err != nil {
			log.Errorw("failed to send periodic invoice for deployment",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"deployment_id", deployment.DeploymentID,
				"error", err)
			// Continue with other deployments even if one fails
			continue
		}
	}

	// Update last invoice timestamp after all deployment invoices are sent
	if err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, now); err != nil {
		log.Errorw("failed to update last processed timestamp after sending periodic invoices",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		// Don't return - invoices were sent successfully
	} else {
		log.Infow("periodic invoices generated and sent successfully",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"deployments", len(periodicUsage.Deployments))
	}
}
