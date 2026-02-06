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

	"github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

func TestPayPerAllocationProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerAllocationProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:     contracts.PayPerAllocation,
				FeePerAllocation: "10.5",
			},
			wantErr: false,
		},
		{
			name: "missing fees_per_allocation",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel: contracts.PayPerAllocation,
			},
			wantErr: true,
			errMsg:  "fees_per_allocation is required",
		},
		{
			name: "invalid fees_per_allocation",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:     contracts.PayPerAllocation,
				FeePerAllocation: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid fees_per_allocation",
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

func TestPayPerAllocationProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerAllocationProcessor(store)

	const contractDID = "test-contract-1"
	lastProcessedAt := time.Now().Add(-2 * time.Hour)

	// Add some allocation events
	addStartAllocationEvent(t, store, contractDID, "alloc-1", "deploy-1", lastProcessedAt.Add(30*time.Minute))
	addStartAllocationEvent(t, store, contractDID, "alloc-2", "deploy-1", lastProcessedAt.Add(1*time.Hour))

	now := time.Now()

	usageData, err := processor.CollectUsage(contractDID, lastProcessedAt, now, "", "")
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contractDID, usageData.ContractDID)
	require.Equal(t, contracts.PayPerAllocation, usageData.PaymentModel)

	usageCount, ok := usageData.Data.(int)
	require.True(t, ok)
	require.GreaterOrEqual(t, usageCount, 2) // At least 2 allocations
}

func TestPayPerAllocationProcessor_CalculatePayment(t *testing.T) {
	const contractDID = "test-contract-1"
	store := setupTestUsageStore(t)
	processor := NewPayPerAllocationProcessor(store)

	contract := &contracts.Contract{
		ContractDID: contractDID,
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:     contracts.PayPerAllocation,
			FeePerAllocation: "10.5",
		},
	}

	tests := []struct {
		name      string
		usageData *contracts.UsageData
		wantItems int
		wantErr   bool
	}{
		{
			name: "valid usage data",
			usageData: &contracts.UsageData{
				ContractDID:  contractDID,
				PaymentModel: contracts.PayPerAllocation,
				Data:         5, // 5 allocations
			},
			wantItems: 1,
			wantErr:   false,
		},
		{
			name: "zero allocations",
			usageData: &contracts.UsageData{
				ContractDID:  contractDID,
				PaymentModel: contracts.PayPerAllocation,
				Data:         0,
			},
			wantItems: 0,
			wantErr:   false,
		},
		{
			name: "invalid usage data type",
			usageData: &contracts.UsageData{
				ContractDID:  contractDID,
				PaymentModel: contracts.PayPerAllocation,
				Data:         "invalid",
			},
			wantItems: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := processor.CalculatePayment(tt.usageData, contract)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, items, tt.wantItems)
				if tt.wantItems > 0 {
					require.NotEmpty(t, items[0].Amount)
					require.Equal(t, 5, items[0].Usages)
				}
			}
		})
	}
}

func TestPayPerAllocationProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerAllocationProcessor(store)

	require.True(t, processor.SupportsManualBilling())
	require.True(t, processor.SupportsAutomaticBilling())
}

func TestPayPerAllocationProcessor_CheckAndGenerateInvoice(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerAllocationProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:       contracts.PayPerAllocation,
			PaymentPeriod:      contracts.PaymentPeriodHour,
			PaymentPeriodCount: 1,
		},
	}

	// Test: period not elapsed should return ErrPeriodNotElapsed
	now := time.Now()
	usageData, err := processor.CheckAndGenerateInvoice(contract, now, now)
	require.Error(t, err)
	require.Nil(t, usageData)
	require.ErrorIs(t, err, ErrPeriodNotElapsed)

	// Test: period elapsed should collect usage
	lastInvoiceAt := now.Add(-2 * time.Hour) // 2 hours ago
	usageData, err = processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contracts.PayPerAllocation, usageData.PaymentModel)
}

// Helper functions for tests
func setupTestUsageStore(t *testing.T) *usage.Store {
	t.Helper()
	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	require.NoError(t, err, "failed to open CloverDB")

	err = db.CreateCollection("contracts_usage")
	require.NoError(t, err, "failed to create collection")

	store, err := usage.New(db)
	require.NoError(t, err, "failed to create store")

	return store
}

func addStartAllocationEvent(t *testing.T, store *usage.Store, contractDID, allocationID, deploymentID string, timestamp time.Time) {
	t.Helper()
	event := events.StartAllocation{
		EventBase:      events.EventBase{Type: events.StartAllocationEvent},
		AllocationBase: events.AllocationBase{AllocationID: allocationID, DeploymentID: deploymentID},
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
