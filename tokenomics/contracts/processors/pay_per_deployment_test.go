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

func TestPayPerDeploymentProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerDeploymentProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:     contracts.PayPerDeployment,
				FeePerDeployment: "25.0",
			},
			wantErr: false,
		},
		{
			name: "missing fee_per_deployment",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel: contracts.PayPerDeployment,
			},
			wantErr: true,
			errMsg:  "fee_per_deployment is required",
		},
		{
			name: "invalid fee_per_deployment",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:     contracts.PayPerDeployment,
				FeePerDeployment: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid fee_per_deployment",
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

func TestPayPerDeploymentProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerDeploymentProcessor(store)

	contractDID := "test-contract-1" //nolint:goconst
	lastProcessedAt := time.Now().Add(-2 * time.Hour)

	// Add some deployment events
	addDeploymentStartEvent(t, store, contractDID, "deploy-1", "orch-1", lastProcessedAt.Add(30*time.Minute))
	addDeploymentStartEvent(t, store, contractDID, "deploy-2", "orch-1", lastProcessedAt.Add(1*time.Hour))

	now := time.Now()

	usageData, err := processor.CollectUsage(contractDID, lastProcessedAt, now)
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contractDID, usageData.ContractDID)
	require.Equal(t, contracts.PayPerDeployment, usageData.PaymentModel)

	usageCount, ok := usageData.Data.(int)
	require.True(t, ok)
	require.GreaterOrEqual(t, usageCount, 2) // At least 2 deployments
}

func TestPayPerDeploymentProcessor_CalculatePayment(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerDeploymentProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:     contracts.PayPerDeployment,
			FeePerDeployment: "25.0",
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
				ContractDID:  "test-contract-1",
				PaymentModel: contracts.PayPerDeployment,
				Data:         3, // 3 deployments
			},
			wantItems: 1,
			wantErr:   false,
		},
		{
			name: "zero deployments",
			usageData: &contracts.UsageData{
				ContractDID:  "test-contract-1",
				PaymentModel: contracts.PayPerDeployment,
				Data:         0,
			},
			wantItems: 0,
			wantErr:   false,
		},
		{
			name: "invalid usage data type",
			usageData: &contracts.UsageData{
				ContractDID:  "test-contract-1",
				PaymentModel: contracts.PayPerDeployment,
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
					require.Equal(t, 3, items[0].Usages)
				}
			}
		})
	}
}

func TestPayPerDeploymentProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewPayPerDeploymentProcessor(store)

	require.True(t, processor.SupportsManualBilling())
	require.False(t, processor.SupportsAutomaticBilling())
}

func addDeploymentStartEvent(t *testing.T, store *usage.Store, contractDID, deploymentID, orchestratorID string, timestamp time.Time) {
	t.Helper()
	event := events.DeploymentStart{
		EventBase:      events.EventBase{Type: events.DeploymentStartEvent},
		DeploymentID:   deploymentID,
		OrchestratorID: orchestratorID,
	}
	eventData, err := json.Marshal(event)
	require.NoError(t, err)

	err = store.AddUsageEvent(usage.Usage{
		ContractDID: contractDID,
		EventType:   events.DeploymentStartEvent,
		Data:        eventData,
		Timestamp:   timestamp,
	})
	require.NoError(t, err)
}
