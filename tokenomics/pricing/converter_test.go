// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package pricing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	cUSDT = "USDT"
	cNTX  = "NTX"
)

// mockOracle is a simple mock implementation of PriceOracle for testing
type mockOracle struct {
	price    float64
	priceErr error
}

func (m *mockOracle) GetPrice(_ context.Context, fromCurrency, toCurrency string) (float64, error) {
	_ = fromCurrency
	_ = toCurrency
	return m.price, m.priceErr
}

func (m *mockOracle) ConvertAmount(_ context.Context, amount string, fromCurrency, toCurrency string) (string, error) {
	_ = fromCurrency
	_ = toCurrency
	if m.priceErr != nil {
		return "", m.priceErr
	}
	// Simplified conversion for testing
	if fromCurrency == cUSDT && toCurrency == cNTX {
		return "2000.00", nil // Assuming rate of 0.05
	}
	_ = amount
	return "", nil
}

func TestPriceConverter_ConvertPaymentItem_NoConversion(t *testing.T) {
	mockOracle := &mockOracle{price: 0.05}
	converter := NewPriceConverter(mockOracle)

	contract := &contracts.Contract{
		PaymentDetails: contracts.PaymentDetails{
			PricingCurrency: "", // Empty - no conversion
			Addresses: []types.PaymentAddressInfo{
				{Currency: cNTX},
			},
		},
	}

	item := &contracts.PaymentItem{
		Amount: "100.00",
	}

	originalAmount := item.Amount
	err := converter.ConvertPaymentItem(context.Background(), item, contract)
	require.NoError(t, err)
	require.Equal(t, originalAmount, item.Amount, "amount should not change when pricing_currency is empty")
	require.Empty(t, item.OriginalAmount, "should not set original amount when no conversion")
}

func TestPriceConverter_ConvertPaymentItem_WithConversion(t *testing.T) {
	mockOracle := &mockOracle{price: 0.05}
	converter := NewPriceConverter(mockOracle)

	contract := &contracts.Contract{
		PaymentDetails: contracts.PaymentDetails{
			PricingCurrency: cUSDT, // Set to USDT - conversion should occur
			Addresses: []types.PaymentAddressInfo{
				{Currency: cNTX},
			},
		},
	}

	item := &contracts.PaymentItem{
		Amount: "100.00", // 100 USDT
	}

	err := converter.ConvertPaymentItem(context.Background(), item, contract)
	require.NoError(t, err)
	require.NotEqual(t, "100.00", item.Amount, "amount should be converted")
	require.Equal(t, "100.00", item.OriginalAmount, "original amount should be preserved")
	require.Equal(t, cUSDT, item.PricingCurrency, "pricing currency should be set")
	require.NotEmpty(t, item.ExchangeRate, "exchange rate should be set")
	require.False(t, item.ConversionTimestamp.IsZero(), "conversion timestamp should be set")
	require.Equal(t, "2000.00", item.Amount, "amount should be converted")
}

func TestPriceConverter_ConvertPaymentItem_SameCurrency(t *testing.T) {
	mockOracle := &mockOracle{price: 0.05}
	converter := NewPriceConverter(mockOracle)

	contract := &contracts.Contract{
		PaymentDetails: contracts.PaymentDetails{
			PricingCurrency: cNTX, // Same as payment currency
			Addresses: []types.PaymentAddressInfo{
				{Currency: cNTX},
			},
		},
	}

	item := &contracts.PaymentItem{
		Amount: "100.00",
	}

	originalAmount := item.Amount
	err := converter.ConvertPaymentItem(context.Background(), item, contract)
	require.NoError(t, err)
	require.Equal(t, originalAmount, item.Amount, "amount should not change when currencies match")
}

func TestPriceConverter_ConvertPaymentItem_APIError(t *testing.T) {
	mockOracle := &mockOracle{priceErr: context.DeadlineExceeded}
	converter := NewPriceConverter(mockOracle)

	contract := &contracts.Contract{
		PaymentDetails: contracts.PaymentDetails{
			PricingCurrency: cUSDT,
			Addresses: []types.PaymentAddressInfo{
				{Currency: cNTX},
			},
		},
	}

	item := &contracts.PaymentItem{
		Amount: "100.00",
	}

	err := converter.ConvertPaymentItem(context.Background(), item, contract)
	require.Error(t, err, "should return error when API fails")
}
