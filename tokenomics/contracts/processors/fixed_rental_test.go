// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func TestFixedRentalProcessor_Validate(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewFixedRentalProcessor(store)

	tests := []struct {
		name           string
		paymentDetails contracts.PaymentDetails
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid payment details",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:       contracts.FixedRental,
				FixedRentalAmount:  "100.0",
				PaymentPeriod:      contracts.PaymentPeriodDay,
				PaymentPeriodCount: 1,
			},
			wantErr: false,
		},
		{
			name: "missing fixed_rental_amount",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:  contracts.FixedRental,
				PaymentPeriod: contracts.PaymentPeriodDay,
			},
			wantErr: true,
			errMsg:  "fixed_rental_amount is required",
		},
		{
			name: "missing payment_period",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:      contracts.FixedRental,
				FixedRentalAmount: "100.0",
			},
			wantErr: true,
			errMsg:  "payment_period is required",
		},
		{
			name: "invalid fixed_rental_amount",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:      contracts.FixedRental,
				FixedRentalAmount: "invalid",
				PaymentPeriod:     contracts.PaymentPeriodDay,
			},
			wantErr: true,
			errMsg:  "invalid fixed_rental_amount",
		},
		{
			name: "invalid payment_period",
			paymentDetails: contracts.PaymentDetails{
				PaymentModel:      contracts.FixedRental,
				FixedRentalAmount: "100.0",
				PaymentPeriod:     "invalid",
			},
			wantErr: true,
			errMsg:  "unsupported payment period",
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

func TestFixedRentalProcessor_CollectUsage(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewFixedRentalProcessor(store)

	usageData, err := processor.CollectUsage("test-contract-1", time.Now(), time.Now(), "", "")
	require.Error(t, err)
	require.Nil(t, usageData)
	require.Contains(t, err.Error(), "does not support manual billing")
}

func TestFixedRentalProcessor_CalculatePayment(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewFixedRentalProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel: contracts.FixedRental,
		},
	}

	fixedRentalUsage := &contracts.FixedRentalUsage{
		PeriodsInvoiced: 2,
		PeriodStart:     time.Now(),
		PeriodEnd:       time.Now().Add(2 * 24 * time.Hour),
		Amount:          "200.0",
		LastInvoiceAt:   time.Now().Add(-2 * 24 * time.Hour),
	}

	usageData := &contracts.UsageData{
		ContractDID:  "test-contract-1",
		PaymentModel: contracts.FixedRental,
		Data:         fixedRentalUsage,
	}

	items, err := processor.CalculatePayment(usageData, contract)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "200.0", items[0].Amount)
	require.Equal(t, 1, items[0].Usages)
}

func TestFixedRentalProcessor_SupportsBilling(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewFixedRentalProcessor(store)

	require.False(t, processor.SupportsManualBilling())
	require.True(t, processor.SupportsAutomaticBilling())
}

func TestFixedRentalProcessor_CheckAndGenerateInvoice(t *testing.T) {
	store := setupTestUsageStore(t)
	processor := NewFixedRentalProcessor(store)

	contract := &contracts.Contract{
		ContractDID: "test-contract-1",
		PaymentDetails: contracts.PaymentDetails{
			PaymentModel:       contracts.FixedRental,
			FixedRentalAmount:  "100.0",
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

	// Test: period elapsed
	lastInvoiceAt = time.Now().Add(-25 * time.Hour) // More than 1 day
	usageData, err = processor.CheckAndGenerateInvoice(contract, lastInvoiceAt, now)
	require.NoError(t, err)
	require.NotNil(t, usageData)
	require.Equal(t, contracts.FixedRental, usageData.PaymentModel)

	fixedRentalUsage, ok := usageData.Data.(*contracts.FixedRentalUsage)
	require.True(t, ok)
	require.NotEmpty(t, fixedRentalUsage.Amount)
	require.Greater(t, fixedRentalUsage.PeriodsInvoiced, 0)
}
