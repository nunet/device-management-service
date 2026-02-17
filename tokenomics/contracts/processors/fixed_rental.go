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

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// FixedRentalProcessor implements PaymentModelProcessor for fixed_rental model.
type FixedRentalProcessor struct {
	store *usage.Store
}

func NewFixedRentalProcessor(store *usage.Store) *FixedRentalProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &FixedRentalProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
// Fixed rental does not support manual billing
func (p *FixedRentalProcessor) CollectUsage(
	_ string,
	_ time.Time,
	_ time.Time,
	_ string, // providerDID (not used for fixed_rental)
	_ string, // headContractDID (not used for fixed_rental)
) (*contracts.UsageData, error) {
	return nil, fmt.Errorf("fixed_rental does not support manual billing")
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *FixedRentalProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	_ *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	fixedRentalUsage, ok := usageData.Data.(*contracts.FixedRentalUsage)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type")
	}

	// Generate unique UUID for this payment item
	uniqueID := uuid.NewString()

	items := []*contracts.PaymentItem{
		{
			UniqueID:     uniqueID, // Generated UUID
			DeploymentID: "",       // Not deployment-based
			Amount:       fixedRentalUsage.Amount,
			Usages:       1,
			Metadata: map[string]interface{}{
				"periods_invoiced": fixedRentalUsage.PeriodsInvoiced,
				"period_start":     fixedRentalUsage.PeriodStart.Format(time.RFC3339),
				"period_end":       fixedRentalUsage.PeriodEnd.Format(time.RFC3339),
				"last_invoice_at":  fixedRentalUsage.LastInvoiceAt.Format(time.RFC3339),
			},
		},
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *FixedRentalProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FixedRentalAmount == "" {
		return fmt.Errorf("fixed_rental_amount is required")
	}
	if paymentDetails.PaymentPeriod == "" {
		return fmt.Errorf("payment_period is required")
	}

	if _, err := convert.StringToFloat64(paymentDetails.FixedRentalAmount, "fixed_rental_amount"); err != nil {
		return err
	}
	if _, err := convert.ParsePaymentPeriod(paymentDetails.PaymentPeriod); err != nil {
		return err
	}
	if paymentDetails.PaymentPeriodCount <= 0 {
		return fmt.Errorf("payment_period_count must be a positive integer, got: %d", paymentDetails.PaymentPeriodCount)
	}
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *FixedRentalProcessor) SupportsManualBilling() bool {
	return false
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *FixedRentalProcessor) SupportsAutomaticBilling() bool {
	return true
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *FixedRentalProcessor) CheckAndGenerateInvoice(
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
	billingCyclesElapsed, periodStart, periodEnd := convert.CalculateElapsedPeriods(
		lastInvoiceAt,
		now,
		periodDuration,
		pd.PaymentPeriodCount,
	)

	if billingCyclesElapsed < 1 {
		return nil, ErrPeriodNotElapsed
	}

	// Parse fixed rental amount
	fixedRentalAmount, err := convert.StringToFloat64(pd.FixedRentalAmount, "fixed_rental_amount")
	if err != nil {
		return nil, err
	}

	// Calculate total amount for all elapsed billing cycles
	// Each billing cycle invoices for the fixedRentalAmount
	periodsToInvoice := billingCyclesElapsed * pd.PaymentPeriodCount
	if pd.PaymentPeriodCount <= 0 {
		periodsToInvoice = billingCyclesElapsed
	}
	totalAmount := fixedRentalAmount * float64(billingCyclesElapsed)

	return &contracts.UsageData{
		ContractDID:  contract.ContractDID,
		PaymentModel: contracts.FixedRental,
		Data: &contracts.FixedRentalUsage{
			PeriodsInvoiced: periodsToInvoice,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			Amount:          formatAmount(totalAmount),
			LastInvoiceAt:   lastInvoiceAt,
		},
	}, nil
}

// GenerateProRatedInvoice generates a pro-rated invoice for a terminated contract
// based on the partial period elapsed (from lastInvoiceAt to now).
// Unlike CheckAndGenerateInvoice, this method does not require a full billing period to have elapsed.
func (p *FixedRentalProcessor) GenerateProRatedInvoice(
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

	// For pro-rating, we invoice for the partial period from lastInvoiceAt to now
	periodStart := lastInvoiceAt
	periodEnd := now

	// Calculate elapsed time
	elapsed := now.Sub(lastInvoiceAt)
	if elapsed <= 0 {
		return nil, fmt.Errorf("no elapsed time for pro-rated invoice")
	}

	// Calculate billing cycle duration
	// The fixed rental amount covers paymentPeriodCount periods (one billing cycle)
	periodCount := pd.PaymentPeriodCount
	if periodCount <= 0 {
		periodCount = 1
	}
	billingCycleDuration := periodDuration * time.Duration(periodCount)

	// Parse fixed rental amount
	fixedRentalAmount, err := convert.StringToFloat64(pd.FixedRentalAmount, "fixed_rental_amount")
	if err != nil {
		return nil, err
	}

	// Calculate pro-rated amount: (elapsed / billing cycle duration) * fixedRentalAmount
	proRatedRatio := float64(elapsed) / float64(billingCycleDuration)
	proRatedAmount := proRatedRatio * fixedRentalAmount

	// For pro-rated invoices, use 0 to indicate partial period
	// The actual ratio is reflected in the amount calculation
	return &contracts.UsageData{
		ContractDID:  contract.ContractDID,
		PaymentModel: contracts.FixedRental,
		Data: &contracts.FixedRentalUsage{
			PeriodsInvoiced: 0, // Pro-rated, not a full period
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			Amount:          formatAmount(proRatedAmount),
			LastInvoiceAt:   lastInvoiceAt,
		},
	}, nil
}
