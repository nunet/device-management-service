// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"encoding/json"
	"time"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	PeersListBehavior    = "/dms/node/peers/list"
	PeerAddrInfoBehavior = "/dms/node/peers/self"
	PeerPingBehavior     = "/dms/node/peers/ping"
	PeerDHTBehavior      = "/dms/node/peers/dht"
	PeerConnectBehavior  = "/dms/node/peers/connect"
	PeerScoreBehavior    = "/dms/node/peers/score"

	OnboardBehavior         = "/dms/node/onboarding/onboard"
	OffboardBehavior        = "/dms/node/onboarding/offboard"
	OnboardStatusBehavior   = "/dms/node/onboarding/status"
	OnboardResourceBehavior = "/dms/node/onboarding/resource"

	ContainerStartBehavior = "/dms/node/container/start"
	ContainerStopBehavior  = "/dms/node/container/stop"
	ContainerListBehavior  = "/dms/node/container/list"

	VMStartBehavior = "/dms/node/vm/start/custom"
	VMStopBehavior  = "/dms/node/vm/stop"
	VMListBehavior  = "/dms/node/vm/list"

	DeploymentListBehavior     = "/dms/node/deployment/list"
	DeploymentStatusBehavior   = "/dms/node/deployment/status"
	DeploymentManifestBehavior = "/dms/node/deployment/manifest"
	DeploymentShutdownBehavior = "/dms/node/deployment/shutdown"

	AllocatedResourcesBehavior = "/dms/node/resources/allocated"
	FreeResourcesBehavior      = "/dms/node/resources/free"
	OnboardedResourcesBehavior = "/dms/node/resources/onboarded"

	LoggerConfigBehavior = "/dms/node/logger/config"

	pingTimeout = 1 * time.Second
)

type PingRequest struct {
	Host string
}

type PingResponse struct {
	Error string
	RTT   int64
}

func (n *Node) handlePeerPing(msg actor.Envelope) {
	defer msg.Discard()

	var request PingRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := PingResponse{}

	res, err := n.network.Ping(context.Background(), request.Host, pingTimeout)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if res.Error != nil {
		resp.Error = res.Error.Error()
	}
	resp.RTT = res.RTT.Milliseconds()
	n.sendReply(msg, resp)
}

type PeersListResponse struct {
	Peers []peer.ID
}

func (n *Node) handlePeersList(msg actor.Envelope) {
	defer msg.Discard()

	// get the underlying libp2p instance and extract the DHT data
	libp2pNet, ok := n.network.(*libp2p.Libp2p)
	if !ok {
		// TODO log
		return
	}

	resp := PeersListResponse{
		Peers: make([]peer.ID, 0),
	}

	for _, v := range libp2pNet.PS.Peers() {
		resp.Peers = append(resp.Peers, v)
	}

	n.sendReply(msg, resp)
}

type PeerAddrInfoResponse struct {
	ID      string `json:"id"`
	Address string `json:"listen_addr"`
}

func (n *Node) handlePeerAddrInfo(msg actor.Envelope) {
	defer msg.Discard()

	stats := n.network.Stat()
	resp := PeerAddrInfoResponse{
		ID:      stats.ID,
		Address: stats.ListenAddr,
	}

	n.sendReply(msg, resp)
}

type PeerDHTResponse struct {
	Peers []kbucket.PeerInfo
}

func (n *Node) handlePeerDHT(msg actor.Envelope) {
	defer msg.Discard()

	// get the underlying libp2p instance and extract the DHT data
	libp2pNet, ok := n.network.(*libp2p.Libp2p)
	if !ok {
		// TODO log
		return
	}

	resp := PeerDHTResponse{
		Peers: libp2pNet.DHT.RoutingTable().GetPeerInfos(),
	}

	n.sendReply(msg, resp)
}

type PeerConnectRequest struct {
	Address string
}

type PeerConnectResponse struct {
	Status string
	Error  string
}

func (n *Node) handlePeerConnect(msg actor.Envelope) {
	defer msg.Discard()

	var request PeerConnectRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	libp2pNet, ok := n.network.(*libp2p.Libp2p)
	if !ok {
		// TODO log
		return
	}

	resp := PeerConnectResponse{}

	peerAddr, err := multiaddr.NewMultiaddr(request.Address)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if err := libp2pNet.Host.Connect(context.Background(), *addrInfo); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Status = "CONNECTED"
	n.sendReply(msg, resp)
}

type OnboardRequest struct {
	Config types.OnboardingConfig
}

type OnboardResponse struct {
	Error  string
	Config types.OnboardingConfig
}

func (n *Node) handleOnboard(msg actor.Envelope) {
	defer msg.Discard()

	resp := OnboardResponse{}

	var request OnboardRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}
	request.Config.MachineResources = machineResources.Resources

	if err := n.onboarder.Onboard(context.Background(), request.Config); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Config = request.Config
	n.sendReply(msg, resp)
}

type OffboardRequest struct {
	Force bool
}

type OffboardResponse struct {
	Success bool
}

func (n *Node) handleOffboard(msg actor.Envelope) {
	defer msg.Discard()

	var request OffboardRequest

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := OffboardResponse{}
	if err := n.onboarder.Offboard(context.Background(), request.Force); err != nil {
		resp.Success = false
		n.sendReply(msg, resp)
		return
	}

	resp.Success = true
	n.sendReply(msg, resp)
}

type OnboardStatusResponse struct {
	Onboarded bool
	Error     string
}

func (n *Node) handleOnboardStatus(msg actor.Envelope) {
	defer msg.Discard()
	resp := OnboardStatusResponse{}

	onboarded, err := n.onboarder.IsOnboarded(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Onboarded = onboarded
	n.sendReply(msg, resp)
}

type OnboardResourceRequest struct {
	Config types.OnboardingConfig
}

type OnboardResourceResponse struct {
	Error  string
	Result types.OnboardingConfig
}

type DeploymentListResponse struct {
	Deployments map[string]string
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	var resp DeploymentListResponse

	resp.Deployments = make(map[string]string)
	for ID, dep := range n.deployments {
		resp.Deployments[ID] = jobs.DeploymentStatusString(dep.Status())
	}

	n.sendReply(msg, resp)
}

type DeploymentStatusRequest struct {
	ID string
}

type DeploymentStatusResponse struct {
	Status string
	Error  string
}

func (n *Node) handleDeploymentStatus(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentStatusRequest
	var resp DeploymentStatusResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.ID]
	n.mx.Unlock()
	if !ok {
		// TODO: check database for persisted deployments data
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Status = jobs.DeploymentStatusString(d.Status())
	n.sendReply(msg, resp)
}

type DeploymentManifestRequest struct {
	ID string
}

type DeploymentManifestResponse struct {
	Manifest jobs.EnsembleManifest
	Error    string
}

func (n *Node) handleDeploymentManifest(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentManifestRequest
	var resp DeploymentManifestResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.ID]
	n.mx.Unlock()
	if !ok {
		// TODO: check database for persisted deployments data
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Manifest = d.Manifest()
	n.sendReply(msg, resp)
}

type DeploymentShutdownRequest struct {
	ID string
}

type DeploymentShutdownResponse struct {
	Error string
}

func (n *Node) handleDeploymentShutdown(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentShutdownRequest
	var resp DeploymentShutdownResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	d, ok := n.deployments[request.ID]
	if !ok {
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	if d.Status() != jobs.DeploymentStatusRunning {
		// maybe-TODO: if it's still provisioning/committing,
		// we should stop the deployment process anyway
		resp.Error = ErrDeploymentNotRunning.Error()
		n.sendReply(msg, resp)
		return
	}

	err := d.Shutdown()
	if err != nil {
		resp.Error = err.Error()
	}

	n.sendReply(msg, resp)
}

type CustomVMStartRequest struct {
	Execution types.ExecutionRequest
}

type CustomVMStartResponse struct {
	Error string
}

func (n *Node) handleVMContainerStart(msg actor.Envelope) {
	defer msg.Discard()

	var request CustomVMStartRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	var executionType jobs.AllocationExecutor
	if request.Execution.EngineSpec.IsType(types.ExecutorTypeFirecracker.String()) {
		executionType = jobs.ExecutorFirecracker
	} else if request.Execution.EngineSpec.IsType(types.ExecutorTypeDocker.String()) {
		executionType = jobs.ExecutorDocker
	}

	resp := CustomVMStartResponse{}

	e, err := n.getExecutor(executionType)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err = e.executor.Start(context.Background(), &request.Execution)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.sendReply(msg, resp)
}

type VMStopRequest struct {
	ExecutionID   string
	ExecutionType jobs.AllocationExecutor
}

type VMStopResponse struct {
	Error string
}

func (n *Node) handleVMContainerStop(msg actor.Envelope) {
	defer msg.Discard()

	var request VMStopRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := VMStopResponse{}

	e, err := n.getExecutor(request.ExecutionType)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err = e.executor.Cancel(context.Background(), request.ExecutionID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.sendReply(msg, resp)
}

type ListVMResponse struct {
	Error         string
	VMS           []types.ExecutionListItem
	ExecutionType jobs.AllocationExecutor
}

func (n *Node) handleVMContainerList(msg actor.Envelope) {
	defer msg.Discard()

	resp := ListVMResponse{}

	e, err := n.getExecutor(resp.ExecutionType)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.VMS = e.executor.List()

	n.sendReply(msg, resp)
}

type PeerScoreResponse struct {
	Score map[string]*network.PeerScoreSnapshot
}

func (n *Node) handlePeerScore(msg actor.Envelope) {
	defer msg.Discard()

	resp := PeerScoreResponse{Score: make(map[string]*network.PeerScoreSnapshot)}
	snapshot := n.network.GetBroadcastScore()
	for p, score := range snapshot {
		resp.Score[p.String()] = score
	}
	n.sendReply(msg, resp)
}

func (n *Node) handleAllocationDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.AllocationDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.AllocationDeploymentResponse{}
	allocations, err := n.createAllocations(msg.From.DID, request.EnsembleID, request.NodeID, request.Allocations)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	resp.Allocations = allocations
	n.sendReply(msg, resp)
}

func (n *Node) handleCommitDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.CommitDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.CommitDeploymentResponse{}
	err := n.commitDeployment(request.EnsembleID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleSubnetCreate(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetCreateRequest
	resp := jobs.SubnetCreateResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err := n.network.CreateSubnet(context.Background(), request.SubnetID, request.RoutingTable)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleSubnetDestroy(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetDestroyRequest
	resp := jobs.SubnetDestroyResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	err := n.network.DestroySubnet(request.SubnetID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

type LoggerConfigRequest struct {
	Interval int
	URL      string
	Level    string
}

type LoggerConfigResponse struct {
	Error string `json:"error,omitempty"`
	OK    bool
}

func (n *Node) handleLoggerConfig(msg actor.Envelope) {
	defer msg.Discard()

	var (
		req  LoggerConfigRequest
		resp LoggerConfigResponse
	)

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if req.Interval != 0 {
		if err := observability.SetFlushInterval(req.Interval); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.Level != "" {
		if err := observability.SetLogLevel(req.Level); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.URL != "" {
		if err := observability.SetElasticsearchEndpoint(req.URL); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	resp.OK = true
	n.sendReply(msg, resp)
}

type resourcesResponse struct {
	Resources types.Resources
	Error     string
}

func (n *Node) getAllocatedResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	allocatedResources, err := n.resourceManager.GetTotalAllocation()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = allocatedResources
	n.sendReply(msg, resp)
}

func (n *Node) getFreeResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	freeResources, err := n.resourceManager.GetFreeResources(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = freeResources.Resources
	n.sendReply(msg, resp)
}

func (n *Node) getOnboardedResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	onboardedResources, err := n.resourceManager.GetOnboardedResources(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = onboardedResources.Resources
	n.sendReply(msg, resp)
}
