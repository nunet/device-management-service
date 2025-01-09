// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package behaviors

import "gitlab.com/nunet/device-management-service/types"

const (
	PeerPingBehavior     = "/dms/node/peers/ping"
	PeersListBehavior    = "/dms/node/peers/list"
	PeerAddrInfoBehavior = "/dms/node/peers/self"
	PeerDHTBehavior      = "/dms/node/peers/dht"
	PeerConnectBehavior  = "/dms/node/peers/connect"
	PeerScoreBehavior    = "/dms/node/peers/score"

	OnboardBehavior       = "/dms/node/onboarding/onboard"
	OffboardBehavior      = "/dms/node/onboarding/offboard"
	OnboardStatusBehavior = "/dms/node/onboarding/status"

	BidRequestTopic              = "/nunet/deployment"
	BidRequestBehavior           = "/dms/deployment/request"
	BidReplyBehavior             = "/dms/deployment/bid"
	VerifyEdgeConstraintBehavior = "/dms/deployment/constraint/edge"
	CommitDeploymentBehavior     = "/dms/deployment/commit"
	AllocationDeploymentBehavior = "/dms/deployment/allocate"
	RevertDeploymentBehavior     = "/dms/deployment/revert"

	NewDeploymentBehavior      = "/dms/node/deployment/new"
	DeploymentListBehavior     = "/dms/node/deployment/list"
	DeploymentLogsBehavior     = "/dms/node/deployment/logs"
	DeploymentStatusBehavior   = "/dms/node/deployment/status"
	DeploymentManifestBehavior = "/dms/node/deployment/manifest"
	DeploymentShutdownBehavior = "/dms/node/deployment/shutdown"

	ResourcesAllocatedBehavior = "/dms/node/resources/allocated"
	ResourcesFreeBehavior      = "/dms/node/resources/free"
	ResourcesOnboardedBehavior = "/dms/node/resources/onboarded"
	HardwareSpecBehavior       = "/dms/node/hardware/spec"
	HardwareUsageBehavior      = "/dms/node/hardware/usage"

	LoggerConfigBehavior = "/dms/node/logger/config"

	CapListBehavior   = "/dms/cap/list"
	CapAnchorBehavior = "/dms/cap/anchor"

	PublicHelloBehavior    = "/public/hello"
	PublicStatusBehavior   = "/public/status"
	BroadcastHelloBehavior = "/broadcast/hello"
	BroadcastHelloTopic    = "/nunet/hello"

	EnsembleNamespace   = "/dms/ensemble/%s"
	AllocationNamespace = "/dms/allocation"
	NodeNamespace       = "/dms/node"

	AllocationStartBehavior     = "/dms/allocation/start"
	AllocationRestartBehavior   = "/dms/allocation/restart"
	RegisterHealthcheckBehavior = "/dms/actor/healthcheck/register"

	SubnetAddPeerBehavior         = AllocationNamespace + "/subnet/add-peer"
	SubnetRemovePeerBehavior      = AllocationNamespace + "/subnet/remove-peer"
	SubnetAcceptPeerBehavior      = AllocationNamespace + "/subnet/accept-peer"
	SubnetMapPortBehavior         = AllocationNamespace + "/subnet/map-port"
	SubnetUnmapPortBehavior       = AllocationNamespace + "/subnet/unmap-port"
	SubnetDNSAddRecordsBehavior   = AllocationNamespace + "/subnet/dns/add-records"
	SubnetDNSRemoveRecordBehavior = AllocationNamespace + "/subnet/dns/remove-record"

	AllocationLogsBehavior     = EnsembleNamespace + "/allocation/logs"
	AllocationShutdownBehavior = EnsembleNamespace + "/allocation/shutdown"
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
)
