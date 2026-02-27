// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrationFeeCalculator_CalculateFee(t *testing.T) {
	calculator := NewOrchestrationFeeCalculator()

	t.Run("fixed fee only", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "1.50",
			Percentage:  "0",
		}
		items := []*PaymentItem{
			{Amount: "10.00"},
			{Amount: "20.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		assert.Equal(t, "1.50000000", fee)
	})

	t.Run("percentage only", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "0",
			Percentage:  "2.5",
		}
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		assert.Equal(t, "2.50000000", fee) // 100 * 2.5 / 100 = 2.5
	})

	t.Run("combined fee", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "0.50",
			Percentage:  "1.5",
		}
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		assert.Equal(t, "2.00000000", fee) // 0.50 + (100 * 1.5 / 100) = 2.0
	})

	t.Run("zero fees", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "0",
			Percentage:  "0",
		}
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		assert.Empty(t, fee)
	})

	t.Run("nil config", func(t *testing.T) {
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, nil)
		require.NoError(t, err)
		assert.Empty(t, fee)
	})

	t.Run("invalid fixed amount format", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "invalid",
			Percentage:  "0",
		}
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.Error(t, err)
		assert.Empty(t, fee)
		assert.Contains(t, err.Error(), "invalid fixed_amount")
	})

	t.Run("invalid percentage format", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "1.00",
			Percentage:  "invalid",
		}
		items := []*PaymentItem{
			{Amount: "100.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.Error(t, err)
		assert.Empty(t, fee)
		assert.Contains(t, err.Error(), "invalid percentage")
	})

	t.Run("empty payment items", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "1.00",
			Percentage:  "0",
		}
		items := []*PaymentItem{}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		assert.Equal(t, "1.00000000", fee) // Fixed fee still applies
	})

	t.Run("multiple payment items", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "1.00",
			Percentage:  "2.0",
		}
		items := []*PaymentItem{
			{Amount: "50.00"},
			{Amount: "30.00"},
			{Amount: "20.00"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.NoError(t, err)
		// 1.00 + (100 * 2.0 / 100) = 3.00
		assert.Equal(t, "3.00000000", fee)
	})

	t.Run("invalid payment item amount", func(t *testing.T) {
		config := &OrchestrationFeeConfig{
			FixedAmount: "1.00",
			Percentage:  "2.0",
		}
		items := []*PaymentItem{
			{Amount: "invalid"},
		}

		fee, err := calculator.CalculateFee(items, config)
		require.Error(t, err)
		assert.Empty(t, fee)
		assert.Contains(t, err.Error(), "invalid payment item amount")
	})
}

func TestOrchestrationFeeCalculator_GenerateOrchestrationFeeItem(t *testing.T) {
	calculator := NewOrchestrationFeeCalculator()

	t.Run("valid fee amount", func(t *testing.T) {
		items := []*PaymentItem{
			{UniqueID: "item1", Amount: "10.00"},
			{UniqueID: "item2", Amount: "20.00"},
		}
		contract := &Contract{
			ContractDID: "did:test:contract",
		}
		baseUniqueID := "base-123"
		feeAmount := "3.00"

		feeItem, err := calculator.GenerateOrchestrationFeeItem(items, contract, baseUniqueID, feeAmount)
		require.NoError(t, err)
		require.NotNil(t, feeItem)

		assert.Equal(t, "base-123-orchestration-fee", feeItem.UniqueID)
		assert.Empty(t, feeItem.DeploymentID)
		assert.Equal(t, "3.00", feeItem.Amount)
		assert.Equal(t, 2, feeItem.Usages)
		assert.True(t, feeItem.IsOrchestrationFee)

		// Check metadata
		assert.Equal(t, "orchestration", feeItem.Metadata["fee_type"])
		assert.Equal(t, 2, feeItem.Metadata["payment_item_count"])
		originalIDs, ok := feeItem.Metadata["original_unique_ids"].([]string)
		require.True(t, ok)
		assert.Equal(t, []string{"item1", "item2"}, originalIDs)
		originalAmounts, ok := feeItem.Metadata["original_amounts"].([]string)
		require.True(t, ok)
		assert.Equal(t, []string{"10.00", "20.00"}, originalAmounts)
	})

	t.Run("empty fee amount", func(t *testing.T) {
		items := []*PaymentItem{
			{UniqueID: "item1", Amount: "10.00"},
		}
		contract := &Contract{
			ContractDID: "did:test:contract",
		}
		baseUniqueID := "base-123"
		feeAmount := ""

		feeItem, err := calculator.GenerateOrchestrationFeeItem(items, contract, baseUniqueID, feeAmount)
		require.NoError(t, err)
		assert.Nil(t, feeItem)
	})

	t.Run("correct unique ID format", func(t *testing.T) {
		items := []*PaymentItem{
			{UniqueID: "item1", Amount: "10.00"},
		}
		contract := &Contract{
			ContractDID: "did:test:contract",
		}
		baseUniqueID := "test-base-id"
		feeAmount := "1.00"

		feeItem, err := calculator.GenerateOrchestrationFeeItem(items, contract, baseUniqueID, feeAmount)
		require.NoError(t, err)
		require.NotNil(t, feeItem)
		assert.Equal(t, "test-base-id-orchestration-fee", feeItem.UniqueID)
	})
}
