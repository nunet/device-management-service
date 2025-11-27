// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package events

import "gitlab.com/nunet/device-management-service/types"

type EventType string

const (
	CreateAllocationEvent   EventType = "CREATE_ALLOCATION_EVENT"
	StartAllocationEvent    EventType = "START_ALLOCATION_EVENT"
	StopAllocationEvent     EventType = "STOP_ALLOCATION_EVENT"
	CompleteAllocationEvent EventType = "COMPLETE_ALLOCATION_EVENT"
	DeploymentStartEvent    EventType = "DEPLOYMENT_START_EVENT"
	DeploymentStopEvent     EventType = "DEPLOYMENT_STOP_EVENT"
)

type CreateAllocation struct {
	Type               EventType       `json:"type"`
	AllocationID       string          `json:"allocation_id"`
	Resources          types.Resources `json:"resources"`
	DeploymentID       string          `json:"deployment_id"`
	ComputeProviderDID string          `json:"compute_provider_did"`
}

type StartAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	DeploymentID       string    `json:"deployment_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
}

type StopAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	DeploymentID       string    `json:"deployment_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
}

type CompleteAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
	DeploymentID       string    `json:"deployment_id"`
}

type DeploymentStart struct {
	Type           EventType `json:"type"`
	DeploymentID   string    `json:"deployment_id"`
	OrchestratorID string    `json:"orchestrator_id"`
}

type DeploymentStop struct {
	Type           EventType `json:"type"`
	DeploymentID   string    `json:"deployment_id"`
	OrchestratorID string    `json:"orchestrator_id"`
}
