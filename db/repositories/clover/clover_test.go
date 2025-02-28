// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package clover

import (
	"fmt"
	"os"
	"testing"

	clover "github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/observability"
)

// setup initializes and sets up the clover database using bbolt under the hood in a temporary dir.
// Additionally, it automatically creates collections for the necessary types.
// TODO: add error handling?
func setup() (*clover.DB, string) {
	// Set observability to no-op mode for testing
	observability.SetNoOpMode(true)

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

// teardown closes the clover database after tests.
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
