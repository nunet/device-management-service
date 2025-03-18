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
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/observability"
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

	var request orchestrator.VerifyEdgeConstraintRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Warnw("verify_edge_constraint_unmarshal_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, orchestrator.VerifyEdgeConstraintResponse{
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
		log.Errorw("commit_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, orchestrator.CommitDeploymentResponse{Error: err.Error()})
	}

	var request orchestrator.CommitDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	log.Infow("commit_deployment_started",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", request.EnsembleID)

	resp := orchestrator.CommitDeploymentResponse{}
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

func (n *Node) saveDeployment(orchestrator orchestrator.Orchestrator) error {
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
		log.Errorw("new_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, NewDeploymentResponse{Status: "ERROR", Error: err.Error()})
	}

	if time.Until(msg.Expiry()) < MinDeploymentTime {
		log.Debugf("deployment time too short")
		handleErr(errors.New("requested deployment time too short"))
		return
	}

	var request NewDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("unmarshal new deployment request: %s", err))
		return
	}

	childCtx := context.WithoutCancel(n.ctx)
	orch, err := n.createOrchestrator(childCtx, request.Ensemble, n.actor)
	if err != nil {
		log.Warnw("orchestrator_creation_failure",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		handleErr(err)
		return
	}

	log.Infow("new_ensemble_deployment_initiated",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", orch.ID())

	n.sendReply(msg, NewDeploymentResponse{
		Status:     "OK",
		EnsembleID: orch.ID(),
	})

	if err := orch.Deploy(msg.Expiry().Add(-orchestrator.MinEnsembleDeploymentTime)); err != nil {
		orch.Stop()
		log.Errorw("ensemble_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"ensembleID", orch.ID(),
			"error", err)
		n.orchestratorRegistry.DeleteOrchestrator(orch.ID())

		return
	}

	// save the deployment
	if err := n.saveDeployment(orch); err != nil {
		log.Errorw("save_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"ensembleID", orch.ID(),
			"error", err)
	}
}

type DeploymentListResponse struct {
	Deployments map[string]string
}

type DeploymentListRequest struct {
	Metadata map[string]string
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("deployment_list_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentListResponse{Deployments: make(map[string]string)})
	}

	var request DeploymentListRequest
	var resp DeploymentListResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment list request: %s", err))
		return
	}

	resp.Deployments = make(map[string]string)
	for ID, dep := range n.orchestratorRegistry.Orchestrators() {
		if len(request.Metadata) > 0 {
			manifest := dep.Manifest()
			shouldInclude := true

			for k, v := range request.Metadata {
				manifestValue, exists := manifest.Metadata[k]
				if !exists || manifestValue != v {
					shouldInclude = false
					break
				}
			}
			if !shouldInclude {
				continue
			}
		}
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
		log.Errorw("deployment_logs_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentLogsResponse{Error: err.Error()})
	}

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment logs: %s", err))
		return
	}

	o, err := n.orchestratorRegistry.GetOrchestrator(request.EnsembleID)
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
		log.Errorw("deployment_status_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentStatusResponse{Error: err.Error()})
	}

	var request DeploymentStatusRequest
	var resp DeploymentStatusResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment status: %s", err))
		return
	}

	o, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
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
		log.Errorw("deployment_manifest_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentManifestResponse{Error: err.Error()})
	}

	var request DeploymentManifestRequest
	var resp DeploymentManifestResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment manifest: %s", err))
		return
	}

	o, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
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
		log.Errorw("deployment_shutdown_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentShutdownResponse{Error: err.Error()})
	}

	var request DeploymentShutdownRequest
	var resp DeploymentShutdownResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	o, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
	if err != nil {
		handleErr(err)
		return
	}

	if o.Status() != jobs.DeploymentStatusRunning {
		log.Debugw("deployment_not_running_for_shutdown",
			"labels", []string{string(observability.LabelDeployment)},
			"deploymentID", request.ID,
			"status", o.Status())
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

	var request orchestrator.RevertDeploymentMessage
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("revert_deployment_unmarshal_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
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
			log.Errorw("revert_deployment_release_failure",
				"labels", []string{string(observability.LabelDeployment)},
				"ensembleID", ensembleID,
				"error", err)
		}
	}
	log.Infow("deployment_reverted",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", ensembleID)
}
