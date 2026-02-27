// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package paymentquote

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
)

const (
	paymentQuotesCollection = "payment_quotes"
	defaultQuoteTTL         = 2 * time.Minute // Quotes expire after 2 minutes
)

type PaymentQuote struct {
	QuoteID         string    `json:"quote_id"`          // Unique quote identifier
	UniqueID        string    `json:"unique_id"`         // Links to transaction
	OriginalAmount  string    `json:"original_amount"`   // Amount in pricing currency (USDT)
	ConvertedAmount string    `json:"converted_amount"`  // Amount in payment currency (NTX)
	PricingCurrency string    `json:"pricing_currency"`  // Original currency (e.g., "USDT")
	PaymentCurrency string    `json:"payment_currency"`  // Payment currency (e.g., "NTX")
	ExchangeRate    string    `json:"exchange_rate"`     // Rate used for conversion
	CreatedAt       time.Time `json:"created_at"`        // Quote creation timestamp
	ExpiresAt       time.Time `json:"expires_at"`        // Quote expiration timestamp
	Used            bool      `json:"used"`              // Whether quote has been used
	UsedAt          time.Time `json:"used_at,omitempty"` // When quote was used (if used)
}

type Store struct {
	db *clover.DB
}

func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	// Ensure collection exists
	hasCollection, err := db.HasCollection(paymentQuotesCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	if !hasCollection {
		if err := db.CreateCollection(paymentQuotesCollection); err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return &Store{db: db}, nil
}

// CreateQuote creates a new payment quote
func (s *Store) CreateQuote(quote PaymentQuote) error {
	if quote.QuoteID == "" {
		return errors.New("quote_id is required")
	}
	if quote.UniqueID == "" {
		return errors.New("unique_id is required")
	}

	bts, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("failed to marshal quote: %w", err)
	}

	doc := document.NewDocumentOf(quote)
	doc.Set("quote_id", quote.QuoteID)
	doc.Set("unique_id", quote.UniqueID)
	doc.Set("created_at", time.Now().UnixNano())
	doc.Set("quote_data", bts)

	return s.db.Insert(paymentQuotesCollection, doc)
}

// GetQuote retrieves a quote by quote_id (does not check expiration - use ValidateQuote for that)
func (s *Store) GetQuote(quoteID string) (*PaymentQuote, error) {
	q := query.NewQuery(paymentQuotesCollection).Where(query.Field("quote_id").Eq(quoteID))
	doc, err := s.db.FindFirst(q)
	if err != nil {
		return nil, fmt.Errorf("failed to find quote: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("quote not found: %s", quoteID)
	}

	data := doc.Get("quote_data")
	var quote PaymentQuote
	if err := json.Unmarshal(data.([]byte), &quote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote: %w", err)
	}

	return &quote, nil
}

// ValidateQuote checks if a quote is valid (not used, not expired)
func (s *Store) ValidateQuote(quoteID string) (*PaymentQuote, error) {
	quote, err := s.GetQuote(quoteID)
	if err != nil {
		return nil, err
	}

	// Check if quote is used
	if quote.Used {
		return nil, fmt.Errorf("quote already used: %s", quoteID)
	}

	// Check if quote is expired
	if time.Now().After(quote.ExpiresAt) {
		return nil, fmt.Errorf("quote expired: %s", quoteID)
	}

	return quote, nil
}

// MarkQuoteAsUsed marks a quote as used
func (s *Store) MarkQuoteAsUsed(quoteID string) error {
	q := query.NewQuery(paymentQuotesCollection).Where(query.Field("quote_id").Eq(quoteID))
	doc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to find quote: %w", err)
	}
	if doc == nil {
		return fmt.Errorf("quote not found: %s", quoteID)
	}

	data := doc.Get("quote_data")
	var quote PaymentQuote
	if err := json.Unmarshal(data.([]byte), &quote); err != nil {
		return fmt.Errorf("failed to unmarshal quote: %w", err)
	}

	if quote.Used {
		return fmt.Errorf("quote already used: %s", quoteID)
	}

	quote.Used = true
	quote.UsedAt = time.Now()

	bts, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("failed to marshal quote: %w", err)
	}

	update := map[string]interface{}{
		"quote_data": bts,
	}

	return s.db.Update(q, update)
}

// GetQuoteByUniqueID retrieves the most recent unused quote for a transaction
func (s *Store) GetQuoteByUniqueID(uniqueID string) (*PaymentQuote, error) {
	q := query.NewQuery(paymentQuotesCollection).
		Where(query.Field("unique_id").Eq(uniqueID)).
		Sort(query.SortOption{Field: "created_at", Direction: -1})

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to find quotes: %w", err)
	}

	// Find first unused quote (expiration check done in ValidateQuote)
	for _, doc := range docs {
		data := doc.Get("quote_data")
		var quote PaymentQuote
		if err := json.Unmarshal(data.([]byte), &quote); err != nil {
			continue
		}

		if !quote.Used {
			return &quote, nil
		}
	}

	return nil, fmt.Errorf("no unused quote found for unique_id: %s", uniqueID)
}

// HasActiveQuote checks if there's an active (unused and not expired) quote for a transaction
func (s *Store) HasActiveQuote(uniqueID string) (*PaymentQuote, error) {
	q := query.NewQuery(paymentQuotesCollection).
		Where(query.Field("unique_id").Eq(uniqueID)).
		Sort(query.SortOption{Field: "created_at", Direction: -1})

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to find quotes: %w", err)
	}

	// Find first active quote (not used and not expired)
	now := time.Now()
	for _, doc := range docs {
		data := doc.Get("quote_data")
		var quote PaymentQuote
		if err := json.Unmarshal(data.([]byte), &quote); err != nil {
			continue
		}

		// Check if quote is unused and not expired
		if !quote.Used && now.Before(quote.ExpiresAt) {
			return &quote, nil
		}
	}

	return nil, nil // No active quote found (not an error)
}

// InvalidateQuote explicitly invalidates a quote (e.g., when user cancels payment)
// This marks the quote as used without actually using it for payment
func (s *Store) InvalidateQuote(quoteID string) error {
	// Same logic as MarkQuoteAsUsed, but we can add a flag if needed
	// For now, marking as used effectively invalidates it
	return s.MarkQuoteAsUsed(quoteID)
}

// CleanupExpiredQuotes removes expired quotes older than the retention period
// This should be called periodically (e.g., daily) to clean up old data
func (s *Store) CleanupExpiredQuotes(retentionPeriod time.Duration) error {
	cutoffTime := time.Now().Add(-retentionPeriod)

	q := query.NewQuery(paymentQuotesCollection).
		Where(query.Field("created_at").Lt(cutoffTime.UnixNano()))

	docs, err := s.db.FindAll(q)
	if err != nil {
		return fmt.Errorf("failed to find expired quotes: %w", err)
	}

	for _, doc := range docs {
		if err := s.db.DeleteById(paymentQuotesCollection, doc.ObjectId()); err != nil {
			// Log warning but continue
			continue
		}
	}

	return nil
}
