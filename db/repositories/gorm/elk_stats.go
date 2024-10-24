// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerGORM is a GORM implementation of the RequestTracker interface.
type RequestTrackerGORM struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTracker creates a new instance of RequestTrackerGORM.
// It initializes and returns a GORM-based repository for RequestTracker entities.
func NewRequestTracker(db *gorm.DB) repositories.RequestTracker {
	return &RequestTrackerGORM{
		NewGenericRepository[types.RequestTracker](db),
	}
}
