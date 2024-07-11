package repositories_gorm

import (
	"errors"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

const structFieldNameDeletedAt = "DeletedAt"

// handleDBError is a utility function that translates GORM database errors into custom repository errors.
// It takes a GORM database error as input and returns a corresponding custom error from the repositories package.
func handleDBError(err error) error {
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return repositories.NotFoundError
		case gorm.ErrInvalidData, gorm.ErrInvalidField, gorm.ErrInvalidValue:
			return repositories.InvalidDataError
		case repositories.ErrParsingModel:
			return err
		default:
			return errors.Join(repositories.DatabaseError, err)
		}
	}
	return nil
}
