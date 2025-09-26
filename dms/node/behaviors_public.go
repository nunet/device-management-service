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
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/sys"
)

type HelloResponse struct {
	DID did.DID
}

type PublicStatusResponse struct {
	Status    string
	Resources types.Resources
}

type NetworkInterfaces struct {
	Name       string   `json:"name"`
	IP         []string `json:"ip"`
	MacAddress string   `json:"mac_address"`
}

type DiscoveryStatus struct {
	Hostname string `json:"hostname"`
	DID      string `json:"did"`
	Network  struct {
		Interfaces []NetworkInterfaces `json:"interfaces"`
		P2P        types.NetworkStats  `json:"p2p"`
	} `json:"network"`
	Onboarded bool `json:"onboarded"`
	Resources struct {
		Total     types.Resources `json:"total"`
		Onboarded types.Resources `json:"onboarded"`
		Allocated types.Resources `json:"allocated"`
		Free      types.Resources `json:"free"`
	} `json:"resources"`
	Config config.Config `json:"config"`

	Errors []string `json:"errors"`
}

type DiscoveryStatusResponse map[string]DiscoveryStatus

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

	n.lock.Lock()
	if st, ok := n.peers[p]; ok {
		st.helloIn = true
	} else if n.network.PeerConnected(p) {
		// rance with connected notification
		st = &peerState{helloIn: true}
		n.peers[p] = st
	}
	n.lock.Unlock()

	n.handleHello(msg)
}

func (n *Node) handleBroadcastHelloBehavior(msg actor.Envelope) {
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

func (n *Node) handleStatusDiscoveryBehavior(msg actor.Envelope) {
	defer msg.Discard()

	var err error

	resp := make(DiscoveryStatusResponse, 0)

	// TODO peerIDs for peers under the controller
	// only self peer for now
	for _, peerID := range []string{n.network.GetHostID().String()} {
		peerInfo := DiscoveryStatus{}

		collectErrors := func(msg string, err error) {
			if err != nil {
				peerInfo.Errors = append(peerInfo.Errors, fmt.Sprintf("%s: %v", msg, err))
			}
		}

		peerInfo.Hostname, err = os.Hostname()
		collectErrors("error getting hostname", err)

		peerInfo.DID = n.actor.Security().DID().String()

		ifaceList, err := sys.GetNetInterfaces()
		collectErrors("error getting network interfaces", err)

		for _, iface := range ifaceList {
			netIface := NetworkInterfaces{}
			netIface.Name = iface.Name
			netIface.MacAddress = iface.HardwareAddr.String()
			ip, err := iface.Addrs()
			collectErrors(fmt.Sprintf("error getting addrs for iface %s", iface.Name), err)
			for _, addr := range ip {
				netIface.IP = append(netIface.IP, addr.String())
			}
			peerInfo.Network.Interfaces = append(peerInfo.Network.Interfaces, netIface)
		}

		peerInfo.Network.P2P = n.network.Stat()

		if n.hardware != nil {
			totalResrc, err := n.hardware.GetMachineResources()
			collectErrors("error getting total machine resource", err)
			peerInfo.Resources.Total = totalResrc.Resources
		} else {
			collectErrors("error getting hardware info", fmt.Errorf("node hardware manager not set"))
		}

		if n.onboarding != nil {
			peerInfo.Onboarded = n.onboarding.Config.IsOnboarded
			if peerInfo.Onboarded {
				peerInfo.Resources.Onboarded = n.onboarding.Config.OnboardedResources

				peerInfo.Resources.Allocated, err = n.resourceManager.GetTotalAllocation()
				collectErrors("error getting allocated resources", err)

				freeResources, err := n.resourceManager.GetFreeResources(context.Background())
				collectErrors("error getting free resources", err)
				peerInfo.Resources.Free = freeResources.Resources
			}
		} else {
			collectErrors("error getting onboarding info", fmt.Errorf("node onboarding not set"))
		}

		peerInfo.Config = n.dmsConfig

		resp[peerID] = peerInfo
	}

	n.sendReply(msg, resp)
}
