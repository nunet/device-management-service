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
)

func TestPayPerTimeUtilizationProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerTimeUtilizationProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.PayPerTimeUtilization,
				FeePerTimeUnit: "0.1",
				TimeUnit:       "hour",
			},
			wantErr: false,
		},
		{
			name: "missing fee_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel: contracts.PayPerTimeUtilization,
				TimeUnit:     "hour",
			},
			wantErr: true,
			errMsg:  "fee_per_time_unit is required",
		},
		{
			name: "missing time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.PayPerTimeUtilization,
				FeePerTimeUnit: "0.1",
			},
			wantErr: true,
			errMsg:  "time_unit is required",
		},
		{
			name: "invalid fee_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.PayPerTimeUtilization,
				FeePerTimeUnit: "invalid",
				TimeUnit:       "hour",
			},
			wantErr: true,
			errMsg:  "invalid fee_per_time_unit",
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

func TestPayPerTimeUtilizationProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerTimeUtilizationProcessor(store)

	contractDID := "test-contract-1"
	lastProcessedAt := time.Now().Add(-2 * time.Hour)

	// Add allocation events
	addStartAllocationEvent(t, store, contractDID, "alloc-1", "deploy-1", lastProcessedAt.Add(30*time.Minute))
	addCompleteAllocationEvent(t, store, contractDID, "alloc-1", "deploy-1", lastProcessedAt.Add(1*time.Hour))

	now := time.Now()

	usageData, err := processor.CollectUsage(contractDID, lastProcessedAt, now, "", "")
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contractDID, usageData.ContractDID)
	require.Equal(t, contracts.PayPerTimeUtilization, usageData.PaymentModel)

	timeUtil, ok := usageData.Data.(*contracts.TimeUtilizationUsage)
	require.True(t, ok)
	require.NotNil(t, timeUtil)
}

func TestPayPerTimeUtilizationProcessor_CalculatePayment(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerTimeUtilizationProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:   contracts.PayPerTimeUtilization,
			FeePerTimeUnit: "0.1",
			TimeUnit:       "hour",
		},
	}

	// Create usage data with 2 hours of utilization
	timeUtil := &contracts.TimeUtilizationUsage{
		Deployments: []contracts.DeploymentTimeUtilization{
			{
				DeploymentID:        "deploy-1",
				TotalUtilizationSec: 7200.0, // 2 hours
			},
		},
	}

	usageData := &contracts.UsageData{
		ContractDID:  "test-contract-1",
		PaymentModel: contracts.PayPerTimeUtilization,
		Data:         timeUtil,
	}

	items, err := processor.CalculatePayment(usageData, contract)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "deploy-1", items[0].DeploymentID)
	require.Equal(t, "0.20000000", items[0].Amount) // 2 hours * 0.1 per hour
}

func TestPayPerTimeUtilizationProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerTimeUtilizationProcessor(store)

	require.True(t, processor.SupportsManualBilling())
	require.True(t, processor.SupportsAutomaticBilling())
}

func addCompleteAllocationEvent(t *testing.T, store *usage.Store, contractDID, allocationID, deploymentID string, timestamp time.Time) {
	t.Helper()
	event := events.CompleteAllocation{
		EventBase:      events.EventBase{Type: events.CompleteAllocationEvent},
		AllocationBase: events.AllocationBase{AllocationID: allocationID, DeploymentID: deploymentID},
	}
	eventData, err := json.Marshal(event)
	require.NoError(t, err)

	err = store.AddUsageEvent(usage.Usage{
		ContractDID: contractDID,
		EventType:   events.CompleteAllocationEvent,
		Data:        eventData,
		Timestamp:   timestamp,
	})
	require.NoError(t, err)
}
