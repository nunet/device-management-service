package events

import "gitlab.com/nunet/device-management-service/types"

type EventType string

const (
	CreateAllocationEvent   EventType = "CREATE_ALLOCATION_EVENT"
	StartAllocationEvent    EventType = "START_ALLOCATION_EVENT"
	StopAllocationEvent     EventType = "STOP_ALLOCATION_EVENT"
	CompleteAllocationEvent EventType = "COMPLETE_ALLOCATION_EVENT"
)

type CreateAllocation struct {
	Type               EventType       `json:"type"`
	AllocationID       string          `json:"allocation_id"`
	Resources          types.Resources `json:"resources"`
	ComputeProviderDID string          `json:"compute_provider_did"`
}

type StartAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
}

type StopAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
}

type CompleteAllocation struct {
	Type               EventType `json:"type"`
	AllocationID       string    `json:"allocation_id"`
	ComputeProviderDID string    `json:"compute_provider_did"`
}
