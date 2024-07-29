package repositories_clover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/models"
)

// TestStorageVolumeRepository is a test suite for the StorageVolumeRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the StorageVolume model behave as expected.
func TestStorageVolumeRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	storageVolRepo := NewStorageVolumeRepository(db)

	// Test Create method
	createdStorageVol, err := storageVolRepo.Create(
		context.Background(),
		models.StorageVolume{
			Path:           "/test",
			ReadOnly:       true,
			EncryptionType: models.EncryptionTypeNull,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdStorageVol.ID)

	// Test Get method
	retrievedStorageVol, err := storageVolRepo.Get(
		context.Background(),
		createdStorageVol.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdStorageVol.Path, retrievedStorageVol.Path)

	// Test Update method
	updatedStorageVol := retrievedStorageVol
	updatedStorageVol.CID = "baf123"

	_, err = storageVolRepo.Update(
		context.Background(),
		updatedStorageVol.ID,
		updatedStorageVol,
	)
	assert.NoError(t, err)

	retrievedStorageVol, err = storageVolRepo.Get(
		context.Background(),
		createdStorageVol.ID,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		updatedStorageVol.CID,
		retrievedStorageVol.CID,
	)

	// Test Delete method
	err = storageVolRepo.Delete(context.Background(), updatedStorageVol.ID)
	assert.NoError(t, err)

	// Test Find method
	storageVol2, err := storageVolRepo.Create(
		context.Background(),
		models.StorageVolume{
			Path:           "/job123",
			ReadOnly:       false,
			EncryptionType: models.EncryptionTypeNull,
		},
	)
	assert.NoError(t, err)

	query := storageVolRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.EQ("Path", storageVol2.Path),
	)
	foundStorageVol, err := storageVolRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, storageVol2.Path, foundStorageVol.Path)

	// Test FindAll method
	storageVol3, err := storageVolRepo.Create(
		context.Background(),
		models.StorageVolume{
			Path:           "/gpt4-data",
			ReadOnly:       true,
			EncryptionType: models.EncryptionTypeNull,
		},
	)
	assert.NoError(t, err)

	allStorageVols, err := storageVolRepo.FindAll(
		context.Background(),
		storageVolRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allStorageVols, 2)

	// Clean up created records
	err = storageVolRepo.Delete(context.Background(), storageVol2.ID)
	err = storageVolRepo.Delete(context.Background(), storageVol3.ID)
}
