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
	resources types.CommittedResources, ports map[int]int,
) error {
	bid, ok := n.getBid(ensembleID)
	if !ok {
		return fmt.Errorf("no bid requests for ensemble id: %s", ensembleID)
	}

	n.lock.Lock()
	defer n.lock.Unlock()

	if bid.expire.Before(time.Now()) {
		return fmt.Errorf("bid request for ensemble id: %s has expired", ensembleID)
	}

	if err := n.allocator.Commit(context.Background(), allocationID, resources, ports, bid.request.V1.PublicPorts.Dynamic, bid.expire.Unix()); err != nil {
		return fmt.Errorf("commit resources for ensemble allocID: %s: %w", allocationID, err)
	}

	return nil
}

func (n *Node) handleCommitDeployment(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error committing deployment: %v", err)
		n.sendReply(msg, jobs.CommitDeploymentResponse{Error: err.Error()})
	}

	var request jobs.CommitDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	log.Infof("committing deployment for %s", request.EnsembleID)

	resp := jobs.CommitDeploymentResponse{}
	allocationID := types.ConstructAllocationID(request.EnsembleID, request.AllocationName)
	request.Resources.AllocationID = allocationID
	err := n.commitDeployment(request.EnsembleID, allocationID, request.Resources, request.PortMapping)
	if err != nil {
		handleErr(err)
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

	handleErr := func(err error) {
		log.Errorf("Error in new deployment: %s", err)
		n.sendReply(msg, NewDeploymentResponse{Status: "ERROR", Error: err.Error()})
	}

	if time.Until(msg.Expiry()) < MinDeploymentTime {
		log.Debugf("deployment time too short")
		handleErr(errors.New("requested deployment time too short"))
		return
	}

	var request NewDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("unmarshalling deployment request: %s", err)
		handleErr(fmt.Errorf("error unmarshalling deployment request: %s", err))
		return
	}

	childCtx := context.WithoutCancel(n.ctx)
	orchestrator, err := n.createOrchestrator(childCtx, request.Ensemble, n.actor)
	if err != nil {
		log.Warnf("creating orchestrator: %s", err)
		handleErr(err)
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

	handleErr := func(err error) {
		log.Errorf("Error handling deployment logs: %s", err)
		n.sendReply(msg, DeploymentLogsResponse{Error: err.Error()})
	}

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment logs: %s", err))
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.EnsembleID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get orchestrator: %s", err))
		return
	}

	data, err := o.GetAllocationLogs(request.AllocationName)
	if err != nil {
		handleErr(fmt.Errorf("failed to get allocation logs: %s", err))
		return
	}

	allocDir, err := o.WriteAllocationLogs(request.AllocationName, data.Stdout, data.Stderr)
	if err != nil {
		handleErr(fmt.Errorf("failed to write allocation logst: %s", err))
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

	handleErr := func(err error) {
		log.Errorf("Error handling deployment status: %s", err)
		n.sendReply(msg, DeploymentStatusResponse{Error: err.Error()})
	}

	var request DeploymentStatusRequest
	var resp DeploymentStatusResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment status: %s", err))
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		// TODO: check database for persisted deployments data
		handleErr(fmt.Errorf("failed to get orchestrator: %s", err))
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

	handleErr := func(err error) {
		log.Errorf("Error handling deployment manifest: %s", err)
		n.sendReply(msg, DeploymentManifestResponse{Error: err.Error()})
	}

	var request DeploymentManifestRequest
	var resp DeploymentManifestResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment manifest: %s", err))
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		// TODO: check database for persisted deployments data
		handleErr(err)
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

	handleErr := func(err error) {
		log.Errorf("Error shutting down deployment: %s", err)
		n.sendReply(msg, DeploymentShutdownResponse{Error: err.Error()})
	}

	var request DeploymentShutdownRequest
	var resp DeploymentShutdownResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	o, err := n.orchestratorProvider.GetOrchestrator(request.ID)
	if err != nil {
		handleErr(err)
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
	delete(n.answeredBids, ensembleID)
	n.lock.Unlock()

	err := n.network.DestroySubnet(request.EnsembleID)
	if err != nil {
		log.Warnf("failed to destroy subnet for ensemble id: %s: %v (it may not have been created or may already been destroyed)", ensembleID, err)
	}

	for _, allocName := range request.AllocsByName {
		allocID := types.ConstructAllocationID(ensembleID, allocName)
		err := n.allocator.Release(context.Background(), allocID)
		if err != nil {
			log.Errorf("revert commit for ensemble id: %s: %s", ensembleID, err)
		}
	}
	log.Infof("deployment reverted: %+v", ensembleID)
}
