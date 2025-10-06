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
	"strconv"
	"strings"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// Deployment behavior data source strategy:
// - Store-based (primary source of truth): handleDeploymentList, handleDeploymentStatus, handleDeploymentManifest
// - In-memory registry (requires active orchestrator): handleDeploymentLogs, handleDeploymentShutdown, handleDeploymentUpdate
// - Store + in-memory: handleNewDeployment (creates in memory, auto-saved to store via status watcher)

// MinDeploymentTime minimum time for deployment
const (
	MinDeploymentTime       = time.Minute - time.Second
	MinUpdateDeploymentTime = 2 * (time.Minute - time.Second) // TODO: tune this
)

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
	err := n.commitDeployment(request.EnsembleID, request.AllocationName, request.Resources, request.PortMapping)
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
	err := n.orchestratorRegistry.SaveOrchestrator(orchestrator)
	if err != nil {
		return fmt.Errorf("save deployment: %w", err)
	}

	log.Debugf("deployment %s of status %s saved", orchestrator.ID(),
		orchestrator.Status().String())

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

	orch, err := n.createOrchestrator(n.ctx, request.Ensemble)
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
		// Orchestrator status is automatically saved to store via status watcher
		orch.Stop()
		log.Errorw("ensemble_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"ensembleID", orch.ID(),
			"error", err)
		n.orchestratorRegistry.DeleteOrchestrator(orch.ID())

		return
	}

	// Orchestrator status is automatically saved to store via status watcher
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

	// Get all deployments from store (primary source of truth)
	allDeployments, err := n.orchestratorRegistry.GetAllDeployments()
	if err != nil {
		handleErr(fmt.Errorf("failed to get deployments from store: %w", err))
		return
	}

	for _, deployment := range allDeployments {
		if shouldIncludeDeployment(deployment, request.Metadata) {
			resp.Deployments[deployment.OrchestratorID] = deployment.Status.String()
		}
	}

	n.sendReply(msg, resp)
}

// shouldIncludeDeployment checks if a deployment should be included based on metadata filter
func shouldIncludeDeployment(deployment *jobtypes.OrchestratorView, metadataFilter map[string]string) bool {
	if len(metadataFilter) == 0 {
		return true
	}

	// Check if all metadata filter conditions are met
	for k, v := range metadataFilter {
		manifestValue, exists := deployment.Manifest.Metadata[k]
		if !exists || manifestValue != v {
			return false
		}
	}

	return true
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

	// For logs, we need an active orchestrator (in-memory registry only)
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

	// Read deployment status from store (primary source of truth)
	deployment, err := n.orchestratorRegistry.GetDeployment(request.ID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get deployment: %s", err))
		return
	}

	resp.Status = deployment.Status.String()
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

	// Read deployment manifest from store (primary source of truth)
	deployment, err := n.orchestratorRegistry.GetDeployment(request.ID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get deployment: %s", err))
		return
	}

	resp.Manifest = deployment.Manifest
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

	err = o.Shutdown()
	if err != nil {
		handleErr(err)
		return
	}

	// force status update and ignore status watcher
	if err := n.orchestratorRegistry.SaveOrchestrator(o); err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleDeploymentRevert(msg actor.Envelope) {
	defer msg.Discard()

	var request orchestrator.DeploymentRevertRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("revert_deployment_unmarshal_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)

		n.sendReply(msg, orchestrator.DeploymentRevertResponse{
			OK:    false,
			Error: fmt.Sprintf("failed to unmarshal revert request: %v", err),
		})
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

	for _, allocID := range request.AllocsByName {
		// Now the allocID comes pre-constructed from the orchestrator, so we use it directly
		// without calling types.ConstructAllocationID again

		// Here we're considering both the committed and uncommitted resources/allocations/ports from the orchestrator
		// TODO: consider the allocation state and perform the necessary actions eg: uncommit, release, etc.
		// This would need addition of a new behavior. UnCommitAllocationBehavior?
		// https://gitlab.com/nunet/device-management-service/-/issues/961
		if a, _ := n.allocator.GetAllocation(allocID); a != nil {
			if err := n.allocator.Release(context.Background(), allocID); err != nil {
				log.Errorw("revert_deployment_release_failure",
					"labels", []string{string(observability.LabelDeployment)},
					"ensembleID", ensembleID,
					"error", err)
			}
		} else {
			log.Debugf("allocation %s not found in allocator, skipping to uncommit", allocID)
			if err := n.allocator.Uncommit(context.Background(), allocID); err != nil {
				log.Errorw("revert_deployment_uncommit_failure",
					"labels", []string{string(observability.LabelDeployment)},
					"ensembleID", ensembleID,
					"error", err,
				)
			} else {
				log.Debugf("successfully uncommitted allocation %s", allocID)
			}
		}
	}

	log.Infow("deployment_reverted",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", ensembleID)

	// Send success response
	n.sendReply(msg, orchestrator.DeploymentRevertResponse{
		OK: true,
	})
}

type UpdateDeploymentRequest struct {
	EnsembleID string
	Ensemble   jobtypes.EnsembleConfig
}

type UpdateDeploymentResponse struct {
	OK    bool
	Error string `json:",omitempty"`
}

func (n *Node) handleDeploymentUpdate(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("deployment update error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, UpdateDeploymentResponse{Error: err.Error()})
	}

	if time.Until(msg.Expiry()) < MinUpdateDeploymentTime {
		handleErr(fmt.Errorf("requested deployment update time too short"))
		return
	}

	var request UpdateDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("unmarshal update deployment request: %s", err))
		return
	}

	orch, err := n.orchestratorRegistry.GetOrchestrator(request.EnsembleID)
	if err != nil {
		handleErr(err)
		return
	}

	if orch.Status() != jobs.DeploymentStatusRunning {
		handleErr(errors.Join(fmt.Errorf("deployment %s is not running(status=%v), cannot update", request.EnsembleID, orch.Status()), ErrorDeploymentNotRunning))
		return
	}

	log.Infof("updating ensemble: %s", orch.ID())
	if err := orch.Update(request.Ensemble, msg.Expiry().Add(-orchestrator.MinEnsembleUpdateTimeout)); err != nil {
		handleErr(fmt.Errorf("error updating ensemble: %s", err))
		return
	}

	n.sendReply(msg, UpdateDeploymentResponse{
		OK: true,
	})

	// update the deployment in db
	if err := n.updateDeployment(orch); err != nil {
		log.Errorf("error saving deployment %s: %s", orch.ID(), err)
	}
}

func (n *Node) updateDeployment(_ orchestrator.Orchestrator) error {
	// TODO
	return nil
}

// Deployment pruning and clearing commands

type DeploymentPruneRequest struct {
	Before string `json:"before,omitempty"`
	All    bool   `json:"all,omitempty"`
}

type DeploymentPruneResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (n *Node) handleDeploymentPrune(msg actor.Envelope) {
	defer msg.Discard()

	log.Infow("deployment_prune_started",
		"labels", []string{string(observability.LabelDeployment)},
		"msg", msg)

	handleErr := func(err error) {
		log.Errorw("deployment_prune_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentPruneResponse{Error: err.Error()})
	}

	var request DeploymentPruneRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment prune request: %s", err))
		return
	}

	if request.All {
		// delete all deployments whose status is greater than Running
		statuses := []jobtypes.DeploymentStatus{
			jobtypes.DeploymentStatusFailed,
			jobtypes.DeploymentStatusCompleted,
		}
		for _, s := range statuses {
			views, err := n.orchestratorRegistry.GetDeploymentsByStatus(s)
			if err != nil {
				handleErr(fmt.Errorf("failed to list deployments by status %s: %w", s.String(), err))
				return
			}
			for _, v := range views {
				if err := n.orchestratorRegistry.DeleteDeployment(v.OrchestratorID); err != nil {
					handleErr(fmt.Errorf("failed to delete deployment %s: %w", v.OrchestratorID, err))
					return
				}
			}
		}
		log.Infow("deployments_pruned_by_status",
			"labels", []string{string(observability.LabelDeployment)},
			"mode", "all_status_gt_running")
		n.sendReply(msg, DeploymentPruneResponse{OK: true})
		return
	}

	if strings.TrimSpace(request.Before) == "" {
		handleErr(errors.New("before must be provided unless --all is used"))
		return
	}

	// parse supported formats: duration (1s,1m,1h,1d) and datetime (RFC3339, common)
	var cutoffTime time.Time
	// Try duration forms first
	before := strings.TrimSpace(request.Before)
	if strings.HasSuffix(before, "d") {
		// days is not a standard Go duration; handle explicitly
		daysStr := strings.TrimSuffix(before, "d")
		if daysStr == "" {
			handleErr(fmt.Errorf("invalid before duration: %s", before))
			return
		}
		if nDays, err := strconv.Atoi(daysStr); err == nil && nDays > 0 {
			cutoffTime = time.Now().AddDate(0, 0, -nDays)
		} else {
			handleErr(fmt.Errorf("invalid before duration days: %s", before))
			return
		}
	} else if dur, err := time.ParseDuration(before); err == nil {
		cutoffTime = time.Now().Add(-dur)
	} else {
		// Try datetime formats
		var parseErr error
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, before); err == nil {
				cutoffTime = t
				parseErr = nil
				break
			}
			parseErr = err
			continue
		}
		if cutoffTime.IsZero() {
			handleErr(fmt.Errorf("invalid before value: %w", parseErr))
			return
		}
	}

	// delete all deployments whose status is greater than Running
	statuses := []jobtypes.DeploymentStatus{
		jobtypes.DeploymentStatusFailed,
		jobtypes.DeploymentStatusCompleted,
	}
	for _, s := range statuses {
		views, err := n.orchestratorRegistry.GetDeploymentsByStatus(s)
		if err != nil {
			handleErr(fmt.Errorf("failed to list deployments by status %s: %w", s.String(), err))
			return
		}
		for _, v := range views {
			if v.CreatedAt.Before(cutoffTime) {
				if err := n.orchestratorRegistry.DeleteDeployment(v.OrchestratorID); err != nil {
					handleErr(fmt.Errorf("failed to delete deployment %s: %w", v.OrchestratorID, err))
					return
				}
			}
		}
	}
	log.Infow("deployments_pruned",
		"labels", []string{string(observability.LabelDeployment)},
		"mode", "before")
	n.sendReply(msg, DeploymentPruneResponse{OK: true})
}

type DeploymentDeleteRequest struct {
	OrchestratorID string `json:"orchestrator_id"`
}

type DeploymentDeleteResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (n *Node) handleDeploymentDelete(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("deployment_delete_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentDeleteResponse{Error: err.Error()})
	}

	var request DeploymentDeleteRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment delete request: %s", err))
		return
	}

	if request.OrchestratorID == "" {
		handleErr(errors.New("orchestrator_id is required"))
		return
	}

	// Delete the specific deployment
	if err := n.orchestratorRegistry.DeleteDeployment(request.OrchestratorID); err != nil {
		handleErr(fmt.Errorf("failed to delete deployment %s: %w", request.OrchestratorID, err))
		return
	}

	log.Infow("deployment_deleted",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestrator_id", request.OrchestratorID)

	n.sendReply(msg, DeploymentDeleteResponse{OK: true})
}
