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
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	contractstore "gitlab.com/nunet/device-management-service/tokenomics/store"
)

const (
	contractActorCapsLifespan = 7 * 24 * time.Hour
	contractPeriodicChecker   = 5 * time.Minute
)

type ContractActor struct {
	*actor.BasicActor
	ContractDID        did.DID
	SolutionEnablerDID did.DID
	ctx                context.Context
	cancel             context.CancelFunc

	contractStore contractstore.Store
	participants  contracts.ContractParticipants
}

func NewContractActor(
	solutionEnabler actor.Handle,
	net network.Network,
	participants contracts.ContractParticipants,
	privKey crypto.PrivKey, pubKey crypto.PubKey,
	contractStore contractstore.Store,
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
		ctx:                ctxActor,
		cancel:             cancel,
		contractStore:      contractStore,
		participants:       participants,
	}

	if err := contractActor.setupBehaviorsAndCapabilities(); err != nil {
		return nil, fmt.Errorf("setting up behaviors and capabilities: %w", err)
	}

	if err := contractActor.SetupParticipantsCapabilities(participants); err != nil {
		return nil, fmt.Errorf("failed to setup participant capabilities: %w", err)
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

		foundContract, err := c.contractStore.GetContract(c.ContractDID.URI)
		if err != nil {
			log.Errorw("contract not found while checking its status",
				"labels", string(observability.LabelContract),
				"contract_did", c.ContractDID.URI)
			return
		}

		for range ticker.C {
			if time.Now().After(foundContract.Duration.EndDate) {
				// if we reach duration then we mark it as completed
				foundContract.CurrentState = contracts.ContractCompleted
				if err := c.contractStore.Upsert(foundContract); err != nil {
					log.Errorw("failed to update contract with status completed",
						"labels", string(observability.LabelContract),
						"contract_did", c.ContractDID.URI,
						"end_date", foundContract.Duration.EndDate)
				}

				log.Infow("contract has reached its end date",
					"labels", string(observability.LabelContract),
					"contract_did", c.ContractDID.URI,
					"end_date", foundContract.Duration.EndDate)
			}
		}
	}()

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

// Setup contract participants capabilities
func (c *ContractActor) SetupParticipantsCapabilities(participants contracts.ContractParticipants) error {
	// Grant capabilities to the primary party
	err := c.Security().Grant(
		participants.Provider,
		c.ContractDID,
		[]ucan.Capability{behaviors.ContractTerminationBehavior, behaviors.ContractCompleteBehavior, behaviors.ContractStatusBehavior, behaviors.ContractSettleBehavior, behaviors.ContractValidationBehavior, behaviors.ContractSignBehavior},
		contractActorCapsLifespan,
	)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities to Provider: %w", err)
	}

	// Grant capabilities to the secondary party
	err = c.Security().Grant(
		participants.Requestor,
		c.ContractDID,
		[]ucan.Capability{behaviors.ContractTerminationBehavior, behaviors.ContractCompleteBehavior, behaviors.ContractStatusBehavior, behaviors.ContractSettleBehavior, behaviors.ContractValidationBehavior, behaviors.ContractSignBehavior},
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
	}

	return contractBehaviors
}

func (c *ContractActor) handleContractSignByParticipants(msg actor.Envelope) {
	defer msg.Discard()
	resp := contracts.ContractSignResponseBehaviour{}

	var req contracts.ContractSignRequestBehaviour
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

	resp := contracts.ContractStatusResponseBehaviour{}
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

	resp := contracts.ContractTerminationResponseBehaviour{}
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

	resp := contracts.ContractCompletionResponseBehaviour{}
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

	resp := contracts.ContractSettleResponseBehaviour{}
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

	resp := contracts.ContractValidateResponseBehaviour{}
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
