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
) (*contracts.UsageData, error) {
	// Use store's abstract query method
	startEvents, endEvents, err := p.store.QueryAllocationEvents(contractDID, lastProcessedAt, now)
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

		items = append(items, &contracts.PaymentItem{
			UniqueID:     "", // Will be set by payment processor
			DeploymentID: deployment.DeploymentID,
			Amount:       formatAmount(amount),
			Usages:       1,
			Metadata: map[string]interface{}{
				"total_utilization_sec": deployment.TotalUtilizationSec,
			},
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
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *PayPerTimeUtilizationProcessor) SupportsManualBilling() bool {
	return true
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PayPerTimeUtilizationProcessor) SupportsAutomaticBilling() bool {
	return false
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
// This payment model does not support automatic billing, so this should never be called.
func (p *PayPerTimeUtilizationProcessor) CheckAndGenerateInvoice(
	_ *contracts.Contract,
	_ time.Time,
	_ time.Time,
) (*contracts.UsageData, error) {
	return nil, fmt.Errorf("pay_per_time_utilization does not support automatic billing")
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
