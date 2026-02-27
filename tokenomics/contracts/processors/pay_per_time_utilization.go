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

// PayPerTimeUtilizationProcessor implements PaymentModelProcessor for pay_per_time_utilization model.
// This processor collects allocation time utilization and calculates payment based on time duration.
var _ contracts.PaymentModelProcessor = (*PayPerTimeUtilizationProcessor)(nil)

type PayPerTimeUtilizationProcessor struct {
	store *usage.Store
}

func NewPayPerTimeUtilizationProcessor(store *usage.Store) *PayPerTimeUtilizationProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &PayPerTimeUtilizationProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
func (p *PayPerTimeUtilizationProcessor) CollectUsage(
	contractDID string,
	lastProcessedAt time.Time,
	now time.Time,
	_ string, // providerDID - unused in this processor
	headContractDID string, // New parameter
) (*contracts.UsageData, error) {
	// Use store's abstract query method
	// If headContractDID is provided, query by Head Contract DID; otherwise query by contractDID
	startEvents, endEvents, err := p.store.QueryAllocationEvents(contractDID, lastProcessedAt, now, headContractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to query allocation events: %w", err)
	}

	// Build allocation windows (processor-specific logic)
	windows := p.buildAllocationWindows(startEvents, endEvents)

	// Group by deployment and calculate time utilization
	deployments := p.calculateDeploymentTimeUtilization(windows, lastProcessedAt, now)

	return &contracts.UsageData{
		ContractDID:  contractDID,
		PaymentModel: contracts.PayPerTimeUtilization,
		Data:         &contracts.TimeUtilizationUsage{Deployments: deployments},
	}, nil
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *PayPerTimeUtilizationProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	contract *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	timeUtil, ok := usageData.Data.(*contracts.TimeUtilizationUsage)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type")
	}

	pd := contract.PaymentDetails
	feePerUnit, err := convert.StringToFloat64(pd.FeePerTimeUnit, "fee_per_time_unit")
	if err != nil {
		return nil, err
	}

	items := make([]*contracts.PaymentItem, 0)
	for _, deployment := range timeUtil.Deployments {
		// Use utility function for time conversion
		timeInUnit, err := convert.SecondsToUnit(deployment.TotalUtilizationSec, pd.TimeUnit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert time: %w", err)
		}

		amount := feePerUnit * timeInUnit

		// Generate unique UUID for this payment item
		uniqueID := uuid.NewString()

		// Build enriched metadata
		metadata := map[string]interface{}{
			"total_utilization_sec": deployment.TotalUtilizationSec,
			"allocation_count":      len(deployment.Allocations),
		}

		// Add allocation details
		allocations := make([]map[string]interface{}, 0, len(deployment.Allocations))
		for _, alloc := range deployment.Allocations {
			allocData := map[string]interface{}{
				"allocation_id":        alloc.AllocationID,
				"duration_sec":         alloc.Duration.Seconds(),
				"start_time":           alloc.StartTime.Format(time.RFC3339),
				"payment_model":        contracts.PayPerTimeUtilization,
				"payment_period":       pd.PaymentPeriod,
				"payment_period_count": pd.PaymentPeriodCount,
				"fee_per_time_unit":    pd.FeePerTimeUnit,
				"time_unit":            pd.TimeUnit,
			}
			if !alloc.EndTime.IsZero() {
				allocData["end_time"] = alloc.EndTime.Format(time.RFC3339)
			}
			allocations = append(allocations, allocData)
		}
		metadata["allocations"] = allocations

		items = append(items, &contracts.PaymentItem{
			UniqueID:     uniqueID, // Generated UUID
			DeploymentID: deployment.DeploymentID,
			Amount:       formatAmount(amount),
			Usages:       1,
			Metadata:     metadata,
		})
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *PayPerTimeUtilizationProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FeePerTimeUnit == "" {
		return fmt.Errorf("fee_per_time_unit is required")
	}
	if paymentDetails.TimeUnit == "" {
		return fmt.Errorf("time_unit is required")
	}
	if _, err := convert.StringToFloat64(paymentDetails.FeePerTimeUnit, "fee_per_time_unit"); err != nil {
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
func (p *PayPerTimeUtilizationProcessor) SupportsManualBilling() bool {
	return true
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PayPerTimeUtilizationProcessor) SupportsAutomaticBilling() bool {
	return true
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *PayPerTimeUtilizationProcessor) CheckAndGenerateInvoice(
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
				headContractDID = contract.ContractDID
			}
		}
	}
	return p.CollectUsage(contract.ContractDID, lastInvoiceAt, now, "", headContractDID)
}

// buildAllocationWindows is processor-specific logic
func (p *PayPerTimeUtilizationProcessor) buildAllocationWindows(
	startEvents, endEvents []*usage.Usage,
) map[string]*allocationWindow {
	windows := make(map[string]*allocationWindow)

	// Process start events
	for _, evt := range startEvents {
		if evt.Timestamp.IsZero() {
			continue
		}

		var data events.StartAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		windows[data.AllocationID] = &allocationWindow{
			allocationID: data.AllocationID,
			deploymentID: data.DeploymentID,
			startTime:    evt.Timestamp,
		}
	}

	// Process end events using shared helper
	for _, evt := range endEvents {
		allocationID, ok := processAllocationEndEvent(evt)
		if !ok {
			continue
		}

		window := windows[allocationID]
		if window != nil {
			window.endTime = evt.Timestamp
			window.isComplete = true
		}
	}

	return windows
}

// calculateDeploymentTimeUtilization groups windows by deployment
func (p *PayPerTimeUtilizationProcessor) calculateDeploymentTimeUtilization(
	windows map[string]*allocationWindow,
	queryStart, queryEnd time.Time,
) []contracts.DeploymentTimeUtilization {
	deploymentMap := make(map[string]*contracts.DeploymentTimeUtilization)

	for _, window := range windows {
		effectiveStart, effectiveEnd, valid := usage.CalculateEffectiveTime(
			window.startTime, window.endTime, window.isComplete, queryStart, queryEnd,
		)
		if !valid {
			continue
		}

		duration := effectiveEnd.Sub(effectiveStart)

		if deploymentMap[window.deploymentID] == nil {
			deploymentMap[window.deploymentID] = &contracts.DeploymentTimeUtilization{
				DeploymentID: window.deploymentID,
				Allocations:  make([]contracts.AllocationTimeUtilization, 0),
			}
		}

		allocUtil := contracts.AllocationTimeUtilization{
			AllocationID: window.allocationID,
			Duration:     duration,
			StartTime:    window.startTime, // Always use actual start time for tracking
		}
		if window.isComplete {
			allocUtil.EndTime = window.endTime
		}

		deploymentMap[window.deploymentID].Allocations = append(
			deploymentMap[window.deploymentID].Allocations,
			allocUtil,
		)
		deploymentMap[window.deploymentID].TotalUtilizationSec += duration.Seconds()
	}

	result := make([]contracts.DeploymentTimeUtilization, 0, len(deploymentMap))
	for _, deployment := range deploymentMap {
		result = append(result, *deployment)
	}

	return result
}
