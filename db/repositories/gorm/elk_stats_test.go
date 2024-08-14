package repositories_gorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// TestRequestTrackerRepository is a test suite for the requestTrackerRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the RequestTracker model behave as expected.
func TestRequestTrackerRepository(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	requestTrackerRepo := NewRequestTrackerRepository(db)

	// Test Create method
	createdRequestTracker, err := requestTrackerRepo.Create(
		context.Background(),
		types.RequestTracker{},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdRequestTracker.ID)

	// Test Get method
	retrievedRequestTracker, err := requestTrackerRepo.Get(
		context.Background(),
		createdRequestTracker.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdRequestTracker.ID, retrievedRequestTracker.ID)

	// Test Update method
	updatedRequestTracker := retrievedRequestTracker
	updatedRequestTracker.Status = "accepted"

	_, err = requestTrackerRepo.Update(
		context.Background(),
		updatedRequestTracker.ID,
		updatedRequestTracker,
	)
	assert.NoError(t, err)
	retrievedRequestTracker, err = requestTrackerRepo.Get(
		context.Background(),
		createdRequestTracker.ID,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		updatedRequestTracker.Status,
		retrievedRequestTracker.Status,
	)

	// Test Delete method
	err = requestTrackerRepo.Delete(context.Background(), updatedRequestTracker.ID)
	assert.NoError(t, err)

	// Test Find method
	requestTracker1, err := requestTrackerRepo.Create(
		context.Background(),
		types.RequestTracker{Status: "started"},
	)
	assert.NoError(t, err)

	query := requestTrackerRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Status", requestTracker1.Status))
	foundRequestTracker, err := requestTrackerRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, requestTracker1.Status, foundRequestTracker.Status)

	// Test FindAll method
	requestTracker2, err := requestTrackerRepo.Create(
		context.Background(),
		types.RequestTracker{Status: "finished"},
	)
	assert.NoError(t, err)

	allRequestTracker, err := requestTrackerRepo.FindAll(
		context.Background(),
		requestTrackerRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allRequestTracker, 2)

	// Clean up created records
	err = requestTrackerRepo.Delete(context.Background(), requestTracker1.ID)
	err = requestTrackerRepo.Delete(context.Background(), requestTracker2.ID)
}
