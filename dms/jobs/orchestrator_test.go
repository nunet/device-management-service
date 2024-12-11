// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/types"
)

type subnetObj struct {
	id           string
	routingTable map[string]string
	peer         map[string]string
	dns          map[string]string
	ports        map[int]int
}

func TestProvision(t *testing.T) {
	addrs, privKey, peer := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
	rootDID, root := actor.MakeRootTrustContext(t)
	actorDID, trust := actor.MakeTrustContext(t, privKey)
	capa := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)
	actr := actor.CreateActor(t, peer, capa)
	require.NoError(t, actr.Start())
	orchestrator, err := NewOrchestrator(context.Background(), "id", actr, peer, job_types.EnsembleConfig{
		V1: &job_types.EnsembleConfigV1{
			Allocations: map[string]job_types.AllocationConfig{
				"allocation1": {
					Executor: "docker",
					Resources: types.Resources{
						CPU: types.CPU{
							ClockSpeed: 2.4,
							Cores:      2,
							Model:      "Intel Core i7",
							Vendor:     "Intel",
						},
						GPUs: []types.GPU{
							{
								Model:      "NVIDIA GeForce GTX 1080",
								Vendor:     "NVIDIA",
								VRAM:       8,
								Index:      0,
								PCIAddress: "0000:01:00.0",
							},
						},
						RAM: types.RAM{
							Size:       16,
							ClockSpeed: 2400,
						},
						Disk: types.Disk{
							Size:       256,
							Model:      "Samsung 970 EVO",
							Vendor:     "Samsung",
							Type:       "SSD",
							Interface:  "NVMe",
							ReadSpeed:  3500,
							WriteSpeed: 2500,
						},
					},
					Execution: types.SpecConfig{},
				},
			},
			Nodes: map[string]job_types.NodeConfig{
				"node1": {
					Allocations: []string{"allocation1"},
				},
			},
		},
	})
	require.NoError(t, err)

	_, privKey1, peer1 := actor.NewLibp2pNetwork(t, addrs)
	rootDID1, root1 := actor.MakeRootTrustContext(t)
	actorDID1, trust1 := actor.MakeTrustContext(t, privKey1)
	cap1 := actor.MakeCapabilityContext(t, actorDID1, rootDID1, trust1, root1)

	// actor.AllowReciprocal(t, cap, root, rootDID, rootDID1, "/dms")
	// actor.AllowReciprocal(t, cap1, root1, rootDID1, rootDID, "/dms")

	actr1 := actor.CreateActor(t, peer1, cap1)
	require.NoError(t, actr1.Start())

	subnets := make(map[string]subnetObj)
	_ = actr1.AddBehavior(SubnetCreateBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		t.Log("got msg for create")
		var request SubnetCreateRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		subnets[actr1.Handle().DID.String()] = subnetObj{
			id:           request.SubnetID,
			routingTable: request.RoutingTable,
			peer:         map[string]string{},
			dns:          map[string]string{},
			ports:        map[int]int{},
		}

		reply, err := actor.ReplyTo(msg, SubnetCreateResponse{
			OK: true,
		})
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	_ = actr1.AddBehavior(AllocationStartBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		response := AllocationStartResponse{
			OK: true,
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	_ = actr1.AddBehavior(SubnetAddPeerBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request SubnetAddPeerRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := SubnetAddPeerResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found" //nolint
		} else {
			subnet.peer[request.IP] = request.PeerID
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	_ = actr1.AddBehavior(SubnetAcceptPeerBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request SubnetAcceptPeerRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := SubnetAcceptPeerResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found"
		} else {
			subnet.routingTable[request.IP] = request.PeerID
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	_ = actr1.AddBehavior(SubnetDNSAddRecordsBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request SubnetDNSAddRecordsRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := SubnetDNSAddRecordsResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found"
		} else {
			for name, ip := range request.Records {
				subnet.dns[name] = ip
			}
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	_ = actr1.AddBehavior(SubnetMapPortBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request SubnetMapPortRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := SubnetMapPortResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found"
		} else {
			srcPort, _ := strconv.Atoi(request.SourcePort)
			destPort, _ := strconv.Atoi(request.DestPort)
			subnet.ports[srcPort] = destPort
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
	})

	nodeID := uuid.New().String()
	manifest := EnsembleManifest{
		ID:           uuid.New().String(),
		Orchestrator: actr.Handle(),
		Allocations: map[string]AllocationManifest{
			"allocation1": {
				ID:       uuid.New().String(),
				NodeID:   nodeID,
				Handle:   actr1.Handle(),
				DNSName:  "actor.com.",
				PrivAddr: "",
				Ports: map[int]int{
					8080: 8888,
				},
			},
		},
		Nodes: map[string]NodeManifest{
			nodeID: {
				ID:        uuid.New().String(),
				Peer:      peer1.Host.ID().String(),
				Handle:    actr1.Handle(),
				PubAddrss: []string{},
				Location:  Location{},
				Allocations: []string{
					"allocation1",
				},
			},
		},
	}

	actrdid, err := did.FromID(actr1.Handle().ID)
	require.NoError(t, err)
	tokenlist, err := cap1.Grant(
		ucan.Delegate,
		actr.Handle().DID,
		actrdid,
		[]string{"/nunet"},
		actor.MakeExpiry(time.Hour),
		0,
		[]ucan.Capability{
			ucan.Capability(AllocationStartBehavior),
			ucan.Capability(SubnetCreateBehavior),
			ucan.Capability(SubnetAddPeerBehavior),
			ucan.Capability(SubnetAcceptPeerBehavior),
			ucan.Capability(SubnetDNSAddRecordsBehavior),
			ucan.Capability(SubnetMapPortBehavior),
		},
	)
	require.NoError(t, err)
	require.NoError(t, cap1.AddRoots([]did.DID{}, tokenlist, ucan.TokenList{}, ucan.TokenList{}))

	err = orchestrator.provision(manifest)
	require.NoError(t, err)

	// TODO 741 - re-enable after provision is fixed
	//
	// ownIP := ""
	// subnet, ok := subnets[actr1.Handle().DID.String()]

	// require.True(t, ok)

	// assert.Equal(t, subnet.id, manifest.ID)
	// for ip, peerID := range subnet.routingTable {
	// 	if peerID == peer1.Host.ID().String() {
	// 		assert.Equal(t, subnet.routingTable[ip], peerID)
	// 		ownIP = ip
	// 	}
	// }

	// assert.Equal(t, subnet.peer[ownIP], peer1.Host.ID().String())
	// assert.Equal(t, subnet.dns["actor.com."], ownIP)

	// assert.Equal(t, subnet.ports[8080], 8888)
}

func TestSupervise(t *testing.T) {
	addrs, privKey, peer := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
	rootDID, root := actor.MakeRootTrustContext(t)
	actorDID, trust := actor.MakeTrustContext(t, privKey)
	capa := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)
	actr := actor.CreateActor(t, peer, capa)
	require.NoError(t, actr.Start())
	orchestrator, err := NewOrchestrator(context.Background(), "id", actr, peer, job_types.EnsembleConfig{
		V1: &job_types.EnsembleConfigV1{
			Allocations: map[string]job_types.AllocationConfig{
				"allocation1": {
					Executor: "docker",
					Resources: types.Resources{
						CPU: types.CPU{
							ClockSpeed: 2.4,
							Cores:      2,
							Model:      "Intel Core i7",
							Vendor:     "Intel",
						},
						GPUs: []types.GPU{
							{
								Model:      "NVIDIA GeForce GTX 1080",
								Vendor:     "NVIDIA",
								VRAM:       8,
								Index:      0,
								PCIAddress: "0000:01:00.0",
							},
						},
						RAM: types.RAM{
							Size:       16,
							ClockSpeed: 2400,
						},
						Disk: types.Disk{
							Size:       256,
							Model:      "Samsung 970 EVO",
							Vendor:     "Samsung",
							Type:       "SSD",
							Interface:  "NVMe",
							ReadSpeed:  3500,
							WriteSpeed: 2500,
						},
					},
					Execution: types.SpecConfig{},
				},
			},
			Nodes: map[string]NodeConfig{
				"node1": {
					Allocations: []string{"allocation1"},
				},
			},
		},
	})
	require.NoError(t, err)

	_, privKey1, peer1 := actor.NewLibp2pNetwork(t, addrs)
	rootDID1, root1 := actor.MakeRootTrustContext(t)
	actorDID1, trust1 := actor.MakeTrustContext(t, privKey1)
	cap1 := actor.MakeCapabilityContext(t, actorDID1, rootDID1, trust1, root1)

	actr1 := actor.CreateActor(t, peer1, cap1)
	require.NoError(t, actr1.Start())

	times := 0
	_ = actr1.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		t.Log("got msg for create")
		var request struct{}
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		t.Log("Responding to healthcheck")
		if times >= 1 {
			t.Log("Not going through")
			return
		}

		reply, err := actor.ReplyTo(msg, nil)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}

		times++
	})

	restartedAllocations := make(map[string]bool)
	ch := make(chan struct{})
	_ = actr1.AddBehavior(RestartAllocationBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request RestartAllocationRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		t.Log("Restarting allocation", request.AllocationID)

		restartedAllocations[request.AllocationID] = true

		response := RestartAllocationResponse{
			OK: true,
		}

		reply, err := actor.ReplyTo(msg, response)
		if err != nil {
			log.Debugf("error creating reply: %s", err)
			return
		}

		if err := actr1.Send(reply); err != nil {
			log.Debugf("error sending  reply: %s", err)
		}
		ch <- struct{}{}
	})

	actrdid, err := did.FromID(actr1.Handle().ID)
	require.NoError(t, err)
	tokenlist, err := cap1.Grant(
		ucan.Delegate,
		actr.Handle().DID,
		actrdid,
		[]string{"/nunet"},
		actor.MakeExpiry(time.Hour),
		0,
		[]ucan.Capability{
			ucan.Capability(actor.HealthCheckBehavior),
			ucan.Capability(RestartAllocationBehavior),
		},
	)
	require.NoError(t, err)
	require.NoError(t, cap1.AddRoots([]did.DID{}, tokenlist, ucan.TokenList{}, ucan.TokenList{}))

	allocations := map[string]actor.Handle{}
	allocations["allocation1"] = actr1.Handle()
	go orchestrator.supervise(allocations, orchestrator.manifest)

	<-time.After(actor.HealthCheckInterval)
	require.Equal(t, 0, len(restartedAllocations))

	<-ch
	require.Equal(t, 1, len(restartedAllocations))
	require.Equal(t, true, restartedAllocations["allocation1"])
}
