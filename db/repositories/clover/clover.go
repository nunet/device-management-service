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

	"github.com/dgraph-io/badger/v3"
	clover "github.com/ostafen/clover/v2"
	badgerstore "github.com/ostafen/clover/v2/store/badger"

	"gitlab.com/nunet/device-management-service/observability"
)

func createCollections(db *clover.DB, collections []string) error {
	for _, c := range collections {
		if err := db.CreateCollection(c); err != nil {
			if err == clover.ErrCollectionExist {
				continue
			}
			err = fmt.Errorf("failed to create collection %s: %w", c, err)
			logger.Errorw("clover_db_init_failure", "collection", c, "error", err)
			return err
		}
	}

	return nil
}

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

	if err := createCollections(db, collections); err != nil {
		return nil, err
	}

	logger.Infow("clover_db_init_success", "path", path, "collections", collections)
	return db, nil
}

// NewMemDB initializes and sets up in-memory database using badger store.
// Additionally, it automatically creates collections for the necessary types.
func NewMemDB(collections []string) (*clover.DB, error) {
	store, err := badgerstore.Open(badger.DefaultOptions("").WithInMemory(true)) // opens a badger in memory database
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory store: %w", err)
	}
	db, err := clover.OpenWithStore(store)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := createCollections(db, collections); err != nil {
		return nil, err
	}

	return db, nil
}
