// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

type (
	NodeManifest        = jobtypes.NodeManifest
	NodeConfig          = jobtypes.NodeConfig
	EnsembleConfig      = jobtypes.EnsembleConfig
	EnsembleManifest    = jobtypes.EnsembleManifest
	DeploymentStatus    = jobtypes.DeploymentStatus
	DeploymentSnapshot  = jobtypes.DeploymentSnapshot
	AllocationManifest  = jobtypes.AllocationManifest
	AllocationExecutor  = jobtypes.AllocationExecutor
	Bid                 = jobtypes.Bid
	BidRequest          = jobtypes.BidRequest
	EnsembleBidRequest  = jobtypes.EnsembleBidRequest
	EdgeConstraint      = jobtypes.EdgeConstraint
	Location            = jobtypes.Location
	LocationConstraints = jobtypes.LocationConstraints
	OrchestratorView    = jobtypes.OrchestratorView
)

const (
	DeploymentStatusPreparing    = jobtypes.DeploymentStatusPreparing
	DeploymentStatusGenerating   = jobtypes.DeploymentStatusGenerating
	DeploymentStatusCommitting   = jobtypes.DeploymentStatusCommitting
	DeploymentStatusProvisioning = jobtypes.DeploymentStatusProvisioning
	DeploymentStatusRunning      = jobtypes.DeploymentStatusRunning
	DeploymentStatusFailed       = jobtypes.DeploymentStatusFailed
	DeploymentStatusShuttingDown = jobtypes.DeploymentStatusShuttingDown
	DeploymentStatusCompleted    = jobtypes.DeploymentStatusCompleted
)
