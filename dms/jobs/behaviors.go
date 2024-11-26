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
	AllocationStartBehavior      = "/dms/deployment/start"
	AllocationGetLogsBehavior    = "/dms/deployment/logs"
	AllocationStartTimeout       = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second

	MaxBidMultiplier = 8

	SubnetCreateBehavior          = "/dms/node/subnet/create"
	SubnetDestroyBehavior         = "/dms/node/subnet/destroy"
	SubnetAddPeerBehavior         = "/dms/node/subnet/add-peer"
	SubnetRemovePeerBehavior      = "/dms/node/subnet/remove-peer"
	SubnetAcceptPeerBehavior      = "/dms/node/subnet/accept-peer"
	SubnetMapPortBehavior         = "/dms/node/subnet/map-port"
	SubnetUnmapPortBehavior       = "/dms/node/subnet/unmap-port"
	SubnetDNSAddRecordBehavior    = "/dms/node/subnet/dns/add-record"
	SubnetDNSRemoveRecordBehavior = "/dms/node/subnet/dns/remove-record"
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
	EnsembleID string
	NodeID     string
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

type AllocationStartRequest struct{}

type AllocationStartResponse struct {
	OK    bool
	Error string
}

type AllocationGetLogsRequest struct{}

type AllocationGetLogsResponse struct {
	Data  []byte
	Error string
}

type RestartAllocationRequest struct {
	AllocationID string
}

type RestartAllocationResponse struct {
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

type SubnetDNSAddRecordRequest struct {
	SubnetID   string
	DomainName string
	IP         string
}

type SubnetDNSAddRecordResponse struct {
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
