// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gorm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// TestHandleDBError tests the handleDBError function for proper error translation.
// It asserts that different GORM errors are correctly translated into corresponding custom repository errors.
func TestHandleDBError(t *testing.T) {
	// Test case: GORM ErrRecordNotFound should result in NotFoundError
	err := handleDBError(gorm.ErrRecordNotFound)
	assert.Equal(t, repositories.ErrNotFound, err)

	// Test case: GORM ErrInvalidData should result in InvalidDataError
	err = handleDBError(gorm.ErrInvalidData)
	assert.Equal(t, repositories.ErrInvalidData, err)

	// Test case: GORM ErrInvalidDB should result in DatabaseError
	// We should check if error HAS a DatabaseError as it may be wrapped with other errors
	err = handleDBError(gorm.ErrInvalidDB)
	assert.ErrorAs(t, err, &repositories.ErrDatabase)

	// Test case: custom ErrParsingModel should result in ErrParsingModel
	// ErrParsingModel is a wrapped error
	err = handleDBError(
		fmt.Errorf("%v: %v", repositories.ErrParsingModel,
			fmt.Errorf("some error"),
		))
	assert.ErrorAs(t, err, &repositories.ErrParsingModel)
}
