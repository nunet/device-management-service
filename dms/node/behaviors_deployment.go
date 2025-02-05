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
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

// MinDeploymentTime minimum time for deployment
const MinDeploymentTime = time.Minute - time.Second

var (
	ErrDeploymentNotFound     = errors.New("deployment not found")
	ErrorDeploymentNotRunning = errors.New("deployment is not running")
)

func (n *Node) handleVerifyEdgeConstraint(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.VerifyEdgeConstraintRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Warnf("unmarshalling constraint request: %s", err)
		n.sendReply(msg, jobs.VerifyEdgeConstraintResponse{
			OK:    false,
			Error: err.Error(),
		})
	}

	// TODO: implement
	// also add to docs (dms/behaviors/README.md, help-caps command and man page)
}

func (n *Node) commitDeployment(
	ensembleID, allocationID string,
	resources types.Resources, ports map[int]int,
) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	bid, ok := n.bids[ensembleID]
	if !ok {
		return fmt.Errorf("no bid requests for ensemble id: %s", ensembleID)
	}

	if bid.expire.Before(time.Now()) {
		return fmt.Errorf("bid request for ensemble id: %s has expired", ensembleID)
	}

	_, alreadyCommited := n.commitedResources[allocationID]
	if alreadyCommited {
		return nil
	}

	if err := n.resourceManager.CommitResources(context.TODO(), types.CommittedResources{
		AllocationID: allocationID,
		Resources:    resources,
	}); err != nil {
		return fmt.Errorf("preallocate resources for ensemble id: %s: %w", allocationID, err)
	}

	n.commitedResources[allocationID] = bid

	if len(ports) > 0 {
		for port := range ports {
			err := n.portAllocator.AllocatePorts(allocationID, []int{port})
			if err != nil {
				return fmt.Errorf("failed to allocate static ports: %w", err)
			}
		}
	}

	if bid.request.V1.PublicPorts.Dynamic > 0 {
		_, err := n.portAllocator.AllocateRandom(allocationID, bid.request.V1.PublicPorts.Dynamic)
		if err != nil {
			return fmt.Errorf("failed to allocate ports: %w", err)
		}
	}

	return nil
}

func (n *Node) handleCommitDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.CommitDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.CommitDeploymentResponse{}
	allocationID := jobs.ConstructAllocationID(request.EnsembleID, request.AllocationName)
	err := n.commitDeployment(request.EnsembleID, allocationID, request.Resources, request.PortMapping)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

type NewDeploymentRequest struct {
	Ensemble jobtypes.EnsembleConfig
}

type NewDeploymentResponse struct {
	Status     string
	EnsembleID string `json:",omitempty"`
	Error      string `json:",omitempty"`
}

func (n *Node) saveDeployment(orchestrator jobs.OrchestratorAPI) error {
	pvkey := orchestrator.ActorPrivateKey()

	pkRaw, err := crypto.MarshalPrivateKey(pvkey)
	if err != nil {
		return fmt.Errorf("convert priv key to raw: %w", err)
	}

	// TODO (not sensitive now): encrypt the orchestrator's pvkey before storing
	view := jobtypes.OrchestratorView{
		OrchestratorID:     orchestrator.ID(),
		Cfg:                orchestrator.Config(),
		Manifest:           orchestrator.Manifest(),
		Status:             orchestrator.Status(),
		DeploymentSnapshot: orchestrator.DeploymentSnapshot(),
		PrivKey:            pkRaw,
	}

	_, err = n.orchestratorRepo.Create(n.ctx, view)
	if err != nil {
		return fmt.Errorf("save deployment on database: %w", err)
	}

	return nil
}

func (n *Node) handleNewDeployment(msg actor.Envelope) {
	defer msg.Discard()

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

	childCtx := context.WithoutCancel(n.ctx)
	orchestrator, err := n.createOrchestrator(childCtx, request.Ensemble, n.actor)
	if err != nil {
		log.Warnf("creating orchestrator: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	log.Infof("deploying ensemble: %s", orchestrator.ID())
	n.sendReply(msg, NewDeploymentResponse{
		Status:     "OK",
		EnsembleID: orchestrator.ID(),
	})

	if err := orchestrator.Deploy(msg.Expiry().Add(-jobs.MinEnsembleDeploymentTime)); err != nil {
		orchestrator.Stop()
		log.Errorf("error creating ensemble: %s", err)
		n.orchestratorProvider.DeleteOrchestrator(orchestrator.ID())

		return
	}

	// save the deployment
	if err := n.saveDeployment(orchestrator); err != nil {
		log.Errorf("error saving deployment %s: %s", orchestrator.ID(), err)
	}
}

type DeploymentListResponse struct {
	Deployments map[string]string
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	var resp DeploymentListResponse

	resp.Deployments = make(map[string]string)
	for ID, dep := range n.orchestratorProvider.Orchestrators() {
		resp.Deployments[ID] = jobtypes.DeploymentStatusString(dep.Status())
	}

	n.sendReply(msg, resp)
}

type DeploymentLogsRequest struct {
	EnsembleID     string
	AllocationName string
}

type DeploymentLogsResponse struct {
	LogsWrittenTo string
	Error         string
}

func (n *Node) handleDeploymentLogs(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.EnsembleID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	data, err := o.GetAllocationLogs(request.AllocationName)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	ensembleDir := filepath.Join(
		n.dmsConfig.WorkDir,
		"deployments",
		request.EnsembleID,
	)
	allocDir := filepath.Join(ensembleDir, request.AllocationName)

	err = n.writeAllocationLogsTo(allocDir, data.Stdout, data.Stderr)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.LogsWrittenTo = allocDir
	n.sendReply(msg, resp)
}

type DeploymentStatusRequest struct {
	ID string
}

type DeploymentStatusResponse struct {
	Status string
	Error  string
}

func (n *Node) handleDeploymentStatus(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentStatusRequest
	var resp DeploymentStatusResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		// TODO: check database for persisted deployments data
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Status = jobtypes.DeploymentStatusString(o.Status())
	n.sendReply(msg, resp)
}

type DeploymentManifestRequest struct {
	ID string
}

type DeploymentManifestResponse struct {
	Manifest jobs.EnsembleManifest `json:"manifest"`
	Error    string                `json:"error,omitempty"`
}

func (n *Node) handleDeploymentManifest(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentManifestRequest
	var resp DeploymentManifestResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		// TODO: check database for persisted deployments data
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Manifest = o.Manifest()
	n.sendReply(msg, resp)
}

type DeploymentShutdownRequest struct {
	ID string
}

type DeploymentShutdownResponse struct {
	OK    bool
	Error string
}

func (n *Node) handleDeploymentShutdown(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentShutdownRequest
	var resp DeploymentShutdownResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if o.Status() != jobs.DeploymentStatusRunning {
		log.Debugf("deployment %q is not running(status=%q), cannot shutdown", request.ID, o.Status())
		// maybe-TODO: if it's still provisioning/committing,
		// we should stop the deployment process anyway
		resp.Error = ErrorDeploymentNotRunning.Error()
		n.sendReply(msg, resp)
		return
	}

	o.Shutdown()
	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) releaseCommit(allocID string) error {
	err := n.resourceManager.UncommitResources(context.TODO(), allocID)
	if err != nil {
		return fmt.Errorf("release resources for ensemble allocID: %s: %w", allocID, err)
	}

	n.portAllocator.Release(allocID)

	n.lock.Lock()
	delete(n.commitedResources, allocID)
	n.lock.Unlock()

	return nil
}

func (n *Node) handleRevertDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.RevertDeploymentMessage
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}
	ensembleID := request.EnsembleID

	// forget bid
	n.lock.Lock()
	delete(n.bids, ensembleID)
	n.lock.Unlock()

	for _, allocName := range request.AllocsByName {
		allocID := jobs.ConstructAllocationID(ensembleID, allocName)

		// try revert commit phase
		n.lock.Lock()
		_, ok := n.commitedResources[allocID]
		n.lock.Unlock()
		if ok {
			err := n.releaseCommit(allocID)
			if err != nil {
				log.Errorf("revert commit for ensemble id: %s: %s", ensembleID, err)
				// we have to try to revert other allocations too, so do not return
			}
		}

		// try revert allocations if exist
		_, err := n.getAllocation(allocID)
		if err == nil {
			// allocation exists
			err := n.releaseAllocation(allocID)
			if err != nil {
				log.Errorf("failed to revert allocation %s: %s", allocID, err)
			}
		}
	}
	log.Infof("deployment reverted: %+v", ensembleID)
}
