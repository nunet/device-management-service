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

// DeploymentStateRequest requested by CPs to query the orchestrator about the state of a deployment
type DeploymentStateRequest struct {
	EnsembleID       string
	AllocationNamess []string
}

// DeploymentStateResponse is a reply by the orchestrator in response to CP request with
// Orchestrator replies OK=true if the CP and its allocations are still considered active for
// the deployment
//
// TODO: better to have well defined states in the future such as ACTIVE, STANDBY, REVERTED etc...
type DeploymentStateResponse struct {
	OK    bool
	Error string
}
