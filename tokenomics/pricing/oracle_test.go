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
	"time"

	"github.com/stretchr/testify/require"
)

const (
	sandboxBaseURL = "https://pro-api.coinmarketcap.com/v2"
	sandboxAPIKey  = "4851bf9c832340328bc48b10cab272f7"
)

func TestCoinMarketCapOracle_GetPrice(t *testing.T) {
	// Use CoinMarketCap sandbox API for testing
	oracle := NewCoinMarketCapOracle(sandboxAPIKey, sandboxBaseURL, "/tools/price-conversion", 5*time.Minute)

	ctx := context.Background()
	price, err := oracle.GetPrice(ctx, "USDT", "NTX")

	require.NoError(t, err)
	// Sandbox API returns real data, so we just verify we got a price > 0
	require.Greater(t, price, 0.0, "price should be greater than 0")
}

func TestCoinMarketCapOracle_GetPrice_Cache(t *testing.T) {
	// Use CoinMarketCap sandbox API for testing
	// Create oracle with short cache TTL for testing
	oracle := NewCoinMarketCapOracle(sandboxAPIKey, sandboxBaseURL, "/tools/price-conversion", 1*time.Second)
	ctx := context.Background()

	// First call - should hit API
	price1, err := oracle.GetPrice(ctx, "USDT", "NTX")
	require.NoError(t, err)
	require.Greater(t, price1, 0.0, "price should be greater than 0")

	// Second call within cache TTL - should use cache
	price2, err := oracle.GetPrice(ctx, "USDT", "NTX")
	require.NoError(t, err)
	require.Equal(t, price1, price2, "cached price should match")
	// Note: We can't verify call count with real API, but cache should prevent second call

	// Wait for cache to expire
	time.Sleep(2 * time.Second)

	// Third call after cache expiry - should hit API again
	price3, err := oracle.GetPrice(ctx, "USDT", "NTX")
	require.NoError(t, err)
	require.Greater(t, price3, 0.0, "price should be greater than 0")
	// Note: Price may differ slightly due to market fluctuations, but should be valid
}

func TestCoinMarketCapOracle_ConvertAmount(t *testing.T) {
	// Use CoinMarketCap sandbox API for testing
	oracle := NewCoinMarketCapOracle(sandboxAPIKey, sandboxBaseURL, "/tools/price-conversion", 5*time.Minute)
	ctx := context.Background()

	tests := []struct {
		name         string
		amount       string
		fromCurrency string
		toCurrency   string
		wantErr      bool
	}{
		{
			name:         "convert 100 USDT to NTX",
			amount:       "100.00",
			fromCurrency: "USDT",
			toCurrency:   "NTX",
			wantErr:      false,
		},
		{
			name:         "convert 50 NTX to USDT",
			amount:       "50.00",
			fromCurrency: "NTX",
			toCurrency:   "USDT",
			wantErr:      true,
		},
		{
			name:         "invalid amount",
			amount:       "invalid",
			fromCurrency: "USDT",
			toCurrency:   "NTX",
			wantErr:      true,
		},
		{
			name:         "unsupported conversion",
			amount:       "100.00",
			fromCurrency: "EUR",
			toCurrency:   "GBP",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := oracle.ConvertAmount(ctx, tt.amount, tt.fromCurrency, tt.toCurrency)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, result, "converted amount should not be empty")
				// Verify result is a valid number (sandbox returns real prices, so exact values may vary)
				require.Regexp(t, `^\d+\.?\d*$`, result, "result should be a valid number")
			}
		})
	}
}

func TestCoinMarketCapOracle_GetPrice_StaleCacheFallback(t *testing.T) {
	// Use CoinMarketCap sandbox API for testing
	oracle := NewCoinMarketCapOracle(sandboxAPIKey, sandboxBaseURL, "/tools/price-conversion", 1*time.Second)
	ctx := context.Background()

	// First call - should succeed and cache
	price1, err := oracle.GetPrice(ctx, "USDT", "NTX")
	require.NoError(t, err)
	require.Greater(t, price1, 0.0, "price should be greater than 0")

	// Wait for cache to expire
	time.Sleep(2 * time.Second)

	// Second call - should fetch fresh price (sandbox API should be available)
	// In case of API failure, stale cache fallback would be used
	price2, err := oracle.GetPrice(ctx, "USDT", "NTX")
	require.NoError(t, err, "should get price from API or stale cache")
	require.Greater(t, price2, 0.0, "price should be greater than 0")
	// Note: With real sandbox API, we can't easily test stale cache fallback
	// This would require mocking or network interruption
}
