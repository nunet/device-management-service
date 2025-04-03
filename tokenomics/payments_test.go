// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build ignore

package tokenomics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PaymentProcessorTestSuite is the test suite for the PaymentProcessor.
type PaymentProcessorTestSuite struct {
	suite.Suite
	paymentProcessor *PaymentProcessor
}

// SetupTest sets up the test suite by initializing a new PaymentProcessor.
func (s *PaymentProcessorTestSuite) SetupTest() {
	s.paymentProcessor = &PaymentProcessor{}
}

// TestPaymentProcessorTestSuite runs the test suite for the PaymentProcessor.
func TestPaymentProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentProcessorTestSuite))
}

// Helper function to create a new Payment for testing
func (s *PaymentProcessorTestSuite) newPayment(paymentType string, paymentMode PaymentMode, price int, platformFee int) Payment {
	return Payment{
		Requestor:   "user1",
		Provider:    "provider1",
		Currency:    "USD",
		Timestamp:   time.Now(),
		PaymentType: paymentType,
		PaymentMode: paymentMode,
		PricingMeta: PricingMetadata{
			Price:       price,
			PlatformFee: platformFee,
			Type:        FixedJobPricingType,
		},
	}
}

// TestDeposit tests the Deposit method of the MockPaymentGateway.
// It checks various payment scenarios and validates if errors are handled correctly.
func (s *PaymentProcessorTestSuite) TestDeposit() {
	cases := map[string]struct {
		contractID int
		payment    Payment
		expErr     string
	}{
		"valid direct fiat payment": {
			contractID: 1,
			payment:    s.newPayment(DirectPaymentType, FiatPayment, 100, 10),
		},
		"valid escrow blockchain payment": {
			contractID: 2,
			payment: Payment{
				Requestor:   "user2",
				Provider:    "provider2",
				Currency:    "BTC",
				Timestamp:   time.Now(),
				PaymentType: EscrowPaymentType,
				PaymentMode: BlockchainPayment,
				PricingMeta: PricingMetadata{
					Price:       200,
					PlatformFee: 20,
					Type:        PeriodicPricingType,
					Periodic: &PeriodicPricing{
						Period: "monthly",
						UsageLimits: UsageLimits{
							MaxCPUHours:         100,
							MaxMemoryUsage:      1000,
							MaxStorageUsage:     500,
							MaxNetworkBandwidth: 1000,
						},
					},
				},
			},
		},
		"invalid payment mode": {
			contractID: 3,
			payment:    s.newPayment(DirectPaymentType, "invalid", 150, 15),
			expErr:     "unsupported payment type: direct",
		},
		"missing payment type": {
			contractID: 4,
			payment:    s.newPayment("", FiatPayment, 250, 25),
			expErr:     "unsupported payment type: ",
		},
	}

	ctx := context.Background()
	for name, tt := range cases {
		tt := tt
		s.Run(name, func() {
			err := s.paymentProcessor.Deposit(ctx, tt.contractID, tt.payment)
			if tt.expErr != "" {
				assert.EqualError(s.T(), err, tt.expErr)
			} else {
				assert.NoError(s.T(), err)
			}
		})
	}
}

// TestCalculatePaymentAmount tests the payment amount calculation
func (s *PaymentProcessorTestSuite) TestCalculatePaymentAmount() {
	cases := map[string]struct {
		pricing PricingMetadata
		expAmt  float64
		expErr  string
	}{
		"valid fixed pricing": {
			pricing: PricingMetadata{
				Price:       100,
				PlatformFee: 10,
				Type:        FixedJobPricingType,
				Fixed:       &FixedPricing{},
			},
			expAmt: 110,
		},
		"valid periodic pricing": {
			pricing: PricingMetadata{
				Price:       200,
				PlatformFee: 20,
				Type:        PeriodicPricingType,
				Periodic: &PeriodicPricing{
					Period: "monthly",
					UsageLimits: UsageLimits{
						MaxCPUHours:         100,
						MaxMemoryUsage:      1000,
						MaxStorageUsage:     500,
						MaxNetworkBandwidth: 1000,
					},
				},
			},
			expAmt: 220,
		},
		"invalid pricing type": {
			pricing: PricingMetadata{
				Price:       300,
				PlatformFee: 30,
				Type:        "invalid",
			},
			expErr: "unsupported pricing type: invalid",
		},
	}

	for name, tt := range cases {
		tt := tt
		s.Run(name, func() {
			amount, err := calculatePaymentAmount(tt.pricing)
			if tt.expErr != "" {
				assert.EqualError(s.T(), err, tt.expErr)
			} else {
				assert.NoError(s.T(), err)
				assert.Equal(s.T(), tt.expAmt, amount)
			}
		})
	}
}
