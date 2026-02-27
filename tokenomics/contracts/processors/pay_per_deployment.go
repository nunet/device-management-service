// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processors

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// PayPerDeploymentProcessor implements PaymentModelProcessor for pay_per_deployment model.
// This processor counts deployments and calculates payment based on a fixed fee per deployment.
var _ contracts.PaymentModelProcessor = (*PayPerDeploymentProcessor)(nil)

type PayPerDeploymentProcessor struct {
	store *usage.Store
}

func NewPayPerDeploymentProcessor(store *usage.Store) *PayPerDeploymentProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &PayPerDeploymentProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
func (p *PayPerDeploymentProcessor) CollectUsage(
	contractDID string,
	lastProcessedAt time.Time,
	now time.Time,
	_ string, // providerDID - unused in this processor
	headContractDID string, // New parameter
) (*contracts.UsageData, error) {
	var usageCount int
	var err error

	// If headContractDID is provided, query by Head Contract DID
	// Otherwise, query by Tail Contract DID (backward compatible)
	if headContractDID != "" {
		filters := usage.EventFilters{
			HeadContractDID: headContractDID,
			EventTypes:      []events.EventType{events.DeploymentStartEvent},
			StartTime:       lastProcessedAt,
			EndTime:         now,
		}
		usageEvents, err := p.store.QueryEvents(filters)
		if err != nil {
			return nil, fmt.Errorf("failed to query events by head contract: %w", err)
		}
		// Count unique deployments from events
		deploymentSet := make(map[string]bool)
		for _, evt := range usageEvents {
			var evtData events.DeploymentStart
			if err := json.Unmarshal(evt.Data, &evtData); err != nil {
				continue
			}
			if evtData.DeploymentID != "" {
				deploymentSet[evtData.DeploymentID] = true
			}
		}
		usageCount = len(deploymentSet)
	} else {
		// Existing logic: query by Tail Contract DID
		usageCount, err = p.store.CountDeploymentsByContract(contractDID, lastProcessedAt, now)
		if err != nil {
			return nil, fmt.Errorf("failed to count deployments: %w", err)
		}
	}

	return &contracts.UsageData{
		ContractDID:  contractDID,
		PaymentModel: contracts.PayPerDeployment,
		Data:         usageCount, // Simple count for this model
	}, nil
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *PayPerDeploymentProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	contract *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	usageCount, ok := usageData.Data.(int)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type for pay_per_deployment")
	}

	if usageCount == 0 {
		return []*contracts.PaymentItem{}, nil
	}

	pd := contract.PaymentDetails
	feePerDeployment, err := convert.StringToFloat64(pd.FeePerDeployment, "fee_per_deployment")
	if err != nil {
		return nil, err
	}

	totalAmount := feePerDeployment * float64(usageCount)

	// Generate unique UUID for this payment item
	uniqueID := uuid.NewString()

	items := []*contracts.PaymentItem{
		{
			UniqueID:     uniqueID, // Generated UUID
			DeploymentID: "",       // Not per-deployment item, but deployment-based model
			Amount:       formatAmount(totalAmount),
			Usages:       usageCount,
			Metadata: map[string]interface{}{
				"deployment_count":     usageCount,
				"fee_per_deployment":   feePerDeployment,
				"total_amount":         totalAmount,
				"payment_model":        contracts.PayPerDeployment,
				"payment_period":       pd.PaymentPeriod,
				"payment_period_count": pd.PaymentPeriodCount,
			},
		},
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *PayPerDeploymentProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FeePerDeployment == "" {
		return fmt.Errorf("fee_per_deployment is required")
	}
	if _, err := convert.StringToFloat64(paymentDetails.FeePerDeployment, "fee_per_deployment"); err != nil {
		return err
	}
	// Validate payment_period if provided (optional for this model)
	if paymentDetails.PaymentPeriod != "" {
		if _, err := convert.ParsePaymentPeriod(paymentDetails.PaymentPeriod); err != nil {
			return err
		}
	}
	// Validate payment_period_count if provided (optional for this model)
	if paymentDetails.PaymentPeriodCount < 0 {
		return fmt.Errorf("payment_period_count must be a positive integer, got: %d", paymentDetails.PaymentPeriodCount)
	}
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *PayPerDeploymentProcessor) SupportsManualBilling() bool {
	return true
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PayPerDeploymentProcessor) SupportsAutomaticBilling() bool {
	return true
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *PayPerDeploymentProcessor) CheckAndGenerateInvoice(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
) (*contracts.UsageData, error) {
	pd := contract.PaymentDetails

	// Parse payment period
	periodDuration, err := convert.ParsePaymentPeriod(pd.PaymentPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid payment_period: %w", err)
	}

	// Calculate elapsed periods
	billingCyclesElapsed, _, _ := convert.CalculateElapsedPeriods(
		lastInvoiceAt,
		now,
		periodDuration,
		pd.PaymentPeriodCount,
	)

	if billingCyclesElapsed < 1 {
		return nil, ErrPeriodNotElapsed
	}

	// Period has elapsed, collect usage
	// Detect contract type from metadata to determine query strategy
	headContractDID := ""
	if contract.Metadata != nil {
		if role, ok := contract.Metadata[contracts.ContractChainRoleMetadataKey]; ok {
			if role == contracts.ContractChainRoleHead {
				// Head Contract: query by head_contract_did = contract's DID
				headContractDID = contract.ContractDID
			}
			// For Tail Contract or P2P, headContractDID remains empty (query by contract_did)
		}
	}
	return p.CollectUsage(contract.ContractDID, lastInvoiceAt, now, "", headContractDID)
}
