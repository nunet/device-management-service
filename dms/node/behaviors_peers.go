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
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

const pingTimeout = 1 * time.Second

type PingRequest struct {
	Host string
}

type PingResponse struct {
	Error string
	RTT   int64
}

func (n *Node) handlePeerPing(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("peer_ping_error", "error", err)
		n.sendReply(msg, PingResponse{Error: err.Error()})
	}

	var request PingRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("peer_ping_unmarshal_error", "error", err)
		return
	}

	resp := PingResponse{}

	res, err := n.network.Ping(context.Background(), request.Host, pingTimeout)
	if err != nil {
		handleErr(err)
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
		log.Debug("peers_list_not_libp2p_network")
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
		log.Debug("peer_dht_not_libp2p_network")
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

	handleErr := func(err error) {
		log.Errorw("peer_connect_error", "error", err)
		n.sendReply(msg, PeerConnectResponse{Error: err.Error()})
	}

	var request PeerConnectRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("peer_connect_unmarshal_error", "error", err)
		return
	}

	libp2pNet, ok := n.network.(*libp2p.Libp2p)
	if !ok {
		log.Debug("peer_connect_not_libp2p_network")
		return
	}

	resp := PeerConnectResponse{}

	peerAddr, err := multiaddr.NewMultiaddr(request.Address)
	if err != nil {
		handleErr(err)
		return
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		handleErr(err)
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
