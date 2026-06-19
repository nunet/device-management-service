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
	"fmt"

	"go.uber.org/multierr"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

func (a *Allocation) handleSubnetAddPeer(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetAddPeerRequest
	resp := behaviors.SubnetAddPeerResponse{}

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
	a.setSubnetID(request.SubnetID)
	a.mergeRoutingTable(map[string]string{request.IP: request.PeerID})

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetAcceptPeers(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetAcceptPeersRequest
	resp := behaviors.SubnetAcceptPeersResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AcceptSubnetPeers(request.SubnetID, request.PartialRoutingTable)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_peer_accepted",
		"labels", []string{},
		"peers", request.PartialRoutingTable)
	a.mergeRoutingTable(request.PartialRoutingTable)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetMapPort(msg actor.Envelope) {
	defer msg.Discard()
	log.Debugw("handle_subnet_map_port_invoked", "from", msg.From)

	var request behaviors.SubnetMapPortRequest
	resp := behaviors.SubnetMapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("subnet_map_port_unmarshal_error",
			"labels", []string{},
			"error", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	mapReq := types.MapPortRequest{
		SubnetID:      request.SubnetID,
		Protocol:      request.Protocol,
		ExecutionPort: request.SourcePort,
		SubnetIP:      request.DestIP,
		SubnetPort:    request.DestPort,
		ExecutorType:  types.ExecutorType(a.Job.Execution.Type),
		CNIBridge:     a.executorInfo.Net.HostBridge,
	}

	log.Infow("subnet_map_port_args",
		"subnetID", mapReq.SubnetID,
		"protocol", mapReq.Protocol,
		"executionPort", mapReq.ExecutionPort,
		"subnetIP", mapReq.SubnetIP,
		"subnetPort", mapReq.SubnetPort,
		"executorType", mapReq.ExecutorType,
		"cniBridge", mapReq.CNIBridge,
	)

	err := a.network.MapPort(mapReq)
	if err != nil {
		log.Errorw("subnet_map_port_error",
			"labels", []string{},
			"error", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	a.portMapping = append(a.portMapping, jobtypes.AllocationPortMapping{
		SubnetID:   request.SubnetID,
		Protocol:   request.Protocol,
		SourceIP:   request.SourceIP,
		SourcePort: request.SourcePort,
		DestIP:     request.DestIP,
		DestPort:   request.DestPort,
	})

	log.Debugw("subnet_port_mapped",
		"labels", []string{},
		"sourcePort", request.SourcePort,
		"subnetID", request.SubnetID)
	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetDNSAddRecords(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetDNSAddRecordsRequest
	resp := behaviors.SubnetDNSAddRecordsResponse{}

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

	a.mergeDNSRecords(request.Records)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetUnmapPort(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetUnmapPortRequest
	resp := behaviors.SubnetUnmapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.UnmapPort(types.MapPortRequest{
		SubnetID:      request.SubnetID,
		Protocol:      request.Protocol,
		ExecutionPort: request.SourcePort,
		SubnetIP:      request.DestIP,
		SubnetPort:    request.DestPort,
		ExecutorType:  types.ExecutorType(a.Job.Execution.Type),
		CNIBridge:     a.executorInfo.Net.HostBridge,
	})
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

func (a *Allocation) handleSubnetDNSRemoveRecords(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetDNSRemoveRecordsRequest
	resp := behaviors.SubnetDNSRemoveRecordsResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	var errs error

	for _, domain := range request.DomainNames {
		err := a.network.RemoveSubnetDNSRecord(request.SubnetID, domain)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("error removing dns record: %w", err))
		}
	}

	if errs != nil {
		resp.Error = errs.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_dns_record_removed",
		"labels", []string{},
		"domains", request.DomainNames,
		"subnetID", request.SubnetID)
	a.removeDNSRecords(request.DomainNames)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetRemovePeers(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.SubnetRemovePeersRequest
	resp := behaviors.SubnetRemovePeersResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.RemoveSubnetPeers(request.SubnetID, request.PartialRoutingTable)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugw("subnet_peer_removed",
		"labels", []string{},
		"peers", request.PartialRoutingTable)
	a.removeRoutingTableEntries(request.PartialRoutingTable)

	resp.OK = true
	a.sendReply(msg, resp)
}
