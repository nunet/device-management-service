// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package transaction

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
	transactionsCollection = "service_provider_transactions"
)

// Transaction struct
type Transaction struct {
	UniqueID            string `json:"unique_id"`
	PaymentValidatorDID string `json:"payment_validator_did"`
	ContractDID         string `json:"contract_did"`
	ToAddress           string `json:"to_address"`
	Amount              string `json:"amount"`
	Status              string `json:"status"`
	TxHash              string `json:"tx_hash"`
}

type Store struct {
	db *clover.DB
}

// New creates a new transaction store
func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &Store{db: db}, nil
}

// Upsert inserts a transaction if it doesn't already exist
func (s *Store) Upsert(t Transaction) error {
	if t.Status == "" {
		t.Status = "unpaid"
	}

	bts, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	q := query.NewQuery(transactionsCollection).Where(query.Field("unique_id").Eq(t.UniqueID))
	existingDoc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	if existingDoc != nil {
		return nil
	}

	doc := document.NewDocumentOf(t)
	doc.Set("unique_id", t.UniqueID)
	doc.Set("contract_did", t.ContractDID)
	doc.Set("created_at", time.Now().UnixNano())
	doc.Set("transaction_data", bts)

	return s.db.Insert(transactionsCollection, doc)
}

// AllTransactions retrieves all transactions from the database
func (s *Store) AllTransactions() ([]*Transaction, error) {
	q := query.NewQuery(transactionsCollection)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all transactions: %w", err)
	}

	allTransactions := make([]*Transaction, 0)
	for _, doc := range docs {
		var t Transaction
		data := doc.Get("transaction_data")
		err = json.Unmarshal(data.([]byte), &t)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal single transaction: %w", err)
		}
		allTransactions = append(allTransactions, &t)
	}

	return allTransactions, nil
}

// MarkAsPaid updates a transaction's status to "paid" given its unique ID
// it returns also the payment provider did
func (s *Store) MarkAsPaid(uniqueID string, txHash string) (string, error) {
	q := query.NewQuery(transactionsCollection).Where(query.Field("unique_id").Eq(uniqueID))
	doc, err := s.db.FindFirst(q)
	if err != nil {
		return "", fmt.Errorf("failed to find transaction: %w", err)
	}
	if doc == nil {
		return "", fmt.Errorf("transaction not found with unique_id: %s", uniqueID)
	}

	data := doc.Get("transaction_data")
	var t Transaction
	if err := json.Unmarshal(data.([]byte), &t); err != nil {
		return "", fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	t.Status = "paid"
	t.TxHash = txHash

	bts, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated transaction: %w", err)
	}

	update := map[string]interface{}{
		"transaction_data": bts,
	}

	return t.PaymentValidatorDID, s.db.Update(q, update)
}
