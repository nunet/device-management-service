// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: keeping here temporarily. We must organize types and behavior payloads.
// issue: https://gitlab.com/nunet/device-management-service/-/issues/893 (part 2)
// TODO (wrong nomenclature): AllocationDeploymentRequest -> EnsembleDeploymentRequest
type AllocationDeploymentRequest struct {
	EnsembleID  string
	NodeID      string
	Allocations map[string]AllocationDeploymentConfig
}

type AllocationDeploymentConfig struct {
	Type             AllocationType
	Executor         AllocationExecutor
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
	Keys             []types.AllocationKey
	Volume           []types.VolumeConfig
	Contracts        map[string]types.ContractConfig
}

type AllocationDeploymentResponse struct {
	OK          bool
	Error       string
	Allocations map[string]actor.Handle
}
