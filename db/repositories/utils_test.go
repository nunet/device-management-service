// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/types"
)

// TestUpdateField tests the updateField function for updating struct fields using reflection.
// It covers cases where the input can be a struct or a pointer to a struct.
func TestUpdateField(t *testing.T) {
	// Test case: Updating a field in a struct (not a pointer)
	modified1, err := UpdateField(types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}, "RAM", types.RAM{Size: 2})
	assert.NoError(t, err)
	assert.Equal(t, float64(2), modified1.RAM.Size)

	// Test case: Updating a field in a struct (using a pointer)
	modified2, err := UpdateField(&types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}, "RAM", types.RAM{Size: 2})
	assert.NoError(t, err)
	assert.Equal(t, float64(2), modified2.RAM.Size)

	// Test case: Attempting to update a non-existent field results in an error
	_, err = UpdateField(types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}, "WRAM", types.RAM{Size: 2})
	assert.Error(t, err)

	// Test case: Attempting to update with an incompatible value results in an error
	_, err = UpdateField(types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}, "WRAM", "a")
	assert.Error(t, err)
}

// TestEmptyValue tests the isEmptyValue function for checking if a struct has non zero value.
// It asserts that the function correctly identifies empty and non-empty structs (or a pointer to a struct).
func TestEmptyValue(t *testing.T) {
	// Test case: nil should be considered empty
	assert.Equal(t, true, IsEmptyValue(nil))

	// Test case: Empty struct and its pointer should be considered empty
	assert.Equal(t, true, IsEmptyValue(types.FreeResources{}))
	assert.Equal(t, true, IsEmptyValue(&types.FreeResources{}))

	// Test case: Struct and its pointer with non-zero field should not be considered empty
	assert.Equal(t, false, IsEmptyValue(types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}))
	assert.Equal(t, false, IsEmptyValue(&types.FreeResources{Resources: types.Resources{RAM: types.RAM{Size: 1}}}))
}
