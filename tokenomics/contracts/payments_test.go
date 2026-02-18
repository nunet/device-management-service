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

func TestPaymentDetails_ValidateOrchestrationFee(t *testing.T) {
	t.Run("valid fixed amount only", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.50",
				Percentage:  "0",
			},
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})

	t.Run("valid percentage only", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "0",
				Percentage:  "2.5",
			},
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})

	t.Run("valid combined", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "0.50",
				Percentage:  "1.5",
			},
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})

	t.Run("both zero", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "0",
				Percentage:  "0",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "orchestration_fee must have at least fixed_amount or percentage")
	})

	t.Run("both empty", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "",
				Percentage:  "",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "orchestration_fee must have at least fixed_amount or percentage")
	})

	t.Run("invalid fixed amount format", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "invalid",
				Percentage:  "0",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid orchestration_fee.fixed_amount")
	})

	t.Run("invalid percentage format", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.00",
				Percentage:  "invalid",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid orchestration_fee.percentage")
	})

	t.Run("percentage out of range - negative", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.00",
				Percentage:  "-1",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "orchestration_fee.percentage must be between 0 and 100")
	})

	t.Run("percentage out of range - over 100", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.00",
				Percentage:  "101",
			},
		}
		err := pd.ValidateOrchestrationFee()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "orchestration_fee.percentage must be between 0 and 100")
	})

	t.Run("percentage at boundary - 0", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.00",
				Percentage:  "0",
			},
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})

	t.Run("percentage at boundary - 100", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: &OrchestrationFeeConfig{
				FixedAmount: "1.00",
				Percentage:  "100",
			},
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})

	t.Run("nil orchestration fee", func(t *testing.T) {
		pd := PaymentDetails{
			OrchestrationFee: nil,
		}
		err := pd.ValidateOrchestrationFee()
		assert.NoError(t, err)
	})
}
