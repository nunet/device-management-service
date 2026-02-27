// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"fmt"
	"strconv"
)

// OrchestrationFeeCalculator calculates orchestration fees for payment items
type OrchestrationFeeCalculator struct{}

// NewOrchestrationFeeCalculator creates a new fee calculator
func NewOrchestrationFeeCalculator() *OrchestrationFeeCalculator {
	return &OrchestrationFeeCalculator{}
}

// CalculateFee calculates the orchestration fee for a batch of payment items
// Returns the fee amount as a string, or empty string if no fee should be charged
func (c *OrchestrationFeeCalculator) CalculateFee(
	paymentItems []*PaymentItem,
	config *OrchestrationFeeConfig,
) (string, error) {
	if config == nil {
		return "", nil // No orchestration fee configured
	}

	var fixedFee float64
	var percentage float64

	// Parse fixed amount
	if config.FixedAmount != "" && config.FixedAmount != "0" {
		var err error
		fixedFee, err = strconv.ParseFloat(config.FixedAmount, 64)
		if err != nil {
			return "", fmt.Errorf("invalid fixed_amount: %w", err)
		}
	}

	// Parse percentage
	if config.Percentage != "" && config.Percentage != "0" {
		var err error
		percentage, err = strconv.ParseFloat(config.Percentage, 64)
		if err != nil {
			return "", fmt.Errorf("invalid percentage: %w", err)
		}
	}

	// If both are zero, no fee
	if fixedFee == 0 && percentage == 0 {
		return "", nil
	}

	// Calculate total amount across all payment items
	var totalAmount float64
	for _, item := range paymentItems {
		itemAmount, err := strconv.ParseFloat(item.Amount, 64)
		if err != nil {
			return "", fmt.Errorf("invalid payment item amount: %w", err)
		}
		totalAmount += itemAmount
	}

	// Calculate: Fixed Fee + (Total Amount × Percentage / 100)
	percentageFee := totalAmount * percentage / 100.0
	totalFee := fixedFee + percentageFee

	// Format and return (8 decimal places, same as formatAmount helper)
	return fmt.Sprintf("%.8f", totalFee), nil
}

// GenerateOrchestrationFeeItem creates a PaymentItem for the orchestration fee
func (c *OrchestrationFeeCalculator) GenerateOrchestrationFeeItem(
	paymentItems []*PaymentItem,
	_ *Contract, // Contract parameter kept for API consistency but not used
	baseUniqueID string,
	feeAmount string,
) (*PaymentItem, error) {
	if feeAmount == "" {
		return nil, nil // No fee to generate
	}

	// Collect original unique IDs for metadata
	originalUniqueIDs := make([]string, len(paymentItems))
	originalAmounts := make([]string, len(paymentItems))
	for i, item := range paymentItems {
		originalUniqueIDs[i] = item.UniqueID
		originalAmounts[i] = item.Amount
	}

	// Create orchestration fee item
	feeItem := &PaymentItem{
		UniqueID:           fmt.Sprintf("%s-orchestration-fee", baseUniqueID),
		DeploymentID:       "", // Empty - applies to entire batch
		Amount:             feeAmount,
		Usages:             len(paymentItems), // Number of payment items in batch
		IsOrchestrationFee: true,
		Metadata: map[string]interface{}{
			"original_unique_ids": originalUniqueIDs,
			"original_amounts":    originalAmounts,
			"fee_type":            "orchestration",
			"payment_item_count":  len(paymentItems),
		},
	}

	return feeItem, nil
}
