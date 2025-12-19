// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processors

import (
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// PayPerAllocationProcessor implements PaymentModelProcessor for pay_per_allocation model.
// This processor counts allocations and calculates payment based on a fixed fee per allocation.
var _ contracts.PaymentModelProcessor = (*PayPerAllocationProcessor)(nil)

type PayPerAllocationProcessor struct {
	store *usage.Store
}

func NewPayPerAllocationProcessor(store *usage.Store) *PayPerAllocationProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &PayPerAllocationProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
func (p *PayPerAllocationProcessor) CollectUsage(
	contractDID string,
	lastProcessedAt time.Time,
	now time.Time,
) (*contracts.UsageData, error) {
	usageCount, err := p.store.CountAllocationsByContractDID(contractDID, lastProcessedAt, now)
	if err != nil {
		return nil, fmt.Errorf("failed to count allocations: %w", err)
	}

	return &contracts.UsageData{
		ContractDID:  contractDID,
		PaymentModel: contracts.PayPerAllocation,
		Data:         usageCount, // Simple count for this model
	}, nil
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *PayPerAllocationProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	contract *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	usageCount, ok := usageData.Data.(int)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type for pay_per_allocation")
	}

	if usageCount == 0 {
		return []*contracts.PaymentItem{}, nil
	}

	pd := contract.PaymentDetails
	feePerAllocation, err := convert.StringToFloat64(pd.FeesPerAllocation, "fees_per_allocation")
	if err != nil {
		return nil, err
	}

	totalAmount := feePerAllocation * float64(usageCount)

	items := []*contracts.PaymentItem{
		{
			UniqueID:     "", // Will be set by payment processor
			DeploymentID: "", // Not deployment-based
			Amount:       formatAmount(totalAmount),
			Usages:       usageCount,
			Metadata: map[string]interface{}{
				"allocation_count": usageCount,
			},
		},
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *PayPerAllocationProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FeesPerAllocation == "" {
		return fmt.Errorf("fees_per_allocation is required")
	}
	if _, err := convert.StringToFloat64(paymentDetails.FeesPerAllocation, "fees_per_allocation"); err != nil {
		return err
	}
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *PayPerAllocationProcessor) SupportsManualBilling() bool {
	return true
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PayPerAllocationProcessor) SupportsAutomaticBilling() bool {
	return false
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *PayPerAllocationProcessor) CheckAndGenerateInvoice(
	_ *contracts.Contract,
	_ time.Time,
	_ time.Time,
) (*contracts.UsageData, error) {
	return nil, fmt.Errorf("pay_per_allocation does not support automatic billing")
}
