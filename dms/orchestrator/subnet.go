package orchestrator

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	netutils "gitlab.com/nunet/device-management-service/network/utils"
	"gitlab.com/nunet/device-management-service/utils"
)

type recordsModificationType int

const (
	add recordsModificationType = iota
	remove
)

var orchestratorJoinTimeout = 2 * time.Minute

type SubnetManifest struct {
	CIDR              string            `json:"cidr"`
	GatewayIP         string            `json:"gateway_ip"`
	BroadcastIP       string            `json:"broadcast_ip"`
	UsedIPs           map[string]bool   `json:"used_ips"`
	RoutingTable      map[string]string `json:"routing_table"` // ip -> peerID
	IndexRoutingTable map[string]string `json:"index_routing_table"`
	DNSRecords        map[string]string `json:"dns_records"`
}

type subnetRequest struct {
	handle actor.Handle
	ip     string
	peerID string
	ports  map[int]int
}

type SubnetCreateRequest struct {
	SubnetID     string
	IP           string
	RoutingTable map[string]string
}

type SubnetCreateResponse struct {
	OK    bool
	Error string
}

type SubnetJoinRequest struct {
	SubnetID string
	PeerID   string
	IP       string

	// map of domain_name:ip
	Records map[string]string
}

type SubnetJoinResponse struct {
	OK    bool
	Error string
}

func newSubnetManifest() (SubnetManifest, error) {
	cidr, err := netutils.GetRandomCIDRInRange(
		24,
		net.ParseIP("10.0.0.0"),
		net.ParseIP("10.255.255.255"),
		[]string{},
	)
	if err != nil {
		return SubnetManifest{}, fmt.Errorf("error getting random CIDR: %w", err)
	}

	parts := strings.Split(strings.Split(cidr, "/")[0], ".")
	gatewayIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "1")
	broadcastIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "255")
	usedIPs := map[string]bool{
		gatewayIP:   true,
		broadcastIP: true,
	}

	return SubnetManifest{
		CIDR:              cidr,
		GatewayIP:         gatewayIP,
		BroadcastIP:       broadcastIP,
		UsedIPs:           usedIPs,
		RoutingTable:      make(map[string]string),
		IndexRoutingTable: make(map[string]string),
		DNSRecords:        make(map[string]string),
	}, nil
}

func (p *Provisioner) addAllocationToSubnet(mf jtypes.EnsembleManifest, allocName string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subnetManifest.IndexRoutingTable[allocName]; ok {
		log.Debugf("allocation %s already in subnet", allocName)
		return nil
	}

	ip, err := netutils.GetNextIP(p.subnetManifest.CIDR, p.subnetManifest.UsedIPs)
	if err != nil {
		return fmt.Errorf("error getting next IP: %w", err)
	}
	allocManifest, ok := mf.Allocations[allocName]
	if !ok {
		return fmt.Errorf("allocation %s not found in manifest", allocName)
	}

	p.subnetManifest.RoutingTable[ip.String()] = mf.Nodes[allocManifest.NodeID].Peer
	p.subnetManifest.IndexRoutingTable[allocName] = ip.String()
	p.subnetManifest.DNSRecords[allocManifest.DNSName] = ip.String()
	p.subnetManifest.UsedIPs[ip.String()] = true

	log.Debugf("Added allocation %s with IP %s to subnet", allocName, ip)

	return nil
}

func (o *BasicOrchestrator) removeAllocationsFromSubnet(
	mf jtypes.EnsembleManifest, allocsNames []string,
) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	for _, allocName := range allocsNames {
		// Get the allocation from the manifest
		allocManifest, ok := mf.Allocations[allocName]
		if !ok {
			log.Warnf(
				"skipping subnet removal: allocation %s not found in manifest",
				allocName,
			)
			continue
		}

		// Get the IP address for this allocation
		ip, ok := o.subnetManifest.IndexRoutingTable[allocName]
		if !ok {
			log.Warnf(
				"skipping subnet removal: allocation %s not found in index routing table",
				allocName,
			)
			continue
		}

		// Remove the IP from the used IPs map
		delete(o.subnetManifest.UsedIPs, ip)

		// Remove the routing table entry
		delete(o.subnetManifest.RoutingTable, ip)

		// Remove the index routing table entry
		delete(o.subnetManifest.IndexRoutingTable, allocName)

		// Remove any DNS records associated with this allocation
		if allocManifest.DNSName != "" {
			delete(o.subnetManifest.DNSRecords, allocManifest.DNSName)
		}

		log.Debugf("Removed allocation %s with IP %s from subnet", allocName, ip)
	}

	return nil
}

func (o *BasicOrchestrator) getAllocIP(allocName string) (string, bool) {
	o.lock.Lock()
	defer o.lock.Unlock()
	ip, ok := o.subnetManifest.IndexRoutingTable[allocName]
	if !ok {
		return "", false
	}
	return ip, true
}

// TODO: make it as a method of subnetManifest
// to be tackled here: https://gitlab.com/nunet/device-management-service/-/issues/909
func (p *Provisioner) newSubnetRequests(mf jtypes.EnsembleManifest) ([]subnetRequest, error) {
	// subnet config requests (add peer, dns, port map)
	subReqs := []subnetRequest{}
	for allocName, allocManifest := range mf.Allocations {
		ip, ok := p.subnetManifest.IndexRoutingTable[allocName]
		if !ok {
			return nil, fmt.Errorf("ip not found for allocation %s", allocName)
		}
		nmf, ok := mf.Nodes[allocManifest.NodeID]
		if !ok {
			return nil, fmt.Errorf("node not found for allocation %s", allocName)
		}
		subReqs = append(subReqs, subnetRequest{
			handle: allocManifest.Handle,
			ip:     ip,
			peerID: nmf.Peer,
			ports:  allocManifest.Ports,
		})
	}

	return subReqs, nil
}

func (p *Provisioner) createSubnet(
	manifestID string,
	subReqs []subnetRequest, routingTable map[string]string,
	subCreateHandles []actor.Handle,
) error {
	errCh := make(chan error, len(subReqs))
	wg := sync.WaitGroup{}

	for _, handle := range subCreateHandles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				handle,
				fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, manifestID),
				SubnetCreateRequest{
					SubnetID:     manifestID,
					RoutingTable: routingTable,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetCreateResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet response: %w", err)
					return
				}
				if !response.OK {
					errCh <- fmt.Errorf("error creating subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(SubnetCreateTimeout):
				errCh <- fmt.Errorf("timeout creating subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("subnet successfully created on peer", handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

// TODO: this step should be together with subnet creation, we have
// to refactor the SubnetCreate handle
func (p *Provisioner) subnetAddPeer(manifestID string, subReqs []subnetRequest) error {
	// 1.b create and plug IPs
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				req.handle,
				behaviors.SubnetAddPeerBehavior,
				behaviors.SubnetAddPeerRequest{
					SubnetID: manifestID,
					IP:       req.ip,
					PeerID:   req.peerID,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-peer message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-peer message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetAddPeerResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout adding peer to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("peer successfully added to subnet on peer", req.handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) addDNSRecords(
	manifestID string,
	subReqs []subnetRequest, dnsRecords map[string]string,
) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				req.handle,
				behaviors.SubnetDNSAddRecordsBehavior,
				behaviors.SubnetDNSAddRecordsRequest{
					SubnetID: manifestID,
					Records:  dnsRecords,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-dns-records message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-dns-records message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetDNSAddRecordsResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error sending dns records to peer: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}

			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout sending dns records to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("DNS records successfully added to subnet on peer", req.handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) removeDNSRecords(
	manifestID string,
	subReqs []subnetRequest, domainNames []string,
) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				req.handle,
				behaviors.SubnetDNSRemoveRecordsBehavior,
				behaviors.SubnetDNSRemoveRecordsRequest{
					SubnetID:    manifestID,
					DomainNames: domainNames,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet remove-dns-records message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet remove-dns-records message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetDNSRemoveRecordsResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet remove-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error sending dns records to be removed from peer: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}

			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout removing dns records from subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("DNS records successfully removed from subnet on peer", req.handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) mapPorts(manifestID string, subReqs []subnetRequest) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))
	for _, req := range subReqs {
		for pubPort := range req.ports {
			wg.Add(1)
			go func() {
				defer wg.Done()
				msg, err := actor.Message(
					p.actor.Handle(),
					req.handle,
					behaviors.SubnetMapPortBehavior,
					behaviors.SubnetMapPortRequest{
						SubnetID:   manifestID,
						Protocol:   "TCP", // TODO: add support in AllocationManifest for protocol
						SourceIP:   "0.0.0.0",
						SourcePort: strconv.Itoa(pubPort),
						DestIP:     req.ip,
						DestPort:   strconv.Itoa(pubPort),
					},
					actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
				)
				if err != nil {
					errCh <- fmt.Errorf("error creating subnet MapPort message: %w", err)
					return
				}

				replyCh, err := p.actor.Invoke(msg)
				if err != nil {
					errCh <- fmt.Errorf("error invoking subnet MapPort message: %w", err)
					return
				}

				var reply actor.Envelope
				select {
				case reply = <-replyCh:
					defer reply.Discard()

					var response behaviors.SubnetMapPortResponse
					if err := json.Unmarshal(reply.Message, &response); err != nil {
						errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
						return
					}

					if !response.OK {
						errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
						return
					}
				case <-time.After(2 * time.Minute):
					errCh <- fmt.Errorf("timeout mapping port for subnet: %w", ErrDeploymentFailed)
					return
				}

				log.Info("port mapping successfully added to subnet on peer", req.handle)
			}()
		}
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

// TODO: maybe this hsould go to the createSubnet method
func (p *Provisioner) orchestratorJoinSubnet(
	manifestID string,
	indexRoutingTable map[string]string, dnsRecords map[string]string,
) error {
	msg, err := actor.Message(
		p.actor.Handle(),
		p.actor.Supervisor(),
		fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, manifestID),
		SubnetJoinRequest{
			SubnetID: manifestID,
			IP:       indexRoutingTable[orchSubnetName],
			PeerID:   p.actor.Handle().Address.HostID,
			Records:  dnsRecords,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
	)
	if err != nil {
		return fmt.Errorf("error creating subnet join message: %w", err)
	}

	replyCh, err := p.actor.Invoke(msg)
	if err != nil {
		return fmt.Errorf("error invoking subnet join message: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		var response SubnetJoinResponse
		if err := json.Unmarshal(reply.Message, &response); err != nil {
			return fmt.Errorf("error unmarshalling subnet join response: %w", err)
		}

		if !response.OK {
			return fmt.Errorf("error joining orchestrator to subnet: %s: %w", response.Error, ErrDeploymentFailed)
		}
	case <-time.After(orchestratorJoinTimeout):
		return fmt.Errorf("timeout joining orchestrator to subnet: %w", ErrDeploymentFailed)
	}

	log.Info("orchestrator successfully joined the subnet")
	return nil
}

// It invokes an subnet update behavior on every peer within the subnet
// with updates on:
//  1. routing table
//  2. dns records
//
// Usually used when allocations are added or removed
func (p *Provisioner) updateSubnetAllocations(
	mf jtypes.EnsembleManifest,
	newDNSRecords, additionsRoutingTable map[string]string,
) error {
	subReqs, err := p.newSubnetRequests(mf)
	if err != nil {
		return fmt.Errorf("error creating subnet requests: %w", err)
	}

	// 1. send extend routing table with acceptPeersBehavior
	err = p.acceptPeersSubnetBehavior(mf.ID, subReqs, additionsRoutingTable)
	if err != nil {
		return fmt.Errorf("error adding peers to subnet: %w", err)
	}

	// 2. dns records
	err = p.addDNSRecords(mf.ID, subReqs, newDNSRecords)
	if err != nil {
		return fmt.Errorf("error adding dns records: %w", err)
	}

	return nil
}

func (p *Provisioner) acceptPeersSubnetBehavior(
	mfID string,
	subReqs []subnetRequest, routingTable map[string]string,
) error {
	return p.modifyRoutingTableAllocations(mfID, add, subReqs, routingTable)
}

func (p *Provisioner) removePeersSubnetBehavior(
	mfID string,
	subReqs []subnetRequest, routingTable map[string]string,
) error {
	return p.modifyRoutingTableAllocations(mfID, remove, subReqs, routingTable)
}

func (p *Provisioner) modifyRoutingTableAllocations(
	mfID string, modificationType recordsModificationType,
	subReqs []subnetRequest, routingTable map[string]string,
) error {
	var (
		behavior       string
		operationName  string
		errorMsgPrefix string
		timeoutMsg     string
		successMsg     string
	)

	switch modificationType {
	case add:
		behavior = behaviors.SubnetAcceptPeersBehavior
		operationName = "accept-peer"
		errorMsgPrefix = "adding new peers to subnet"
		timeoutMsg = "timeout adding new peers to subnet"
		successMsg = "new peers added to subnet"
	case remove:
		behavior = behaviors.SubnetRemovePeersBehavior
		operationName = "remove-peer"
		errorMsgPrefix = "removing peers from subnet"
		timeoutMsg = "timeout removing peers from subnet"
		successMsg = "peers removed from subnet"
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func(request subnetRequest) {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				request.handle,
				behavior,
				behaviors.SubnetAcceptPeersRequest{ // Add/remove payload are equal
					SubnetID:            mfID,
					PartialRoutingTable: routingTable,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet %s message: %w", operationName, err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet %s message: %w", operationName, err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetAcceptPeersResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet %s response: %w", operationName, err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error %s: %s: %w", errorMsgPrefix, response.Error, ErrDeploymentFailed)
					return
				}

			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("%s: %w", timeoutMsg, ErrDeploymentFailed)
				return
			}

			log.Info(successMsg, request.handle)
		}(req)
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) revertSubnetAllocationsUpdate(
	mf jtypes.EnsembleManifest,
	dnsRecords, partialRountingTable map[string]string,
) {
	subReqs, err := p.newSubnetRequests(mf)
	if err != nil {
		log.Errorf("Reverting subnet allocations update: error creating subnet requests: %w", err)
	}

	// 1. remove from routing table
	err = p.removePeersSubnetBehavior(mf.ID, subReqs, partialRountingTable)
	if err != nil {
		log.Errorf("Reverting subnet allocations update: error removing peers from subnet: %w", err)
	}

	// 2. remove DNS records
	err = p.removeDNSRecords(mf.ID, subReqs, utils.MapKeysToSlice(dnsRecords))
	if err != nil {
		log.Errorf("Reverting subnet allocations update: error removing dns records: %w", err)
	}
}
