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
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	contractstore "gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

const (
	contractActorCapsLifespan = 7 * 24 * time.Hour
	contractPeriodicChecker   = 5 * time.Minute
)

// FixedRentalBillingCheckerInterval controls how often the contract actor checks
// whether a Fixed Rental invoice should be generated.
// In production this is set to 15 minutes to balance timeliness and load.
var FixedRentalBillingCheckerInterval = 15 * time.Minute

// PeriodicBillingCheckerInterval controls how often the contract actor checks
// whether a Periodic invoice should be generated.
// Set to 1 minute for testing, will be changed to 15 minutes after E2E tests pass.
var PeriodicBillingCheckerInterval = 1 * time.Minute

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

	// Fixed Rental and Periodic billing routines (only start for applicable contracts)
	// Check payment model before starting the billing routine to avoid unnecessary
	// goroutines for contracts with other payment models
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get contract to check payment model for billing routine",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		// Continue - billing routine won't start, but actor still starts
		return nil
	}

	switch contract.PaymentDetails.PaymentModel {
	case contracts.FixedRental:
		go c.startFixedRentalBilling()
		log.Infow("started fixed rental billing routine",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
	case contracts.Periodic:
		go c.startPeriodicBilling()
		log.Infow("started periodic billing routine",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
	}

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
		[]ucan.Capability{behaviors.ContractEventsBehavior, behaviors.ContractTerminationBehavior, behaviors.ContractCompleteBehavior, behaviors.ContractStatusBehavior, behaviors.ContractSettleBehavior, behaviors.ContractValidationBehavior, behaviors.ContractSignBehavior},
		contractActorCapsLifespan,
	)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities to Provider: %w", err)
	}

	// Grant capabilities to the secondary party
	err = c.Security().Grant(
		participants.Requestor,
		c.ContractDID,
		[]ucan.Capability{behaviors.ContractEventsBehavior, behaviors.ContractTerminationBehavior, behaviors.ContractCompleteBehavior, behaviors.ContractStatusBehavior, behaviors.ContractSettleBehavior, behaviors.ContractValidationBehavior, behaviors.ContractSignBehavior},
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
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

	// Extract event type for efficient indexing (optional - AddUsageEvent will extract if not provided)
	var eventType events.EventType
	var eventMap map[string]interface{}
	if err := json.Unmarshal(req.Payload, &eventMap); err == nil {
		if typeStr, ok := eventMap["type"].(string); ok {
			eventType = events.EventType(typeStr)
		}
	}

	// Store with event_type (will be extracted from JSON if not provided)
	err := c.usageStore.AddUsageEvent(usage.Usage{
		ContractDID: c.ContractDID.URI,
		EventType:   eventType, // Optional - extracted from JSON if empty
		Data:        req.Payload,
	})
	if err != nil {
		resp.Error = err.Error()
		c.sendReply(msg, resp)
		return
	}

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

// parsePaymentPeriod converts a payment period string to a time.Duration
func parsePaymentPeriod(period string) (time.Duration, error) {
	switch period {
	case contracts.PaymentPeriodMinute:
		return time.Minute, nil
	case contracts.PaymentPeriodHour:
		return time.Hour, nil
	case contracts.PaymentPeriodDay:
		return 24 * time.Hour, nil
	case contracts.PaymentPeriodWeek:
		return 7 * 24 * time.Hour, nil
	case contracts.PaymentPeriodMonth:
		// Approximate: 30 days (could be enhanced to handle exact calendar months)
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid payment_period: %s", period)
	}
}

func (c *ContractActor) calculateFixedRentalInvoice(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
) (*contracts.FixedRentalUsage, error) {
	pd := contract.PaymentDetails

	// Parse payment period
	periodDuration, err := parsePaymentPeriod(pd.PaymentPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid payment_period: %w", err)
	}

	// Calculate elapsed time
	elapsed := now.Sub(lastInvoiceAt)

	// Calculate number of periods that have elapsed
	periodsElapsed := int(elapsed / periodDuration)

	// Get payment_period_count (default to 1 if not set or 0)
	paymentPeriodCount := pd.PaymentPeriodCount
	if paymentPeriodCount <= 0 {
		paymentPeriodCount = 1 // Default: invoice every period
	}

	// Only invoice when enough periods have elapsed (paymentPeriodCount periods)
	// Calculate how many billing cycles have elapsed
	billingCyclesElapsed := periodsElapsed / paymentPeriodCount
	if billingCyclesElapsed < 1 {
		// Not enough periods have elapsed for a billing cycle
		return nil, ErrPeriodNotElapsed
	}

	// Parse fixed rental amount (amount per invoice, not per period)
	// With paymentPeriodCount, we invoice this fixed amount every N periods
	fixedRentalAmount, err := strconv.ParseFloat(pd.FixedRentalAmount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid fixed_rental_amount: %w", err)
	}

	// Calculate total amount for all elapsed billing cycles
	// Each billing cycle invoices for the fixedRentalAmount
	// Example: fixedRentalAmount=10, paymentPeriodCount=2 means invoice 10.00 every 2 periods
	totalAmount := fixedRentalAmount * float64(billingCyclesElapsed)

	// Calculate period boundaries
	// Periods invoiced represents the number of periods covered by all billing cycles
	periodsToInvoice := billingCyclesElapsed * paymentPeriodCount
	periodStart := lastInvoiceAt.Truncate(periodDuration)
	if periodStart.Before(lastInvoiceAt) {
		periodStart = periodStart.Add(periodDuration)
	}

	// Calculate period end based on periods actually invoiced
	periodEnd := periodStart.Add(periodDuration * time.Duration(periodsToInvoice))

	return &contracts.FixedRentalUsage{
		PeriodsInvoiced: periodsToInvoice,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Amount:          fmt.Sprintf("%.8f", totalAmount),
		LastInvoiceAt:   lastInvoiceAt,
	}, nil
}

// calculateProRatedInvoiceForTermination calculates a pro-rated invoice for a terminated contract.
// This function always generates a pro-rated invoice for ANY elapsed time, regardless of billing cycle.
// Unlike calculateProRatedInvoice, this does not return ErrFullPeriodElapsed and always pro-rates.
func (c *ContractActor) calculateProRatedInvoiceForTermination(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
	periodDuration time.Duration,
) (*contracts.FixedRentalUsage, error) {
	pd := contract.PaymentDetails

	// Calculate elapsed time since last invoice
	elapsed := now.Sub(lastInvoiceAt)

	// Parse fixed rental amount (amount per invoice)
	fixedRentalAmount, err := strconv.ParseFloat(pd.FixedRentalAmount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid fixed_rental_amount: %w", err)
	}

	// Get payment_period_count (default to 1 if not set or 0)
	paymentPeriodCount := pd.PaymentPeriodCount
	if paymentPeriodCount <= 0 {
		paymentPeriodCount = 1
	}

	// Calculate billing cycle duration
	billingCycleDuration := periodDuration * time.Duration(paymentPeriodCount)

	// For terminated contracts, always pro-rate based on billing cycle
	// Pro-rate based on what portion of the billing cycle has elapsed
	proRatedRatio := float64(elapsed) / float64(billingCycleDuration)
	proRatedAmount := fixedRentalAmount * proRatedRatio

	// Ensure we don't generate negative or zero amounts for very small elapsed times
	if proRatedAmount <= 0 {
		// If elapsed time is negligible or zero, return nil (no invoice needed)
		return nil, nil
	}

	// Calculate period boundaries
	periodStart := lastInvoiceAt.Truncate(periodDuration)
	if periodStart.Before(lastInvoiceAt) {
		periodStart = periodStart.Add(periodDuration)
	}

	return &contracts.FixedRentalUsage{
		PeriodsInvoiced: 1, // Single pro-rated period
		PeriodStart:     periodStart,
		PeriodEnd:       now,
		Amount:          fmt.Sprintf("%.8f", proRatedAmount),
		LastInvoiceAt:   lastInvoiceAt,
	}, nil
}

func (c *ContractActor) calculatePeriodicInvoice(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
) (*contracts.PeriodicUsage, error) {
	pd := contract.PaymentDetails

	// Parse payment period
	periodDuration, err := parsePaymentPeriod(pd.PaymentPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid payment_period: %w", err)
	}

	// Calculate elapsed time
	elapsed := now.Sub(lastInvoiceAt)

	// Calculate number of periods that have elapsed
	periodsElapsed := int(elapsed / periodDuration)

	// Get payment_period_count (default to 1 if not set or 0)
	paymentPeriodCount := pd.PaymentPeriodCount
	if paymentPeriodCount <= 0 {
		paymentPeriodCount = 1
	}

	// Only invoice when enough periods have elapsed (paymentPeriodCount periods)
	billingCyclesElapsed := periodsElapsed / paymentPeriodCount
	if billingCyclesElapsed < 1 {
		// Not enough periods have elapsed for a billing cycle
		return nil, ErrPeriodNotElapsed
	}

	// Calculate billing period boundaries
	periodStart := lastInvoiceAt.Truncate(periodDuration)
	if periodStart.Before(lastInvoiceAt) {
		periodStart = periodStart.Add(periodDuration)
	}

	periodsToInvoice := billingCyclesElapsed * paymentPeriodCount
	periodEnd := periodStart.Add(periodDuration * time.Duration(periodsToInvoice))

	// Calculate deployment runtime for this billing period
	deployments, err := c.usageStore.CalculateDeploymentTimeUtilizationByContract(
		contract.ContractDID,
		periodStart,
		periodEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate deployment time utilization: %w", err)
	}

	// Edge Case 1: No deployments during period - skip invoice
	if len(deployments) == 0 {
		return nil, ErrNoDeployments
	}

	// Calculate total time across all deployments
	var totalTimeSec float64
	for _, deployment := range deployments {
		totalTimeSec += deployment.TotalUtilizationSec
	}

	// If totalTimeSec is zero or negative, skip invoice
	if totalTimeSec <= 0 {
		return nil, ErrNoDeployments
	}

	// Parse fee per time unit
	feePerUnit, err := strconv.ParseFloat(pd.FeePerTimeUnit, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid fee_per_time_unit: %w", err)
	}

	// Convert total time to the specified time unit
	var timeInUnit float64
	switch pd.TimeUnit {
	case contracts.TimeUnitSecond:
		timeInUnit = totalTimeSec
	case contracts.TimeUnitMinute:
		timeInUnit = totalTimeSec / 60.0
	case contracts.TimeUnitHour:
		timeInUnit = totalTimeSec / 3600.0
	default:
		return nil, fmt.Errorf("unsupported time_unit: %s", pd.TimeUnit)
	}

	// Calculate total amount (sum across all deployments for this period)
	// Note: Each deployment will get its own invoice, but this provides the combined total
	totalAmount := feePerUnit * timeInUnit

	return &contracts.PeriodicUsage{
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		LastInvoiceAt:   lastInvoiceAt,
		Deployments:     deployments,
		TotalTimeSec:    totalTimeSec,
		Amount:          fmt.Sprintf("%.8f", totalAmount),
		PeriodsInvoiced: periodsToInvoice,
	}, nil
}

func (c *ContractActor) calculateProRatedPeriodicInvoiceForTermination(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
	periodDuration time.Duration,
) (*contracts.PeriodicUsage, error) {
	pd := contract.PaymentDetails

	// Calculate deployment runtime from last invoice to termination
	deployments, err := c.usageStore.CalculateDeploymentTimeUtilizationByContract(
		contract.ContractDID,
		lastInvoiceAt,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate deployment time utilization: %w", err)
	}

	// Edge Case 1: If no deployments, skip invoice
	if len(deployments) == 0 {
		return nil, nil // No invoice needed
	}

	// Calculate total time across all deployments
	var totalTimeSec float64
	for _, deployment := range deployments {
		totalTimeSec += deployment.TotalUtilizationSec
	}

	if totalTimeSec <= 0 {
		return nil, nil // No invoice needed
	}

	// Parse fee per time unit
	feePerUnit, err := strconv.ParseFloat(pd.FeePerTimeUnit, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid fee_per_time_unit: %w", err)
	}

	// Convert total time to the specified time unit
	var timeInUnit float64
	switch pd.TimeUnit {
	case contracts.TimeUnitSecond:
		timeInUnit = totalTimeSec
	case contracts.TimeUnitMinute:
		timeInUnit = totalTimeSec / 60.0
	case contracts.TimeUnitHour:
		timeInUnit = totalTimeSec / 3600.0
	default:
		return nil, fmt.Errorf("unsupported time_unit: %s", pd.TimeUnit)
	}

	// Calculate pro-rated amount
	proRatedAmount := feePerUnit * timeInUnit

	// Ensure we don't generate negative or zero amounts for very small elapsed times
	if proRatedAmount <= 0 {
		// If elapsed time is negligible or zero, return nil (no invoice needed)
		return nil, nil
	}

	// Calculate period boundaries
	periodStart := lastInvoiceAt.Truncate(periodDuration)
	if periodStart.Before(lastInvoiceAt) {
		periodStart = periodStart.Add(periodDuration)
	}

	return &contracts.PeriodicUsage{
		PeriodStart:     periodStart,
		PeriodEnd:       now,
		LastInvoiceAt:   lastInvoiceAt,
		Deployments:     deployments,
		TotalTimeSec:    totalTimeSec,
		Amount:          fmt.Sprintf("%.8f", proRatedAmount),
		PeriodsInvoiced: 1, // Single pro-rated period
	}, nil
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

// checkAndGenerateFixedRentalInvoice checks if an invoice is needed and generates it.
// Returns true if the billing routine should stop (contract terminated/completed), false otherwise.
func (c *ContractActor) checkAndGenerateFixedRentalInvoice() bool {
	// Get current contract state
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get contract for fixed rental billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	// Defensive check: Only process Fixed Rental contracts
	// Note: This should always be true since we only start this routine for Fixed Rental contracts,
	// but we keep it as a safety check in case payment model changes or for future flexibility
	if contract.PaymentDetails.PaymentModel != contracts.FixedRental {
		return true // Stop routine if payment model changed
	}

	// Check if contract is terminated - generate final invoice (pro-rated or regular)
	if contract.CurrentState == contracts.ContractTerminated {
		// Generate final invoice for elapsed time since last invoice
		// Logic:
		// 1. If elapsed < periodDuration: generate pro-rated invoice for partial period
		// 2. If elapsed >= periodDuration: generate regular invoice for full period(s)
		lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
		if err != nil {
			log.Errorw("failed to get last processed timestamp for terminated contract final invoice",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			return true // Stop billing routine - error getting last invoice, can't generate final invoice
		}

		if lastInvoiceAt.IsZero() {
			// No previous invoice, nothing to pro-rate
			return true // Stop billing routine - contract terminated, nothing to invoice
		}

		now := time.Now()
		elapsed := now.Sub(lastInvoiceAt)
		if elapsed <= 0 {
			// No elapsed time, nothing to invoice
			return true // Stop billing routine - contract terminated, nothing to invoice
		}

		periodDuration, err := parsePaymentPeriod(contract.PaymentDetails.PaymentPeriod)
		if err != nil {
			log.Errorw("failed to parse payment period for terminated contract final invoice",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			return true // Stop billing routine - can't parse payment period
		}

		// For terminated contracts, always generate an invoice for elapsed time
		// Try regular billing first (if enough periods have elapsed)
		fixedRentalUsage, err := c.calculateFixedRentalInvoice(contract, lastInvoiceAt, now)
		if err != nil {
			if !errors.Is(err, ErrPeriodNotElapsed) {
				// Error other than "period not elapsed" - log and try pro-rated fallback
				log.Warnw("failed to calculate regular invoice for terminated contract, falling back to pro-rated",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"error", err)
			}
			// Period not elapsed or other error - fall back to pro-rated invoice
			// For terminated contracts, we always generate a pro-rated invoice for any elapsed time
			proRatedUsage, err := c.calculateProRatedInvoiceForTermination(contract, lastInvoiceAt, now, periodDuration)
			if err != nil {
				log.Errorw("failed to calculate pro-rated invoice for terminated contract",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"error", err)
				return true // Stop billing routine - error calculating pro-rated invoice
			}

			if proRatedUsage != nil && proRatedUsage.Amount != "" {
				// Pro-rated invoice generated for partial elapsed time
				c.sendFixedRentalInvoice(contract, proRatedUsage, now)
			}
			// Stop billing routine after final invoice
			return true
		}

		if fixedRentalUsage != nil {
			// Regular invoice generated for full period(s) that have elapsed
			c.sendFixedRentalInvoice(contract, fixedRentalUsage, now)
			// Stop billing routine after final regular invoice
			return true
		}
	}

	// Check if contract is completed
	if contract.CurrentState == contracts.ContractCompleted {
		return true // Stop billing routine - contract is completed
	}

	// Check if contract has passed its end date
	if time.Now().After(contract.Duration.EndDate) {
		return true // Stop billing routine - contract has expired
	}

	// Get last invoice timestamp
	lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get last processed timestamp for fixed rental billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	now := time.Now()

	// If this is the first invoice, use current time as baseline
	// We don't invoice for time before the billing routine started tracking
	// Check for both zero time and Unix(0) which GetLastProcessedAt returns when no record exists
	unixEpoch := time.Unix(0, 0)
	if lastInvoiceAt.IsZero() || lastInvoiceAt.Equal(unixEpoch) {
		// For the first invoice, start from now (when billing routine starts)
		// This ensures we don't invoice for historical periods before the contract was active
		lastInvoiceAt = now
		// Save this as the initial timestamp to establish the baseline
		if err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, now); err != nil {
			log.Errorw("failed to save initial timestamp for fixed rental billing",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			// Continue anyway - will try again next check
		}
		// Return false to continue checking - not enough time has passed yet for first invoice
		return false
	}

	// Calculate if invoice is needed
	fixedRentalUsage, err := c.calculateFixedRentalInvoice(contract, lastInvoiceAt, now)
	if err != nil {
		if errors.Is(err, ErrPeriodNotElapsed) {
			// Not enough time has passed yet, check again next time
			return false // Continue checking
		}
		// Other error occurred - log but continue checking (might be transient error)
		log.Errorw("failed to calculate fixed rental invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	// Generate and send invoice
	c.sendFixedRentalInvoice(contract, fixedRentalUsage, now)

	// Continue running - invoice generated, but contract still active
	return false
}

func (c *ContractActor) startFixedRentalBilling() {
	// Check immediately on start to catch any invoices that should have been generated
	shouldStop := c.checkAndGenerateFixedRentalInvoice()
	if shouldStop {
		log.Infow("fixed rental billing routine stopping after initial check - contract terminated or completed",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	ticker := time.NewTicker(FixedRentalBillingCheckerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			log.Infow("fixed rental billing routine stopped",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		case <-ticker.C:
			// Check if we should stop (contract terminated or completed)
			shouldStop := c.checkAndGenerateFixedRentalInvoice()
			if shouldStop {
				log.Infow("fixed rental billing routine stopping - contract terminated or completed",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI)
				return
			}
		}
	}
}

func (c *ContractActor) startPeriodicBilling() {
	// Perform initial check immediately
	shouldStop := c.checkAndGeneratePeriodicInvoice()
	if shouldStop {
		log.Infow("periodic billing routine stopping - contract terminated or completed",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI)
		return
	}

	ticker := time.NewTicker(PeriodicBillingCheckerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			shouldStop := c.checkAndGeneratePeriodicInvoice()
			if shouldStop {
				log.Infow("periodic billing routine stopping - contract terminated or completed",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI)
				return
			}
		case <-c.ctx.Done():
			log.Infow("periodic billing routine stopping - context cancelled",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}
	}
}

// checkAndGeneratePeriodicInvoice checks if an invoice is needed and generates it.
// Returns true if the billing routine should stop (contract terminated/completed), false otherwise.
func (c *ContractActor) checkAndGeneratePeriodicInvoice() bool {
	// Get current contract state
	contract, err := c.contractStore.GetContract(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get contract for periodic billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	// Defensive check: Only process Periodic contracts
	if contract.PaymentDetails.PaymentModel != contracts.Periodic {
		return true // Stop routine if payment model changed
	}

	// Check if contract is terminated - generate final invoice (pro-rated or regular)
	if contract.CurrentState == contracts.ContractTerminated {
		// Generate final invoice for elapsed time since last invoice
		lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
		if err != nil {
			log.Errorw("failed to get last processed timestamp for terminated contract final invoice",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			return true // Stop billing routine - error getting last invoice
		}

		if lastInvoiceAt.IsZero() {
			// No previous invoice, nothing to invoice
			return true // Stop billing routine
		}

		now := time.Now()
		elapsed := now.Sub(lastInvoiceAt)
		if elapsed <= 0 {
			// No elapsed time, nothing to invoice
			return true // Stop billing routine
		}

		periodDuration, err := parsePaymentPeriod(contract.PaymentDetails.PaymentPeriod)
		if err != nil {
			log.Errorw("failed to parse payment period for terminated contract final invoice",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			return true // Stop billing routine
		}

		// For terminated contracts, always generate an invoice for elapsed time
		// Try regular billing first (if enough periods have elapsed)
		periodicUsage, err := c.calculatePeriodicInvoice(contract, lastInvoiceAt, now)
		if err != nil {
			if !errors.Is(err, ErrPeriodNotElapsed) && !errors.Is(err, ErrNoDeployments) {
				log.Warnw("failed to calculate regular invoice for terminated contract, falling back to pro-rated",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"error", err)
			}
			// Period not elapsed or no deployments - fall back to pro-rated invoice
			// For terminated contracts, we always generate a pro-rated invoice for any elapsed time
			proRatedUsage, err := c.calculateProRatedPeriodicInvoiceForTermination(contract, lastInvoiceAt, now, periodDuration)
			if err != nil {
				log.Errorw("failed to calculate pro-rated invoice for terminated contract",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"error", err)
				return true // Stop billing routine
			}

			if proRatedUsage != nil && proRatedUsage.Amount != "" {
				// Pro-rated invoice generated for partial elapsed time
				c.sendPeriodicInvoice(contract, proRatedUsage, now)
			}
			// Stop billing routine after final invoice
			return true
		}

		if periodicUsage != nil {
			// Regular invoice generated for full period(s) that have elapsed
			c.sendPeriodicInvoice(contract, periodicUsage, now)
			// Stop billing routine after final regular invoice
			return true
		}

		return true // Stop billing routine after handling termination
	}

	// Check if contract is completed or expired
	if contract.CurrentState == contracts.ContractCompleted ||
		time.Now().After(contract.Duration.EndDate) {
		return true // Stop billing routine
	}

	// Get last invoice timestamp
	lastInvoiceAt, err := c.usageStore.GetLastProcessedAt(c.ContractDID.URI)
	if err != nil {
		log.Errorw("failed to get last processed timestamp for periodic billing",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false // Continue checking (might be transient error)
	}

	// Initialize lastInvoiceAt if zero (first invoice)
	now := time.Now()
	if lastInvoiceAt.IsZero() {
		// For first invoice, use contract start date as baseline
		lastInvoiceAt = contract.Duration.StartDate
		if err := c.usageStore.SaveLastProcessedAt(c.ContractDID.URI, lastInvoiceAt); err != nil {
			log.Errorw("failed to save initial last processed timestamp",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"error", err)
			return false
		}
	}

	// Try to calculate invoice for elapsed periods
	periodicUsage, err := c.calculatePeriodicInvoice(contract, lastInvoiceAt, now)
	if err != nil {
		if errors.Is(err, ErrPeriodNotElapsed) {
			// Period hasn't elapsed yet - this is normal, continue checking
			return false
		}
		if errors.Is(err, ErrNoDeployments) {
			// Edge Case 1: No deployments during period - skip invoice with log
			log.Infow("skipping periodic invoice - no deployments active during billing period",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI,
				"period_start", lastInvoiceAt,
				"period_end", now)
			// Update lastInvoiceAt to skip this period
			// Calculate period boundaries and move to next period start
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
			return false // Continue checking for next period
		}
		// Other error occurred - log and continue (might be transient)
		log.Errorw("failed to calculate periodic invoice",
			"labels", string(observability.LabelContract),
			"contract_did", c.ContractDID.URI,
			"error", err)
		return false
	}

	if periodicUsage != nil {
		// Invoice calculated successfully - send it (will generate one invoice per deployment)
		c.sendPeriodicInvoice(contract, periodicUsage, now)
		return false // Continue billing routine
	}

	return false
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

		// Create unique ID for this deployment's invoice
		uniqueID := fmt.Sprintf("%s-periodic-%s-%d", contract.ContractDID, deployment.DeploymentID, now.Unix())

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
