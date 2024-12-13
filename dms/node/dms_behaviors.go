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
	"fmt"
	"path/filepath"
	"time"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
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

	OnboardBehavior       = "/dms/node/onboarding/onboard"
	OffboardBehavior      = "/dms/node/onboarding/offboard"
	OnboardStatusBehavior = "/dms/node/onboarding/status"

	ContainerStartBehavior = "/dms/node/container/start"
	ContainerStopBehavior  = "/dms/node/container/stop"
	ContainerListBehavior  = "/dms/node/container/list"

	VMStartBehavior = "/dms/node/vm/start/custom"
	VMStopBehavior  = "/dms/node/vm/stop"
	VMListBehavior  = "/dms/node/vm/list"

	DeploymentListBehavior     = "/dms/node/deployment/list"
	DeploymentStatusBehavior   = "/dms/node/deployment/status"
	DeploymentLogsBehavior     = "/dms/node/deployment/logs"
	DeploymentManifestBehavior = "/dms/node/deployment/manifest"
	DeploymentShutdownBehavior = "/dms/node/deployment/shutdown"

	ResourcesAllocatedBehavior = "/dms/node/resources/allocated"
	ResourcesFreeBehavior      = "/dms/node/resources/free"
	ResourcesOnboardedBehavior = "/dms/node/resources/onboarded"

	HardwareSpecBehavior  = "/dms/node/hardware/spec"
	HardwareUsageBehavior = "/dms/node/hardware/usage"

	LoggerConfigBehavior      = "/dms/node/logger/config"
	RestartAllocationBehavior = "/dms/node/allocation/restart"
	StopAllocationBehavior    = "/dms/node/allocation/stop"

	CapListBehavior   = "/dms/cap/list"
	CapAnchorBehavior = "/dms/cap/anchor"

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
	NoGPU  bool
	GPUs   string
	Config types.OnboardingConfig
}

type OnboardResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Config  types.OnboardingConfig `json:"config,omitempty"`
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

	config, err := n.onboarder.Onboard(context.Background(), request.Config)
	if err != nil {
		resp.Error = err.Error()
		resp.Success = false
		n.sendReply(msg, resp)
		return
	}

	resp.Config = config
	resp.Success = true
	n.sendReply(msg, resp)
}

type OffboardRequest struct{}

type OffboardResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (n *Node) handleOffboard(msg actor.Envelope) {
	defer msg.Discard()

	resp := OffboardResponse{}
	if err := n.onboarder.Offboard(context.Background()); err != nil {
		resp.Success = false
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Success = true
	n.sendReply(msg, resp)
}

type OnboardStatusResponse struct {
	Onboarded bool   `json:"onboarded"`
	Error     string `json:"error,omitempty"`
}

func (n *Node) handleOnboardStatus(msg actor.Envelope) {
	defer msg.Discard()
	resp := OnboardStatusResponse{}

	onboarded, err := n.onboarder.IsOnboarded()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Onboarded = onboarded
	n.sendReply(msg, resp)
}

type DeploymentListResponse struct {
	Deployments map[string]string
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	var resp DeploymentListResponse

	resp.Deployments = make(map[string]string)
	for ID, dep := range n.deployments {
		resp.Deployments[ID] = job_types.DeploymentStatusString(dep.Status())
	}

	n.sendReply(msg, resp)
}

type DeploymentLogsRequest struct {
	EnsembleID     string
	AllocationName string
}

type DeploymentLogsResponse struct {
	LogsWrittenTo string
	Error         string
}

func (n *Node) handleDeploymentLogs(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.EnsembleID]
	n.mx.Unlock()
	if !ok {
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	data, err := d.GetAllocationLogs(request.AllocationName)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	ensembleDir := filepath.Join(
		n.dmsConfig.WorkDir,
		"deployments",
		request.EnsembleID,
	)

	err = n.fs.MkdirAll(ensembleDir, 0o744)
	if err != nil {
		resp.Error = fmt.Sprintf("failed to create ensemble directory %s: %s", ensembleDir, err.Error())
		n.sendReply(msg, resp)
		return
	}

	writeLogsTo := filepath.Join(ensembleDir, fmt.Sprintf("%s.logs", request.AllocationName))
	err = n.fs.WriteFile(writeLogsTo, data, 0o644)
	if err != nil {
		resp.Error = fmt.Sprintf("failed to write logs to %s: %s", writeLogsTo, err.Error())
		n.sendReply(msg, resp)
		return
	}

	resp.LogsWrittenTo = writeLogsTo
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

	resp.Status = job_types.DeploymentStatusString(d.Status())
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
	OK    bool
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
		log.Debugf("deployment %q is not running(status=%q), cannot shutdown", request.ID, d.Status())
		// maybe-TODO: if it's still provisioning/committing,
		// we should stop the deployment process anyway
		resp.Error = ErrDeploymentNotRunning.Error()
		n.sendReply(msg, resp)
		return
	}

	d.Shutdown()
	resp.OK = true
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
		executionType = job_types.ExecutorFirecracker
	} else if request.Execution.EngineSpec.IsType(types.ExecutorTypeDocker.String()) {
		executionType = job_types.ExecutorDocker
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

func (n *Node) handleSubnetCreate(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetCreateRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.SubnetCreateResponse{}
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
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.SubnetDestroyResponse{}
	err := n.network.DestroySubnet(request.SubnetID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleAllocationDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.AllocationDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.AllocationDeploymentResponse{}
	if err := n.registerDynamicBehaviors(request.EnsembleID); err != nil {
		err = fmt.Errorf("failed to register dynamic behaviors: %w", err)
		log.Error(err)
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	allocations, err := n.createAllocations(
		msg.From.DID,
		request.EnsembleID,
		request.Allocations,
		msg.From,
	)
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
	allocationID := request.EnsembleID + "_" + request.AllocationName
	err := n.commitDeployment(request.EnsembleID, allocationID, request.Resources)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

type LoggerConfigRequest struct {
	Interval       int    `json:"interval,omitempty"`
	URL            string `json:"url,omitempty"`
	Level          string `json:"level,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
	APMURL         string `json:"apm_url,omitempty"`
	ElasticEnabled *bool  `json:"elastic_enabled,omitempty"`
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
	if req.APIKey != "" { // Handle API Key
		if err := observability.SetAPIKey(req.APIKey); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.APMURL != "" { // Handle APM URL
		if err := observability.SetAPMURL(req.APMURL); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.ElasticEnabled != nil { // Handle Elasticsearch Enabled
		if err := observability.EnableElasticsearchLogging(*req.ElasticEnabled); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

type resourcesResponse struct {
	OK        bool
	Resources types.Resources
	Error     string `json:"error,omitempty"`
}

func (n *Node) handleAllocatedResources(msg actor.Envelope) {
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

func (n *Node) handleFreeResources(msg actor.Envelope) {
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

func (n *Node) handleOnboardedResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	onboardedResources, err := n.resourceManager.GetOnboardedResources(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = onboardedResources.Resources
	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleHardwareUsage(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	hardwareUsage, err := n.hardware.GetUsage()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = hardwareUsage
	n.sendReply(msg, resp)
}

type CapListRequest struct {
	Context string
}

type CapListResponse struct {
	OK      bool
	Error   string
	Roots   []did.DID
	Require ucan.TokenList
	Provide ucan.TokenList
	Revoke  ucan.TokenList
}

func (n *Node) handleCapList(msg actor.Envelope) {
	defer msg.Discard()
	var request CapListRequest
	resp := CapListResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	roots, require, provide, revoke := n.rootCap.ListRoots()
	resp.OK = true
	resp.Roots = roots
	resp.Require = require
	resp.Provide = provide
	resp.Revoke = revoke
	n.sendReply(msg, resp)
}

type CapAnchorRequest struct {
	Root    []did.DID
	Require ucan.TokenList
	Provide ucan.TokenList
	Revoke  ucan.TokenList
}

type CapAnchorResponse struct {
	OK    bool
	Error string
}

func (n *Node) handleCapAnchor(msg actor.Envelope) {
	defer msg.Discard()
	var request CapAnchorRequest
	resp := CapAnchorResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if err := n.rootCap.AddRoots(nil, request.Require, request.Provide, request.Revoke); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	cfg := config.GetConfig()
	if err := SaveCapabilityContext(n.rootCap, cfg); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}
