package transaction

import (
	"testing"

	"github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/require"
)

// setupTestDB initializes a temporary Clover DB for tests
func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	require.NoError(t, err)

	err = db.CreateCollection(transactionsCollection)
	require.NoError(t, err)

	store, err := New(db)
	require.NoError(t, err)

	return store
}

func TestUpsert_NewTransaction(t *testing.T) {
	store := setupTestDB(t)

	tx := Transaction{
		UniqueID:            "tx1",
		PaymentValidatorDID: "val1",
		ContractDID:         "contract1",
		ToAddress:           "address1",
		Amount:              "100",
	}

	err := store.Upsert(tx)
	require.NoError(t, err)

	txs, err := store.AllTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 1)

	stored := txs[0]
	require.Equal(t, "tx1", stored.UniqueID)
	require.Equal(t, "unpaid", stored.Status)
}

func TestUpsert_ExistingTransaction(t *testing.T) {
	store := setupTestDB(t)

	tx := Transaction{
		UniqueID: "tx2",
		Amount:   "200",
	}

	// first time
	err := store.Upsert(tx)
	require.NoError(t, err)

	// again, should not duplicate
	err = store.Upsert(tx)
	require.NoError(t, err)

	txs, err := store.AllTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 1)
}

func TestMarkAsPaid(t *testing.T) {
	store := setupTestDB(t)

	tx := Transaction{
		UniqueID: "tx3",
		Amount:   "300",
	}
	err := store.Upsert(tx)
	require.NoError(t, err)

	_, err = store.MarkAsPaid("tx3", "h2")
	require.NoError(t, err)

	txs, err := store.AllTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 1)

	stored := txs[0]
	require.Equal(t, "paid", stored.Status)
}

func TestMarkAsPaid_NotFound(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.MarkAsPaid("nonexistent", "h1")
	require.Error(t, err)
}

func TestAllTransactions_Multiple(t *testing.T) {
	store := setupTestDB(t)

	txs := []Transaction{
		{UniqueID: "txA", Amount: "10"},
		{UniqueID: "txB", Amount: "20"},
	}

	for _, tx := range txs {
		err := store.Upsert(tx)
		require.NoError(t, err)
	}

	all, err := store.AllTransactions()
	require.NoError(t, err)
	require.Len(t, all, 2)
}
