package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// Update updates a running ensemble.
//
// Removed allocations/nodes shall NOT be reverted in case
// new deployments fail.
//
// TODO: if one of the deployment fails, revert all other deployments
//
// TODO: we may have to see how to handle DependsOn, for now we'll ignore it
func (o *BasicOrchestrator) Update(modifiedCfg jtypes.EnsembleConfig, expiry time.Time) error {
	o.setStatus(jtypes.DeploymentStatusUpdating)
	defer o.setStatus(jtypes.DeploymentStatusRunning)

	if time.Now().After(expiry) {
		return fmt.Errorf("update expiry time has already passed")
	}

	if err := modifiedCfg.Validate(); err != nil {
		return fmt.Errorf("invalid ensemble configuration: %w", err)
	}

	err := validateEnsembleUpdate(o.cfg, modifiedCfg)
	if err != nil {
		return fmt.Errorf("invalid ensemble update: %w", err)
	}

	// 0. Save current state for potential rollback
	// currentConfig := o.cfg.Clone()
	// currentManifest := o.manifest.Clone()

	// 1. teardown removed nodes and allocations
	err = o.handleEnsembleRemovals(modifiedCfg)
	if err != nil {
		return fmt.Errorf("handling ensemble removals: %w", err)
	}

	// 2. deploy new nodes
	err = o.handleNewAllocations(modifiedCfg, expiry)
	if err != nil {
		return fmt.Errorf("deploying new nodes: %w", err)
	}

	// 4. start supervisor for new allocations
	o.supervisor.Update(jtypes.NewManifestReader(o.manifest))

	log.Warnf("Updated Manifest: %s", o.manifest)
	return nil
}

// handleNewAllocations deployes new allocations to the running ensemble.
//   - It does NOT remove allocations from existent nodes
//
// It is similar to o.deploy but it adds:
//  1. extends subnet's routing table and dns records
//  2. updating all other nodes's subnets on the ensemble.
//
// It _implictly_ updates o.manifest by the use of o.commit and
// o.provision
//
// TODO: 6. revert subnet updates
func (o *BasicOrchestrator) handleNewAllocations(
	modifiedCfg jtypes.EnsembleConfig, expiry time.Time,
) error {
	existingNodes := make(map[string]string)
	for n, node := range o.manifest.Nodes {
		existingNodes[n] = node.Peer
	}

	newConfig, err := newConfigForDeploymentUpdate(
		o.cfg,
		modifiedCfg,
		existingNodes,
	)
	if err != nil {
		return fmt.Errorf("creating new nodes config: %w", err)
	}

	if len(newConfig.Allocations()) == 0 {
		return nil
	}

	addNodesAndAllocsToCfg := func() {
		for name := range newConfig.Nodes() {
			if node, ok := modifiedCfg.Node(name); ok {
				o.lock.Lock()
				o.cfg.AddNodeAndAllocations(name, node, newConfig.Allocations())
				o.lock.Unlock()
			}
		}
	}

	updateManifest := func(manifest jtypes.EnsembleManifest) {
		currentManifest := o.Manifest()
		for n, node := range manifest.Nodes {
			for _, alloc := range node.Allocations {
				currentManifest.Allocations[alloc] = manifest.Allocations[alloc]
			}
			if nmf, ok := currentManifest.Node(n); ok {
				node.Allocations = append(node.Allocations, nmf.Allocations...)
			}
			currentManifest.Nodes[n] = node
		}
		o.updateManifest(currentManifest)
	}

deploy:
	for time.Now().Before(expiry) {
		// 1. bid
		bidC, err := NewBidCoordinator(o.id, o.actor)
		if err != nil {
			return fmt.Errorf("failed to create bidder: %w", err)
		}

		bidC.getNonce()
		candidate, err := bidC.bid(jtypes.NewEnsembleCfgReader(newConfig), o.DeploymentSnapshot().Candidates, expiry)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				log.Warnf("candidate deployment not found, redeploying: %v", err)
				continue deploy
			}

			return fmt.Errorf("failed to bid: %v", err)
		}

		newManifest := o.newManifest(newConfig)
		for alloc, amf := range o.manifest.Allocations {
			newManifest.Allocations[alloc] = amf
		}

		// 2. commit
		committer := NewCommitter(o.ctx, o.id, o.actor)
		updatedManifest, err := committer.commit(
			jtypes.NewEnsembleCfgReader(newConfig),
			jtypes.NewManifestReader(newManifest), candidate)
		if err != nil {
			log.Warnf("committing for new nodes: %w", err)
			continue deploy
		}

		// 3. extend subnet manifest with new nodes
		provisioner := NewProvisioner(o.ctx, o.cancel, o.actor, o.subnetManifest)
		addedDNSRecords := make(map[string]string, len(updatedManifest.Allocations))
		routingTableExtension := make(map[string]string, len(updatedManifest.Allocations))
		for allocName, alloc := range updatedManifest.Allocations {
			err := provisioner.addAllocationToSubnet(updatedManifest, allocName)
			if err != nil {
				log.Warnf(
					"error adding allocation %s to subnet: %w",
					allocName, err)

				err := o.removeAllocationsFromSubnet(
					updatedManifest,
					utils.MapKeysToSlice(updatedManifest.Allocations),
				)
				if err != nil {
					log.Warnf("error removing allocations from subnet: %w", err)
				}
				continue deploy
			}

			ip, ok := o.getAllocIP(allocName)
			if !ok {
				log.Warnf("allocation %s not found in subnet", allocName)
			}

			addedDNSRecords[alloc.DNSName] = ip
			routingTableExtension[ip] = updatedManifest.Nodes[alloc.NodeID].Peer
		}

		// 4. provision subnet
		skip := utils.MapKeysToSlice(existingNodes)
		mfAfterSubnet, err := provisioner.provisionSubnet(updatedManifest, skip...)
		if err != nil {
			o.revert(newConfig, updatedManifest)
			log.Warnf("provision subnet for new nodes (will revert deployment): %w", err)
			continue deploy
		}

		// 5. update existent nodes with new subnet information
		err = provisioner.updateSubnetAllocations(mfAfterSubnet, addedDNSRecords, routingTableExtension)
		if err != nil {
			o.revert(newConfig, updatedManifest)
			provisioner.revertSubnetAllocationsUpdate(mfAfterSubnet, addedDNSRecords, routingTableExtension)
			log.Warnf("updating subnet allocations for new nodes (will revert deployment): %w", err)
			continue deploy
		}

		// 6. provision allocations
		mfAFterProvisionAllocs, err := provisioner.provisionAllocations(newConfig, mfAfterSubnet)
		if err != nil {
			o.revert(newConfig, updatedManifest)
			provisioner.revertSubnetAllocationsUpdate(mfAfterSubnet, addedDNSRecords, routingTableExtension)
			log.Warnf("provisioning allocations for new nodes (will revert deployment): %w", err)
			continue deploy
		}

		log.Info("updated: new nodes deployed successfully")

		// 7. update config and manifest with added nodes
		updateManifest(mfAFterProvisionAllocs)
		addNodesAndAllocsToCfg()
		o.deploymentSnapshot.Candidates = candidate
		return nil
	}

	return nil
}

// handleEnsembleRemovals handles both removals of nodes and allocations in
// a best effort basis.
func (o *BasicOrchestrator) handleEnsembleRemovals(modifiedCfg jtypes.EnsembleConfig) error {
	var errs error
	log.Infof("removing nodes and allocations from config %+v", modifiedCfg.V1)

	// 1. teardown removed nodes
	removeNodesCfg, err := newConfigForRemovedNodes(
		o.cfg, modifiedCfg,
	)
	if err != nil {
		errs = multierr.Append(errs, err)
	} else if len(removeNodesCfg.Nodes()) > 0 {
		mf, err := manifestOnlyForNodes(o.manifest, utils.MapKeysToSlice(removeNodesCfg.Nodes()))
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			o.revert(removeNodesCfg, mf)
			o.removeNodesAllocationsFromCfg(utils.MapKeysToSlice(removeNodesCfg.Nodes()))
		}
	}

	// 2. teardown allocations from existent nodes
	for n := range o.cfg.Nodes() {
		allocs := identifyRemovedAllocations(o.cfg, modifiedCfg, n)
		if len(allocs) == 0 {
			continue
		}

		errCh := make(chan error, len(allocs))
		wg := sync.WaitGroup{}
		for alloc := range allocs {
			amf, ok := o.manifest.Allocation(alloc)
			if !ok {
				errCh <- fmt.Errorf("allocation %s not found in manifest", alloc)
				continue
			}
			wg.Add(1)
			go func(h actor.Handle, allocID string) {
				defer wg.Done()
				msg, err := actor.Message(
					o.actor.Handle(),
					h,
					fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, o.manifest.ID),
					AllocationStopRequest{
						AllocationID: allocID,
					},
					actor.WithMessageExpiry(actor.MakeExpiry(AllocationShutdownTimeout)),
				)
				if err != nil {
					log.Errorf("error creating stop message for alloc: %s: %v", allocID, err)
					errCh <- err
					return
				}

				// invoke the stop message
				replyCh, err := o.actor.Invoke(msg)
				if err != nil {
					log.Errorf("error invoking stop message for %s: %v", allocID, err)
					errCh <- err
					return
				}

				// wait for the reply
				var reply actor.Envelope
				select {
				case reply = <-replyCh:
					defer reply.Discard()
					var resp AllocationStopResponse
					if err := json.Unmarshal(reply.Message, &resp); err != nil {
						log.Errorf("error unmarshalling stop allocation response: %s", err)
						errCh <- err
						return
					}
					if !resp.OK {
						log.Errorf("failed to stop allocation %s", allocID)
						errCh <- fmt.Errorf("failed to stop allocation %s", allocID)
						return
					}
				case <-time.After(AllocationShutdownTimeout):
					log.Errorf("timeout stopping allocation %s", allocID)
					errCh <- fmt.Errorf("timeout stopping allocation %s", allocID)
					return
				}
				log.Infof("allocation %s stopped", allocID)
			}(o.manifest.Nodes[n].Handle, amf.ID)
		}
		wg.Wait()
		close(errCh)

		err := o.removeAllocationsFromSubnet(o.manifest, utils.MapKeysToSlice(allocs))
		if err != nil {
			log.Errorf(
				"removeNodeFromManifest: error removing allocations from subnet: %v",
				err)
		}
		func(allocs []string) {
			o.lock.Lock()
			defer o.lock.Unlock()
			for _, alloc := range allocs {
				delete(o.manifest.Allocations, alloc)
				delete(o.cfg.V1.Allocations, alloc)
			}
			nodeAllocs := modifiedCfg.V1.Nodes[n].Allocations
			nmf := o.manifest.Nodes[n]
			nmf.Allocations = nodeAllocs
			o.manifest.Nodes[n] = nmf

			ncfg := o.cfg.V1.Nodes[n]
			ncfg.Allocations = nodeAllocs
			o.cfg.V1.Nodes[n] = ncfg
		}(utils.MapKeysToSlice(allocs))

		if errCh != nil {
			errs = multierr.Append(errs, aggregateErrors(errCh))
		}
	}
	if errs != nil {
		log.Errorf("error removing allocations for nodes: %v", errs)
	}

	return errs
}

func (o *BasicOrchestrator) removeNodesAllocationsFromCfg(nodes []string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, node := range nodes {
		o.cfg.RemoveNodeAndAllocations(node)
	}
}
