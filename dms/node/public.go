// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	PublicHelloBehavior    = "/public/hello"
	PublicStatusBehavior   = "/public/status"
	BroadcastHelloBehavior = "/broadcast/hello"
	BroadcastHelloTopic    = "/nunet/hello"
)

type HelloResponse struct {
	DID did.DID
}

type PublicStatusResponse struct {
	Status    string
	Resources types.Resources
}

func (n *Node) publicHelloBehavior(msg actor.Envelope) {
	pubk, err := did.PublicKeyFromDID(msg.From.DID)
	if err != nil {
		log.Debugf("failed to extract public key from DID: %s", err)
		return
	}

	p, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		log.Debugf("failed to extract peer ID from public key: %s", err)
		return
	}

	n.mx.Lock()
	if st, ok := n.peers[p]; ok {
		st.helloIn = true
	} else if n.network.PeerConnected(p) {
		// rance with connected notification
		st = &peerState{helloIn: true}
		n.peers[p] = st
	}
	n.mx.Unlock()

	n.handleHello(msg)
}

func (n *Node) broadcastHelloBehavior(msg actor.Envelope) {
	n.handleHello(msg)
}

func (n *Node) handleHello(msg actor.Envelope) {
	defer msg.Discard()
	log.Debugf("hello from %s", msg.From.Address.HostID)

	resp := HelloResponse{DID: n.actor.Security().DID()}
	n.sendReply(msg, resp)
}

func (n *Node) publicStatusBehavior(msg actor.Envelope) {
	defer msg.Discard()

	var resp PublicStatusResponse
	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		resp.Status = "ERROR"
	} else {
		resp.Status = "OK"
		resp.Resources = machineResources.Resources
	}

	n.sendReply(msg, resp)
}
