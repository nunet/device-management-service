// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package clover

import (
	clover "github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// RequestTrackerClover is a Clover implementation of the RequestTracker interface.
type RequestTrackerClover struct {
	repositories.GenericRepository[types.RequestTracker]
}

// NewRequestTracker creates a new instance of RequestTrackerClover.
// It initializes and returns a Clover-based repository for RequestTracker entities.
func NewRequestTracker(db *clover.DB) repositories.RequestTracker {
	endTrace := observability.StartTrace("clover_db_repo_init_duration")
	defer endTrace()

	repo := &RequestTrackerClover{
		NewGenericRepository[types.RequestTracker](db),
	}

	logger.Infow("clover_db_repo_init_success", "repository", "RequestTracker")
	return repo
}
