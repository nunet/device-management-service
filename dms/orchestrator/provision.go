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
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	netutils "gitlab.com/nunet/device-management-service/network/utils"
	"gitlab.com/nunet/device-management-service/observability"
)

func (o *BasicOrchestrator) provision(
	cfg jtypes.EnsembleConfig, manifest jtypes.EnsembleManifest,
) error {
	o.setStatus(jtypes.DeploymentStatusProvisioning)
	log.Infow("provisioning ensemble manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id,
	)

	// 1. provision subnet
	err := o.provisionSubnet(manifest)
	if err != nil {
		return fmt.Errorf("provisioning subnet: %w", err)
	}

	// 2. start allocations
	err = o.provisionAllocations(cfg, manifest)
	if err != nil {
		return fmt.Errorf("provisioning allocations: %w", err)
	}

	return nil
}

func (o *BasicOrchestrator) provisionSubnet(manifest jtypes.EnsembleManifest) error {
	for allocName, allocManifest := range manifest.Allocations {
		ip, err := netutils.GetNextIP(o.subnetManifest.CIDR, o.subnetManifest.UsedIPs)
		log.Debug("Generated IP", ip, "for alllocation", allocName)
		if err != nil {
			return fmt.Errorf("error getting next IP: %w", err)
		}
		o.subnetManifest.RoutingTable[ip.String()] = manifest.Nodes[allocManifest.NodeID].Peer
		o.subnetManifest.IndexRoutingTable[allocName] = ip.String()
		o.subnetManifest.UsedIPs[ip.String()] = true

		if _, ok := manifest.Allocations[allocName]; ok {
			o.updateAllocationIP(allocName, ip.String())
		}
	}

	dnsRecords := make(map[string]string)
	for allocName, allocManifest := range manifest.Allocations {
		dnsRecords[allocManifest.DNSName] = o.subnetManifest.IndexRoutingTable[allocName]
	}

	// handles to request subnetcreate
	subCreateHandles := []actor.Handle{}
	for _, node := range manifest.Nodes {
		subCreateHandles = append(subCreateHandles, node.Handle)
	}

	// subnet config requests (add peer, dns, port map)
	subReqs := []subnetRequest{}
	for allocName, allocManifest := range manifest.Allocations {
		subReqs = append(subReqs, subnetRequest{
			handle: allocManifest.Handle,
			ip:     o.subnetManifest.IndexRoutingTable[allocName],
			peerID: manifest.Nodes[allocManifest.NodeID].Peer,
			ports:  allocManifest.Ports,
		})
	}

	if manifest.Subnet.Join { // orchestrator should join the subnet
		ip, err := netutils.GetNextIP(o.subnetManifest.CIDR, o.subnetManifest.UsedIPs)
		log.Debug("Generated IP %s for orchestrator", ip)
		if err != nil {
			return fmt.Errorf("error getting next IP: %w", err)
		}
		o.subnetManifest.RoutingTable[ip.String()] = o.actor.Handle().Address.HostID
		o.subnetManifest.IndexRoutingTable[orchSubnetName] = ip.String()
		o.subnetManifest.UsedIPs[ip.String()] = true

		subCreateHandles = append(subCreateHandles, o.actor.Supervisor())
		dnsRecords[orchSubnetName] = o.subnetManifest.IndexRoutingTable[orchSubnetName]
	}

	// 1.a create subnet in each peer
	err := o.createSubnet(subReqs, o.subnetManifest.RoutingTable, subCreateHandles)
	if err != nil {
		return fmt.Errorf("error creating subnet: %w", err)
	}

	// if orchestrator should join subnet, setup with one behavior
	// this doesn't look very good but let's address with #893
	if manifest.Subnet.Join {
		err := o.orchestratorJoinSubnet(o.subnetManifest.IndexRoutingTable, dnsRecords)
		if err != nil {
			return fmt.Errorf("error joining subnet: %w", err)
		}
	}

	// 1.b create and plug IPs
	err = o.subnetAddPeer(subReqs)
	if err != nil {
		return fmt.Errorf("error adding peers to subnet: %w", err)
	}

	// 1.c configure DNS
	err = o.addDNSRecords(subReqs, dnsRecords)
	if err != nil {
		return fmt.Errorf("error adding dns records to subnet: %w", err)
	}

	// 1.d configure port mapping
	err = o.mapPorts(subReqs)
	if err != nil {
		return fmt.Errorf("error adding port mappings to subnet: %w", err)
	}

	return nil
}

func (o *BasicOrchestrator) provisionAllocations(
	cfg jtypes.EnsembleConfig, manifest jtypes.EnsembleManifest,
) error {
	var wg sync.WaitGroup
	var aggErr error

	if o.isOnlyTaskManifest(manifest) {
		go o.monitorOnlyTaskManifest(manifest)
	}

	interim := map[string][]string{} // a map of verteces to edges (their dependencies)
	for allocName, allocCfg := range cfg.Allocations() {
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
			go func(allocManifest jtypes.AllocationManifest) {
				defer wg.Done()

				msg, err := actor.Message(
					o.actor.Handle(),
					allocManifest.Handle,
					behaviors.AllocationStartBehavior,
					behaviors.AllocationStartRequest{
						SubnetIP:    o.subnetManifest.IndexRoutingTable[allocName],
						GatewayIP:   o.subnetManifest.GatewayIP,
						PortMapping: allocManifest.Ports,
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

				log.Infof("allocation successfully started on peer %s for allocation %s", &allocManifest.Handle.DID, manifest.ID)
				allocStatuses[allocName] = jtypes.AllocationRunning
			}(manifest.Allocations[allocName])

			wg.Wait()

			for allocName, status := range allocStatuses {
				o.updateAllocationStatus(allocName, status)
			}

			close(errCh)
			if aggregateErrors(errCh) != nil {
				return aggErr
			}
		}
	}

	return nil
}

func (o *BasicOrchestrator) updateAllocationStatus(allocName string, s jtypes.AllocationStatus) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if alloc, ok := o.manifest.Allocations[allocName]; ok {
		alloc.Status = s
		o.manifest.Allocations[allocName] = alloc
	} else {
		log.Warnf("allocation %s not found in manifest", allocName)
	}
}

// updateAllocationIP
func (o *BasicOrchestrator) updateAllocationIP(allocName string, ip string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if alloc, ok := o.manifest.Allocations[allocName]; ok {
		alloc.PrivAddr = ip
		o.manifest.Allocations[allocName] = alloc
	} else {
		log.Warnf("allocation %s not found in manifest", allocName)
	}
}

func (o *BasicOrchestrator) isOnlyTaskManifest(m jtypes.EnsembleManifest) bool {
	for _, a := range m.Allocations {
		if a.Type != jtypes.AllocationTypeTask {
			return false
		}
	}
	return true
}

// monitorOnlyTaskManifest will be responsible for tearing down
// the orchestrator after all tasks are terminated.
func (o *BasicOrchestrator) monitorOnlyTaskManifest(m jtypes.EnsembleManifest) {
	if !o.isOnlyTaskManifest(m) {
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
			for name := range m.Allocations {
				if !m.IsTerminatedTask(name) {
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
