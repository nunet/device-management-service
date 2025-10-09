// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package store

import (
	"testing"

	"github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	if err != nil {
		t.Fatalf("failed to open CloverDB: %v", err)
	}

	err = db.CreateCollection(contractsCollection)
	if err != nil {
		t.Fatalf("failed to create collection %s: %v", contractsCollection, err)
	}

	err = db.CreateCollection(contractsKeysCollection)
	if err != nil {
		t.Fatalf("failed to create collection %s: %v", contractsKeysCollection, err)
	}

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store
}

func TestInsertAndGetContract(t *testing.T) {
	store := setupTestDB(t)

	contract := &contracts.Contract{
		ContractDID: "contract123",
	}

	err := store.Upsert(contract)
	assert.NoError(t, err, "Insert should succeed")

	result, err := store.GetContract("contract123")
	assert.NoError(t, err, "GetContract should succeed")
	assert.NotNil(t, result, "Returned contract should not be nil")
	assert.Equal(t, "contract123", result.ContractDID, "ContractID should match")
}

func TestInsertContractKeyCapAndGet(t *testing.T) {
	store := setupTestDB(t)

	capabilities := ContractKey{
		ContractDID: "did:example:456",
		Key:         []byte("key456"),
	}

	err := store.InsertContractKey(capabilities)
	assert.NoError(t, err, "InsertContractKeyCap should succeed")

	result, err := store.GetContractKey("did:example:456")
	assert.NoError(t, err, "GetContractKeyCap should succeed")
	assert.NotNil(t, result, "Returned contract key cap should not be nil")

	assert.Equal(t, capabilities.ContractDID, result.ContractDID)
	assert.Equal(t, capabilities.Key, result.Key)
}

func TestGetContractNotFound(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetContract("nonexistent")
	assert.Error(t, err, "GetContract should return error for non-existing contract")
}

func TestGetContractKeyCapNotFound(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetContractKey("did:nonexistent")
	assert.Error(t, err, "GetContractKeyCap should return error for non-existing contract")
}

func TestInsertNilContract(t *testing.T) {
	store := setupTestDB(t)

	err := store.Upsert(nil)
	assert.EqualError(t, err, "contract is nil")
}

func TestNewStoreNilDB(t *testing.T) {
	s, err := New(nil)
	assert.Nil(t, s)
	assert.EqualError(t, err, "db is nil")
}

func TestGetAllContracts(t *testing.T) {
	store := setupTestDB(t)

	contracts := []*contracts.Contract{
		{ContractDID: "did:contract:1"},
		{ContractDID: "did:contract:2"},
		{ContractDID: "did:contract:3"},
	}

	for _, c := range contracts {
		err := store.Upsert(c)
		assert.NoError(t, err, "Insert should succeed for %s", c.ContractDID)
	}

	allContracts, err := store.GetAllContracts()
	assert.NoError(t, err, "GetAllContracts should succeed")
	assert.Len(t, allContracts, len(contracts), "Number of contracts returned should match inserted")

	didSet := make(map[string]bool)
	for _, c := range allContracts {
		didSet[c.ContractDID] = true
	}

	for _, expected := range contracts {
		assert.True(t, didSet[expected.ContractDID], "Expected contract %s not found", expected.ContractDID)
	}
}
