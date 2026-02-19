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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/utils/convert"
)

const (
	USDT = "USDT"
	NTX  = "NTX"
)

var allowedConversions = map[string]string{
	USDT: NTX, // NTX to USDT
}

// PriceOracle defines the interface for fetching cryptocurrency prices
type PriceOracle interface {
	// GetPrice fetches the current exchange rate for a trading pair
	// Returns price as a float64 (e.g., 0.05 means 1 NTX = 0.05 USDT)
	GetPrice(ctx context.Context, fromCurrency, toCurrency string) (float64, error)

	// ConvertAmount converts an amount from one currency to another
	ConvertAmount(ctx context.Context, amount string, fromCurrency, toCurrency string) (string, error)
}

// CoinMarketCapOracle implements PriceOracle using CoinMarketCap API
type CoinMarketCapOracle struct {
	apiKey       string
	baseURL      string
	endpointPath string // API endpoint path (e.g., "/tools/price-conversion")
	httpClient   *http.Client
	cache        *PriceCache
}

// PriceCache stores cached exchange rates
type PriceCache struct {
	rates map[string]CachedRate
	ttl   time.Duration
	mu    sync.RWMutex
}

type CachedRate struct {
	Rate      float64
	Timestamp time.Time
}

// NewCoinMarketCapOracle creates a new CoinMarketCap oracle instance
// baseURL is the API endpoint base URL (e.g., "https://pro-api.coinmarketcap.com/v1")
// If baseURL is empty, defaults to "https://pro-api.coinmarketcap.com/v1"
// endpointPath is the API endpoint path (e.g., "/tools/price-conversion")
// If endpointPath is empty, defaults to "/tools/price-conversion"
func NewCoinMarketCapOracle(apiKey string, baseURL string, endpointPath string, cacheTTL time.Duration) *CoinMarketCapOracle {
	// Use default base URL if not specified
	if baseURL == "" {
		baseURL = "https://pro-api.coinmarketcap.com/v2"
	}

	// Use default endpoint path if not specified
	if endpointPath == "" {
		endpointPath = "/tools/price-conversion"
	}

	return &CoinMarketCapOracle{
		apiKey:       apiKey,
		baseURL:      baseURL,
		endpointPath: endpointPath,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache: &PriceCache{
			rates: make(map[string]CachedRate),
			ttl:   cacheTTL,
		},
	}
}

// GetPrice implements PriceOracle.GetPrice
func (o *CoinMarketCapOracle) GetPrice(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	// Normalize currency symbols
	fromCurrency = normalizeCurrency(fromCurrency)
	toCurrency = normalizeCurrency(toCurrency)

	// Check cache first
	cacheKey := fmt.Sprintf("%s/%s", fromCurrency, toCurrency)
	if rate, found := o.cache.Get(cacheKey); found {
		return rate, nil
	}

	// Fetch from API
	rate, err := o.fetchPriceFromAPI(ctx, fromCurrency, toCurrency)
	if err != nil {
		// Try to get stale cache as fallback
		if staleRate, found := o.cache.GetStale(cacheKey); found {
			// Using stale cache as fallback - API fetch failed
			return staleRate, nil
		}
		return 0, fmt.Errorf("failed to fetch price: %w", err)
	}

	// Update cache
	o.cache.Set(cacheKey, rate)

	return rate, nil
}

// ConvertAmount implements PriceOracle.ConvertAmount
func (o *CoinMarketCapOracle) ConvertAmount(
	ctx context.Context,
	amount string,
	fromCurrency, toCurrency string,
) (string, error) {
	if v, ok := allowedConversions[fromCurrency]; !ok || v != toCurrency {
		return "", fmt.Errorf("unsupported conversion: %s to %s", fromCurrency, toCurrency)
	}

	// Parse amount
	amountFloat, err := convert.StringToFloat64(amount, "amount")
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	// Get exchange rate
	rate, err := o.GetPrice(ctx, fromCurrency, toCurrency)
	if err != nil {
		return "", err
	}

	convertedAmount := amountFloat * rate

	return formatAmount(convertedAmount), nil
}

// fetchPriceFromAPI fetches price from CoinMarketCap API
// Uses the configurable baseURL and endpointPath from the oracle instance
func (o *CoinMarketCapOracle) fetchPriceFromAPI(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	// CoinMarketCap API endpoint for price conversion
	url := fmt.Sprintf("%s%s?amount=1&symbol=%s&convert=%s",
		o.baseURL, o.endpointPath, fromCurrency, toCurrency)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("X-CMC_PRO_API_KEY", o.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse v2 API response - data is an array in v2
	var apiResponse struct {
		Data []struct {
			Amount float64 `json:"amount"`
			Symbol string  `json:"symbol"`
			Quote  map[string]struct {
				Price       float64 `json:"price"`
				LastUpdated string  `json:"last_updated,omitempty"`
			} `json:"quote"`
		} `json:"data"`
		Status struct {
			ErrorCode    int    `json:"error_code"`
			ErrorMessage string `json:"error_message,omitempty"`
		} `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return 0, fmt.Errorf("failed to decode API response: %w", err)
	}

	// Check for API errors
	if apiResponse.Status.ErrorCode != 0 {
		return 0, fmt.Errorf("API error: %s (code: %d)", apiResponse.Status.ErrorMessage, apiResponse.Status.ErrorCode)
	}

	// In v2, data is an array, so we take the first element
	if len(apiResponse.Data) == 0 {
		return 0, fmt.Errorf("no data in API response")
	}

	quote, ok := apiResponse.Data[0].Quote[toCurrency]
	if !ok {
		return 0, fmt.Errorf("currency %s not found in response", toCurrency)
	}

	return quote.Price, nil
}

// normalizeCurrency normalizes currency symbols for API calls
func normalizeCurrency(currency string) string {
	// Map common variations to standard symbols
	currencyMap := map[string]string{
		"USDT": "USDT",
		"NTX":  "NTX",
	}

	if normalized, ok := currencyMap[strings.ToUpper(currency)]; ok {
		return normalized
	}
	return strings.ToUpper(currency)
}

// formatAmount formats a float64 amount as a string with 8 decimal places
func formatAmount(amount float64) string {
	return fmt.Sprintf("%.8f", amount)
}

// PriceCache methods
func (c *PriceCache) Get(key string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.rates[key]
	if !ok {
		return 0, false
	}

	if time.Since(cached.Timestamp) > c.ttl {
		return 0, false // Cache expired
	}

	return cached.Rate, true
}

func (c *PriceCache) GetStale(key string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.rates[key]
	if !ok {
		return 0, false
	}

	return cached.Rate, true // Return even if stale
}

func (c *PriceCache) Set(key string, rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rates[key] = CachedRate{
		Rate:      rate,
		Timestamp: time.Now(),
	}
}
