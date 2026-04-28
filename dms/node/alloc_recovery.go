// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	dockerexecutor "gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func (n *Node) createAllocStatePersist(ap jobtypes.AllocationsStatePersist) error {
	ctxT, cancel := context.WithTimeout(n.ctx, 5*time.Second)
	defer cancel()

	// check if record already exists for this ensemble
	query := n.allocsPersistRepo.GetQuery()
	query.Conditions = append(
		query.Conditions, repositories.EQ("EnsembleID", ap.EnsembleID))
	existing, err := n.allocsPersistRepo.FindAll(ctxT, query)
	if err != nil {
		return fmt.Errorf("failed to query existing persisted alloc record: %w", err)
	}

	switch len(existing) {
	case 0:
		// create new record
		ap := jobtypes.AllocationsStatePersist{
			EnsembleID:   ap.EnsembleID,
			Orchestrator: ap.Orchestrator,
			SubnetCIDR:   ap.SubnetCIDR,
			RoutingTable: ap.RoutingTable,
			Allocations:  make(map[string]jobtypes.AllocationState),
		}
		_, err = n.allocsPersistRepo.Create(ctxT, ap)
		if err != nil {
			return fmt.Errorf("failed to create persisted alloc record: %w", err)
		}
		log.Infof("created persisted alloc record for ensemble %s", ap.EnsembleID)
	case 1:
		log.Infof("persisted alloc record already exists for ensemble %s", ap.EnsembleID)
	default:
		log.Warnf("multiple persisted alloc records found for ensemble %s, expected only 1", ap.EnsembleID)
		return fmt.Errorf("multiple persisted alloc records found for ensemble %s", ap.EnsembleID)
	}

	return nil
}

func (n *Node) saveAllocationsState(as jobtypes.AllocationState) error {
	ctxT, cancel := context.WithTimeout(n.ctx, 5*time.Second)
	defer cancel()

	// check if record already exists for this ensemble
	query := n.allocsPersistRepo.GetQuery()
	query.Conditions = append(
		query.Conditions, repositories.EQ("EnsembleID", as.DeploymentID))
	existing, err := n.allocsPersistRepo.FindAll(ctxT, query)
	if err != nil {
		return fmt.Errorf("failed to query existing persisted alloc record: %w", err)
	}

	switch len(existing) {
	case 0:
		// create new record
		ap := jobtypes.AllocationsStatePersist{
			EnsembleID:   as.DeploymentID,
			Orchestrator: as.Orchestrator,
			Allocations:  map[string]jobtypes.AllocationState{as.AllocationID: as},
		}
		_, err = n.allocsPersistRepo.Create(ctxT, ap)
		if err != nil {
			return fmt.Errorf("failed to create persisted alloc record: %w", err)
		}
		log.Infof("created persisted alloc record for ensemble %s", ap.EnsembleID)
	case 1:
		// update existing record
		// only updating the allocations and orch info, keeping the same subnet info
		existing[0].Allocations[as.AllocationID] = as
		_, err = n.allocsPersistRepo.Update(ctxT, existing[0].ID, existing[0])
		if err != nil {
			return fmt.Errorf("failed to update persisted alloc record: %w", err)
		}
	default:
		log.Warnf("multiple persisted alloc records found for ensemble %s, expected only 1", as.DeploymentID)
	}

	return nil
}

// deletePersistedAllocationState removes the persisted state for a single
// allocation from the persisted ensemble record. If it was the last allocation,
// the whole persisted record is deleted.
func (n *Node) deletePersistedAllocationState(allocationID string) error {
	ctxT, cancel := context.WithTimeout(n.ctx, 5*time.Second)
	defer cancel()

	// XXX don't like fetching all
	all, err := n.allocsPersistRepo.FindAll(ctxT, n.allocsPersistRepo.GetQuery())
	if err != nil {
		return fmt.Errorf("failed to get all persisted alloc record for deletion: %w", err)
	}
	if len(all) == 0 {
		return nil // no records
	}
	for _, record := range all {
		if _, ok := record.Allocations[allocationID]; ok {
			existing := record
			delete(existing.Allocations, allocationID)

			switch len(existing.Allocations) {
			case 0:
				// nNo allocations left, delete the full record
				if err := n.allocsPersistRepo.Delete(ctxT, existing.ID); err != nil {
					return fmt.Errorf("failed to delete persisted alloc record: %w", err)
				}
				log.Infof("deleted persisted alloc record %s as it had no more allocations", existing.ID)
			default:
				_, err = n.allocsPersistRepo.Update(ctxT, existing.ID, existing)
				if err != nil {
					return fmt.Errorf("failed to update persisted alloc record: %w", err)
				}
				log.Infof("updated persisted alloc record %s after deleting allocation %s", existing.ID, allocationID)
			}
			return nil
		}
	}

	return nil
}

// evalPersistedAllocs: decides to restore allocations or not
func (n *Node) evalPersistedAllocs() {
	log.Infof("evalPersistedAllocs: start")
	defer log.Infof("evalPersistedAllocs: done")

	ctxT, cancel := context.WithTimeout(n.ctx, 5*time.Second)
	defer cancel()

	aps, err := n.allocsPersistRepo.FindAll(ctxT, n.allocsPersistRepo.GetQuery())
	if err != nil {
		log.Errorf("evalPersistedAllocs: unable to fetch persisted allocations: %v", err)
		n.readyForBids.Store(true)
		return
	}
	log.Infof("evalPersistedAllocs: found %d persisted allocations", len(aps))

	wg := sync.WaitGroup{}
	for _, persisted := range aps {
		ap := persisted
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.restoreAlloc(ap); err != nil {
				return
			}
			go n.restoreAllocConfirm(ap)
		}()
	}
	wg.Wait()
	n.readyForBids.Store(true)
}

// restoreAlloc restores allocations without requiring confirmation from orchestrator
// on failure the persisted record is removed and executions cleaned up
func (n *Node) restoreAlloc(ap jobtypes.AllocationsStatePersist) error {
	if ap.Orchestrator.Empty() || len(ap.Allocations) == 0 {
		log.Errorf("evalPersistedAllocs: persisted alloc %s has invalid orchestrator(%+v) or empty allocations (%d), deleting record", ap.EnsembleID, ap.Orchestrator, len(ap.Allocations))
		n.deleteAllocPersist(ap.ID)
		return fmt.Errorf("invalid persisted allocation record")
	}

	log.Infof("evalPersistedAllocs: restoring allocation id=%s ensemble=%s (local)", ap.ID, ap.EnsembleID)

	if err := n.restoreAllocations(ap); err != nil {
		log.Errorf("evalPersistedAllocs: failed to restore persisted alloc %s: %v", ap.ID, err)
		n.rmUnrecoveredExec(ap)
		n.deleteAllocPersist(ap.ID)
		return err
	}
	return nil
}

// restoreAllocConfirm waits for the orchestrator, confirms deployment state,
// it runs async so the node can continue while orch confirmation is running.
// restoration is rolledback on failure
func (n *Node) restoreAllocConfirm(ap jobtypes.AllocationsStatePersist) {
	ok, err := n.waitForOrch(ap)
	if err != nil {
		log.Errorf("evalPersistedAllocs: deployment state check failed for persisted alloc %s: %v", ap.ID, err)
		n.rollbackRestoredAlloc(ap)
		n.rmUnrecoveredExec(ap)
		n.deleteAllocPersist(ap.ID)
		return
	}
	if !ok {
		log.Infof("evalPersistedAllocs: rollback restored persisted allocation %s, orchestrator returned ok=false", ap.ID)
		n.rollbackRestoredAlloc(ap)
		n.rmUnrecoveredExec(ap)

		n.deleteAllocPersist(ap.ID)
		return
	}

	if err := n.createAllocStatePersist(ap); err != nil {
		log.Warnf("evalPersistedAllocs: ensure persisted alloc record after restore failed for ensemble %s: %v", ap.EnsembleID, err)
	}
}

func (n *Node) waitForOrch(ap jobtypes.AllocationsStatePersist) (bool, error) {
	const (
		initialCooldown = 15 * time.Second
		maxCooldown     = time.Hour
		giveUpAfter     = 3 * time.Hour
		dialTimeout     = 3 * time.Second
		queryTimeout    = time.Minute
		invokeRetries   = 3
		invokeRetryWait = 30 * time.Second
	)

	aIDs := make([]string, 0, len(ap.Allocations))
	for allocID := range ap.Allocations {
		aIDs = append(aIDs, types.AllocationNameFromID(allocID))
	}

	deploymentStateBehavior := fmt.Sprintf(behaviors.DeploymentStateBehavior, ap.EnsembleID)
	req := behaviors.DeploymentStateRequest{
		EnsembleID:       ap.EnsembleID,
		AllocationNamess: aIDs,
	}

	peerAddr := fmt.Sprintf("/p2p/%s", ap.Orchestrator.Address.HostID)
	deadline := time.Now().Add(giveUpAfter)
	wait := initialCooldown
	waitWithinDeadline := func(sleep time.Duration, dialAttempt int) error {
		if rem := time.Until(deadline); sleep > rem && rem > 0 {
			sleep = rem
		}
		if sleep <= 0 {
			return fmt.Errorf("unable to confirm deployment state within %v (%d attempts)", giveUpAfter, dialAttempt)
		}
		select {
		case <-time.After(sleep):
			return nil
		case <-n.ctx.Done():
			return fmt.Errorf("orchestrator state wait cancelled: %w", n.ctx.Err())
		}
	}

	for dialAttempt := 1; ; dialAttempt++ {
		log.Infof("evalPersistedAllocs: starting dial attempt %d for ensemble %s orchestrator %s", dialAttempt, ap.EnsembleID, ap.Orchestrator.Address.HostID)
		ctxT, cancelP := context.WithTimeout(n.ctx, dialTimeout)
		err := n.network.Connect(ctxT, peerAddr)
		cancelP()
		if err == nil {
			ctxP, cancelP := context.WithTimeout(n.ctx, dialTimeout)
			pingResult, pingErr := n.network.Ping(ctxP, ap.Orchestrator.Address.HostID, 3*time.Second)
			cancelP()
			if pingErr != nil || !pingResult.Success {
				err = fmt.Errorf("ping orchestrator: pingError: %w, pingResult: %+v", pingErr, pingResult)
			} else {
				for invokeAttempt := 1; invokeAttempt <= invokeRetries; invokeAttempt++ {
					log.Infof("evalPersistedAllocs: invoking deployment state behavior attempt %d/%d for ensemble %s", invokeAttempt, invokeRetries, ap.EnsembleID)
					reply, invokeErr := n.invokeBehaviour(
						ap.Orchestrator,
						deploymentStateBehavior,
						req,
						queryTimeout,
					)
					if invokeErr != nil {
						log.Warnf("evalPersistedAllocs: invoke deployment state behavior failed on attempt %d/%d for ensemble %s: %v", invokeAttempt, invokeRetries, ap.EnsembleID, invokeErr)
						if invokeAttempt == invokeRetries {
							log.Warnf("evalPersistedAllocs: invoke deployment state behavior failed after %d attempts (ping=successful) ensemble %s: %v", invokeRetries, ap.EnsembleID, invokeErr)
							return false, nil
						}
						err = fmt.Errorf("invoke deployment state behavior: %w", invokeErr)
						log.Infof("evalPersistedAllocs: waiting %s before retry after invoke failure", invokeRetryWait)
						if waitErr := waitWithinDeadline(invokeRetryWait, dialAttempt); waitErr != nil {
							log.Warnf("evalPersistedAllocs: waitWithinDeadline failed: %v - error: %v", waitErr, err)
							return false, fmt.Errorf("%w: %w", waitErr, err)
						}
						continue
					}

					if reply.Message == nil {
						err = fmt.Errorf("empty deployment state reply")
						log.Warnf("evalPersistedAllocs: empty deployment state reply for ensemble %s", ap.EnsembleID)
					} else {
						var resp behaviors.DeploymentStateResponse
						if unmarshalErr := json.Unmarshal(reply.Message, &resp); unmarshalErr != nil {
							err = fmt.Errorf("unmarshal deployment state response: %w", unmarshalErr)
							log.Warnf("evalPersistedAllocs: failed to unmarshal deployment state response for ensemble %s: %v", ap.EnsembleID, unmarshalErr)
						} else {
							log.Infof("evalPersistedAllocs: successfully received deployment state response ok=%v for ensemble %s", resp.OK, ap.EnsembleID)
							if dialAttempt > 1 {
								log.Infof("evalPersistedAllocs: deployment state confirmed after %d attempts for ensemble %s", dialAttempt, ap.EnsembleID)
							}
							log.Infof("evalPersistedAllocs: deployment state response ok=%v for alloc %s", resp.OK, ap.ID)
							return resp.OK, nil
						}
					}
					break
				}
			}
		}

		log.Debugf("evalPersistedAllocs: orch(%s) not ready for state check yet, attempt %d: %v", ap.Orchestrator.Address.HostID, dialAttempt, err)

		if time.Now().After(deadline) {
			log.Warnf("evalPersistedAllocs: deadline exceeded for ensemble %s after %d attempts", ap.EnsembleID, dialAttempt)
			return false, fmt.Errorf("unable to confirm deployment state within %v (%d attempts): %w", giveUpAfter, dialAttempt, err)
		}

		sleep := wait
		log.Infof("evalPersistedAllocs: orch(%s) state check pending, cool off %s before next attempt (max %s) - err: %v",
			ap.Orchestrator.Address.HostID, sleep, maxCooldown, err)

		if waitErr := waitWithinDeadline(sleep, dialAttempt); waitErr != nil {
			return false, fmt.Errorf("%w: %w", waitErr, err)
		}

		if tw := wait * 2; tw > maxCooldown {
			wait = maxCooldown
		} else {
			wait = tw
		}
	}
}

func (n *Node) rollbackRestoredAlloc(ap jobtypes.AllocationsStatePersist) {
	log.Infof("rollbackRestoredAllocation: rolling back ensemble %s", ap.EnsembleID)

	subnetIDs := make(map[string]struct{})
	subnetIDs[ap.EnsembleID] = struct{}{}
	for _, allocConf := range ap.Allocations {
		if allocConf.SubnetID != "" {
			subnetIDs[allocConf.SubnetID] = struct{}{}
		}
	}

	for subnetID := range subnetIDs {
		if err := n.network.DestroySubnet(subnetID); err != nil {
			log.Warnf("rollbackRestoredAllocation: failed to destroy subnet %s: %v", subnetID, err)
		}
	}

	for allocID := range ap.Allocations {
		if err := n.allocator.Release(context.Background(), allocID); err != nil {
			log.Errorw("revert_deployment_release_failure",
				"labels", []string{string(observability.LabelDeployment)},
				"ensembleID", ap.EnsembleID,
				"error", err)
		}
		if err := n.allocator.Uncommit(context.Background(), allocID); err != nil {
			log.Errorw("revert_deployment_uncommit_failure",
				"labels", []string{string(observability.LabelDeployment)},
				"ensembleID", ap.EnsembleID,
				"error", err,
			)
		}
	}

	log.Infow("restore_rolled_back",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", ap.EnsembleID,
	)
}

func (n *Node) rmUnrecoveredExec(ap jobtypes.AllocationsStatePersist) {
	for allocID, allocConf := range ap.Allocations {
		execType := strings.TrimSpace(strings.ToLower(allocConf.Execution.Type))
		if execType == "" {
			execType = types.ExecutorTypeDocker.String()
		}

		if execType != types.ExecutorTypeDocker.String() {
			log.Warnw("terminate_unrecoverable_execution_skipped",
				"allocationID", allocID,
				"executionType", execType,
				"reason", "unsupported executor type for explicit cleanup",
			)
			continue
		}

		metadata, err := n.getExecutor(jobtypes.AllocationExecutor(execType))
		if err != nil {
			log.Warnw("terminate_unrecoverable_execution_skipped",
				"allocationID", allocID,
				"executionType", execType,
				"reason", "executor unavailable",
				"error", err,
			)
			continue
		}

		dockerExec, ok := metadata.executor.(*dockerexecutor.Executor)
		if !ok {
			log.Warnw("terminate_unrecoverable_execution_skipped",
				"allocationID", allocID,
				"executionType", execType,
				"reason", "executor is not docker backend",
			)
			continue
		}

		ctxT, cancel := context.WithTimeout(n.ctx, 10*time.Second)
		err = dockerExec.TerminateByJobID(ctxT, allocID, 5)
		cancel()

		if err != nil {
			log.Warnw("terminate_unrecoverable_execution_failed",
				"allocationID", allocID,
				"executionType", execType,
				"error", err,
			)
			continue
		}

		log.Infow("terminate_unrecoverable_execution_success",
			"allocationID", allocID,
			"executionType", execType,
		)
	}
}

func (n *Node) deleteAllocPersist(recordID string) {
	ctxTD, cancelD := context.WithTimeout(n.ctx, 5*time.Second)
	defer cancelD()

	if err := n.allocsPersistRepo.Delete(ctxTD, recordID); err != nil {
		log.Errorf("evalPersistedAllocs: unable to delete persisted allocation record %s: %v", recordID, err)
		return
	}
	log.Infof("evalPersistedAllocs: deleted persisted allocation record %s", recordID)
}

func (n *Node) restoreAllocations(ap jobtypes.AllocationsStatePersist) error {
	allocDepCfg := make(map[string]jobtypes.AllocationDeploymentConfig)
	restoreErr := error(nil)
	defer func() {
		if restoreErr != nil {
			log.Errorf("restoreAllocations: failed to restore allocations: %v", restoreErr)
			n.rollbackRestoredAlloc(ap)
		}
	}()

	// commit
	for allocID, allocConf := range ap.Allocations {
		ctxT, cancel := context.WithTimeout(n.ctx, 5*time.Second)
		err := n.allocator.RestoreAllocCommit(
			ctxT, allocID, allocConf.Resources, allocConf.Ports, allocConf.DynamicPortsNum,
		)
		cancel()
		if err != nil {
			restoreErr = fmt.Errorf("restore allocation commit %s: %w", allocID, err)
			return restoreErr
		}

		privKeyBytes, err := base64.StdEncoding.DecodeString(allocConf.PrivKeyB64)
		if err != nil {
			restoreErr = fmt.Errorf("decode private key for allocation %s: %w", allocID, err)
			return restoreErr
		}

		identity, err := crypto.BytesToPrivateKey(privKeyBytes)
		if err != nil {
			restoreErr = fmt.Errorf("parse private key for allocation %s: %w", allocID, err)
			return restoreErr
		}

		// prepare allocation deployment config
		allocDepCfg[allocID] = jobtypes.AllocationDeploymentConfig{
			Type:             allocConf.Type,
			ProvisionScripts: allocConf.ProvisionScripts,
			Execution:        allocConf.Execution,
			Executor:         jobtypes.ExecutorDocker,
			Keys:             allocConf.Keys,
			Volume:           allocConf.Volume,
			Contracts:        allocConf.Contracts,
			NetState:         allocConf.NetState,
			Resources:        allocConf.Resources,
			Identity:         identity,
		}
	}

	allocs, err := n.createAllocations(ap.EnsembleID, allocDepCfg, ap.Orchestrator)
	if err != nil {
		restoreErr = fmt.Errorf("create allocations for persisted alloc %s: %w", ap.EnsembleID, err)
		return restoreErr
	}
	log.Infof("restored allocations: %+v", allocs)

	createdSubnets := make(map[string]struct{})
	for _, allocConf := range ap.Allocations {
		if _, ok := createdSubnets[allocConf.SubnetID]; !ok {
			err = n.network.CreateSubnet(allocConf.SubnetID, ap.SubnetCIDR, ap.RoutingTable)
			if err != nil {
				restoreErr = fmt.Errorf("create subnet on restore of %s: %w", ap.EnsembleID, err)
				return restoreErr
			}
			createdSubnets[allocConf.SubnetID] = struct{}{}
		}

		joinReq := orchestrator.SubnetJoinRequest{
			SubnetID:     allocConf.SubnetID,
			PeerID:       n.hostID,
			IP:           allocConf.NetState.SubnetIP,
			RoutingTable: allocConf.RoutingTable,
			Records:      allocConf.DNSRecords,
		}

		err = n.subnetJoin(joinReq)
		if err != nil {
			restoreErr = fmt.Errorf("join subnet on restore of %s: %w", ap.EnsembleID, err)
			return restoreErr
		}

		for _, allocPConf := range allocConf.PortMapping {
			err := n.network.MapPort(allocConf.SubnetID, allocPConf.Protocol, allocPConf.SourceIP, allocPConf.SourcePort, allocPConf.DestIP, allocPConf.DestPort)
			if err != nil {
				restoreErr = fmt.Errorf("map ports on restore of %s: %w", ap.EnsembleID, err)
				return restoreErr
			}
		}

		// TODO issue #1154 - better handle transient allocations
		subnetStatusMx.Lock()
		subnetStatus[allocConf.SubnetID] = 1
		subnetStatusMx.Unlock()

	}

	// add enssemble behaviors
	err = n.addEnsembleBehaviors(ap.EnsembleID)
	if err != nil {
		restoreErr = fmt.Errorf("add ensemble behaviors for restored alloc %s: %w", ap.EnsembleID, err)
		return restoreErr
	}

	for allocID := range allocs {
		alloc, err := n.allocator.GetAllocation(allocID)
		if err != nil {
			restoreErr = fmt.Errorf("get allocation %s after restore: %w", allocID, err)
			return restoreErr
		}
		if alloc == nil {
			restoreErr = fmt.Errorf("get allocation %s after restore: allocation is nil", allocID)
			return restoreErr
		}

		alloc.ApplyPersistedNetworkMetadata(
			ap.Allocations[allocID].SubnetID,
			ap.Allocations[allocID].RoutingTable,
			ap.Allocations[allocID].DNSRecords,
			ap.Allocations[allocID].PortMapping,
		)

		err = alloc.Run(n.ctx, ap.Allocations[allocID].NetState.SubnetIP, ap.Allocations[allocID].NetState.GatewayIP, ap.Allocations[allocID].NetState.PortMapping)
		if err != nil {
			restoreErr = fmt.Errorf("run allocation %s after restore: %w", allocID, err)
			return restoreErr
		}
	}

	return nil
}
