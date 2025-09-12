// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/types"
)

type CreateContractRequestBehaviour struct {
	SolutionEnablerDID    did.DID              `json:"solution_enabler_did"`
	PaymentValidatorDID   did.DID              `json:"payment_validator_did"`
	ResourceConfiguration types.Resources      `json:"resource_configuration"`
	TerminationOption     *TerminationOption   `json:"termination_option"`
	Penalties             []PenaltyClause      `json:"penalties"`
	PaymentDetails        PaymentDetails       `json:"payment_details"`
	ContractTerms         interface{}          `json:"contract_terms"`
	ContractParticipants  ContractParticipants `json:"contract_participants"`
	Duration              DurationDetails      `json:"duration"`
}

type ContractPaymentStatusRequest struct {
	UniqueID string `json:"unique_id"`
}

type ContractPaymentStatusResponse struct {
	UniqueID string `json:"unique_id"`
	Paid     bool   `json:"paid"`
	Error    string `json:"error"`
}

type CollectUsagesAndForwardToPaymentProvidersReponse struct {
	Error       string `json:"error"`
	TotalUsages int    `json:"total_usages"`
}

type ContractListLocalTransactionsRequest struct{}

type ContractListLocalTransactionsResponse struct {
	Error        string                     `json:"error"`
	Transactions []*transaction.Transaction `json:"transactions"`
}

type ContractConfirmLocalTransactionRequest struct {
	UniqueID string `json:"unique_id"`
	TxHash   string `json:"tx_hash"`
}

type ContractConfirmLocalTransactionResponse struct {
	Error string `json:"error"`
}

type TransactionForServiceProviderRequest struct {
	UniqueID            string `json:"unique_id"`
	PaymentValidatorDID string `json:"payment_validator_did"`
	ContractDID         string `json:"contract_did"`
	ToAddress           string `json:"to_address"`
	Amount              string `json:"amount"`
}

type TransactionForServiceProviderResponse struct {
	Error string `json:"error"`
}

type ContractUsageRequestBehavior struct {
	UniqueID string   `json:"unique_id"`
	Contract Contract `json:"contract"`
	Usages   int      `json:"usages"`
}

type ContractUsageResponseBehavior struct {
	Error string `json:"error"`
}

type ContractEventRequestBehaviour struct {
	Payload []byte `json:"payload"`
}

type ContractEventResponseBehaviour struct {
	Error string `json:"error"`
}

type ContractPaymentValidationRequestBehavior struct {
	TxHash   string `json:"tx_hash"`
	UniqueID string `json:"unique_id"`
}

type ContractPaymentValidationResponseBehavior struct {
	Error string `json:"error"`
}

type PaymentValidateRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}

type PaymentValidateResponseBehaviour struct {
	Error string `json:"error"`
}

type ContractListIncomingResponseBehaviour struct {
	Contracts []*Contract `json:"contracts"`
	Error     string      `json:"error,omitempty"`
}

type ContractApproveLocalRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}

type ContractApproveLocalResponseBehaviour struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ContractVerificationResponse represents the response structure for contract verification
type ContractVerificationResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type ContractStatusRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}
type ContractStatusResponseBehaviour struct {
	Error    string   `json:"error"`
	Contract Contract `json:"contract"`
}

type CreateContractResponseBehaviour struct {
	ContractDID string `json:"contract_did"`
	PubKey      string `json:"pub_key"`
	Error       string `json:"error"`
}

type ProposeContractResponseBehaviour struct {
	Signature Signature `json:"signature"`
	Error     string    `json:"error"`
}

type ContractTerminationRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}
type ContractTerminationResponseBehaviour struct {
	Error string `json:"error"`
}

type ContractCompletionRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}
type ContractCompletionResponseBehaviour struct {
	Error string `json:"error"`
}

type ContractSettleRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}
type ContractSettleResponseBehaviour struct {
	Error string `json:"error"`
}

type ContractValidateRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
}
type ContractValidateResponseBehaviour struct {
	Valid         bool   `json:"valid"`
	CurrentStatus string `json:"current_status"`
	Error         string `json:"error"`
}

type ContractSignRequestBehaviour struct {
	ContractDID string `json:"contract_did"`
	Signature   []byte `json:"signature"`
}
type ContractSignResponseBehaviour struct {
	Error    string   `json:"error"`
	Contract Contract `json:"contract"`
}

// ContractState represents the possible states of a contract
type ContractState string

const (
	ContractDraft      ContractState = "DRAFT"
	ContractProposed   ContractState = "PROPOSED"
	ContractAccepted   ContractState = "ACCEPTED" // TODO add accepted status
	ContractActive     ContractState = "ACTIVE"   // Not expired and not canceled
	ContractPaused     ContractState = "PAUSED"   // in case of dispute
	ContractUpdate     ContractState = "UPDATED"
	ContractTerminated ContractState = "TERMINATED"
	ContractCompleted  ContractState = "COMPLETED"
	ContractSettled    ContractState = "SETTLED"
)

// ContractEvent represents events that can trigger state transitions
type ContractEvent string

const (
	EventPropose   ContractEvent = "PROPOSE"
	EventAccepted  ContractEvent = "ACCEPTED"
	EventActivate  ContractEvent = "ACTIVATE"
	EventDispute   ContractEvent = "DISPUTE"
	EventUpdate    ContractEvent = "UPDATED"
	EventTerminate ContractEvent = "TERMINATE" // TODO: termination
	EventComplete  ContractEvent = "COMPLETE"
	EventSettle    ContractEvent = "SETTLE"
)

// StateTransitionError represents an invalid state transition
type StateTransitionError struct {
	Current ContractState
	Event   ContractEvent
	Message string
}

func (e *StateTransitionError) Error() string {
	return e.Message
}

// ContractStateTransition represents a valid state transition
type ContractStateTransition struct {
	FromState ContractState
	Event     ContractEvent
	ToState   ContractState
}

// StateTransition represents a historical state change
type StateTransition struct {
	FromState   ContractState `json:"from_state"`
	ToState     ContractState `json:"to_state"`
	Event       ContractEvent `json:"event"`
	Timestamp   time.Time     `json:"timestamp"`
	InitiatedBy did.DID       `json:"initiated_by"`
}

// Contract represents the contract details between nodes
type Contract struct {
	ContractDID           string               `json:"contract_did"`
	SolutionEnablerDID    did.DID              `json:"solution_enabler_did"`
	PaymentValidatorDID   did.DID              `json:"payment_validator_did"`
	ResourceConfiguration types.Resources      `json:"resource_configuration"`
	TerminationOption     *TerminationOption   `json:"termination_option,omitempty"`
	Penalties             []PenaltyClause      `json:"penalties"`
	Duration              *DurationDetails     `json:"duration,omitempty"`
	ContractParticipants  ContractParticipants `json:"participants"`
	PaymentDetails        PaymentDetails       `json:"payment_details"` // Zero value: zero value of payments.Payment struct
	Paid                  bool                 `json:"paid"`
	Signatures            []Signature          `json:"signatures"`     // Changed to slice of Signature
	Settled               bool                 `json:"settled"`        // Example default: false
	Verification          jobs.Status          `json:"verification"`   // Zero value: zero value of jobs.Status
	ContractProof         []byte               `json:"contract_proof"` // Example default: "Pending"
	CurrentState          ContractState        `json:"current_state"`  // state tracking
	ContractTerms         interface{}          `json:"contract_terms"` // To store contract agreement terms
	TerminationStarted    time.Time            `json:"termination_started"`
	Transitions           []StateTransition    `json:"transitions"`
}

func (c *Contract) Sign(key did.Provider) ([]byte, error) {
	cBytes, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract data: %w", err)
	}
	sig, err := key.Sign(cBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the contract")
	}
	return sig, nil
}

type ContractParticipants struct {
	Provider  did.DID `json:"provider"`
	Requestor did.DID `json:"requestor"`
}

// TerminationOption specifies termination rules for long-running jobs.
type TerminationOption struct {
	Allowed      bool          `json:"allowed"`
	NoticePeriod time.Duration `json:"notice_period"` // e.g., "30 days"
}

// DurationDetails defines the duration for hire.
type DurationDetails struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type PenaltyClause struct {
	Condition string  `json:"condition"` // e.g., "Uptime < 99.9%"
	Penalty   float64 `json:"penalty"`
}

// Signature represents a digital signature on the contract
type Signature struct {
	DID        did.DID `json:"did"`       // The DID of the signer
	Signatures []byte  `json:"signature"` // The actual signature bytes
}

func GenerateContractID(req CreateContractRequestBehaviour) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func NewContract(contractDID string, req CreateContractRequestBehaviour) *Contract {
	return &Contract{
		ContractDID:           contractDID,
		SolutionEnablerDID:    req.SolutionEnablerDID,
		PaymentValidatorDID:   req.PaymentValidatorDID,
		ResourceConfiguration: req.ResourceConfiguration,
		TerminationOption:     req.TerminationOption,
		Penalties:             req.Penalties,
		Duration:              &req.Duration,
		ContractParticipants:  req.ContractParticipants,
		PaymentDetails:        req.PaymentDetails,
		ContractTerms:         req.ContractTerms,
		CurrentState:          ContractDraft,
		Transitions:           []StateTransition{},
	}
}
