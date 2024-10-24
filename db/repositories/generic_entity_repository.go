// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package repositories

import (
	"context"
)

// GenericEntityRepository is an interface defining basic CRUD operations for repositories handling a single record.
type GenericEntityRepository[T ModelType] interface {
	// Save adds or updates a single record in the repository.
	Save(ctx context.Context, data T) (T, error)
	// Get retrieves the single record from the repository.
	Get(ctx context.Context) (T, error)
	// Clear removes the record and its history from the repository.
	Clear(ctx context.Context) error
	// History retrieves previous records from the repository constrained by the query.
	History(ctx context.Context, qiery Query[T]) ([]T, error)
	// GetQuery returns an empty query instance for the repository's type.
	GetQuery() Query[T]
}
