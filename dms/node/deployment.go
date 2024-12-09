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
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	NewDeploymentBehavior = "/dms/node/deployment/new"

	// Minimum time for deployment
	MinDeploymentTime = time.Minute - time.Second

	RestoreDeadlineCommitting   = 1 * time.Minute
	RestoreDeadlineProvisioning = 1 * time.Minute
	RestoreDeadlineRunning      = 5 * time.Minute
	bidStateGCInterval          = time.Minute
	bidStateTimeout             = 5 * time.Minute
)

type NewDeploymentRequest struct {
	Ensemble job_types.EnsembleConfig
}

type NewDeploymentResponse struct {
	Status     string
	EnsembleID string `json:",omitempty"`
	Error      string `json:",omitempty"`
}

func (n *Node) newDeployment(msg actor.Envelope) {
	defer msg.Discard()

	log.Infof("new deployment: %+v", msg)
	if time.Until(msg.Expiry()) < MinDeploymentTime {
		log.Debugf("deployment time too short")
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  "requested deployment time too short",
		})
		return
	}

	var request NewDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("unmarshalling deployment request: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	orchestrator, err := n.createOrchestrator(request.Ensemble)
	if err != nil {
		log.Warnf("creating orchestrator: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	n.mx.Lock()
	n.deployments[orchestrator.ID()] = orchestrator
	n.mx.Unlock()

	log.Infof("deploying ensemble: %s", orchestrator.ID())
	n.sendReply(msg, NewDeploymentResponse{
		Status:     "OK",
		EnsembleID: orchestrator.ID(),
	})

	if err := orchestrator.Deploy(msg.Expiry().Add(-jobs.MinEnsembleDeploymentTime)); err != nil {
		orchestrator.Stop()
		log.Errorf("error creating ensemble: %s", err)
		n.mx.Lock()
		delete(n.deployments, orchestrator.ID())
		n.mx.Unlock()

		return
	}

	// save the deployment
	n.mx.Lock()
	if err := n.saveDeployment(orchestrator); err != nil {
		log.Errorf("error saving deployment %s: %s", orchestrator.ID(), err)
	}
	n.mx.Unlock()
}

func (n *Node) deploymentVerifyEdgeConstraint(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.VerifyEdgeConstraintRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Warnf("error unmarshalling constraint request: %s", err)
		n.sendReply(msg, jobs.VerifyEdgeConstraintResponse{
			OK:    false,
			Error: err.Error(),
		})
	}

	// TODO
}

func (n *Node) createOrchestrator(ensemble job_types.EnsembleConfig) (*jobs.Orchestrator, error) {
	orch, err := jobs.NewOrchestrator(n.actor, n.network, ensemble)
	if err != nil {
		return nil, err
	}

	return orch, nil
}

func (n *Node) saveDeployments() error {
	n.mx.Lock()
	defer n.mx.Unlock()

	var failed []string
	for oid, deployment := range n.deployments {
		if err := n.saveDeployment(deployment); err != nil {
			log.Errorf("error saving deployment %s: %s", oid, err)
			failed = append(failed, oid)
		}
	}

	if len(failed) != 0 {
		return fmt.Errorf("failed to save deployments: %v", failed)
	}

	return nil
}

func (n *Node) saveDeployment(deployment *jobs.Orchestrator) error {
	view := job_types.OrchestratorView{
		DeploymentID:       deployment.ID(),
		Cfg:                deployment.Config(),
		Manifest:           deployment.Manifest(),
		Status:             deployment.Status(),
		DeploymentSnapshot: deployment.DeploymentSnapshot(),
	}

	_, err := n.orchestratorRepo.Create(n.ctx, view)
	if err != nil {
		return fmt.Errorf("couldn't save deployment on database: %w", err)
	}

	return nil
}

func (n *Node) restoreDeployments() error {
	query := n.orchestratorRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.LTE("Status", job_types.DeploymentStatusRunning),
	)

	orchestratorsViews, err := n.orchestratorRepo.FindAll(n.ctx, query)
	if err != nil {
		if err == repositories.ErrNotFound {
			return nil
		}
		return fmt.Errorf("couldn't query the database for hanging deployments: %w", err)
	}

	var failedToRestore []string
	for _, d := range orchestratorsViews {
		if d.Status < job_types.DeploymentStatusCommitting {
			log.Warnf(
				"deployment %s will not be restaured because it was not previously committed",
				d.DeploymentID,
			)
			continue
		}

		// Check restore deadline based on deployment status
		// TODO: on the compute provider side, if the deployer stops to answer
		// for more than the restore deadlines, they should free any resources alocated
		// and consider the deployment as canceled
		var restoreDeadline time.Duration
		switch d.Status {
		case job_types.DeploymentStatusCommitting:
			restoreDeadline = RestoreDeadlineCommitting
		case job_types.DeploymentStatusProvisioning:
			restoreDeadline = RestoreDeadlineProvisioning
		case job_types.DeploymentStatusRunning:
			restoreDeadline = RestoreDeadlineRunning
		default:
			log.Warnf("Unknown restorable deployment status for %s, skipping restoration", d.DeploymentID)
			continue
		}

		if time.Since(d.CreatedAt) > restoreDeadline {
			log.Warnf("Deployment %s has exceeded its restore deadline, skipping restoration", d.DeploymentID)
			continue
		}

		orchestrator, err := jobs.RestoreDeployment(n.actor, n.network, d.DeploymentID, d.Cfg, d.Manifest, d.Status, d.DeploymentSnapshot)
		if err != nil {
			log.Errorf("couldn't restore orchestrator of id %s; Error: %w", d.DeploymentID, err)
			failedToRestore = append(failedToRestore, d.DeploymentID)
			continue
		}

		log.Debugf("deployment %s restored!\n", orchestrator.ID())
		n.deployments[orchestrator.ID()] = orchestrator
	}

	if len(failedToRestore) > 0 {
		return fmt.Errorf("failed to restore the following deployment(s): %v", failedToRestore)
	}

	return nil
}

func (n *Node) handleBidRequest(msg actor.Envelope) {
	defer msg.Discard()

	log.Debugf("got a bid request from: %+v", &msg.From.Address)

	onboarded, err := n.onboarder.IsOnboarded()
	if err != nil {
		log.Debugf("got some error while checking onboarding: %w", err)
		return
	}
	if !onboarded {
		log.Debugf("node not onboarded. ignoring bid request...")
		return
	}

	var request job_types.EnsembleBidRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	_, bidExists := n.getBid(request.ID)
	if bidExists {
		log.Debugf("bid already sent for request: %s", request.ID)
		return
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		log.Debugf("failed to get machine resources")
		return
	}

	// randomize the order of bid request checks
	rand.Shuffle(len(request.Request), func(i, j int) {
		request.Request[i], request.Request[j] = request.Request[j], request.Request[i]
	})

	// find the first bid request that matches
	var toAnswer job_types.BidRequest
	var found bool
loop:
	for _, v := range request.Request {
		// check if it is a V1 request
		if v.V1 == nil {
			log.Debug("bid request not v1")
			continue
		}

		// check if we are excluded
		hostID := n.actor.Handle().Address.HostID
		for _, p := range request.PeerExclusion {
			if p == hostID {
				log.Debug("bid request has exclusion")
				continue loop
			}
		}

		// if the desired executable is not found stop
		for _, e := range v.V1.Executors {
			executor, err := n.getExecutor(e)
			if err != nil {
				log.Debugf("failed to get executor: %+v", e)
				continue loop
			}
			if executor.executionType == job_types.ExecutorDocker {
				if v.V1.GeneralRequirements.PrivilegedDocker {
					if !n.dmsConfig.AllowPrivilegedDocker {
						log.Debugf("node does not allow privileged docker")
						continue loop
					}
				}
			}
		}

		comparisonResult, err := machineResources.Compare(v.V1.Resources)
		if err != nil {
			log.Debugf("failed to compare machine resources")
			continue loop
		}

		if comparisonResult != types.Better {
			log.Debugf("resource comparison - not better - result: %+v", comparisonResult)
			continue
		}

		found = true
		toAnswer = v
		break
	}

	if !found {
		log.Debugf("bid requirements were not satisfied")
		return
	}

	// TODO-MR: handle static port allocation via port allocator

	// handle dynamic port allocs
	// TODO: dynamic port allocs should be on committing phase
	allocKey := request.ID
	ports, err := n.portAllocator.Allocate(allocKey, toAnswer.V1.PublicPorts.Dynamic)
	if err != nil {
		log.Debugf("failed to allocate ports")
		return
	}
	cleanup := func() {
		n.portAllocator.Release(allocKey)
	}

	log.Debugf("signing bid with did: %+v", n.actor.Security().DID())
	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		cleanup()
		return
	}
	log.Debugf("signing bid with provider: %+v", provider)

	bid := job_types.Bid{
		V1: &job_types.BidV1{
			EnsembleID: request.ID,
			NodeID:     toAnswer.V1.NodeID,
			Peer:       n.hostID,
			Location: job_types.Location{
				Region:  n.hostLocation.HostContinent,
				Country: n.hostLocation.HostCountry,
				City:    n.hostLocation.HostCity,
			},
			Handle: n.actor.Handle(),
		},
	}

	err = bid.Sign(provider)
	if err != nil {
		cleanup()
		return
	}

	n.sendReply(msg, bid)
	n.rememberBid(request.ID, toAnswer, ports)
}

func (n *Node) handleRevertDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.RevertDeploymentMessage
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}
	ensembleID := request.EnsembleID

	// try revert bid if exists
	// TODO: remove this part if port allocation be moved to committing phase
	n.mx.Lock()
	_, ok := n.bids[ensembleID]
	n.mx.Unlock()
	if ok {
		n.portAllocator.Release(ensembleID)
	}

	// try revert commit phase
	n.mx.Lock()
	_, ok = n.commitedResources[ensembleID]
	n.mx.Unlock()
	if ok {
		err := n.releaseCommit(ensembleID)
		if err != nil {
			log.Errorf("failed to revert commit for ensemble id: %s: %s", ensembleID, err)
			// we have to try to revert allocation too, so do not return
		}
	}

	// try revert allocations if exist
	for _, allocID := range request.AllocationsIDs {
		err := n.releaseAllocation(allocID)
		if err != nil {
			log.Errorf("failed to revert allocation %s: %s", allocID, err)
		}
	}
	log.Infof("deployment reverted: %+v", ensembleID)
}

func (n *Node) releaseCommit(eid string) error {
	err := n.resourceManager.UncommitResources(context.TODO(), eid)
	if err != nil {
		return fmt.Errorf("failed to release resources for ensemble id: %s: %w", eid, err)
	}

	n.mx.Lock()
	delete(n.commitedResources, eid)
	n.mx.Unlock()

	return nil
}

func (n *Node) releaseAllocation(allocID string) error {
	n.mx.Lock()
	alloc, ok := n.allocations[allocID]
	n.mx.Unlock()
	if !ok {
		log.Debugf("allocation %s not found (it may be already released)", allocID)
		return nil
	}

	err := alloc.Stop(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to stop allocation %s: %w", allocID, err)
	}

	n.mx.Lock()
	delete(n.allocations, allocID)
	n.mx.Unlock()

	return nil
}

func (n *Node) rememberBid(eid string, req job_types.BidRequest, ports []int) {
	n.mx.Lock()
	defer n.mx.Unlock()

	_, exists := n.bids[eid]
	if exists {
		// we have an older bid
		n.portAllocator.Release(eid)
	}

	n.bids[eid] = &bidState{
		expire:  time.Now().Add(bidStateTimeout),
		request: req,
		ports:   ports,
	}
}

func (n *Node) getBid(eid string) (*bidState, bool) {
	n.mx.Lock()
	defer n.mx.Unlock()

	b, exists := n.bids[eid]

	return b, exists
}

func (n *Node) gcBidState() {
	ticker := time.NewTicker(bidStateGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.doGCBidState()

		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) doGCBidState() {
	now := time.Now()

	n.mx.Lock()
	defer n.mx.Unlock()

	for k, bs := range n.bids {
		if bs.expire.Before(now) {
			n.portAllocator.Release(k)
			delete(n.bids, k)
		}
	}
}
