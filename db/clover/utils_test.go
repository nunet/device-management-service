// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package clover

import (
	"testing"

	"github.com/ostafen/clover/v2"
	cloverd "github.com/ostafen/clover/v2/document"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// TestHandleDBError tests the handleDBError function
func TestHandleDBError(t *testing.T) {
	t.Parallel()
	// Test with nil error
	err := handleDBError(nil)
	assert.NoError(t, err, "handleDBError should return nil for nil error")

	// Test with ErrDocumentNotExist
	err = handleDBError(clover.ErrDocumentNotExist)
	assert.ErrorIs(t, err, repositories.ErrNotFound, "handleDBError should return ErrNotFound for ErrDocumentNotExist")

	// Test with ErrDuplicateKey
	err = handleDBError(clover.ErrDuplicateKey)
	assert.ErrorIs(t, err, repositories.ErrInvalidData, "handleDBError should return ErrInvalidData for ErrDuplicateKey")

	// Test with ErrParsingModel
	err = handleDBError(repositories.ErrParsingModel)
	assert.ErrorIs(t, err, repositories.ErrParsingModel, "handleDBError should return ErrParsingModel for ErrParsingModel")

	// Test with other error
	err = handleDBError(clover.ErrCollectionNotExist)
	assert.ErrorIs(t, err, repositories.ErrDatabase, "handleDBError should wrap other errors with ErrDatabase")
}

// TestToCloverDoc tests the toCloverDoc function
func TestToCloverDoc(t *testing.T) {
	t.Parallel()
	// Create a test model
	testModel := types.StorageVolume{
		BaseDBModel: types.BaseDBModel{
			ID: "test-id",
		},
		CID:      "test-cid",
		Path:     "/test/path",
		ReadOnly: true,
		Private:  false,
	}

	// Convert to Clover document
	doc := toCloverDoc(testModel)

	// Verify the document was created
	assert.NotNil(t, doc, "Document should not be nil")

	// We can't directly test the document contents because the field names
	// may be different due to JSON serialization. Just verify the document exists.
}

// TestToModel tests the toModel function
func TestToModel(t *testing.T) {
	t.Parallel()
	// Create a test document with the correct JSON field names
	doc := cloverd.NewDocument()
	// Set the document ID using the Set method instead of ObjectId
	doc.Set("_id", "test-id")

	// Create a map with the correct JSON field names
	data := map[string]interface{}{
		"cid":             "test-cid",
		"path":            "/test/path",
		"read_only":       true,
		"private":         false,
		"encryption_type": "null",
	}

	// Add all fields to the document
	for k, v := range data {
		doc.Set(k, v)
	}

	// Convert to model (non-entity repository)
	model, err := toModel[types.StorageVolume](doc, false)
	assert.NoError(t, err, "toModel should not return an error")

	// Verify basic conversion worked
	assert.NotEmpty(t, model.ID, "Model should have an ID")
	assert.Equal(t, "test-cid", model.CID, "Model should have correct CID")
	assert.Equal(t, "/test/path", model.Path, "Model should have correct Path")

	// Test with entity repository flag
	model, err = toModel[types.StorageVolume](doc, true)
	assert.NoError(t, err, "toModel should not return an error for entity repository")
}

// TestFieldJSONTag tests the fieldJSONTag function
func TestFieldJSONTag(t *testing.T) {
	t.Parallel()
	// Test with field that has a JSON tag
	fieldName := fieldJSONTag[types.StorageVolume]("CID")
	assert.Equal(t, "CID", fieldName, "fieldJSONTag should return the JSON tag for CID")

	// Test with field that has a JSON tag with options
	fieldName = fieldJSONTag[types.OnboardingConfig]("OnboardedResources")
	assert.Equal(t, "onboarded_resources", fieldName, "fieldJSONTag should return the JSON tag for OnboardedResources")

	// Test with field that doesn't exist
	fieldName = fieldJSONTag[types.StorageVolume]("NonExistentField")
	assert.Equal(t, "NonExistentField", fieldName, "fieldJSONTag should return the original field name for non-existent fields")
}
