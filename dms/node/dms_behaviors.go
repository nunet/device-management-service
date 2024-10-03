package node

import (
	"context"
	"encoding/json"
	"time"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	PeersListBehavior       = "/dms/node/peers/list"
	PeerAddrInfoBehavior    = "/dms/node/peers/self"
	PeerPingBehavior        = "/dms/node/peers/ping"
	PeerDHTBehavior         = "/dms/node/peers/dht"
	PeerConnectBehavior     = "/dms/node/peers/connect"
	PeerScoreBehavior       = "/dms/node/peers/score"
	OnboardBehavior         = "/dms/node/onboarding/onboard"
	OffboardBehavior        = "/dms/node/onboarding/offboard"
	OnboardStatusBehavior   = "/dms/node/onboarding/status"
	OnboardResourceBehavior = "/dms/node/onboarding/resource"
	CustomVMStartBehavior   = "/dms/node/vm/start/custom"
	VMStopBehavior          = "/dms/node/vm/stop"
	VMListBehavior          = "/dms/node/vm/list"

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
	Config types.CapacityForNunet
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

	onboardResult, err := n.onboarder.Onboard(context.Background(), request.Config)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Config = *onboardResult
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
	Config types.CapacityForNunet
}

type OnboardResourceResponse struct {
	Error  string
	Result types.OnboardingConfig
}

func (n *Node) handleOnboardResource(msg actor.Envelope) {
	defer msg.Discard()

	var request OnboardResourceRequest

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := OnboardResourceResponse{}

	result, err := n.onboarder.ResourceConfig(context.Background(), request.Config)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Result = *result

	n.sendReply(msg, resp)
}

type CustomVMStartRequest struct {
	Execution types.ExecutionRequest
}

type CustomVMStartResponse struct {
	Error string
}

func (n *Node) handleCustomVMStart(msg actor.Envelope) {
	defer msg.Discard()

	var request CustomVMStartRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := CustomVMStartResponse{}

	err := n.executor.Start(context.Background(), &request.Execution)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.sendReply(msg, resp)
}

type VMStopRequest struct {
	ExecutionID string
}

type VMStopResponse struct {
	Error string
}

func (n *Node) handleVMStop(msg actor.Envelope) {
	defer msg.Discard()

	var request VMStopRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		// TODO log
		return
	}

	resp := VMStopResponse{}

	err := n.executor.Cancel(context.Background(), request.ExecutionID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.sendReply(msg, resp)
}

type ListVMResponse struct {
	Error string
	VMS   []types.ExecutionListItem
}

func (n *Node) handleListVM(msg actor.Envelope) {
	defer msg.Discard()

	resp := ListVMResponse{
		VMS: n.executor.List(),
	}

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
