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

	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/observability"
)

// NewDB initializes and sets up the clover database using bbolt under the hood.
// Additionally, it automatically creates collections for the necessary types.
func NewDB(path string, collections []string) (*clover.DB, error) {
	endTrace := observability.StartTrace("clover_db_init_duration")
	defer endTrace()

	db, err := clover.Open(path)
	if err != nil {
		logger.Errorw("clover_db_init_failure", "error", fmt.Errorf("failed to connect to database: %w", err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	for _, c := range collections {
		if err := db.CreateCollection(c); err != nil {
			if err == clover.ErrCollectionExist {
				continue
			}
			err = fmt.Errorf("failed to create collection %s: %w", c, err)
			logger.Errorw("clover_db_init_failure", "collection", c, "error", err)
			return nil, err
		}
	}

	logger.Infow("clover_db_init_success", "path", path, "collections", collections)
	return db, nil
}
