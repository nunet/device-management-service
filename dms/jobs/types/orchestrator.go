// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"time"

	"gitlab.com/nunet/device-management-service/types"
)

type DeploymentStatus int

const (
	DeploymentStatusPreparing DeploymentStatus = iota
	DeploymentStatusGenerating
	DeploymentStatusCommitting
	DeploymentStatusProvisioning
	DeploymentStatusRunning
	DeploymentStatusFailed
	DeploymentStatusShuttingDown
	DeploymentStatusCompleted
)

func DeploymentStatusString(d DeploymentStatus) string {
	switch d {
	case DeploymentStatusPreparing:
		return "Preparing"
	case DeploymentStatusGenerating:
		return "Generating"
	case DeploymentStatusCommitting:
		return "Committing"
	case DeploymentStatusProvisioning:
		return "Provisioning"
	case DeploymentStatusRunning:
		return "Running"
	case DeploymentStatusFailed:
		return "Failed"
	case DeploymentStatusShuttingDown:
		return "ShuttingDown"
	case DeploymentStatusCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}

type OrchestratorView struct {
	types.BaseDBModel
	OrchestratorID     string
	Cfg                EnsembleConfig
	Manifest           EnsembleManifest
	Status             DeploymentStatus
	DeploymentSnapshot DeploymentSnapshot
	PrivKey            []byte
}

type DeploymentSnapshot struct {
	// candidates keeps state of candidates while committing.
	Candidates map[string]Bid

	// Expiry is the time passed as an argument when calling Deploy()
	Expiry time.Time
}
