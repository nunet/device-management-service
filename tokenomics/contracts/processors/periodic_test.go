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

func TestPeriodicProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPeriodicProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.Periodic,
				FeePerTimeUnit: "0.1",
				TimeUnit:       "hour",
				PaymentPeriod:  contracts.PaymentPeriodDay,
			},
			wantErr: false,
		},
		{
			name: "missing fee_per_time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:  contracts.Periodic,
				TimeUnit:      "hour",
				PaymentPeriod: contracts.PaymentPeriodDay,
			},
			wantErr: true,
			errMsg:  "fee_per_time_unit is required",
		},
		{
			name: "missing time_unit",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.Periodic,
				FeePerTimeUnit: "0.1",
				PaymentPeriod:  contracts.PaymentPeriodDay,
			},
			wantErr: true,
			errMsg:  "time_unit is required",
		},
		{
			name: "missing payment_period",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:   contracts.Periodic,
				FeePerTimeUnit: "0.1",
				TimeUnit:       "hour",
			},
			wantErr: true,
			errMsg:  "payment_period is required",
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

func TestPeriodicProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPeriodicProcessor(store)

	usageData, err := processor.CollectUsage("test-contract-1", time.Now(), time.Now(), "", "")
	require.Error(t, err)
	require.Nil(t, usageData)
	require.Contains(t, err.Error(), "does not support manual billing")
}

func TestPeriodicProcessor_CalculatePayment(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPeriodicProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:   contracts.Periodic,
			FeePerTimeUnit: "0.1",
			TimeUnit:       "hour",
		},
	}

	periodicUsage := &contracts.PeriodicUsage{
		PeriodStart:   time.Now().Add(-24 * time.Hour),
		PeriodEnd:     time.Now(),
		LastInvoiceAt: time.Now().Add(-24 * time.Hour),
		Deployments: []contracts.DeploymentTimeUtilization{
			{
				DeploymentID:        "deploy-1",
				TotalUtilizationSec: 3600.0, // 1 hour
			},
		},
		TotalTimeSec:    3600.0,
		Amount:          "0.1",
		PeriodsInvoiced: 1,
	}

	usageData := &contracts.UsageData{
		ContractDID:  "test-contract-1",
		PaymentModel: contracts.Periodic,
		Data:         periodicUsage,
	}

	items, err := processor.CalculatePayment(usageData, contract)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "deploy-1", items[0].DeploymentID)
	require.Equal(t, "0.10000000", items[0].Amount) // 1 hour * 0.1 per hour
}

func TestPeriodicProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPeriodicProcessor(store)

	require.False(t, processor.SupportsManualBilling())
	require.True(t, processor.SupportsAutomaticBilling())
}

func TestPeriodicProcessor_CheckAndGenerateInvoice(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPeriodicProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:       contracts.Periodic,
			FeePerTimeUnit:     "0.1",
			TimeUnit:           "hour",
			PaymentPeriod:      contracts.PaymentPeriodDay,
			PaymentPeriodCount: 1,
		},
	}

	// Test: period not elapsed
	lastInvoiceAt := time.Now().Add(-30 * time.Minute)
	now := time.Now()
	usageData, err := processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	require.Error(t, err)
	require.Nil(t, usageData)
	require.Contains(t, err.Error(), "billing period has not elapsed")

	// Test: period elapsed but no deployments
	lastInvoiceAt = time.Now().Add(-25 * time.Hour) // More than 1 day
	usageData, err = processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	require.Error(t, err)
	require.Nil(t, usageData)
	require.Contains(t, err.Error(), "no deployments active")

	// Test: period elapsed with deployments
	// Add deployment events
	addDeploymentStartEvent(t, store, contract.ContractDID, "deploy-1", "orch-1", lastInvoiceAt.Add(1*time.Hour))
	addDeploymentStopEvent(t, store, contract.ContractDID, "deploy-1", "orch-1", lastInvoiceAt.Add(2*time.Hour))

	now = time.Now()
	usageData, err = processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contracts.Periodic, usageData.PaymentModel)

	periodicUsage, ok := usageData.Data.(*contracts.PeriodicUsage)
	require.True(t, ok)
	require.NotEmpty(t, periodicUsage.Amount)
	require.Greater(t, len(periodicUsage.Deployments), 0)
}

func addDeploymentStopEvent(t *testing.T, store *usage.Store, contractDID, deploymentID, orchestratorID string, timestamp time.Time) {
	t.Helper()
	event := events.DeploymentStop{
		EventBase:      events.EventBase{Type: events.DeploymentStopEvent},
		DeploymentID:   deploymentID,
		OrchestratorID: orchestratorID,
	}
	eventData, err := json.Marshal(event)
	require.NoError(t, err)

	err = store.AddUsageEvent(usage.Usage{
		ContractDID: contractDID,
		EventType:   events.DeploymentStopEvent,
		Data:        eventData,
		Timestamp:   timestamp,
	})
	require.NoError(t, err)
}
