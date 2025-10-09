// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package payment

import (
	"testing"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	if err != nil {
		t.Fatalf("failed to open CloverDB: %v", err)
	}
	err = db.CreateCollection(paymentsCollection)
	require.NoError(t, err)

	store, err := New(db)
	require.NoError(t, err)

	return store
}

func samplePayment(uniqueID, contractDID string, usages int, paid bool) Payment {
	return Payment{
		UniqueID: uniqueID,
		Contract: contracts.Contract{
			ContractDID: contractDID,
		},
		Usages: usages,
		Paid:   paid,
	}
}

func TestInsert_InsertNewPayment(t *testing.T) {
	store := setupTestDB(t)

	p := samplePayment("p1", "cc1", 5, true)

	err := store.Insert(p)
	require.NoError(t, err)

	payments, err := store.AllPayments()
	require.NoError(t, err)
	require.Len(t, payments, 1)

	require.Equal(t, "p1", payments[0].UniqueID)
	require.Equal(t, "cc1", payments[0].Contract.ContractDID)
	require.Equal(t, 5, payments[0].Usages)
	require.True(t, payments[0].Paid)
}

func TestInsert_IgnoreDuplicateUniqueID(t *testing.T) {
	store := setupTestDB(t)

	p1 := samplePayment("p1", "c1", 5, true)
	err := store.Insert(p1)
	require.NoError(t, err)

	// inserting duplicate with same unique_id but different usages/paid
	p2 := samplePayment("p1", "c1", 99, false)
	err = store.Insert(p2)
	require.NoError(t, err)

	payments, err := store.AllPayments()
	require.NoError(t, err)
	require.Len(t, payments, 1)

	// original data should still be there, not replaced
	require.Equal(t, 5, payments[0].Usages)
	require.True(t, payments[0].Paid)
}

func TestInsert_MultiplePaymentsSameContract(t *testing.T) {
	store := setupTestDB(t)

	p1 := samplePayment("p1", "c1", 1, false)
	p2 := samplePayment("p2", "c1", 2, true)

	err := store.Insert(p1)
	require.NoError(t, err)
	err = store.Insert(p2)
	require.NoError(t, err)

	payments, err := store.AllPayments()
	require.NoError(t, err)
	require.Len(t, payments, 2)

	ids := []string{payments[0].UniqueID, payments[1].UniqueID}
	require.Contains(t, ids, "p1")
	require.Contains(t, ids, "p2")
}

func TestAllPayments_Empty(t *testing.T) {
	store := setupTestDB(t)

	payments, err := store.AllPayments()
	require.NoError(t, err)
	require.Empty(t, payments)
}

func TestGetByUniqueID_Found(t *testing.T) {
	store := setupTestDB(t)

	p := samplePayment("p1", "c1", 10, true)
	err := store.Insert(p)
	require.NoError(t, err)

	got, err := store.GetByUniqueID("p1")
	require.NoError(t, err)
	require.NotNil(t, got)

	require.Equal(t, "p1", got.UniqueID)
	require.Equal(t, "c1", got.Contract.ContractDID)
	require.Equal(t, 10, got.Usages)
	require.True(t, got.Paid)
}

func TestGetByUniqueID_NotFound(t *testing.T) {
	store := setupTestDB(t)

	got, err := store.GetByUniqueID("does-not-exist")
	require.Error(t, err)
	require.Nil(t, got)
	require.Contains(t, err.Error(), "not found")
}

func TestGetByUniqueID_InvalidData(t *testing.T) {
	store := setupTestDB(t)

	doc := document.NewDocument()
	doc.Set("unique_id", "bad1")
	doc.Set("payment_data", []byte("this-is-not-json-bytes"))

	err := store.db.Insert(paymentsCollection, doc)
	require.NoError(t, err)

	got, err := store.GetByUniqueID("bad1")
	require.Error(t, err)
	require.Nil(t, got)
	require.Contains(t, err.Error(), "failed to unmarshal")
}

func TestUpdate_Success(t *testing.T) {
	store := setupTestDB(t)

	p := samplePayment("p1", "c1", 5, false)
	err := store.Insert(p)
	require.NoError(t, err)

	p.Usages = 10
	p.Paid = true
	err = store.Update(&p)
	require.NoError(t, err)

	updated, err := store.GetByUniqueID("p1")
	require.NoError(t, err)
	require.Equal(t, 10, updated.Usages)
	require.True(t, updated.Paid)
	require.Equal(t, "c1", updated.Contract.ContractDID) // unchanged
}

func TestUpdate_NotFound(t *testing.T) {
	store := setupTestDB(t)

	p := samplePayment("does-not-exist", "c1", 1, false)
	err := store.Update(&p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestUpdate_EmptyUniqueID(t *testing.T) {
	store := setupTestDB(t)

	p := samplePayment("", "c1", 1, false)
	err := store.Update(&p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unique_id is required")
}
