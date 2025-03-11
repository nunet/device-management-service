// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
)

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

func (o *BasicOrchestrator) provision() error {
	log.Infof("provisioning ensemble manifest: %+v", o.manifest)

	// 1. create subnet
	// 1.a generate routing table
	cidr, err := netutils.GetRandomCIDRInRange(
		24,
		net.ParseIP("10.0.0.0"),
		net.ParseIP("10.255.255.255"),
		[]string{},
	)
	if err != nil {
		return fmt.Errorf("error getting random CIDR: %w", err)
	}

	parts := strings.Split(strings.Split(cidr, "/")[0], ".")
	gatewayIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "1")
	broadcastIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "255")
	usedIPs := map[string]bool{
		gatewayIP:   true,
		broadcastIP: true,
	}

	routingTable := make(map[string]string)
	indexRoutingTable := make(map[string]string)
	for allocationID, manifest := range o.manifest.Allocations {
		ip, err := netutils.GetNextIP(cidr, usedIPs)
		log.Debug("Generated IP", ip, "for alllocation", allocationID)
		if err != nil {
			return fmt.Errorf("error getting next IP: %w", err)
		}
		routingTable[ip.String()] = o.manifest.Nodes[manifest.NodeID].Peer
		indexRoutingTable[allocationID] = ip.String()
		usedIPs[ip.String()] = true

		o.lock.Lock()
		if alloc, ok := o.manifest.Allocations[allocationID]; ok {
			alloc.PrivAddr = ip.String()
			o.manifest.Allocations[allocationID] = alloc
		}
		o.lock.Unlock()
	}

	dnsRecords := make(map[string]string)
	for allocationID, manifest := range o.manifest.Allocations {
		dnsRecords[manifest.DNSName] = indexRoutingTable[allocationID]
	}

	type subnetRequest struct {
		handle actor.Handle
		ip     string
		peerID string
		ports  map[int]int
	}

	// handles to request subnetcreate
	subCreateHandles := []actor.Handle{}
	for _, node := range o.manifest.Nodes {
		subCreateHandles = append(subCreateHandles, node.Handle)
	}

	// subnet config requests (add peer, dns, port map)
	subReqs := []subnetRequest{}
	for allocationID, manifest := range o.manifest.Allocations {
		subReqs = append(subReqs, subnetRequest{
			handle: manifest.Handle,
			ip:     indexRoutingTable[allocationID],
			peerID: o.manifest.Nodes[manifest.NodeID].Peer,
			ports:  manifest.Ports,
		})
	}

	if o.manifest.Subnet.Join { // orchestrator should join the subnet
		ip, err := netutils.GetNextIP(cidr, usedIPs)
		log.Debug("Generated IP %s for orchestrator", ip)
		if err != nil {
			return fmt.Errorf("error getting next IP: %w", err)
		}
		routingTable[ip.String()] = o.actor.Handle().Address.HostID
		indexRoutingTable[orchSubnetName] = ip.String()
		usedIPs[ip.String()] = true

		subCreateHandles = append(subCreateHandles, o.actor.Supervisor())
		dnsRecords[orchSubnetName] = indexRoutingTable[orchSubnetName]
	}

	errCh := make(chan error, len(subReqs))
	wg := sync.WaitGroup{}
	for _, handle := range subCreateHandles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				handle,
				fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, o.manifest.ID),
				SubnetCreateRequest{
					SubnetID:     o.manifest.ID,
					RoutingTable: routingTable,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
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

	var aggErr error
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// if orchestrator should join subnet, setup with one behavior
	// this doesn't look very good but let's address with #893
	if o.manifest.Subnet.Join {
		go func() {
			msg, err := actor.Message(
				o.actor.Handle(),
				o.actor.Supervisor(),
				fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, o.manifest.ID),
				SubnetJoinRequest{
					SubnetID: o.manifest.ID,
					IP:       indexRoutingTable[orchSubnetName],
					PeerID:   o.actor.Handle().Address.HostID,
					Records:  dnsRecords,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet join message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet join message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetJoinResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet join response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error joining orchestrator to subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout joining orchestrator to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("orchestrator successfully joined the subnet")
		}()
	}

	// 1.b create and plug IPs
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(subReqs))
	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				req.handle,
				behaviors.SubnetAddPeerBehavior,
				behaviors.SubnetAddPeerRequest{
					SubnetID: o.manifest.ID,
					IP:       req.ip,
					PeerID:   req.peerID,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-peer message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
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
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 1.c configure DNS
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				req.handle,
				behaviors.SubnetDNSAddRecordsBehavior,
				behaviors.SubnetDNSAddRecordsRequest{
					SubnetID: o.manifest.ID,
					Records:  dnsRecords,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-dns-records message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
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
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 1.d configure port mapping
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(subReqs))
	for _, req := range subReqs {
		for pubPort := range req.ports {
			wg.Add(1)
			go func() {
				defer wg.Done()
				msg, err := actor.Message(
					o.actor.Handle(),
					req.handle,
					behaviors.SubnetMapPortBehavior,
					behaviors.SubnetMapPortRequest{
						SubnetID:   o.manifest.ID,
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

				replyCh, err := o.actor.Invoke(msg)
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
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 2. start the allocations
	if o.isOnlyTaskEnsemble() {
		go o.monitorOnlyTasksEnsemble()
	}

	interim := map[string][]string{} // a map of verteces to edges (their dependencies)
	for allocName, allocCfg := range o.cfg.Allocations() {
		interim[allocName] = allocCfg.DependsOn
	}

	orderedAllocs, err := orderByDependency(interim)
	if err != nil {
		return err
	}

	allocStatuses := make(map[string]jtypes.AllocationStatus)
	for _, allocs := range orderedAllocs {
		wg = sync.WaitGroup{}
		for _, allocName := range allocs {
			errCh := make(chan error, len(allocs))
			wg.Add(1)
			go func(manifest jtypes.AllocationManifest) {
				defer wg.Done()

				msg, err := actor.Message(
					o.actor.Handle(),
					manifest.Handle,
					behaviors.AllocationStartBehavior,
					behaviors.AllocationStartRequest{
						SubnetIP:    indexRoutingTable[allocName],
						GatewayIP:   gatewayIP,
						PortMapping: manifest.Ports,
					},
					actor.WithMessageExpiry(actor.MakeExpiry(AllocationStartTimeout)),
				)
				if err != nil {
					errCh <- fmt.Errorf("error creating allocation start message: %w", err)
					return
				}

				replyCh, err := o.actor.Invoke(msg)
				if err != nil {
					errCh <- fmt.Errorf("error invoking allocation start: %w", err)
					return
				}

				ticker := time.NewTicker(AllocationStartTimeout)
				defer ticker.Stop()

				var reply actor.Envelope
				select {
				case reply = <-replyCh:
					defer reply.Discard()

					var response behaviors.AllocationStartResponse
					if err := json.Unmarshal(reply.Message, &response); err != nil {
						errCh <- fmt.Errorf("error unmarshalling allocation start response: %w", err)
						return
					}

					if !response.OK {
						allocStatuses[allocName] = jtypes.AllocationFailed
						errCh <- fmt.Errorf("error starting allocation: %s: %w", response.Error, ErrDeploymentFailed)
						return
					}
				case <-ticker.C:
					errCh <- fmt.Errorf("timeout starting allocation: %w", ErrDeploymentFailed)
					return
				}

				o.lock.Lock()
				if alloc, ok := o.manifest.Allocations[allocName]; ok {
					alloc.Status = jtypes.AllocationRunning
					o.manifest.Allocations[allocName] = alloc
				}
				o.lock.Unlock()

				log.Infof("allocation successfully started on peer %s for allocation %s", &manifest.Handle.DID, manifest.ID)
				allocStatuses[allocName] = jtypes.AllocationRunning
			}(o.manifest.Allocations[allocName])

			wg.Wait()

			o.lock.Lock()
			for allocName, status := range allocStatuses {
				if alloc, ok := o.manifest.Allocations[allocName]; ok {
					alloc.Status = status
					o.manifest.Allocations[allocName] = alloc
				} else {
					log.Warnf("allocation %s not found in manifest", allocName)
				}
			}
			o.lock.Unlock()

			close(errCh)
			aggErr = nil
			for err := range errCh {
				if aggErr == nil {
					aggErr = err
					continue
				} else if err != nil {
					aggErr = fmt.Errorf("%w\n%w", aggErr, err)
				}
			}
			if aggErr != nil {
				return aggErr
			}
		}
	}

	return nil
}

func (o *BasicOrchestrator) isOnlyTaskEnsemble() bool {
	for _, a := range o.manifest.Allocations {
		if a.Type != jtypes.AllocationTypeTask {
			return false
		}
	}
	return true
}

// monitorOnlyTasksEnsemble will be responsible for tearing down
// the orchestrator after all tasks are terminated.
func (o *BasicOrchestrator) monitorOnlyTasksEnsemble() {
	if !o.isOnlyTaskEnsemble() {
		return
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
selectLoop:
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			for name := range o.manifest.Allocations {
				if !o.manifest.IsTerminatedTask(name) {
					continue selectLoop
				}
			}
			log.Infof("All tasks are terminated, shutting down orchestrator.")

			o.setStatus(jtypes.DeploymentStatusCompleted)
			o.cancel()
			return
		}
	}
}
