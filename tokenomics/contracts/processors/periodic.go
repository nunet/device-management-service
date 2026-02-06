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

	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// PeriodicProcessor implements PaymentModelProcessor for periodic model.
var _ contracts.PaymentModelProcessor = (*PeriodicProcessor)(nil)

type PeriodicProcessor struct {
	store *usage.Store
}

func NewPeriodicProcessor(store *usage.Store) *PeriodicProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &PeriodicProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
// Periodic does not support manual billing
func (p *PeriodicProcessor) CollectUsage(
	_ string,
	_ time.Time,
	_ time.Time,
	_ string, // providerDID (not used for periodic)
	_ string, // headContractDID (not used for periodic)
) (*contracts.UsageData, error) {
	return nil, fmt.Errorf("periodic does not support manual billing")
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *PeriodicProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	contract *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	periodicUsage, ok := usageData.Data.(*contracts.PeriodicUsage)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type")
	}

	pd := contract.PaymentDetails
	feePerUnit, err := convert.StringToFloat64(pd.FeePerTimeUnit, "fee_per_time_unit")
	if err != nil {
		return nil, err
	}

	items := make([]*contracts.PaymentItem, 0)
	for _, deployment := range periodicUsage.Deployments {
		// Convert deployment time to time unit
		timeInUnit, err := convert.SecondsToUnit(deployment.TotalUtilizationSec, pd.TimeUnit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert time: %w", err)
		}

		amount := feePerUnit * timeInUnit

		items = append(items, &contracts.PaymentItem{
			UniqueID:     "", // Will be set by payment processor
			DeploymentID: deployment.DeploymentID,
			Amount:       formatAmount(amount),
			Usages:       1,
			Metadata: map[string]interface{}{
				"total_utilization_sec": deployment.TotalUtilizationSec,
				"period_start":          periodicUsage.PeriodStart,
				"period_end":            periodicUsage.PeriodEnd,
			},
		})
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *PeriodicProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FeePerTimeUnit == "" {
		return fmt.Errorf("fee_per_time_unit is required")
	}
	if paymentDetails.TimeUnit == "" {
		return fmt.Errorf("time_unit is required")
	}
	if paymentDetails.PaymentPeriod == "" {
		return fmt.Errorf("payment_period is required")
	}

	if _, err := convert.StringToFloat64(paymentDetails.FeePerTimeUnit, "fee_per_time_unit"); err != nil {
		return err
	}
	if _, err := convert.ParsePaymentPeriod(paymentDetails.PaymentPeriod); err != nil {
		return err
	}
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *PeriodicProcessor) SupportsManualBilling() bool {
	return false
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PeriodicProcessor) SupportsAutomaticBilling() bool {
	return true
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *PeriodicProcessor) CheckAndGenerateInvoice(
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

	// Query deployment start and stop events
	startEvents, stopEvents, err := p.store.QueryDeploymentEvents(contract.ContractDID, periodStart, periodEnd, headContractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployment events: %w", err)
	}

	// Build deployment windows (processor-specific logic)
	windows := p.buildDeploymentWindows(startEvents, stopEvents)

	// Calculate deployment time utilization
	deployments, totalTimeSec := p.calculateDeploymentTimeUtilization(windows, periodStart, periodEnd, now)

	// Edge Case: No deployments during period - skip invoice
	if len(deployments) == 0 {
		return nil, ErrNoDeployments
	}

	// If totalTimeSec is zero or negative, skip invoice
	if totalTimeSec <= 0 {
		return nil, ErrNoDeployments
	}

	// Parse fee per time unit
	feePerUnit, err := convert.StringToFloat64(pd.FeePerTimeUnit, "fee_per_time_unit")
	if err != nil {
		return nil, err
	}

	// Convert total time to the specified time unit
	timeInUnit, err := convert.SecondsToUnit(totalTimeSec, pd.TimeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to convert time: %w", err)
	}

	// Calculate total amount (sum across all deployments for this period)
	// Note: Each deployment will get its own invoice, but this provides the combined total
	totalAmount := feePerUnit * timeInUnit

	periodsToInvoice := billingCyclesElapsed * pd.PaymentPeriodCount
	if pd.PaymentPeriodCount <= 0 {
		periodsToInvoice = billingCyclesElapsed
	}

	return &contracts.UsageData{
		ContractDID:  contract.ContractDID,
		PaymentModel: contracts.Periodic,
		Data: &contracts.PeriodicUsage{
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			LastInvoiceAt:   lastInvoiceAt,
			Deployments:     deployments,
			TotalTimeSec:    totalTimeSec,
			Amount:          formatAmount(totalAmount),
			PeriodsInvoiced: periodsToInvoice,
		},
	}, nil
}

// GenerateProRatedInvoice generates a pro-rated invoice for a terminated contract
// based on actual deployment time within the partial period (from lastInvoiceAt to now).
// Unlike CheckAndGenerateInvoice, this method does not require a full billing period to have elapsed.
func (p *PeriodicProcessor) GenerateProRatedInvoice(
	contract *contracts.Contract,
	lastInvoiceAt time.Time,
	now time.Time,
) (*contracts.UsageData, error) {
	pd := contract.PaymentDetails

	// For pro-rating, we invoice for the actual deployment time from lastInvoiceAt to now
	periodStart := lastInvoiceAt
	periodEnd := now

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

	// Query deployment start and stop events in the partial period
	startEvents, stopEvents, err := p.store.QueryDeploymentEvents(contract.ContractDID, periodStart, periodEnd, headContractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployment events: %w", err)
	}

	// Build deployment windows (processor-specific logic)
	windows := p.buildDeploymentWindows(startEvents, stopEvents)

	// Calculate deployment time utilization based on actual runtime
	deployments, totalTimeSec := p.calculateDeploymentTimeUtilization(windows, periodStart, periodEnd, now)

	// Edge Case: No deployments during partial period - skip invoice
	if len(deployments) == 0 {
		return nil, ErrNoDeployments
	}

	// If totalTimeSec is zero or negative, skip invoice
	if totalTimeSec <= 0 {
		return nil, ErrNoDeployments
	}

	// Parse fee per time unit
	feePerUnit, err := convert.StringToFloat64(pd.FeePerTimeUnit, "fee_per_time_unit")
	if err != nil {
		return nil, err
	}

	// Convert total time to the specified time unit
	timeInUnit, err := convert.SecondsToUnit(totalTimeSec, pd.TimeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to convert time: %w", err)
	}

	// Calculate pro-rated amount based on actual deployment time
	// This is the key difference: we charge for actual runtime, not a full period
	totalAmount := feePerUnit * timeInUnit

	return &contracts.UsageData{
		ContractDID:  contract.ContractDID,
		PaymentModel: contracts.Periodic,
		Data: &contracts.PeriodicUsage{
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			LastInvoiceAt:   lastInvoiceAt,
			Deployments:     deployments,
			TotalTimeSec:    totalTimeSec,
			Amount:          formatAmount(totalAmount),
			PeriodsInvoiced: 0, // Pro-rated, not a full period
		},
	}, nil
}

// buildDeploymentWindows is processor-specific logic for building deployment windows from events
func (p *PeriodicProcessor) buildDeploymentWindows(
	startEvents, stopEvents []*usage.Usage,
) map[string]*deploymentWindow {
	windows := make(map[string]*deploymentWindow)

	// Process start events
	for _, evt := range startEvents {
		if evt.EventType != events.DeploymentStartEvent {
			continue
		}

		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue
		}

		var data events.DeploymentStart
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		if windows[data.DeploymentID] == nil {
			windows[data.DeploymentID] = &deploymentWindow{
				deploymentID: data.DeploymentID,
				startTime:    eventTime,
				isComplete:   false,
			}
		}
	}

	// Process stop events
	for _, evt := range stopEvents {
		if evt.EventType != events.DeploymentStopEvent {
			continue
		}

		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue
		}

		var data events.DeploymentStop
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		window := windows[data.DeploymentID]
		if window != nil {
			window.endTime = eventTime
			window.isComplete = true
		}
	}

	return windows
}

// calculateDeploymentTimeUtilization calculates deployment time utilization from windows
func (p *PeriodicProcessor) calculateDeploymentTimeUtilization(
	windows map[string]*deploymentWindow,
	queryStart, queryEnd, now time.Time,
) ([]contracts.DeploymentTimeUtilization, float64) {
	deployments := make([]contracts.DeploymentTimeUtilization, 0)
	var totalTimeSec float64

	for _, window := range windows {
		// Use CalculateEffectiveTime helper to determine if deployment is relevant
		effectiveQueryEnd := queryEnd
		if !window.isComplete {
			effectiveQueryEnd = now
		}

		effectiveStart, effectiveEnd, valid := usage.CalculateEffectiveTime(
			window.startTime,
			window.endTime,
			window.isComplete,
			queryStart,
			effectiveQueryEnd,
		)
		if !valid {
			continue
		}

		// Calculate duration from effective start to effective end
		duration := effectiveEnd.Sub(effectiveStart)
		durationSec := duration.Seconds()

		if durationSec <= 0 {
			continue
		}

		deployments = append(deployments, contracts.DeploymentTimeUtilization{
			DeploymentID:        window.deploymentID,
			Allocations:         []contracts.AllocationTimeUtilization{}, // Empty - tracking at deployment level
			TotalUtilizationSec: durationSec,
		})

		totalTimeSec += durationSec
	}

	return deployments, totalTimeSec
}
