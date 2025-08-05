// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"context"
	"time"
)

type (
	PaymentMode string
	PricingType string
	PaymentType string
)

const (
	// Payment modes
	FiatPayment       PaymentMode = "fiat"
	BlockchainPayment PaymentMode = "blockchain"
)

const (
	// Payment types
	EscrowPaymentType PaymentType = "escrow"
	DirectPaymentType PaymentType = "direct"
)

const (
	// Pricing types
	FixedJobPricingType PricingType = "fixed"
	PeriodicPricingType PricingType = "periodic"
)

// PaymentGateway defines the operations for managing payments and settlements
type PaymentGateway interface {
	Deposit(ctx context.Context, contractID string, payment Payment) error
	ProcessPayment(paymentType PaymentType, amount float64) error
}

// Payment represents a payment transaction
type Payment struct {
	Requestor   string
	Provider    string
	Currency    string
	Timestamp   time.Time
	PaymentType PaymentType // PaymentType (like escrow vs. direct)
	PaymentMode PaymentMode
	PricingMeta PricingMetadata
}

// PeriodicPricing represents the details for periodic pricing
type PeriodicPricing struct {
	Period      DurationDetails
	UsageLimits UsageLimits
}

// UsageLimits represents the usage limits or quotas for periodic pricing
type UsageLimits struct {
	MaxCPUHours         int
	MaxMemoryUsage      int
	MaxStorageUsage     int
	MaxNetworkBandwidth int
}

type PricingMetadata struct {
	Price       float32
	PlatformFee float32
	Type        PricingType
	Periodic    *PeriodicPricing
}
