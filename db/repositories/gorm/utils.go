package gorm

import (
	"errors"

	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// handleDBError is a utility function that translates GORM database errors into custom repository errors.
// It takes a GORM database error as input and returns a corresponding custom error from the repositories package.
func handleDBError(err error) error {
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return repositories.ErrNotFound
		case gorm.ErrInvalidData, gorm.ErrInvalidField, gorm.ErrInvalidValue:
			return repositories.ErrInvalidData
		case repositories.ErrParsingModel:
			return err
		default:
			return errors.Join(repositories.ErrDatabase, err)
		}
	}
	return nil
}
