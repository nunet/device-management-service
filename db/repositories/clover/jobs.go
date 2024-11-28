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
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// OrchestratorViewClover is a Clover implementation of the StorageVolume interface.
type OrchestratorViewClover struct {
	repositories.GenericRepository[job_types.OrchestratorView]
}

// NewOrchestratorView creates a new instance of OrchestratorViewClover.
// It initializes and returns a Clover-based repository for StorageVolume entities.
func NewOrchestratorView(db *clover.DB) repositories.OrchestratorView {
	return &OrchestratorViewClover{
		NewGenericRepository[job_types.OrchestratorView](db),
	}
}
