package jobs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network/libp2p"
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
	cap := actor.MakeCapabilityContext(t, actorDID, rootDID, trust, root)
	actr := actor.CreateActor(t, peer, cap)
	require.NoError(t, actr.Start())
	orchestrator := NewOrchestrator(uuid.New().String(), EnsembleConfig{}, actr)

	_, privKey1, peer1 := actor.NewLibp2pNetwork(t, addrs)
	rootDID1, root1 := actor.MakeRootTrustContext(t)
	actorDID1, trust1 := actor.MakeTrustContext(t, privKey1)
	cap1 := actor.MakeCapabilityContext(t, actorDID1, rootDID1, trust1, root1)

	actr1 := actor.CreateActor(t, peer1, cap1)
	require.NoError(t, actr1.Start())

	subnets := make(map[string]subnetObj)
	actr1.AddBehavior(libp2p.SubnetCreateBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		fmt.Println("got msg for create")
		var request libp2p.SubnetCreateRequest
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

		reply, err := actor.ReplyTo(msg, libp2p.SubnetCreateResponse{
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

	actr1.AddBehavior(libp2p.SubnetAddPeerBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request libp2p.SubnetAddPeerRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := libp2p.SubnetAddPeerResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found"
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

	actr1.AddBehavior(libp2p.SubnetAcceptPeerBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request libp2p.SubnetAcceptPeerRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := libp2p.SubnetAcceptPeerResponse{
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

	actr1.AddBehavior(libp2p.SubnetDNSAddRecordBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request libp2p.SubnetDNSAddRecordRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := libp2p.SubnetDNSAddRecordResponse{
			OK: true,
		}

		subnet, ok := subnets[actr1.Handle().DID.String()]
		if !ok {
			response.OK = false
			response.Error = "subnet not found"
		} else {
			subnet.dns[request.DomainName] = request.IP
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

	actr1.AddBehavior(libp2p.SubnetMapPortBehavior, func(msg actor.Envelope) {
		defer msg.Discard()

		var request libp2p.SubnetMapPortRequest
		if err := json.Unmarshal(msg.Message, &request); err != nil {
			return
		}

		response := libp2p.SubnetMapPortResponse{
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

	nodeId := uuid.New().String()
	manifest := EnsembleManifest{
		ID:           uuid.New().String(),
		Orchestrator: actr.Handle(),
		Allocations: map[string]AllocationManifest{
			"allocation1": {
				ID:       uuid.New().String(),
				NodeID:   nodeId,
				Handle:   actr1.Handle(),
				DNSName:  "actor.com.",
				PrivAddr: "",
				Ports: map[int]int{
					8080: 8888,
				},
			},
		},
		Nodes: map[string]NodeManifest{
			nodeId: {
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
			// ucan.Capability("/dms"),
			ucan.Capability(libp2p.SubnetCreateBehavior),
			ucan.Capability(libp2p.SubnetAddPeerBehavior),
			ucan.Capability(libp2p.SubnetAcceptPeerBehavior),
			ucan.Capability(libp2p.SubnetDNSAddRecordBehavior),
			ucan.Capability(libp2p.SubnetMapPortBehavior),
		},
	)
	require.NoError(t, err)
	require.NoError(t, cap1.AddRoots([]did.DID{}, tokenlist, ucan.TokenList{}))

	err = orchestrator.provision(manifest)
	require.NoError(t, err)

	ownIP := ""
	subnet, ok := subnets[actr1.Handle().DID.String()]
	require.True(t, ok)

	assert.Equal(t, subnet.id, manifest.ID)
	for ip, peerId := range subnet.routingTable {
		if peerId == peer1.Host.ID().String() {
			assert.Equal(t, subnet.routingTable[ip], peerId)
			ownIP = ip
		}
	}

	assert.Equal(t, subnet.peer[ownIP], peer1.Host.ID().String())
	assert.Equal(t, subnet.dns["actor.com."], ownIP)

	assert.Equal(t, subnet.ports[8080], 8888)
}
