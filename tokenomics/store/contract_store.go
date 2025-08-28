// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

const (
	contractsCollection     = "contracts"
	contractsKeysCollection = "contracts_keys"
)

type ContractKey struct {
	ContractDID string `json:"contract_did"`
	Key         []byte `json:"key"`
}

type Store struct {
	db *clover.DB
}

// New contract store
func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return &Store{
		db: db,
	}, nil
}

// Insert inserts or updates a contract in CloverDB (Upsert behavior)
func (s *Store) Upsert(contract *contracts.Contract) error {
	if contract == nil {
		return errors.New("contract is nil")
	}

	bts, err := json.Marshal(contract)
	if err != nil {
		return fmt.Errorf("failed to marshal contract: %w", err)
	}

	q := query.NewQuery(contractsCollection).Where(query.Field("contract_did").Eq(contract.ContractDID))
	existingDoc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to check existing contract: %w", err)
	}

	if existingDoc != nil {
		// Update the existing document
		update := document.NewDocument()
		update.Set("contract_did", contract.ContractDID)
		update.Set("updated_at", time.Now().Unix())
		update.Set("contract_data", bts)

		return s.db.Update(q, update.AsMap())
	}

	// Insert a new document
	doc := document.NewDocumentOf(contract)
	doc.Set("contract_did", contract.ContractDID)
	doc.Set("created_at", time.Now().Unix())
	doc.Set("contract_data", bts)

	return s.db.Insert(contractsCollection, doc)
}

// GetContract retrieves a contract by ContractDID
func (s *Store) GetContract(contractDID string) (*contracts.Contract, error) {
	if contractDID == "" {
		return nil, errors.New("contractDID is empty")
	}

	q := query.NewQuery(contractsCollection).Where(query.Field("contract_did").Eq(contractDID))
	doc, err := s.db.FindFirst(q)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("failed to find contract by ID: %w", err)
	}

	var contract contracts.Contract
	data := doc.Get("contract_data")
	contractData, ok := data.([]byte)
	if !ok {
		return nil, errors.New("no contract data available")
	}
	err = json.Unmarshal(contractData, &contract)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal contract: %w", err)
	}

	return &contract, nil
}

// InsertContractKey inserts key
func (s *Store) InsertContractKey(c ContractKey) error {
	bts, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal contract key: %w", err)
	}

	doc := document.NewDocumentOf(c)
	doc.Set("contract_did", c.ContractDID)
	doc.Set("created_at", time.Now().Unix())
	doc.Set("key_data", bts)

	return s.db.Insert(contractsKeysCollection, doc)
}

// GetContractKey retrieves a contract by ContractDID
func (s *Store) GetContractKey(contractDID string) (*ContractKey, error) {
	q := query.NewQuery(contractsKeysCollection).Where(query.Field("contract_did").Eq(contractDID))
	doc, err := s.db.FindFirst(q)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("failed to find contract by did: %w", err)
	}

	var key ContractKey
	data := doc.Get("key_data")
	err = json.Unmarshal(data.([]byte), &key)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal contract: %w", err)
	}

	return &key, nil
}

// GetAllContracts retrieves all contracts from the database
func (s *Store) GetAllContracts() ([]*contracts.Contract, error) {
	q := query.NewQuery(contractsCollection)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all contracts: %w", err)
	}

	allContracts := make([]*contracts.Contract, 0)
	for _, doc := range docs {
		var contract contracts.Contract
		data := doc.Get("contract_data")
		err = json.Unmarshal(data.([]byte), &contract)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal single contract: %w", err)
		}

		allContracts = append(allContracts, &contract)
	}

	return allContracts, nil
}
