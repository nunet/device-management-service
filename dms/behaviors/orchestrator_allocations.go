// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package behaviors

import (
	"gitlab.com/nunet/device-management-service/types"
)

// Behavior payloads for behaviors invoked between allocations and
// orchestrators

// TODO: keep here temporarily. We must organize types and behavior payloads.
// issue: https://gitlab.com/nunet/device-management-service/-/issues/893

// HealthCheckType represents the type of health check performed
type HealthCheckType string

const (
	HealthCheckTypeNone     HealthCheckType = "none"     // No healthcheck configured
	HealthCheckTypeSelf     HealthCheckType = "self"     // Allocation self-reported health
	HealthCheckTypeExecutor HealthCheckType = "executor" // Executor-level health check
)

type SubnetAddPeerRequest struct {
	SubnetID string
	PeerID   string
	IP       string
}

type SubnetAddPeerResponse struct {
	OK    bool
	Error string
}

type SubnetDNSAddRecordsRequest struct {
	SubnetID string
	// map of domain name:ip
	Records map[string]string
}

type SubnetDNSAddRecordsResponse struct {
	OK    bool
	Error string
}

type SubnetDNSRemoveRecordsRequest struct {
	SubnetID    string
	DomainNames []string
}

type SubnetDNSRemoveRecordsResponse struct {
	OK    bool
	Error string
}

type SubnetMapPortRequest struct {
	SubnetID   string
	Protocol   string
	SourceIP   string
	SourcePort string
	DestIP     string
	DestPort   string
}

type SubnetMapPortResponse struct {
	OK    bool
	Error string
}

type SubnetUnmapPortRequest struct {
	SubnetID   string
	Protocol   string
	SourceIP   string
	SourcePort string
	DestIP     string
	DestPort   string
}

type SubnetUnmapPortResponse struct {
	OK    bool
	Error string
}

type SubnetAcceptPeersRequest struct {
	SubnetID            string
	PartialRoutingTable map[string]string // ip -> peerID
}

type SubnetAcceptPeersResponse struct {
	OK    bool
	Error string
}

type SubnetRemovePeersRequest struct {
	SubnetID            string
	PartialRoutingTable map[string]string // ip -> peerID
}

type SubnetRemovePeersResponse struct {
	OK    bool
	Error string
}

type AllocationStartRequest struct {
	SubnetIP    string
	GatewayIP   string
	PortMapping map[int]int
}

type AllocationStartResponse struct {
	OK    bool
	Error string
}

type AllocationStatsRequest struct{}

type AllocationStatsResponse struct {
	OK    bool                 `json:"ok"`
	Error string               `json:"error,omitempty"`
	Stats *types.ExecutorStats `json:"stats,omitempty"`
}

type RegisterHealthcheckRequest struct {
	EnsembleID  string
	HealthCheck types.HealthCheckManifest
}

type RegisterHealthcheckResponse struct {
	OK    bool
	Error string
}

type HealthCheckResponse struct {
	OK    bool
	Error string
}

type TaskTerminationNotification struct {
	AllocationID string `json:"allocation_id"`
	Status       string `json:"status"`

	Error TerminationError `json:"error"`

	Stdout []byte `json:"stdout"`
	Stderr []byte `json:"stderr"`
}

// TerminationError holds information necessary to handle
// failure recovery given retry policies.
type TerminationError struct {
	Err      string `json:"err"`
	ExitCode int    `json:"exit_code"`
	// Killed is used to identify if the application was killed
	// by external means, rather than app exiting itself
	Killed bool `json:"killed"`
}

type AllocationRestartResponse struct {
	OK    bool
	Error string
}

// AllocationLivenessNotification is sent periodically by service allocations
// to report their health and status to the orchestrator
// NOTE: This is for observability only, pull-based healthchecks remain authoritative
type AllocationLivenessNotification struct {
	AllocationID   string `json:"allocation_id"`
	Status         string `json:"status"`          // Current allocation status
	Timestamp      int64  `json:"timestamp"`       // Unix timestamp of report
	SequenceNumber int64  `json:"sequence_number"` // Monotonic counter to detect missed heartbeats

	// Health information (self-reported)
	Health HealthStatus `json:"health"`

	// Optional: Resource usage metrics
	ResourceUsage *AllocationResourceUsage `json:"resource_usage,omitempty"`

	Version string `json:"version"` // Protocol version for future compatibility
}

// AllocationResourceUsage contains resource metrics from the executor
type AllocationResourceUsage struct {
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	MemoryUsedBytes  int64   `json:"memory_used_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
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
