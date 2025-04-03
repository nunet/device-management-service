// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
)

type SubnetAddPeerRequest struct {
	SubnetID string
	PeerID   string
	IP       string
}

type SubnetAddPeerResponse struct {
	OK    bool
	Error string
}

func (a *Allocation) handleSubnetAddPeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetAddPeerRequest
	resp := SubnetAddPeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AddSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_peer_added",
		"labels", []string{},
		"peerID", request.PeerID,
		"subnetID", request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
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

func (a *Allocation) handleSubnetAcceptPeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetAcceptPeerRequest
	resp := SubnetAcceptPeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AcceptSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_peer_accepted",
		"labels", []string{},
		"peerID", request.PeerID,
		"subnetID", request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
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

func (a *Allocation) handleSubnetMapPort(msg actor.Envelope) {
	defer msg.Discard()
	log.Debugw("handle_subnet_map_port_invoked", "from", msg.From)

	var request SubnetMapPortRequest
	resp := SubnetMapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("subnet_map_port_unmarshal_error",
			"labels", []string{},
			"error", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.MapPort(request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort)
	if err != nil {
		log.Debugw("subnet_map_port_error",
			"labels", []string{},
			"error", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_port_mapped",
		"labels", []string{},
		"sourcePort", request.SourcePort,
		"subnetID", request.SubnetID)
	resp.OK = true
	a.sendReply(msg, resp)
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

func (a *Allocation) handleSubnetDNSAddRecords(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetDNSAddRecordsRequest
	resp := SubnetDNSAddRecordsResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AddSubnetDNSRecords(request.SubnetID, request.Records)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_dns_records_added",
		"labels", []string{},
		"records", request.Records,
		"subnetID", request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
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

func (a *Allocation) handleSubnetUnmapPort(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetUnmapPortRequest
	resp := SubnetUnmapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.UnmapPort(
		request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort,
	)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_port_unmapped",
		"labels", []string{},
		"sourcePort", request.SourcePort,
		"subnetID", request.SubnetID)
	resp.OK = true
	a.sendReply(msg, resp)
}

type SubnetDNSRemoveRecordRequest struct {
	SubnetID   string
	DomainName string
}

type SubnetDNSRemoveRecordResponse struct {
	OK    bool
	Error string
}

func (a *Allocation) handleSubnetDNSRemoveRecord(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetDNSRemoveRecordRequest
	resp := SubnetDNSRemoveRecordResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.RemoveSubnetDNSRecord(request.SubnetID, request.DomainName)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_dns_record_removed",
		"labels", []string{},
		"domainName", request.DomainName,
		"subnetID", request.SubnetID)
	resp.OK = true
	a.sendReply(msg, resp)
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

func (a *Allocation) handleSubnetRemovePeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetRemovePeerRequest
	resp := SubnetRemovePeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.RemoveSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_peer_removed",
		"labels", []string{},
		"peerID", request.PeerID,
		"subnetID", request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
}
