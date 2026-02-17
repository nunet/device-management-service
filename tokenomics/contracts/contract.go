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

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/types"
)

type CreateContractRequest struct {
	Metadata              map[string]string    `json:"metadata"`
	SolutionEnablerDID    did.DID              `json:"solution_enabler_did"`
	PaymentValidatorDID   did.DID              `json:"payment_validator_did"`
	ResourceConfiguration types.Resources      `json:"resource_configuration"`
	TerminationOption     *TerminationOption   `json:"termination_option"`
	Penalties             []PenaltyClause      `json:"penalties"`
	PaymentDetails        PaymentDetails       `json:"payment_details"`
	ContractTerms         interface{}          `json:"contract_terms"`
	ContractParticipants  ContractParticipants `json:"contract_participants"`
	Duration              DurationDetails      `json:"duration"`
	DisableBilling        bool                 `json:"disable_billing,omitempty"` // If true, disables all billing (automatic and manual)
}

type ContractPaymentStatusRequest struct {
	UniqueID string `json:"unique_id"`
}

type ContractPaymentStatusResponse struct {
	UniqueID string `json:"unique_id"`
	Paid     bool   `json:"paid"`
	Error    string `json:"error"`
}

type CollectUsagesAndForwardToPaymentProvidersRequest struct {
	ContractDID string `json:"contract_did,omitempty"` // If empty, processes all contracts
}

// AllocationTimeUtilization represents time utilization for a single allocation
type AllocationTimeUtilization struct {
	AllocationID string        `json:"allocation_id"`
	Duration     time.Duration `json:"duration"` // Total time the allocation ran
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time,omitempty"` // Empty if still running
}

// DeploymentTimeUtilization represents time utilization for a deployment
type DeploymentTimeUtilization struct {
	DeploymentID        string                      `json:"deployment_id"`
	Allocations         []AllocationTimeUtilization `json:"allocations"`
	TotalUtilizationSec float64                     `json:"total_utilization_sec"` // Total seconds across all allocations
}

// TimeUtilizationUsage represents usage data for pay_per_time_utilization model
type TimeUtilizationUsage struct {
	Deployments []DeploymentTimeUtilization `json:"deployments"`
}

// AllocationResourceUtilization represents resource utilization for a single allocation
type AllocationResourceUtilization struct {
	AllocationID string          `json:"allocation_id"`
	Resources    types.Resources `json:"resources"` // CPU cores, RAM GB, Disk GB, GPU count
	Duration     time.Duration   `json:"duration"`  // How long allocation ran
	StartTime    time.Time       `json:"start_time"`
	EndTime      time.Time       `json:"end_time,omitempty"`
	// Calculated costs (for invoice details)
	CPUCost   string `json:"cpu_cost,omitempty"`
	RAMCost   string `json:"ram_cost,omitempty"`
	DiskCost  string `json:"disk_cost,omitempty"`
	GPUCost   string `json:"gpu_cost,omitempty"`
	TotalCost string `json:"total_cost,omitempty"`
}

// DeploymentResourceUtilization tracks all allocations in a deployment
type DeploymentResourceUtilization struct {
	DeploymentID        string                          `json:"deployment_id"`
	Allocations         []AllocationResourceUtilization `json:"allocations"`
	TotalUtilizationSec float64                         `json:"total_utilization_sec"`
	TotalCost           string                          `json:"total_cost,omitempty"`
}

// ResourceUtilizationUsage represents resource utilization data
type ResourceUtilizationUsage struct {
	Deployments []DeploymentResourceUtilization `json:"deployments"`
}

// FixedRentalUsage represents usage data for fixed_rental payment model
type FixedRentalUsage struct {
	PeriodsInvoiced int       `json:"periods_invoiced"` // Number of full periods invoiced
	PeriodStart     time.Time `json:"period_start"`     // Start of the first period in this invoice
	PeriodEnd       time.Time `json:"period_end"`       // End of the last period in this invoice
	Amount          string    `json:"amount"`           // Total amount for this invoice
	LastInvoiceAt   time.Time `json:"last_invoice_at"`  // Timestamp of last invoice (before this one)
}

// PeriodicUsage represents usage data for periodic payment model
type PeriodicUsage struct {
	PeriodStart     time.Time                   `json:"period_start"`     // Start of billing period
	PeriodEnd       time.Time                   `json:"period_end"`       // End of billing period
	LastInvoiceAt   time.Time                   `json:"last_invoice_at"`  // Timestamp of last invoice
	Deployments     []DeploymentTimeUtilization `json:"deployments"`      // Deployment runtime data
	TotalTimeSec    float64                     `json:"total_time_sec"`   // Total deployment time in seconds
	Amount          string                      `json:"amount"`           // Calculated amount
	PeriodsInvoiced int                         `json:"periods_invoiced"` // Number of periods covered
}

type ContractUsageResult struct {
	ContractDID         string                    `json:"contract_did"`
	PaymentModel        PaymentModel              `json:"payment_model"`
	Usages              int                       `json:"usages"` // For backward compatibility
	Error               string                    `json:"error,omitempty"`
	TimeUtilization     *TimeUtilizationUsage     `json:"time_utilization,omitempty"`     // For pay_per_time_utilization
	ResourceUtilization *ResourceUtilizationUsage `json:"resource_utilization,omitempty"` // For pay_per_resource_utilization
	FixedRentalDetails  *FixedRentalUsage         `json:"fixed_rental_details,omitempty"` // For fixed_rental
	PeriodicDetails     *PeriodicUsage            `json:"periodic_details,omitempty"`     // For periodic
}

type CollectUsagesAndForwardToPaymentProvidersReponse struct {
	Error       string                `json:"error"`
	TotalUsages int                   `json:"total_usages"`
	Results     []ContractUsageResult `json:"results,omitempty"` // Per-contract results
}

type ContractListLocalTransactionsRequest struct{}

type ContractListLocalTransactionsResponse struct {
	Error        string                     `json:"error"`
	Transactions []*transaction.Transaction `json:"transactions"`
}

type ContractConfirmLocalTransactionRequest struct {
	UniqueID   string `json:"unique_id"`
	TxHash     string `json:"tx_hash"`
	Blockchain string `json:"blockchain"`
}

type ContractConfirmLocalTransactionResponse struct {
	Error string `json:"error"`
}

type TransactionForServiceProviderRequest struct {
	UniqueID            string                     `json:"unique_id"`
	PaymentValidatorDID string                     `json:"payment_validator_did"`
	ContractDID         string                     `json:"contract_did"`
	ToAddress           []types.PaymentAddressInfo `json:"to_address"`
	Amount              string                     `json:"amount"`
	Status              string                     `json:"status,omitempty"`  // optional status, defaults to "unpaid" if empty
	TxHash              string                     `json:"tx_hash,omitempty"` // optional transaction hash
	Metadata            map[string]interface{}     `json:"metadata,omitempty"`
}

type TransactionForServiceProviderResponse struct {
	Error string `json:"error"`
}

type ContractUsageRequest struct {
	UniqueID            string                    `json:"unique_id"`
	Contract            Contract                  `json:"contract"`
	Usages              int                       `json:"usages"`                         // For backward compatibility
	TimeUtilization     *TimeUtilizationUsage     `json:"time_utilization,omitempty"`     // For pay_per_time_utilization
	ResourceUtilization *ResourceUtilizationUsage `json:"resource_utilization,omitempty"` // For pay_per_resource_utilization
	FixedRentalDetails  *FixedRentalUsage         `json:"fixed_rental_details,omitempty"` // For fixed_rental
	PeriodicDetails     *PeriodicUsage            `json:"periodic_details,omitempty"`     // For periodic
}

type ContractUsageResponse struct {
	Error string `json:"error"`
}

type ContractEventRequest struct {
	Payload []byte `json:"payload"`
}

type ContractEventResponse struct {
	Error string `json:"error"`
}

type ContractPaymentValidationRequest struct {
	TxHash     string `json:"tx_hash"`
	UniqueID   string `json:"unique_id"`
	Blockchain string `json:"blockchain"`
}

type ContractPaymentValidationResponse struct {
	Error string `json:"error"`
}

type PaymentValidateRequest struct {
	ContractDID string `json:"contract_did"`
}

type PaymentValidateResponse struct {
	Error string `json:"error"`
}

type ContractListIncomingRole string

const (
	ContractRoleProvider  ContractListIncomingRole = "provider"
	ContractRoleRequestor ContractListIncomingRole = "requestor"
)

type ContractListIncomingRequest struct {
	Role ContractListIncomingRole `json:"role,omitempty"`
}

type ContractListIncomingResponse struct {
	Contracts []*Contract `json:"contracts"`
	Error     string      `json:"error,omitempty"`
}

type ContractApproveLocalRequest struct {
	ContractDID string `json:"contract_did"`
}

type ContractApproveLocalResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ContractVerificationResponse represents the response structure for contract verification
type ContractVerificationResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type ContractStatusRequest struct {
	ContractDID string `json:"contract_did"`
}
type ContractStatusResponse struct {
	Error    string   `json:"error"`
	Contract Contract `json:"contract"`
}

type CreateContractResponse struct {
	ContractRequest CreateContractRequest `json:"contract_request"`
	ContractDID     string                `json:"contract_did"`
	PubKey          string                `json:"pub_key"`
	Error           string                `json:"error"`
}

type ProposeContractRequest struct {
	Contract          Contract     `json:"contract"`
	CreatorOfContract actor.Handle `json:"creator_of_contract"`
}

type ProposeContractResponse struct {
	Signature Signature `json:"signature"`
	Error     string    `json:"error"`
}

type ContractTerminationRequest struct {
	ContractDID string `json:"contract_did"`
}
type ContractTerminationResponse struct {
	Error string `json:"error"`
}

type ContractCompletionRequest struct {
	ContractDID string `json:"contract_did"`
}
type ContractCompletionResponse struct {
	Error string `json:"error"`
}

type ContractSettleRequest struct {
	ContractDID string `json:"contract_did"`
}
type ContractSettleResponse struct {
	Error string `json:"error"`
}

type ContractValidateRequest struct {
	ContractDID string `json:"contract_did"`
}
type ContractValidateResponse struct {
	Valid         bool   `json:"valid"`
	CurrentStatus string `json:"current_status"`
	Error         string `json:"error"`
}

// ContractChainVerificationRequest requests verification of a contract chain
type ContractChainVerificationRequest struct {
	SolutionEnablerDID string `json:"solution_enabler_did"` // Contract host DID
	ContractDID        string `json:"contract_did"`         // Contract DID (Orch ↔ Org)
	OrganizationDID    string `json:"organization_did"`     // Organization DID (from Contract A)
	OrchestratorDID    string `json:"orchestrator_did"`     // Orchestrator DID
	ProviderDID        string `json:"provider_did"`         // Provider DID
}

// ContractChainVerificationResponse contains the chain verification result
type ContractChainVerificationResponse struct {
	Valid                bool      `json:"valid"`
	OrganizationDID      string    `json:"organization_did,omitempty"`
	OrchestratorContract *Contract `json:"orchestrator_contract,omitempty"` // Contract A
	ProviderContract     *Contract `json:"provider_contract,omitempty"`     // Contract B
	Error                string    `json:"error,omitempty"`
}

type ContractSignRequest struct {
	ContractDID string `json:"contract_did"`
	Signature   []byte `json:"signature"`
}
type ContractSignResponse struct {
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

// Contract chain role constants for metadata
const (
	ContractChainRoleMetadataKey = "contract_chain_role"
	ContractChainRoleHead        = "head"
	ContractChainRoleTail        = "tail"
)

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
	DisableBilling        bool                 `json:"disable_billing,omitempty"` // If true, disables all billing (automatic and manual)
	Metadata              map[string]string    `json:"metadata,omitempty"`        // Contract metadata (e.g., contract_chain_role)
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

// Validate validates the CreateContractRequest and returns an error if any required fields are missing or invalid
func (req *CreateContractRequest) Validate() error {
	// Validate SolutionEnablerDID
	if req.SolutionEnablerDID.Empty() {
		return fmt.Errorf("solution_enabler_did is required")
	}

	// Validate PaymentValidatorDID
	if req.PaymentValidatorDID.Empty() {
		return fmt.Errorf("payment_validator_did is required")
	}

	// Validate ContractParticipants
	if req.ContractParticipants.Provider.Empty() {
		return fmt.Errorf("contract_participants.provider is required")
	}
	if req.ContractParticipants.Requestor.Empty() {
		return fmt.Errorf("contract_participants.requestor is required")
	}

	// Validate PaymentDetails
	if req.PaymentDetails.PaymentModel == "" {
		return fmt.Errorf("payment_details.payment_model is required")
	}

	// Validate Duration
	if req.Duration.StartDate.IsZero() {
		return fmt.Errorf("duration.start_date is required")
	}
	if req.Duration.EndDate.IsZero() {
		return fmt.Errorf("duration.end_date is required")
	}
	if !req.Duration.EndDate.After(req.Duration.StartDate) {
		return fmt.Errorf("duration.end_date must be after duration.start_date")
	}

	// Validate payment_period and payment_period_count for models that require them
	if req.PaymentDetails.PaymentModel == FixedRental || req.PaymentDetails.PaymentModel == Periodic {
		// Validate payment_period is required and valid
		if req.PaymentDetails.PaymentPeriod == "" {
			return fmt.Errorf("payment_details.payment_period is required for payment model %s", req.PaymentDetails.PaymentModel)
		}
		// Validate payment_period is one of the valid values
		validPeriods := map[string]bool{
			PaymentPeriodMinute: true,
			PaymentPeriodHour:   true,
			PaymentPeriodDay:    true,
			PaymentPeriodWeek:   true,
			PaymentPeriodMonth:  true,
		}
		if !validPeriods[req.PaymentDetails.PaymentPeriod] {
			return fmt.Errorf("payment_details.payment_period must be one of: minute, hour, day, week, month, got: %s", req.PaymentDetails.PaymentPeriod)
		}

		// Validate payment_period_count is positive
		if req.PaymentDetails.PaymentPeriodCount <= 0 {
			return fmt.Errorf("payment_details.payment_period_count must be a positive integer for payment model %s, got: %d", req.PaymentDetails.PaymentModel, req.PaymentDetails.PaymentPeriodCount)
		}
	}

	return nil
}

func GenerateContractID(req CreateContractRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// SetPeriodicityDefaults sets default values for PaymentPeriod and PaymentPeriodCount if not provided.
// Default: PaymentPeriod = "hour", PaymentPeriodCount = 1
// This ensures all contracts have periodicity configured for automatic billing.
func SetPeriodicityDefaults(pd *PaymentDetails) {
	if pd.PaymentPeriod == "" {
		pd.PaymentPeriod = PaymentPeriodHour
	}
	if pd.PaymentPeriodCount <= 0 {
		pd.PaymentPeriodCount = 1
	}
}

func NewContract(contractDID string, req CreateContractRequest) *Contract {
	// Set periodicity defaults if not provided
	SetPeriodicityDefaults(&req.PaymentDetails)

	// Copy metadata from request if provided
	metadata := make(map[string]string)
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}

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
		DisableBilling:        req.DisableBilling,
		Metadata:              metadata,
	}
}
