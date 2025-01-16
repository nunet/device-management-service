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

	log.Debugf("added peer: %q to subnet: %q", request.PeerID, request.SubnetID)

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

	log.Debugf("accepted peer: %q to subnet: %q", request.PeerID, request.SubnetID)

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
	log.Debugf("behavior handleSubnetMapPort invoked by: %+v", msg.From)

	var request SubnetMapPortRequest
	resp := SubnetMapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("error unmarshalling subnet map port request: %s", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.MapPort(request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort)
	if err != nil {
		log.Debugf("error mapping port: %s", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugf("mapped port: %d to subnet: %q", request.SourcePort, request.SubnetID)

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

	log.Debugf("added dns record(s): %q to subnet: %q", request.Records, request.SubnetID)

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

	log.Debugf("unmapped port: %d from subnet: %q", request.SourcePort, request.SubnetID)

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

	log.Debugf("removed dns record: %q from subnet: %q", request.DomainName, request.SubnetID)

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

	log.Debugf("removed peer: %q from subnet: %q", request.PeerID, request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
}
