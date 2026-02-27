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
	"sync"
	"time"
)

// UsageData represents collected usage data
type UsageData struct {
	ContractDID  string
	PaymentModel PaymentModel
	Data         interface{} // Model-specific data (TimeUtilizationUsage, ResourceUtilizationUsage, etc.)
}

// PaymentItem represents a single payment to process
type PaymentItem struct {
	UniqueID     string
	DeploymentID string // Empty for non-deployment models
	Amount       string // Final amount in payment currency (NTX after conversion)
	Usages       int
	Metadata     map[string]interface{} // Model-specific metadata

	// Fields for price conversion tracking (optional)
	OriginalAmount      string    `json:"original_amount,omitempty"`      // Amount in pricing currency (USDT)
	PricingCurrency     string    `json:"pricing_currency,omitempty"`     // Currency of original amount
	ExchangeRate        string    `json:"exchange_rate,omitempty"`        // Rate used for conversion
	ConversionTimestamp time.Time `json:"conversion_timestamp,omitempty"` // When conversion occurred
	IsOrchestrationFee  bool      `json:"is_orchestration_fee,omitempty"` // Indicates if this is an orchestration fee transaction
}

// PaymentModelProcessor defines the clear, shared interface for all payment model processors.
// Each processor is a self-contained strategy with direct store access.
// All payment models must implement this interface.
type PaymentModelProcessor interface {
	// CollectUsage collects usage data for manual billing.
	// This method is called when a user manually triggers invoice generation.
	// Processors have full control over how they query and process events from the store.
	// Returns UsageData containing model-specific usage information.
	// providerDID is optional - if provided, filters events by provider for per-node billing.
	// headContractDID is optional - if provided, queries events by Head Contract DID instead of Tail Contract DID
	CollectUsage(
		contractDID string,
		lastProcessedAt time.Time,
		now time.Time,
		providerDID string, // Optional: if provided, filters events by provider
		headContractDID string, // Optional: if provided, queries by Head Contract DID
	) (*UsageData, error)

	// CalculatePayment calculates payment items from usage data.
	// This method converts UsageData into PaymentItems that can be processed.
	// Each PaymentItem represents a single payment transaction (one per deployment for deployment-based models).
	// Returns a slice of PaymentItems, each with calculated amount and metadata.
	CalculatePayment(
		usageData *UsageData,
		contract *Contract,
	) ([]*PaymentItem, error)

	// Validate validates payment model configuration.
	// This method checks that PaymentDetails contains all required fields for this payment model.
	// Returns an error if validation fails, nil if valid.
	Validate(paymentDetails PaymentDetails) error

	// SupportsManualBilling indicates whether this payment model supports manual invoice generation.
	// Models like FixedRental and Periodic return false (automatic billing only).
	// Models like PayPerAllocation, PayPerDeployment, etc. return true.
	SupportsManualBilling() bool

	// SupportsAutomaticBilling indicates whether this payment model supports automatic periodic billing.
	// Models like FixedRental and Periodic return true.
	// Other models return false.
	SupportsAutomaticBilling() bool

	// CheckAndGenerateInvoice checks if an invoice should be generated and collects usage data.
	// This method is called periodically by the contract actor for automatic billing models.
	// It checks if enough time has elapsed since the last invoice and collects usage if so.
	// Returns UsageData if invoice should be generated, nil if not yet time, error on failure.
	// Only called for models where SupportsAutomaticBilling() returns true.
	CheckAndGenerateInvoice(
		contract *Contract,
		lastInvoiceAt time.Time,
		now time.Time,
	) (*UsageData, error)
}

// PaymentModelRegistry manages payment model processors.
// All processors must be registered before use.
type PaymentModelRegistry struct {
	processors map[PaymentModel]PaymentModelProcessor
	mu         sync.RWMutex
}

var globalRegistry = &PaymentModelRegistry{
	processors: make(map[PaymentModel]PaymentModelProcessor),
}

// RegisterPaymentModelProcessor registers a payment model processor.
// Should be called during application initialization.
func RegisterPaymentModelProcessor(model PaymentModel, processor PaymentModelProcessor) {
	if processor == nil {
		panic(fmt.Sprintf("cannot register nil processor for payment model: %s", model))
	}
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.processors[model] = processor
}

// GetPaymentModelProcessor returns the processor for a payment model.
// Returns an error if no processor is registered for the given model.
func GetPaymentModelProcessor(model PaymentModel) (PaymentModelProcessor, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	processor, ok := globalRegistry.processors[model]
	if !ok {
		return nil, fmt.Errorf("no processor registered for payment model: %s", model)
	}
	return processor, nil
}

// MustGetPaymentModelProcessor returns the processor or panics (for initialization).
func MustGetPaymentModelProcessor(model PaymentModel) PaymentModelProcessor {
	processor, err := GetPaymentModelProcessor(model)
	if err != nil {
		panic(err)
	}
	return processor
}
