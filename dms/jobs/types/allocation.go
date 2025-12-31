// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"gitlab.com/nunet/device-management-service/types"
)

// AllocationStatus is a representation of the execution status
type AllocationStatus string

const (
	AllocationPending    AllocationStatus = "pending"
	AllocationRunning    AllocationStatus = "running"
	AllocationStopped    AllocationStatus = "stopped"
	AllocationFailed     AllocationStatus = "failed"
	AllocationCompleted  AllocationStatus = "completed"
	AllocationTerminated AllocationStatus = "terminated"
	AllocationStandby    AllocationStatus = "standby" // Allocation is in standby mode (redundancy)
)

// HealthCheckType represents the type of health check performed
type HealthCheckType string

const (
	HealthCheckTypeNone     HealthCheckType = "none"     // No healthcheck configured
	HealthCheckTypeSelf     HealthCheckType = "self"     // Allocation self-reported health
	HealthCheckTypeExecutor HealthCheckType = "executor" // Executor-level health check
)

// AllocationLivenessNotification is sent periodically by service allocations
// to report their health and status to the orchestrator
// NOTE: This is for observability only, pull-based healthchecks remain authoritative
type AllocationLivenessNotification struct {
	AllocationID   string `json:"allocation_id"`
	Status         string `json:"status"`          // Current allocation status
	Timestamp      int64  `json:"timestamp"`       // Unix timestamp of report
	SequenceNumber int64  `json:"sequence_number"` // Monotonic counter to detect missed heartbeats

	// Health information (self-reported)
	HealthCheckEnabled bool         `json:"health_check_enabled"`
	Health             HealthStatus `json:"health"`

	// Optional: Resource usage metrics
	ResourceUsage *AllocationResourceUsage `json:"resource_usage,omitempty"`

	Version string `json:"version"` // Protocol version for future compatibility
}

// AllocationResourceUsage contains resource metrics from the executor
type AllocationResourceUsage struct {
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	MemoryUsedBytes  uint64  `json:"memory_used_bytes"`
	MemoryLimitBytes uint64  `json:"memory_limit_bytes"`
	NetworkRxBytes   uint64  `json:"network_rx_bytes"`
	NetworkTxBytes   uint64  `json:"network_tx_bytes"`
}

// HealthStatus contains self-reported health check results
type HealthStatus struct {
	Healthy       bool            `json:"healthy"`
	LastCheckTime int64           `json:"last_check_time"`
	Message       string          `json:"message,omitempty"`
	CheckType     HealthCheckType `json:"check_type,omitempty"` // Type of health check performed
}

// AllocationStatusUpdate is sent when allocation status changes significantly
// (e.g., starting, stopping, failing)
type AllocationStatusUpdate struct {
	AllocationID string `json:"allocation_id"`
	OldStatus    string `json:"old_status"`
	NewStatus    string `json:"new_status"`
	Timestamp    int64  `json:"timestamp"`
	Reason       string `json:"reason,omitempty"`
}

type AllocationInfo struct {
	AllocationID   string                  `json:"allocation_id"`
	Status         AllocationStatus        `json:"status"`
	HeartbeatSeq   int64                   `json:"heartbeat_seq"`
	HasHealthCheck bool                    `json:"has_health_check"`
	Health         string                  `json:"health"`
	ResourceLimit  types.Resources         `json:"resource_limit"`
	ResourceUsage  AllocationResourceUsage `json:"resource_usage"`
	DNSName        string                  `json:"dns_name"`
	IP             string                  `json:"ip"`
	Timestamp      int64                   `json:"timestamp"`
}
