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
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// PayPerResourceUtilizationProcessor implements PaymentModelProcessor for pay_per_resource_utilization model.
// This processor collects resource utilization and calculates payment based on resources × time.
var _ contracts.PaymentModelProcessor = (*PayPerResourceUtilizationProcessor)(nil)

type PayPerResourceUtilizationProcessor struct {
	store *usage.Store
}

func NewPayPerResourceUtilizationProcessor(store *usage.Store) *PayPerResourceUtilizationProcessor {
	if store == nil {
		panic("usage store cannot be nil")
	}
	return &PayPerResourceUtilizationProcessor{store: store}
}

// CollectUsage implements PaymentModelProcessor.CollectUsage
func (p *PayPerResourceUtilizationProcessor) CollectUsage(
	contractDID string,
	lastProcessedAt time.Time,
	now time.Time,
	_ string, // providerDID - unused in this processor
	headContractDID string, // New parameter
) (*contracts.UsageData, error) {
	// Use store's abstract query methods
	// If headContractDID is provided, query by Head Contract DID; otherwise query by contractDID
	startEvents, endEvents, err := p.store.QueryAllocationEvents(contractDID, lastProcessedAt, now, headContractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to query allocation events: %w", err)
	}

	// Query create events for resource fallback
	createEvents, err := p.store.QueryCreateAllocationEvents(contractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to query create allocation events: %w", err)
	}

	// Build resource map from create events (fallback)
	allocationResources := make(map[string]types.Resources)
	for _, evt := range createEvents {
		var data events.CreateAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}
		allocationResources[data.AllocationID] = data.Resources
	}

	// Build allocation windows with resources
	windows := p.buildAllocationWindowsWithResources(startEvents, endEvents, allocationResources)

	// Group by deployment and calculate resource utilization
	deployments := p.calculateDeploymentResourceUtilization(windows, lastProcessedAt, now)

	return &contracts.UsageData{
		ContractDID:  contractDID,
		PaymentModel: contracts.PayPerResourceUtilization,
		Data:         &contracts.ResourceUtilizationUsage{Deployments: deployments},
	}, nil
}

// CalculatePayment implements PaymentModelProcessor.CalculatePayment
func (p *PayPerResourceUtilizationProcessor) CalculatePayment(
	usageData *contracts.UsageData,
	contract *contracts.Contract,
) ([]*contracts.PaymentItem, error) {
	resourceUtil, ok := usageData.Data.(*contracts.ResourceUtilizationUsage)
	if !ok {
		return nil, fmt.Errorf("invalid usage data type")
	}

	pd := contract.PaymentDetails
	feePerCPUCore, err := convert.StringToFloat64(pd.FeePerCPUCorePerTimeUnit, "fee_per_cpu_core_per_time_unit")
	if err != nil {
		return nil, err
	}

	feePerRAMGB, err := convert.StringToFloat64(pd.FeePerRAMGBPerTimeUnit, "fee_per_ram_gb_per_time_unit")
	if err != nil {
		return nil, err
	}

	feePerDiskGB, err := convert.StringToFloat64(pd.FeePerDiskGBPerTimeUnit, "fee_per_disk_gb_per_time_unit")
	if err != nil {
		return nil, err
	}

	var feePerGPU float64
	if pd.FeePerGPUPerTimeUnit != "" {
		feePerGPU, err = convert.StringToFloat64(pd.FeePerGPUPerTimeUnit, "fee_per_gpu_per_time_unit")
		if err != nil {
			return nil, err
		}
	}

	items := make([]*contracts.PaymentItem, 0)
	for _, deployment := range resourceUtil.Deployments {
		var deploymentTotalCost float64

		// Process each allocation in the deployment
		for _, allocation := range deployment.Allocations {
			const bytesInGB = 1024 * 1024 * 1024

			// Convert duration to time unit
			timeInUnit, err := convert.DurationToUnit(allocation.Duration, pd.ResourceTimeUnit)
			if err != nil {
				return nil, fmt.Errorf("failed to convert duration: %w", err)
			}

			// Calculate costs per resource
			// Note: RAM.Size and Disk.Size are in bytes, need to convert to GB
			cpuCost := float64(allocation.Resources.CPU.Cores) * feePerCPUCore * timeInUnit
			ramCostGB := float64(allocation.Resources.RAM.Size) / float64(bytesInGB) // Convert bytes to GB (binary)
			ramCost := ramCostGB * feePerRAMGB * timeInUnit
			diskCostGB := float64(allocation.Resources.Disk.Size) / float64(bytesInGB) // Convert bytes to GB (binary)
			diskCost := diskCostGB * feePerDiskGB * timeInUnit

			var gpuCost float64
			if len(allocation.Resources.GPUs) > 0 && feePerGPU > 0 {
				gpuCost = float64(len(allocation.Resources.GPUs)) * feePerGPU * timeInUnit
			}

			allocationCost := cpuCost + ramCost + diskCost + gpuCost
			deploymentTotalCost += allocationCost
		}

		items = append(items, &contracts.PaymentItem{
			UniqueID:     "", // Will be set by payment processor
			DeploymentID: deployment.DeploymentID,
			Amount:       formatAmount(deploymentTotalCost),
			Usages:       1,
			Metadata: map[string]interface{}{
				"total_utilization_sec": deployment.TotalUtilizationSec,
			},
		})
	}

	return items, nil
}

// Validate implements PaymentModelProcessor.Validate
func (p *PayPerResourceUtilizationProcessor) Validate(paymentDetails contracts.PaymentDetails) error {
	if paymentDetails.FeePerCPUCorePerTimeUnit == "" {
		return fmt.Errorf("fee_per_cpu_core_per_time_unit is required")
	}
	if paymentDetails.FeePerRAMGBPerTimeUnit == "" {
		return fmt.Errorf("fee_per_ram_gb_per_time_unit is required")
	}
	if paymentDetails.FeePerDiskGBPerTimeUnit == "" {
		return fmt.Errorf("fee_per_disk_gb_per_time_unit is required")
	}
	if paymentDetails.ResourceTimeUnit == "" {
		return fmt.Errorf("resource_time_unit is required")
	}

	if _, err := convert.StringToFloat64(paymentDetails.FeePerCPUCorePerTimeUnit, "fee_per_cpu_core_per_time_unit"); err != nil {
		return err
	}
	if _, err := convert.StringToFloat64(paymentDetails.FeePerRAMGBPerTimeUnit, "fee_per_ram_gb_per_time_unit"); err != nil {
		return err
	}
	if _, err := convert.StringToFloat64(paymentDetails.FeePerDiskGBPerTimeUnit, "fee_per_disk_gb_per_time_unit"); err != nil {
		return err
	}
	if paymentDetails.FeePerGPUPerTimeUnit != "" {
		if _, err := convert.StringToFloat64(paymentDetails.FeePerGPUPerTimeUnit, "fee_per_gpu_per_time_unit"); err != nil {
			return err
		}
	}
	return nil
}

// SupportsManualBilling implements PaymentModelProcessor.SupportsManualBilling
func (p *PayPerResourceUtilizationProcessor) SupportsManualBilling() bool {
	return true
}

// SupportsAutomaticBilling implements PaymentModelProcessor.SupportsAutomaticBilling
func (p *PayPerResourceUtilizationProcessor) SupportsAutomaticBilling() bool {
	return true
}

// CheckAndGenerateInvoice implements PaymentModelProcessor.CheckAndGenerateInvoice
func (p *PayPerResourceUtilizationProcessor) CheckAndGenerateInvoice(
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

// buildAllocationWindowsWithResources builds windows with resources from events
func (p *PayPerResourceUtilizationProcessor) buildAllocationWindowsWithResources(
	startEvents, endEvents []*usage.Usage,
	allocationResources map[string]types.Resources,
) map[string]*allocationWindowWithResources {
	windows := make(map[string]*allocationWindowWithResources)

	// Process start events
	for _, evt := range startEvents {
		if evt.Timestamp.IsZero() {
			continue
		}

		var data events.StartAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		// Get resources from StartAllocationEvent (primary source)
		resources := data.Resources

		// Fallback: If resources not in StartAllocationEvent, use CreateAllocationEvent
		if resources.CPU.Cores == 0 && resources.RAM.Size == 0 {
			if createRes, ok := allocationResources[data.AllocationID]; ok {
				resources = createRes
			}
		}

		windows[data.AllocationID] = &allocationWindowWithResources{
			allocationWindow: allocationWindow{
				allocationID: data.AllocationID,
				deploymentID: data.DeploymentID,
				startTime:    evt.Timestamp,
			},
			resources: resources,
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

// calculateDeploymentResourceUtilization groups windows by deployment with resources
func (p *PayPerResourceUtilizationProcessor) calculateDeploymentResourceUtilization(
	windows map[string]*allocationWindowWithResources,
	queryStart, queryEnd time.Time,
) []contracts.DeploymentResourceUtilization {
	deploymentMap := make(map[string]*contracts.DeploymentResourceUtilization)

	for _, window := range windows {
		effectiveStart, effectiveEnd, valid := usage.CalculateEffectiveTime(
			window.startTime, window.endTime, window.isComplete, queryStart, queryEnd,
		)
		if !valid {
			continue
		}

		duration := effectiveEnd.Sub(effectiveStart)

		if deploymentMap[window.deploymentID] == nil {
			deploymentMap[window.deploymentID] = &contracts.DeploymentResourceUtilization{
				DeploymentID: window.deploymentID,
				Allocations:  make([]contracts.AllocationResourceUtilization, 0),
			}
		}

		allocUtil := contracts.AllocationResourceUtilization{
			AllocationID: window.allocationID,
			Resources:    window.resources,
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

	result := make([]contracts.DeploymentResourceUtilization, 0, len(deploymentMap))
	for _, deployment := range deploymentMap {
		result = append(result, *deployment)
	}

	return result
}

type allocationWindowWithResources struct {
	allocationWindow
	resources types.Resources
}
