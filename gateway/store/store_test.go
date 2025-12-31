package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/gateway/provider"
)

// setupTestDB initializes a temporary Clover DB for tests
func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	require.NoError(t, err)

	err = db.CreateCollection("provisioned_resources")
	require.NoError(t, err)

	s, err := New(db)
	require.NoError(t, err)

	return s
}

func TestNew_NilDB(t *testing.T) {
	s, err := New(nil)
	assert.Error(t, err)
	assert.Nil(t, s)
}

func TestNew_Success(t *testing.T) {
	s := setupTestDB(t)
	assert.NotNil(t, s)
}

func TestInsert_NilInput(t *testing.T) {
	s := setupTestDB(t)

	err := s.Insert(nil)
	assert.Error(t, err)
}

func TestInsert_Success(t *testing.T) {
	s := setupTestDB(t)

	r := &ProvisionedResources{
		ProviderName:        "aws",
		Orchestrator:        "orch",
		ProvisionedVMPeerID: "peer1",
		Resource:            provider.Server{ID: "server123"},
		CreatedAt:           time.Now(),
	}

	err := s.Insert(r)
	assert.NoError(t, err)

	// Verify through All()
	all, err := s.All()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "server123", all[0].Resource.ID)
}

func TestAll_Success(t *testing.T) {
	s := setupTestDB(t)

	r := &ProvisionedResources{
		ProviderName:        "aws",
		Orchestrator:        "orch",
		ProvisionedVMPeerID: "peer-alpha",
		Resource:            provider.Server{ID: "srv-1"},
		CreatedAt:           time.Now(),
	}

	require.NoError(t, s.Insert(r))

	all, err := s.All()
	require.NoError(t, err)
	require.Len(t, all, 1)

	assert.Equal(t, "srv-1", all[0].Resource.ID)
}

func TestDelete_EmptyID(t *testing.T) {
	s := setupTestDB(t)

	err := s.Delete("")
	assert.Error(t, err)
}

func TestDelete_Success(t *testing.T) {
	s := setupTestDB(t)

	r := &ProvisionedResources{
		ProviderName:        "aws",
		Orchestrator:        "orch",
		ProvisionedVMPeerID: "peer2",
		Resource:            provider.Server{ID: "delete-me"},
		CreatedAt:           time.Now(),
	}

	require.NoError(t, s.Insert(r))

	err := s.Delete("delete-me")
	assert.NoError(t, err)

	all, err := s.All()
	require.NoError(t, err)
	assert.Len(t, all, 0)
}
