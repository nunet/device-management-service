// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

type (
	NodeManifest        = job_types.NodeManifest
	NodeConfig          = job_types.NodeConfig
	EnsembleConfig      = job_types.EnsembleConfig
	EnsembleManifest    = job_types.EnsembleManifest
	DeploymentStatus    = job_types.DeploymentStatus
	DeploymentSnapshot  = job_types.DeploymentSnapshot
	AllocationManifest  = job_types.AllocationManifest
	AllocationExecutor  = job_types.AllocationExecutor
	Bid                 = job_types.Bid
	BidRequest          = job_types.BidRequest
	EnsembleBidRequest  = job_types.EnsembleBidRequest
	EdgeConstraint      = job_types.EdgeConstraint
	Location            = job_types.Location
	LocationConstraints = job_types.LocationConstraints
	OrchestratorView    = job_types.OrchestratorView
)

const (
	DeploymentStatusPreparing    = job_types.DeploymentStatusPreparing
	DeploymentStatusGenerating   = job_types.DeploymentStatusGenerating
	DeploymentStatusCommitting   = job_types.DeploymentStatusCommitting
	DeploymentStatusProvisioning = job_types.DeploymentStatusProvisioning
	DeploymentStatusRunning      = job_types.DeploymentStatusRunning
	DeploymentStatusFailed       = job_types.DeploymentStatusFailed
	DeploymentStatusShuttingDown = job_types.DeploymentStatusShuttingDown
	DeploymentStatusCompleted    = job_types.DeploymentStatusCompleted
)
