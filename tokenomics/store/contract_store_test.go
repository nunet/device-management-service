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
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/types"
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

func TestFindTailContract_SingleTailContract(t *testing.T) {
	store := setupTestDB(t)

	// Organization DID (from Head Contract)
	orgDID, err := did.FromString("did:org:123")
	require.NoError(t, err)

	// Compute Provider DID
	providerDID, err := did.FromString("did:provider:456")
	require.NoError(t, err)

	// Create Head Contract (should be skipped)
	headContract := &contracts.Contract{
		ContractDID: "did:head:contract",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  orgDID,
			Requestor: did.DID{URI: "did:orchestrator:789"},
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(headContract)
	require.NoError(t, err)

	// Create Tail Contract (should be found)
	tailContractObj := &contracts.Contract{
		ContractDID: "did:tail:contract",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(tailContractObj)
	require.NoError(t, err)

	// Create unrelated contract (should not be found)
	unrelatedContract := &contracts.Contract{
		ContractDID: "did:unrelated:contract",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  did.DID{URI: "did:other:provider"},
			Requestor: did.DID{URI: "did:other:requestor"},
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(unrelatedContract)
	require.NoError(t, err)

	// Find Tail Contract
	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract",
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	tailContractConfig, err := FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.NoError(t, err)
	require.NotNil(t, tailContractConfig, "Should find exactly one Tail Contract")
	assert.Equal(t, "did:tail:contract", tailContractConfig.DID)
}

func TestFindTailContract_MultipleTailContracts(t *testing.T) {
	store := setupTestDB(t)

	// Organization DID (from Head Contract)
	orgDID, err := did.FromString("did:org:123")
	require.NoError(t, err)

	// Compute Provider DID
	providerDID, err := did.FromString("did:provider:456")
	require.NoError(t, err)

	// Create multiple Tail Contracts
	tailContract1 := &contracts.Contract{
		ContractDID: "did:tail:contract:1",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(tailContract1)
	require.NoError(t, err)

	tailContract2 := &contracts.Contract{
		ContractDID: "did:tail:contract:2",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractActive,
	}
	err = store.Upsert(tailContract2)
	require.NoError(t, err)

	// Find Tail Contracts - should error when multiple are found
	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract",
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	_, err = FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple tail contracts found")
}

func TestFindTailContract_NoTailContractFound(t *testing.T) {
	store := setupTestDB(t)

	// Find Tail Contract when none exists
	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract",
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	tailContracts, err := FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.Error(t, err)
	assert.Nil(t, tailContracts, "Should return nil when no Tail Contracts found")
}

func TestFindTailContract_InvalidHeadContractConfig(t *testing.T) {
	store := setupTestDB(t)

	// Missing Provider field
	headContractConfig := types.ContractConfig{
		DID: "did:head:contract",
		// Provider is empty
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	_, err = FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "head contract config is missing provider DID")
}

func TestFindTailContract_TailContractNotActive(t *testing.T) {
	store := setupTestDB(t)

	// Organization DID
	orgDID, err := did.FromString("did:org:123")
	require.NoError(t, err)

	// Compute Provider DID
	providerDID, err := did.FromString("did:provider:456")
	require.NoError(t, err)

	// Create Tail Contract in DRAFT state (should be excluded)
	tailContractDraft := &contracts.Contract{
		ContractDID: "did:tail:contract:draft",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractDraft, // Not active
	}
	err = store.Upsert(tailContractDraft)
	require.NoError(t, err)

	// Create Tail Contract in ACCEPTED state (should be found)
	tailContractActive := &contracts.Contract{
		ContractDID: "did:tail:contract:active",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(tailContractActive)
	require.NoError(t, err)

	// Find Tail Contracts
	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract",
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	tailContract, err := FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.NoError(t, err)
	require.NotNil(t, tailContract, "Should find only active Tail Contract")
	assert.Equal(t, "did:tail:contract:active", tailContract.DID)
}

func TestFindTailContract_HeadContractExistsLocally(t *testing.T) {
	store := setupTestDB(t)

	// Organization DID
	orgDID, err := did.FromString("did:org:123")
	require.NoError(t, err)

	// Compute Provider DID
	providerDID, err := did.FromString("did:provider:456")
	require.NoError(t, err)

	// Create Head Contract (exists locally, should be skipped)
	headContract := &contracts.Contract{
		ContractDID: "did:head:contract",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  orgDID,
			Requestor: did.DID{URI: "did:orchestrator:789"},
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(headContract)
	require.NoError(t, err)

	// Create Tail Contract
	tailContractObj := &contracts.Contract{
		ContractDID: "did:tail:contract",
		ContractParticipants: contracts.ContractParticipants{
			Provider:  providerDID,
			Requestor: orgDID,
		},
		CurrentState: contracts.ContractAccepted,
	}
	err = store.Upsert(tailContractObj)
	require.NoError(t, err)

	// Find Tail Contracts (Head Contract should be skipped)
	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract", // Same DID as local Head Contract
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	tailContractConfig, err := FindTailContractFromContracts(headContractConfig, "did:provider:456", allContracts)

	require.NoError(t, err)
	require.NotNil(t, tailContractConfig, "Should find Tail Contract but skip Head Contract")
	assert.Equal(t, "did:tail:contract", tailContractConfig.DID)
	assert.NotEqual(t, "did:head:contract", tailContractConfig.DID, "Head Contract should be skipped")
}

func TestFindTailContract_InvalidComputeProviderDID(t *testing.T) {
	store := setupTestDB(t)

	headContractConfig := types.ContractConfig{
		DID:      "did:head:contract",
		Provider: "did:org:123",
	}
	allContracts, err := store.GetAllContracts()
	require.NoError(t, err)
	_, err = FindTailContractFromContracts(headContractConfig, "invalid-did", allContracts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid compute provider DID")
}
