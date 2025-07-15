package orchestrator

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/multierr"

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
	err = o.deployNewNodes(o.cfg.Clone(), modifiedCfg, expiry)
	if err != nil {
		return fmt.Errorf("deploying new nodes: %w", err)
	}

	// 3. deploy new allocations into existent nodes
	// TODO
	// (if not best-efforts, we should revert deployed new nodes)

	// 4. start supervisor for new allocations
	o.supervisor.Update(jtypes.NewManifestReader(o.manifest))

	log.Warnf("Updated Manifest: %s", o.manifest)
	return nil
}

// deployNewNodes deployes *only new nodes* to the running ensemble.
//   - It does NOT update existent nodes with new allocations.
//   - It does NOT remove allocations from existent nodes
//
// It is similar to o.deploy but it adds:
//  1. extends subnet's routing table and dns records
//  2. updating all other nodes's subnets on the ensemble.
//
// It _implictly_ updates o.manifest by the use of o.commit and
// o.provision
//
// Another difference from o.deploy is that this function is not updating
// snapshots and ensemble status.
//
// TODO: 6. revert subnet updates
func (o *BasicOrchestrator) deployNewNodes(
	oldCfg, newCfg jtypes.EnsembleConfig, expiry time.Time,
) error {
	if len(identifyNewNodes(oldCfg, newCfg)) == 0 {
		return nil
	}
	log.Info("deploying added nodes")

	alreadyDeployedNodes := o.ManifestNodesPeerIDs()

	newConfig, err := newConfigForAddedNodes(
		alreadyDeployedNodes,
		oldCfg,
		newCfg,
	)
	if err != nil {
		return fmt.Errorf("creating new nodes config: %w", err)
	}

	addNodesAndAllocsToCfg := func() {
		for name, node := range newConfig.Nodes() {
			o.lock.Lock()
			o.cfg.AddNodeAndAllocations(name, node, newConfig.Allocations())
			o.lock.Unlock()
		}
	}

	updateManifest := func(manifest jtypes.EnsembleManifest) {
		currentManifest := o.Manifest()
		for n, node := range manifest.Nodes {
			currentManifest.Nodes[n] = node
			for _, alloc := range node.Allocations {
				currentManifest.Allocations[alloc] = manifest.Allocations[alloc]
			}
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

		candidate, err := bidC.bid(jtypes.NewEnsembleCfgReader(newConfig), expiry)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				log.Warnf("candidate deployment not found, redeploying: %v", err)
				continue deploy
			}

			return fmt.Errorf("failed to bid: %v", err)
		}

		newManifest := o.newManifest(newConfig)

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
		mfAfterSubnet, err := provisioner.provisionSubnet(updatedManifest)
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

		return nil
	}

	return nil
}

// handleEnsembleRemovals handles both removals of nodes and allocations in
// a best effort basis.
func (o *BasicOrchestrator) handleEnsembleRemovals(modifiedCfg jtypes.EnsembleConfig) error {
	if len(identifyRemovedNodes(o.cfg, modifiedCfg)) == 0 {
		return nil
	}

	var errs error
	log.Infof("removing nodes and allocations from config %+v", modifiedCfg.V1)

	// 1. teardown removed nodes
	removeNodesCfg, err := newConfigForRemovedNodes(
		o.cfg, modifiedCfg,
	)
	if err != nil {
		errs = multierr.Append(errs, err)
	} else {
		mf, err := manifestOnlyForNodes(o.manifest, utils.MapKeysToSlice(removeNodesCfg.Nodes()))
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			o.revert(removeNodesCfg, mf)
			o.removeNodesAllocationsFromCfg(utils.MapKeysToSlice(removeNodesCfg.Nodes()))
		}
	}

	// 2. teardown allocations from existent nodes
	// TODO

	return errs
}

func (o *BasicOrchestrator) removeNodesAllocationsFromCfg(nodes []string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, node := range nodes {
		o.cfg.RemoveNodeAndAllocations(node)
	}
}
