package clover

import (
	"fmt"
	"os"
	"testing"

	clover "github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/assert"
)

// setup initializes and sets up the clover database using bbolt under the hood in a temporary dir.
// Additionally, it automatically creates collections for the necessary types.
// TODO: add error handling?
func setup() (*clover.DB, string) {
	path, err := tempDir()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	db, err := clover.Open(path)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// Create collections
	collections := []string{"car", "computer_specs"}

	for _, collection := range collections {
		if err := db.CreateCollection(collection); err != nil {
			return nil, ""
		}
	}

	return db, path
}

// teardown closes the GORM database connection after tests.
func teardown(db *clover.DB, path string) {
	// close the clover database
	db.Close()
	os.RemoveAll(path)
}

func TestNewDB(t *testing.T) {
	path, err := tempDir()
	assert.NoError(t, err)
	defer os.RemoveAll(path)

	collections := []string{"test_collection1", "test_collection2"}

	db, err := NewDB(path, collections)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Check if collections were created
	for _, collection := range collections {
		exists, err := db.HasCollection(collection)
		assert.NoError(t, err)
		assert.True(t, exists, "Collection %s should exist", collection)
	}

	// Try to create an existing collection
	err = db.CreateCollection(collections[0])
	assert.Error(t, err)

	// Close the database
	err = db.Close()
	assert.NoError(t, err)
}

func tempDir() (string, error) {
	dir, err := os.MkdirTemp("", "nunet-test-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return dir, nil
}
