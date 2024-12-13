// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	EnsembleNamespace   = "/dms/ensemble/%s"
	AllocationNamespace = "/dms/allocation"
	NodeNamespace       = "/dms/node"
)

const (
	BidRequestTopic    = "/nunet/deployment"
	BidRequestBehavior = "/dms/deployment/request"
	BidRequestTimeout  = 5 * time.Second
	BidReplyBehavior   = "/dms/deployment/bid"

	VerifyEdgeConstraintBehavior = "/dms/deployment/constraint/edge"
	VerifyEdgeConstraintTimeout  = 5 * time.Second

	CommitDeploymentBehavior     = "/dms/deployment/commit"
	CommitDeploymentTimeout      = 3 * time.Second
	AllocationDeploymentBehavior = "/dms/deployment/allocate"
	AllocationDeploymentTimeout  = 5 * time.Second
	RevertDeploymentBehavior     = "/dms/deployment/revert"

	AllocationStartBehavior    = "/dms/allocation/start"
	AllocationRestartBehavior  = "/dms/allocation/restart"
	AllocationGetLogsBehavior  = "/dms/allocation/logs"
	AllocationStartTimeout     = 5 * time.Second
	AllocationStopBehavior     = "/dms/allocation/stop"
	AllocationStopTimeout      = 3 * time.Second
	AllocationShutdownBehavior = "/dms/allocation/shutdown"
	AllocationShutdownTimeout  = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second

	MaxBidMultiplier = 8
)

var (
	SubnetCreateBehavior = types.Behavior{
		DynamicTemplate: EnsembleNamespace + "/node/subnet/create",
		Static:          NodeNamespace + "/subnet/create",
	}
	SubnetDestroyBehavior = types.Behavior{
		DynamicTemplate: EnsembleNamespace + "/node/subnet/destroy",
		Static:          NodeNamespace + "/subnet/destroy",
	}

	RegisterHealthcheckBehavior = "/dms/actor/healthcheck/register"

	SubnetAddPeerBehavior         = AllocationNamespace + "/subnet/add-peer"
	SubnetRemovePeerBehavior      = AllocationNamespace + "/subnet/remove-peer"
	SubnetAcceptPeerBehavior      = AllocationNamespace + "/subnet/accept-peer"
	SubnetMapPortBehavior         = AllocationNamespace + "/subnet/map-port"
	SubnetUnmapPortBehavior       = AllocationNamespace + "/subnet/unmap-port"
	SubnetDNSAddRecordsBehavior   = AllocationNamespace + "/subnet/dns/add-records"
	SubnetDNSRemoveRecordBehavior = AllocationNamespace + "/subnet/dns/remove-record"
)

type VerifyEdgeConstraintRequest struct {
	EnsembleID string // the ensemble identifier
	S, T       string // the peer IDs of the edge S->T
	RTT        uint   //  maximum RTT in ms (if > 0)
	BW         uint   // minim BW in Kbps
}

type VerifyEdgeConstraintResponse struct {
	OK    bool
	Error string
}

type CommitDeploymentRequest struct {
	EnsembleID   string
	AllocationID string
	NodeID       string
	Resources    types.Resources
}

type CommitDeploymentResponse struct {
	OK    bool
	Error string
}

type AllocationDeploymentRequest struct {
	EnsembleID  string
	NodeID      string
	Allocations map[string]AllocationDeploymentConfig
}

type AllocationDeploymentConfig struct {
	Executor         AllocationExecutor
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
}

type AllocationDeploymentResponse struct {
	OK          bool
	Error       string
	Allocations map[string]actor.Handle
}

type RevertDeploymentMessage struct {
	EnsembleID     string
	AllocationsIDs []string
}

type AllocationStartRequest struct {
	SubnetIP    string
	PortMapping map[int]int
}

type AllocationStartResponse struct {
	OK    bool
	Error string
}

type AllocationGetLogsRequest struct{}

type AllocationGetLogsResponse struct {
	Data  []byte
	Error string
}

type AllocationShutdownRequest struct {
	AllocationID string
}

type AllocationShutdownResponse struct {
	OK    bool
	Error string
}

type AllocationRestartResponse struct {
	OK    bool
	Error string
}

type SubnetCreateRequest struct {
	SubnetID     string
	IP           string
	RoutingTable map[string]string
}

type SubnetCreateResponse struct {
	OK    bool
	Error string
}

type SubnetDestroyRequest struct {
	SubnetID string
}

type SubnetDestroyResponse struct {
	OK    bool
	Error string
}

type SubnetAddPeerRequest struct {
	SubnetID string
	PeerID   string
	IP       string
}

type SubnetAddPeerResponse struct {
	OK    bool
	Error string
}

type SubnetRemovePeerRequest struct {
	IP       string
	SubnetID string
	PeerID   string
}

type SubnetRemovePeerResponse struct {
	OK    bool
	Error string
}

type SubnetAcceptPeerRequest struct {
	SubnetID string
	PeerID   string
	IP       string
}

type SubnetAcceptPeerResponse struct {
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

type SubnetDNSAddRecordsRequest struct {
	SubnetID string
	// map of domain name:ip
	Records map[string]string
}

type SubnetDNSAddRecordsResponse struct {
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

type SubnetDNSRemoveRecordRequest struct {
	SubnetID   string
	DomainName string
}

type SubnetDNSRemoveRecordResponse struct {
	OK    bool
	Error string
}

type AllocationStopRequest struct {
	AllocationID string
}

type AllocationStopResponse struct {
	OK    bool
	Error string
}

type RegisterHealthcheckRequest struct {
	EnsembleID  string
	HealthCheck types.HealthCheckManifest
}

type RegisterHealthcheckResponse struct {
	OK    bool
	Error string
}
