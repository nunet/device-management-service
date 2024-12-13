// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"time"

	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
)

// Contract represents the structure for P2P agreements.
type Contract struct {
	DID                   did.DID             `json:"did"` // DID
	ResourceConfiguration types.Resources     `json:"resource_configuration"`
	PerformanceMetrics    []PerformanceMetric `json:"performance_metrics"`
	ContractValue         float64             `json:"contract_value"`
	Payment               PaymentDetails      `json:"payment"`
	Duration              *DurationDetails    `json:"duration,omitempty"`
	TerminationOption     *TerminationOption  `json:"termination_option,omitempty"`
	Penalties             []PenaltyClause     `json:"penalties"`
	SettlementProvider    string              `json:"settlement_provider"`
	AcceptedProofOfWork   ProofOfWork         `json:"accepted_proof_of_work"`
	DisputeResolution     string              `json:"dispute_resolution"`
	ContractHost          string              `json:"contract_host"`
	ContractLogic         []byte              `json:"contract_logic"`
	ExtraData             []byte              `json:"extra_data"` // arbitrary data that we can add to the contract
}

// PerformanceMetric represents a single performance metric and target value.
type PerformanceMetric struct {
	Name        string  `json:"name"`
	TargetValue float64 `json:"targetValue"`
	Unit        string  `json:"unit"`
}

// PaymentDetails contains payment type and schedule details.
type PaymentDetails struct {
	Type             string         `json:"type"`
	Amount           float64        `json:"amount"`
	Frequency        *time.Duration `json:"frequency,omitempty"`          // e.g., 30 days for recurring
	NumberOfPayments *int           `json:"number_of_payments,omitempty"` // Optional, for recurring payments
}

// DurationDetails defines the duration for hire.
type DurationDetails struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// TerminationOption specifies termination rules for long-running jobs.
type TerminationOption struct {
	Allowed      bool   `json:"allowed"`
	NoticePeriod string `json:"notice_period"` // e.g., "30 days"
}

// PenaltyClause defines the penalty for non-performance.
type PenaltyClause struct {
	Condition string  `json:"condition"` // e.g., "Uptime < 99.9%"
	Penalty   float64 `json:"penalty"`
}

// ProofOfWork represents the acceptable proof of work for settlement.
type ProofOfWork struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}
