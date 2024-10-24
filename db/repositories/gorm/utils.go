// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
