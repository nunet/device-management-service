// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	netutils "gitlab.com/nunet/device-management-service/network/utils"
	"gitlab.com/nunet/device-management-service/observability"
)

const orchSubnetName = "orchestrator"

var monitorOnlyTaskManifestInterval = time.Second * 10

// Provisioner handles the provisioning process for ensemble deployment
type Provisioner struct {
	ctx            context.Context
	cancel         context.CancelFunc
	actor          actor.Actor
	subnetManifest jtypes.SubnetManifest

	lock sync.Mutex
}

// NewProvisioner creates a new Provisioner instance
func NewProvisioner(
	ctx context.Context,
	cancel context.CancelFunc,
	actor actor.Actor,
	subnetManifest jtypes.SubnetManifest,
) *Provisioner {
	return &Provisioner{
		ctx:            ctx,
		cancel:         cancel,
		actor:          actor,
		subnetManifest: subnetManifest,
	}
}

// Provision handles the provisioning process and returns the updated manifest
func (p *Provisioner) Provision(
	cfgReader jtypes.EnsembleCfgReader,
	manifestReader jtypes.ManifestReader,
) (jtypes.EnsembleManifest, error) {
	cfg := cfgReader.Read()
	manifest := manifestReader.Read()

	log.Infow("provisioning ensemble manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", manifest.ID,
	)

	// 1. provision subnet
	manifest, err := p.provisionSubnet(manifest)
	if err != nil {
		return manifest, fmt.Errorf("provisioning subnet: %w", err)
	}

	// 2. start allocations
	manifest, err = p.provisionAllocations(cfg, manifest)
	if err != nil {
		return manifest, fmt.Errorf("provisioning allocations: %w", err)
	}

	return manifest, nil
}

func (p *Provisioner) provisionSubnet(manifest jtypes.EnsembleManifest, skipCreate ...string) (jtypes.EnsembleManifest, error) {
	for allocName := range manifest.Allocations {
		err := p.addAllocationToSubnet(manifest, allocName)
		if err != nil {
			return manifest,
				fmt.Errorf("error adding allocation %s to subnet: %w", allocName, err)
		}

		err = manifest.UpdateAllocation(allocName, func(alloc *jtypes.AllocationManifest) {
			alloc.PrivAddr = p.subnetManifest.IndexRoutingTable[allocName]
		})
		if err != nil {
			return manifest, fmt.Errorf("error updating allocation %s: %w", allocName, err)
		}
	}

	// handles to request subnetcreate
	subCreateHandles := []actor.Handle{}
	// subnet config requests (add peer, dns, port map)
	subReqs := []subnetRequest{}
	for _, node := range manifest.Nodes {
		if !slices.Contains(skipCreate, node.ID) {
			subCreateHandles = append(subCreateHandles, node.Handle)
		}
		for _, alloc := range node.Allocations {
			amf := manifest.Allocations[alloc]
			subReqs = append(subReqs, subnetRequest{
				handle: amf.Handle,
				ip:     p.subnetManifest.IndexRoutingTable[alloc],
				peerID: manifest.Nodes[amf.NodeID].Peer,
				ports:  amf.Ports,
			})
		}
	}

	if manifest.Subnet.Join { // orchestrator should join the subnet
		if _, ok := p.subnetManifest.IndexRoutingTable[orchSubnetName]; !ok {
			ip, err := netutils.GetNextIP(p.subnetManifest.CIDR, p.subnetManifest.UsedIPs)
			log.Debug("Generated IP %s for orchestrator", ip)
			if err != nil {
				return manifest, fmt.Errorf("error getting next IP: %w", err)
			}
			p.subnetManifest.RoutingTable[ip.String()] = p.actor.Handle().Address.HostID
			p.subnetManifest.IndexRoutingTable[orchSubnetName] = ip.String()
			p.subnetManifest.UsedIPs[ip.String()] = true

			subCreateHandles = append(subCreateHandles, p.actor.Supervisor())
			p.subnetManifest.DNSRecords[orchSubnetName] = p.subnetManifest.IndexRoutingTable[orchSubnetName]
		}
	}

	// 1.a create subnet in each peer
	err := p.createSubnet(manifest.ID, p.subnetManifest.RoutingTable, subCreateHandles)
	if err != nil {
		return manifest, fmt.Errorf("error creating subnet: %w", err)
	}

	// if orchestrator should join subnet, setup with one behavior
	// this doesn't look very good but let's address with #893
	if manifest.Subnet.Join {
		err := p.orchestratorJoinSubnet(manifest.ID, p.subnetManifest.IndexRoutingTable, p.subnetManifest.RoutingTable, p.subnetManifest.DNSRecords)
		if err != nil {
			return manifest, fmt.Errorf("error joining subnet: %w", err)
		}
	}

	// 1.b create and plug IPs
	err = p.subnetAddPeer(manifest.ID, subReqs)
	if err != nil {
		return manifest, fmt.Errorf("error adding peers to subnet: %w", err)
	}

	// 1.c configure DNS
	err = p.addDNSRecords(manifest.ID, subReqs, p.subnetManifest.DNSRecords)
	if err != nil {
		return manifest, fmt.Errorf("error adding dns records to subnet: %w", err)
	}

	// 1.d configure port mapping
	err = p.mapPorts(manifest.ID, subReqs)
	if err != nil {
		return manifest, fmt.Errorf("error adding port mappings to subnet: %w", err)
	}

	return manifest, nil
}

func (p *Provisioner) provisionAllocations(
	cfg jtypes.EnsembleConfig, manifest jtypes.EnsembleManifest,
) (jtypes.EnsembleManifest, error) {
	var wg sync.WaitGroup
	var aggErr error

	interim := map[string][]string{} // a map of verteces to edges (their dependencies)
	for allocName, allocCfg := range cfg.Allocations() {
		interim[allocName] = allocCfg.DependsOn
	}

	orderedAllocs, err := orderByDependency(interim)
	if err != nil {
		return manifest, err
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
					p.actor.Handle(),
					allocManifest.Handle,
					behaviors.AllocationStartBehavior,
					behaviors.AllocationStartRequest{
						SubnetIP:    p.subnetManifest.IndexRoutingTable[allocName],
						GatewayIP:   p.subnetManifest.GatewayIP,
						PortMapping: allocManifest.Ports,
					},
					actor.WithMessageExpiry(actor.MakeExpiry(AllocationStartTimeout)),
				)
				if err != nil {
					errCh <- fmt.Errorf("error creating allocation start message: %w", err)
					return
				}

				replyCh, err := p.actor.Invoke(msg)
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
				err := manifest.UpdateAllocation(allocName, func(alloc *jtypes.AllocationManifest) {
					alloc.Status = status
				})
				if err != nil {
					aggErr = err
				}
			}

			close(errCh)
			if aggregateErrors(errCh) != nil {
				return manifest, aggErr
			}
		}
	}

	return manifest, nil
}
