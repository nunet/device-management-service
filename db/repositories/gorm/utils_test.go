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
