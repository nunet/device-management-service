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
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

// PriceConverter handles currency conversion for payment items
type PriceConverter struct {
	oracle PriceOracle
}

// NewPriceConverter creates a new price converter
func NewPriceConverter(oracle PriceOracle) *PriceConverter {
	return &PriceConverter{
		oracle: oracle,
	}
}

// GetOracle returns the underlying price oracle
func (c *PriceConverter) GetOracle() PriceOracle {
	return c.oracle
}

// ConvertPaymentItem converts a payment item's amount from pricing currency to payment currency
// This method is OPTIONAL - if pricing_currency is not set, it returns immediately with no conversion
func (c *PriceConverter) ConvertPaymentItem(
	ctx context.Context,
	item *contracts.PaymentItem,
	contract *contracts.Contract,
) error {
	// OPTIONAL FEATURE: If pricing_currency is not specified, skip conversion entirely
	// This ensures backward compatibility - existing contracts work unchanged
	pricingCurrency := contract.PaymentDetails.PricingCurrency
	if pricingCurrency == "" || pricingCurrency == "NTX" { //nolint:goconst
		// No conversion needed - fallback to previous behavior
		// Payment item amount is used as-is, exactly as before
		return nil
	}

	// Determine payment currency from address
	paymentCurrency := "NTX" // Default
	if len(contract.PaymentDetails.Addresses) > 0 {
		paymentCurrency = contract.PaymentDetails.Addresses[0].Currency
	}

	// If currencies match, no conversion needed
	if pricingCurrency == paymentCurrency {
		return nil
	}

	// Only convert if pricing currency is a stable asset and payment currency is NTX
	// At this point, we know pricingCurrency is set to a stable asset (not empty, not "NTX")
	if paymentCurrency == "NTX" {
		originalAmount := item.Amount

		// Convert amount
		convertedAmount, err := c.oracle.ConvertAmount(ctx, originalAmount, pricingCurrency, paymentCurrency)
		if err != nil {
			return fmt.Errorf("failed to convert amount: %w", err)
		}

		// Get exchange rate for metadata
		rate, err := c.oracle.GetPrice(ctx, pricingCurrency, paymentCurrency)
		if err != nil {
			return fmt.Errorf("failed to get exchange rate: %w", err)
		}

		// Update payment item
		item.OriginalAmount = originalAmount
		item.PricingCurrency = pricingCurrency
		item.Amount = convertedAmount
		item.ExchangeRate = formatAmount(rate)
		item.ConversionTimestamp = time.Now()

		// Add to metadata
		if item.Metadata == nil {
			item.Metadata = make(map[string]interface{})
		}
		item.Metadata["price_conversion"] = map[string]interface{}{
			"original_amount":  originalAmount,
			"pricing_currency": pricingCurrency,
			"converted_amount": convertedAmount,
			"payment_currency": paymentCurrency,
			"exchange_rate":    formatAmount(rate),
			"conversion_time":  time.Now().Format(time.RFC3339),
		}
	}

	return nil
}
