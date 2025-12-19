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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/types"
)

func TestPayPerResourceUtilizationProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerResourceUtilizationProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:             contracts.PayPerResourceUtilization,
				FeePerCPUCorePerTimeUnit: "0.1",
				FeePerRAMGBPerTimeUnit:   "0.05",
				FeePerDiskGBPerTimeUnit:  "0.02",
				ResourceTimeUnit:         "hour",
			},
			wantErr: false,
		},
		{
			name: "missing fee_per_cpu_core_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:            contracts.PayPerResourceUtilization,
				FeePerRAMGBPerTimeUnit:  "0.05",
				FeePerDiskGBPerTimeUnit: "0.02",
				ResourceTimeUnit:        "hour",
			},
			wantErr: true,
			errMsg:  "fee_per_cpu_core_per_time_unit is required",
		},
		{
			name: "missing fee_per_ram_gb_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:             contracts.PayPerResourceUtilization,
				FeePerCPUCorePerTimeUnit: "0.1",
				FeePerDiskGBPerTimeUnit:  "0.02",
				ResourceTimeUnit:         "hour",
			},
			wantErr: true,
			errMsg:  "fee_per_ram_gb_per_time_unit is required",
		},
		{
			name: "missing fee_per_disk_gb_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:             contracts.PayPerResourceUtilization,
				FeePerCPUCorePerTimeUnit: "0.1",
				FeePerRAMGBPerTimeUnit:   "0.05",
				ResourceTimeUnit:         "hour",
			},
			wantErr: true,
			errMsg:  "fee_per_disk_gb_per_time_unit is required",
		},
		{
			name: "missing resource_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:             contracts.PayPerResourceUtilization,
				FeePerCPUCorePerTimeUnit: "0.1",
				FeePerRAMGBPerTimeUnit:   "0.05",
				FeePerDiskGBPerTimeUnit:  "0.02",
			},
			wantErr: true,
			errMsg:  "resource_time_unit is required",
		},
		{
			name: "invalid fee_per_cpu_core_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:             contracts.PayPerResourceUtilization,
				FeePerCPUCorePerTimeUnit: "invalid",
				FeePerRAMGBPerTimeUnit:   "0.05",
				FeePerDiskGBPerTimeUnit:  "0.02",
				ResourceTimeUnit:         "hour",
			},
			wantErr: true,
			errMsg:  "invalid fee_per_cpu_core_per_time_unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.Validate(tt.paymentDetails)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPayPerResourceUtilizationProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerResourceUtilizationProcessor(store)

	contractDID := "test-contract-1"
	lastProcessedAt := time.Now().Add(-2 * time.Hour)

	// Add allocation events with resources
	resources := types.Resources{
		CPU:  types.CPU{Cores: 2},
		RAM:  types.RAM{Size: 4 * 1024 * 1024 * 1024},   // 4 GB
		Disk: types.Disk{Size: 20 * 1024 * 1024 * 1024}, // 20 GB
	}
	addStartAllocationEventWithResources(t, store, contractDID, "alloc-1", "deploy-1", resources, lastProcessedAt.Add(30*time.Minute))
	addCompleteAllocationEvent(t, store, contractDID, "alloc-1", "deploy-1", lastProcessedAt.Add(1*time.Hour))

	now := time.Now()

	usageData, err := processor.CollectUsage(contractDID, lastProcessedAt, now)
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contractDID, usageData.ContractDID)
	require.Equal(t, contracts.PayPerResourceUtilization, usageData.PaymentModel)

	resourceUtil, ok := usageData.Data.(*contracts.ResourceUtilizationUsage)
	require.True(t, ok)
	require.NotNil(t, resourceUtil)
}

func TestPayPerResourceUtilizationProcessor_CalculatePayment(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerResourceUtilizationProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:             contracts.PayPerResourceUtilization,
			FeePerCPUCorePerTimeUnit: "0.1",
			FeePerRAMGBPerTimeUnit:   "0.05",
			FeePerDiskGBPerTimeUnit:  "0.02",
			ResourceTimeUnit:         "hour",
		},
	}

	// Create usage data with 1 hour of utilization
	resourceUtil := &contracts.ResourceUtilizationUsage{
		Deployments: []contracts.DeploymentResourceUtilization{
			{
				DeploymentID:        "deploy-1",
				TotalUtilizationSec: 3600.0, // 1 hour
				Allocations: []contracts.AllocationResourceUtilization{
					{
						AllocationID: "alloc-1",
						Resources: types.Resources{
							CPU:  types.CPU{Cores: 2},
							RAM:  types.RAM{Size: 4 * 1024 * 1024 * 1024},   // 4 GB
							Disk: types.Disk{Size: 20 * 1024 * 1024 * 1024}, // 20 GB
						},
						Duration: 1 * time.Hour,
					},
				},
			},
		},
	}

	usageData := &contracts.UsageData{
		ContractDID:  "test-contract-1",
		PaymentModel: contracts.PayPerResourceUtilization,
		Data:         resourceUtil,
	}

	items, err := processor.CalculatePayment(usageData, contract)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "deploy-1", items[0].DeploymentID)
	// CPU: 2 cores * 0.1 * 1 hour = 0.2
	// RAM: 4 GB * 0.05 * 1 hour = 0.2
	// Disk: 20 GB * 0.02 * 1 hour = 0.4
	// Total: 0.8
	require.Equal(t, "0.80000000", items[0].Amount)
}

func TestPayPerResourceUtilizationProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerResourceUtilizationProcessor(store)

	require.True(t, processor.SupportsManualBilling())
	require.False(t, processor.SupportsAutomaticBilling())
}

func addStartAllocationEventWithResources(t *testing.T, store *usage.Store, contractDID, allocationID, deploymentID string, resources types.Resources, timestamp time.Time) {
	t.Helper()
	event := events.StartAllocation{
		EventBase:      events.EventBase{Type: events.StartAllocationEvent},
		AllocationBase: events.AllocationBase{AllocationID: allocationID, DeploymentID: deploymentID},
		Resources:      resources,
	}
	eventData, err := json.Marshal(event)
	require.NoError(t, err)

	err = store.AddUsageEvent(usage.Usage{
		ContractDID: contractDID,
		EventType:   events.StartAllocationEvent,
		Data:        eventData,
		Timestamp:   timestamp,
	})
	require.NoError(t, err)
}
