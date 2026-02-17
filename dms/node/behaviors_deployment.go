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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/gateway/provider"
	"gitlab.com/nunet/device-management-service/gateway/store"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// Deployment behavior data source strategy:
// - Store-based (primary source of truth): handleDeploymentList, handleDeploymentStatus, handleDeploymentManifest
// - In-memory registry (requires active orchestrator): handleDeploymentLogs, handleDeploymentShutdown, handleDeploymentUpdate
// - Store + in-memory: handleNewDeployment (creates in memory, auto-saved to store via status watcher)

// MinDeploymentTime minimum time for deployment
const (
	MinDeploymentTime             = time.Minute - time.Second
	MinUpdateDeploymentTime       = 2 * (time.Minute - time.Second) // TODO: tune this
	allocationStatsRequestTimeout = 20 * time.Second
	maxRetries                    = 5
	retryDelay                    = time.Second
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

	log.Debugw("deployment_saved", "labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", orchestrator.ID(), "stats", orchestrator.Status().String())

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

	if request.Ensemble.V1 == nil {
		handleErr(errors.New("empty ensemble config"))
		return
	}

	if request.Ensemble.Contracts() != nil {
		for _, contract := range request.Ensemble.Contracts() {
			if contract.DID == "" {
				handleErr(errors.New("contract DID is required"))
				return
			}
		}

		// retrieve contract information from contract host
		for k, contract := range request.Ensemble.Contracts() {
			// Call the contract host node (not contract actor) to get contract info
			// Use HandleFromDID which defaults to "root" inbox for node-level behaviors
			destination, err := actor.HandleFromDID(contract.Host)
			if err != nil {
				handleErr(fmt.Errorf("failed to get contract host handle: %w", err))
				return
			}

			// invoke behavior to retrieve contract information
			reply, err := n.invokeBehaviour(
				destination,
				behaviors.ContractInfoBehavior,
				behaviors.ContractInfoRequest{
					ContractDID: contract.DID,
				},
				invokeMessageTimeout,
			)
			if err != nil {
				handleErr(fmt.Errorf("failed to invoke contract info for %s: %w", contract.DID, err))
				return
			}

			var contractInfoResp behaviors.ContractInfoResponse
			if err := json.Unmarshal(reply.Message, &contractInfoResp); err != nil {
				handleErr(fmt.Errorf("failed to unmarshal contract info response: %w", err))
				return
			}

			if !contractInfoResp.OK {
				handleErr(fmt.Errorf("contract info error: %s", contractInfoResp.Error))
				return
			}

			contract.Provider = contractInfoResp.Provider
			contract.Requestor = contractInfoResp.Requestor
			request.Ensemble.V1.Contracts[k] = contract
		}
	}

	orch, err := n.createOrchestrator(n.ctx, request.Ensemble, request.Ensemble.Contracts())
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
		// Manually save the failed status before stopping to ensure it persists
		if err := n.orchestratorRegistry.SaveOrchestrator(orch); err != nil {
			log.Warnw("failed to save failed orchestrator", "error", err)
		}
		orch.Stop()
		log.Errorw("ensemble_deployment_error",
			"labels", []string{string(observability.LabelDeployment)},
			"ensembleID", orch.ID(),
			"error", err)

		return
	}
	// Orchestrator status is automatically saved to store via status watcher
}

type DeploymentListRequest struct {
	// Existing metadata filter (backward compatible)
	Metadata map[string]string `json:"metadata,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Max number of results (default: no limit, for backward compat)
	Offset int `json:"offset,omitempty"` // Number of results to skip (default: 0)

	// Status filter (for JSON API - parsed from strings in CLI)
	Status []jobtypes.DeploymentStatus `json:"status,omitempty"` // Filter by one or more statuses

	// Date filters (for JSON API)
	CreatedAfter  *time.Time `json:"created_after,omitempty"`  // Filter by CreatedAt >= value
	CreatedBefore *time.Time `json:"created_before,omitempty"` // Filter by CreatedAt <= value
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`  // Filter by UpdatedAt >= value
	UpdatedBefore *time.Time `json:"updated_before,omitempty"` // Filter by UpdatedAt <= value

	// Sorting
	SortBy string `json:"sort_by,omitempty"` // Field to sort by (e.g., "created_at", "-created_at" for desc)
}

type DeploymentListResponse struct {
	// Enhanced deployment information
	Deployments []DeploymentInfo `json:"deployments"`

	// Pagination metadata
	Total      int  `json:"total"`                 // Total number of deployments matching filters
	HasMore    bool `json:"has_more"`              // Whether there are more results available
	NextOffset int  `json:"next_offset,omitempty"` // Offset for next page (if has_more is true)
}

type DeploymentInfo struct {
	OrchestratorID string            `json:"orchestrator_id"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("deployment_list_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		n.sendReply(msg, DeploymentListResponse{Deployments: []DeploymentInfo{}})
	}

	var request DeploymentListRequest
	var resp DeploymentListResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment list request: %s", err))
		return
	}

	// Build query from request
	query := orchestrator.DeploymentQuery{
		Limit:  request.Limit,
		Offset: request.Offset,
		SortBy: request.SortBy,
	}

	// Status filter
	if len(request.Status) > 0 {
		query.StatusFilter = request.Status
	}

	// Date filters
	if request.CreatedAfter != nil {
		query.CreatedAfter = request.CreatedAfter
	}
	if request.CreatedBefore != nil {
		query.CreatedBefore = request.CreatedBefore
	}
	if request.UpdatedAfter != nil {
		query.UpdatedAfter = request.UpdatedAfter
	}
	if request.UpdatedBefore != nil {
		query.UpdatedBefore = request.UpdatedBefore
	}

	// Query deployments from store
	deployments, total, err := n.orchestratorRegistry.QueryDeployments(query)
	if err != nil {
		handleErr(fmt.Errorf("failed to query deployments: %w", err))
		return
	}

	// Apply metadata filtering (in-memory, as metadata is in deployment_data)
	filteredDeployments := make([]DeploymentInfo, 0)
	for _, deployment := range deployments {
		if shouldIncludeDeployment(deployment, request.Metadata) {
			info := DeploymentInfo{
				OrchestratorID: deployment.OrchestratorID,
				Status:         deployment.Status.String(),
				CreatedAt:      deployment.CreatedAt,
				UpdatedAt:      deployment.UpdatedAt,
				CompletedAt:    deployment.CompletedAt,
				Metadata:       deployment.Manifest.Metadata,
			}
			filteredDeployments = append(filteredDeployments, info)
		}
	}

	// Calculate pagination metadata
	resp.Deployments = filteredDeployments
	resp.Total = total
	resp.HasMore = request.Limit > 0 && (request.Offset+len(filteredDeployments) < total)
	if resp.HasMore {
		resp.NextOffset = request.Offset + len(filteredDeployments)
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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_logs_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
		n.sendReply(msg, DeploymentLogsResponse{Error: err.Error()})
	}

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment logs: %w", err))
		return
	}

	// For logs, we need an active orchestrator (in-memory registry only)
	o, err := n.orchestratorRegistry.GetOrchestrator(request.EnsembleID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get orchestrator: %w", err))
		return
	}
	orchestratorID = o.ID()

	data, err := o.GetAllocationLogs(request.AllocationName)
	if err != nil {
		handleErr(fmt.Errorf("failed to get allocation logs: %w", err))
		return
	}

	allocDir, err := o.WriteAllocationLogs(request.AllocationName, data.Stdout, data.Stderr)
	if err != nil {
		handleErr(fmt.Errorf("failed to write allocation logst: %w", err))
		return
	}

	resp.LogsWrittenTo = allocDir
	n.sendReply(msg, resp)
}

type DeploymentStatusRequest struct {
	ID           string `json:"id"`
	IncludeUsage bool   `json:"include_usage,omitempty"`
}

type DeploymentStatusResponse struct {
	Status          string                             `json:"status"`
	Error           string                             `json:"error"`
	AllocationInfo  map[string]jobtypes.AllocationInfo `json:"allocation_info,omitempty"`
	AllocationUsage map[string]*types.ExecutorStats    `json:"allocation_usage,omitempty"`
}

func (n *Node) handleDeploymentStatus(msg actor.Envelope) {
	defer msg.Discard()

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_status_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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
	orchestratorID = deployment.ID

	resp.Status = deployment.Status.String()

	if deployment.Status == jobtypes.DeploymentStatusRunning {
		orch, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
		if err != nil {
			handleErr(fmt.Errorf("failed to get orchestrator: %s", err))
			return
		}
		resp.AllocationInfo = orch.AllocationInfo()

		if request.IncludeUsage {
			manifest := orch.Manifest()
			usage := make(map[string]*types.ExecutorStats, len(manifest.Allocations))

			for allocID, allocManifest := range manifest.Allocations {
				reply, err := n.invokeBehaviour(
					allocManifest.Handle,
					behaviors.AllocationStatsBehavior,
					behaviors.AllocationStatsRequest{},
					allocationStatsRequestTimeout,
				)
				if err != nil {
					handleErr(fmt.Errorf("failed to invoke allocation stats for %s: %w", allocID, err))
					return
				}

				var statsResp behaviors.AllocationStatsResponse
				if err := json.Unmarshal(reply.Message, &statsResp); err != nil {
					handleErr(fmt.Errorf("failed to decode allocation stats response for %s: %w", allocID, err))
					return
				}

				if !statsResp.OK {
					handleErr(fmt.Errorf("allocation %s stats error: %s", allocID, statsResp.Error))
					return
				}

				usage[allocID] = statsResp.Stats
			}

			resp.AllocationUsage = usage
		}
	}

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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_manifest_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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

	// if deployment is running, get the latest manifest directly from orchestrator
	if deployment.Status == jobtypes.DeploymentStatusRunning {
		orch, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
		if err != nil {
			handleErr(fmt.Errorf("failed to get orchestrator: %s", err))
			return
		}
		resp.Manifest = orch.Manifest()
		n.sendReply(msg, resp)
		return
	}

	resp.Manifest = deployment.Manifest
	n.sendReply(msg, resp)
}

func (n *Node) handleDeploymentInfo(msg actor.Envelope) {
	defer msg.Discard()

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_info_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
		n.sendReply(msg, DeploymentInfoResponse{Error: err.Error()})
	}

	var request DeploymentInfoRequest
	var resp DeploymentInfoResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling deployment info request: %s", err))
		return
	}

	if request.ID == "" {
		handleErr(errors.New("deployment ID is required"))
		return
	}

	// Get deployment from store (primary source of truth)
	deployment, err := n.orchestratorRegistry.GetDeployment(request.ID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get deployment: %s", err))
		return
	}
	orchestratorID = deployment.OrchestratorID

	resp.ID = deployment.OrchestratorID
	resp.Status = deployment.Status.String()

	// Try to get orchestrator (may exist even for non-running deployments)
	orch, err := n.orchestratorRegistry.GetOrchestrator(request.ID)
	hasOrchestrator := err == nil

	var manifest jobs.EnsembleManifest
	var allocationInfo map[string]jobtypes.AllocationInfo
	if hasOrchestrator {
		// If orchestrator exists, get latest manifest from it
		manifest = orch.Manifest()
		resp.Manifest = &manifest
		resp.Allocations = make(map[string]AllocationDetails)

		// Get allocation info
		allocationInfo = orch.AllocationInfo()
		for allocID, info := range allocationInfo {
			resp.Allocations[allocID] = AllocationDetails{
				AllocationID:   info.AllocationID,
				Status:         string(info.Status),
				HeartbeatSeq:   info.HeartbeatSeq,
				HasHealthCheck: info.HasHealthCheck,
				Health:         info.Health,
				ResourceLimit:  info.ResourceLimit,
				ResourceUsage:  info.ResourceUsage,
				DNSName:        info.DNSName,
				IP:             info.IP,
				Timestamp:      info.Timestamp,
			}
		}
	} else {
		// If orchestrator doesn't exist, use manifest from store
		manifest = deployment.Manifest
		resp.Manifest = &manifest
		resp.Allocations = make(map[string]AllocationDetails)
		allocationInfo = make(map[string]jobtypes.AllocationInfo)
	}

	// Collect resource usage if requested (works for both running and completed deployments if orchestrator exists)
	if request.IncludeUsage && hasOrchestrator {
		usage := make(map[string]*types.ExecutorStats, len(manifest.Allocations))

		// Build a map from allocation ID to manifest key for matching
		allocIDToManifestKey := make(map[string]string, len(allocationInfo))
		for allocID := range allocationInfo {
			parsedID, err := types.ParseAllocationID(allocID)
			if err != nil {
				log.Warnw("failed to parse allocation ID for usage mapping",
					"allocationID", allocID,
					"error", err)
				continue
			}
			manifestKey := parsedID.ManifestKey()
			allocIDToManifestKey[allocID] = manifestKey
		}

		for manifestKey, allocManifest := range manifest.Allocations {
			// Find the corresponding allocation ID
			var matchingAllocID string
			for allocID, mappedKey := range allocIDToManifestKey {
				if mappedKey == manifestKey {
					matchingAllocID = allocID
					break
				}
			}

			if matchingAllocID == "" {
				log.Warnw("could not find matching allocation ID for manifest key",
					"manifestKey", manifestKey)
				continue
			}

			reply, err := n.invokeBehaviour(
				allocManifest.Handle,
				behaviors.AllocationStatsBehavior,
				behaviors.AllocationStatsRequest{},
				allocationStatsRequestTimeout,
			)
			if err != nil {
				log.Warnw("failed to get allocation stats",
					"allocation", matchingAllocID,
					"error", err)
				continue
			}

			var statsResp behaviors.AllocationStatsResponse
			if err := json.Unmarshal(reply.Message, &statsResp); err != nil {
				log.Warnw("failed to decode allocation stats",
					"allocation", matchingAllocID,
					"error", err)
				continue
			}

			if statsResp.OK && statsResp.Stats != nil {
				usage[matchingAllocID] = statsResp.Stats
				// Update allocation details with stats
				if details, exists := resp.Allocations[matchingAllocID]; exists {
					details.ExecutorStats = statsResp.Stats
					resp.Allocations[matchingAllocID] = details
				}
			} else {
				log.Warnw("allocation stats error",
					"allocation", matchingAllocID,
					"error", statsResp.Error)
			}
		}

		resp.Usage = usage
	}

	// Collect logs if requested (works for both running and completed deployments if orchestrator exists)
	if request.IncludeLogs {
		// Build a map from allocation ID to manifest key for matching
		manifestKeyToAllocID := make(map[string]string, len(allocationInfo))
		for allocID := range allocationInfo {
			parsedID, err := types.ParseAllocationID(allocID)
			if err != nil {
				log.Warnw("failed to parse allocation ID for logs mapping",
					"allocationID", allocID,
					"error", err)
				continue
			}
			manifestKey := parsedID.ManifestKey()
			manifestKeyToAllocID[manifestKey] = allocID
		}

		// Determine which allocations to get logs for
		allocationsToLog := make(map[string]bool)
		if len(request.AllocationNames) == 0 {
			// Get logs for all allocations in manifest
			for manifestKey := range manifest.Allocations {
				allocationsToLog[manifestKey] = true
			}
		} else {
			// Get logs for specified allocation names (could be config names or manifest keys)
			// Try to match them to manifest keys
			for _, requestedName := range request.AllocationNames {
				found := false
				// First try as manifest key directly
				if _, exists := manifest.Allocations[requestedName]; exists {
					allocationsToLog[requestedName] = true
					found = true
				} else {
					// Try as config name - search through manifest allocations
					for manifestKey, allocManifest := range manifest.Allocations {
						if allocManifest.RedundancyGroup == requestedName {
							allocationsToLog[manifestKey] = true
							found = true
							break
						}
					}
				}
				if !found {
					log.Warnw("requested allocation name not found in manifest",
						"requestedName", requestedName)
				}
			}
		}

		// Get logs for each allocation
		for manifestKey := range allocationsToLog {
			// Find the corresponding allocation ID
			allocID, exists := manifestKeyToAllocID[manifestKey]
			if !exists {
				log.Warnw("could not find matching allocation ID for manifest key",
					"manifestKey", manifestKey)
				continue
			}

			logsData, err := orch.GetAllocationLogs(manifestKey)
			if err != nil {
				log.Warnw("failed to get allocation logs",
					"allocation", allocID,
					"manifestKey", manifestKey,
					"error", err)
				// Add error to allocation details
				if details, exists := resp.Allocations[allocID]; exists {
					details.Logs = &AllocationLogs{
						Error: err.Error(),
					}
					resp.Allocations[allocID] = details
				}
				continue
			}

			// Write logs to directory (reuse existing logic)
			allocDir, err := orch.WriteAllocationLogs(manifestKey, logsData.Stdout, logsData.Stderr)
			if err != nil {
				log.Warnw("failed to write allocation logs",
					"allocation", allocID,
					"manifestKey", manifestKey,
					"error", err)
				if details, exists := resp.Allocations[allocID]; exists {
					details.Logs = &AllocationLogs{
						Error: err.Error(),
					}
					resp.Allocations[allocID] = details
				}
				continue
			}

			// Add logs to allocation details
			if details, exists := resp.Allocations[allocID]; exists {
				details.Logs = &AllocationLogs{
					StdoutPath:    filepath.Join(allocDir, "stdout.log"),
					StderrPath:    filepath.Join(allocDir, "stderr.log"),
					LogsWrittenTo: allocDir,
				}
				resp.Allocations[allocID] = details
			}
		}
	}

	n.sendReply(msg, resp)
}

type DeploymentInfoRequest struct {
	ID              string   `json:"id"`                         // Deployment ID (required)
	IncludeUsage    bool     `json:"include_usage,omitempty"`    // Include resource usage stats
	IncludeLogs     bool     `json:"include_logs,omitempty"`     // Include logs for all allocations
	AllocationNames []string `json:"allocation_names,omitempty"` // Specific allocations to include logs for (empty = all)
}

type AllocationLogs struct {
	StdoutPath    string `json:"stdout_path,omitempty"`     // Path to stdout.log file
	StderrPath    string `json:"stderr_path,omitempty"`     // Path to stderr.log file
	LogsWrittenTo string `json:"logs_written_to,omitempty"` // Directory path where logs are written
	Error         string `json:"error,omitempty"`           // Error retrieving logs (if any)
}

type AllocationDetails struct {
	// From AllocationInfo
	AllocationID   string                           `json:"allocation_id"`
	Status         string                           `json:"status"`
	HeartbeatSeq   int64                            `json:"heartbeat_seq"`
	HasHealthCheck bool                             `json:"has_health_check"`
	Health         string                           `json:"health"`
	ResourceLimit  types.Resources                  `json:"resource_limit"`
	ResourceUsage  jobtypes.AllocationResourceUsage `json:"resource_usage"`
	DNSName        string                           `json:"dns_name"`
	IP             string                           `json:"ip"`
	Timestamp      int64                            `json:"timestamp"`

	// Optional: Resource usage stats (if IncludeUsage=true)
	ExecutorStats *types.ExecutorStats `json:"executor_stats,omitempty"`

	// Optional: Logs (if IncludeLogs=true)
	Logs *AllocationLogs `json:"logs,omitempty"`
}

type DeploymentInfoResponse struct {
	// Basic deployment information
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`

	// Full manifest
	Manifest *jobs.EnsembleManifest `json:"manifest,omitempty"`

	// Allocation information
	Allocations map[string]AllocationDetails `json:"allocations,omitempty"`

	// Optional: Resource usage (if IncludeUsage=true)
	Usage map[string]*types.ExecutorStats `json:"usage,omitempty"`
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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_shutdown_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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
			"orchestratorID", request.ID,
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

	orchestratorID := ""
	var request orchestrator.DeploymentRevertRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw("revert_deployment_unmarshal_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)

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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment update error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_prune_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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
				orchestratorID = v.OrchestratorID
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

	orchestratorID := ""
	handleErr := func(err error) {
		log.Errorw("deployment_delete_error",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err,
			"orchestratorID", orchestratorID,
		)
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
	orchestratorID = request.OrchestratorID

	// Delete the specific deployment
	if err := n.orchestratorRegistry.DeleteDeployment(request.OrchestratorID); err != nil {
		handleErr(fmt.Errorf("failed to delete deployment %s: %w", request.OrchestratorID, err))
		return
	}

	log.Infow("deployment_deleted",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", request.OrchestratorID)

	n.sendReply(msg, DeploymentDeleteResponse{OK: true})
}

// handleBidSigning is for new provisioned dmses to sign bids
func (n *Node) handleBidSigning(msg actor.Envelope) {
	defer msg.Discard()

	var req jobtypes.SignPromiseBidRequest
	err := json.Unmarshal(msg.Message, &req)
	if err != nil {
		n.sendReply(msg, jobtypes.PromiseBidSigningResponse{
			Error: err.Error(),
		})
		return
	}

	bid := jobtypes.Bid{
		V1: &jobtypes.BidV1{
			EnsembleID: req.Bid.EnsembleID(),
			NodeID:     req.Bid.NodeID(),
			Peer:       n.hostID,
			Location:   req.Bid.Location(),
			Handle:     n.actor.Handle(),
		},
	}

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		log.Debugw("provider_retrieval_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		n.sendReply(msg, jobtypes.PromiseBidSigningResponse{
			Error: err.Error(),
		})

		return
	}

	err = bid.Sign(provider)
	if err != nil {
		log.Debugw("provider_sign_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		n.sendReply(msg, jobtypes.PromiseBidSigningResponse{
			Error: err.Error(),
		})

		return
	}

	n.storeBid(bid.EnsembleID(), req.Nounce, req.BidRequest)

	n.sendReply(msg, jobtypes.PromiseBidSigningResponse{
		Bid: bid,
	})
}

func (n *Node) handlePromiseBid(msg actor.Envelope) {
	defer msg.Discard()

	log.Debug("handling promise bids")

	var req jobtypes.PromiseBidRequest
	err := json.Unmarshal(msg.Message, &req)
	if err != nil {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: err.Error(),
		})

		return
	}

	// check if we already signed this bid
	bid, ok := n.getBid(req.Bid.EnsembleID())
	if !ok {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: fmt.Sprintf("bid with ensemble id %s not found", req.Bid.EnsembleID()),
		})
		return
	}

	allProviders := n.serverProviderRegistry.All()
	log.Debugf("checking %d providers for provisioning", len(allProviders))

	ctx, cancel := context.WithCancel(n.ctx)
	defer cancel()

	var (
		wg             sync.WaitGroup
		found          int32
		targetPlan     provider.Plan
		targetProvider provider.Provider
		mu             sync.Mutex
	)

	for _, pp := range allProviders {
		wg.Add(1)
		go func(pp provider.Provider) {
			defer wg.Done()

			if atomic.LoadInt32(&found) == 1 {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			plans, err := pp.ListPlans(ctx)
			if err != nil {
				return
			}

			matchedPlan, err := pp.SelectMatchingPlan(plans, bid.request.V1.Resources)
			if err != nil || matchedPlan == nil {
				return
			}

			if atomic.CompareAndSwapInt32(&found, 0, 1) {
				mu.Lock()
				targetPlan = *matchedPlan
				targetProvider = pp
				mu.Unlock()
				cancel() // stop others
			}
		}(pp)
	}

	wg.Wait()

	if atomic.LoadInt32(&found) == 0 || targetProvider == nil {
		log.Debug("targetProvider is nil — no matching plan found")
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: fmt.Sprintf("no suitable plan found for bid with ensemble: %s", req.Bid.EnsembleID()),
		})
		return
	}

	// TODO: for now image is empty, maybe make it part of the plan object
	server, err := targetProvider.ProvisionServer(n.ctx, targetPlan, targetPlan.Name, "", msg.From.DID.String())
	if err != nil {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: fmt.Sprintf("failed to provision server for bid with ensemble: %s", req.Bid.EnsembleID()),
		})
		return
	}

	log.Debugf("successfully provisioned server %s using provider %s", server.ID, targetProvider.Name())

	connected := false

	for i := 1; i <= maxRetries; i++ {
		err = n.network.Connect(n.ctx, fmt.Sprintf("%s/p2p/%s", server.ListenAddr, server.PeerID))
		if err == nil {
			connected = true
			break
		}

		time.Sleep(retryDelay)
	}

	if !connected {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: fmt.Sprintf("failed to connect to provisioned resource for bid: %s", req.Bid.EnsembleID()),
		})
		err := targetProvider.DeleteServer(ctx, server.ID)
		if err != nil {
			log.Errorf("failed to delete provisioned instance: %v", err)
		}
		return
	}

	err = n.addProvisionedResources(msg.From.DID.String(), targetProvider, server)
	if err != nil {
		log.Errorf("failed to record provisioned resource in store: %v", err)
	}

	destination, err := actor.HandleFromPeerID(server.PeerID)
	if err != nil {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: err.Error(),
		})
		return
	}
	signReq := jobtypes.SignPromiseBidRequest{
		Bid:        req.Bid,
		BidRequest: bid.request,
		Nounce:     bid.nonce,
	}

	envelope, err := n.invokeBehaviour(destination, behaviors.PromiseBidSigningBehavior, signReq, invokeMessageTimeout)
	if err != nil {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: err.Error(),
		})
		return
	}

	var signedBidPayload jobtypes.PromiseBidSigningResponse
	err = json.Unmarshal(envelope.Message, &signedBidPayload)
	if err != nil {
		n.sendReply(msg, jobtypes.ConvertedPromiseBidResponse{
			Error: err.Error(),
		})
		return
	}

	resp := jobtypes.ConvertedPromiseBidResponse{
		Bid: signedBidPayload.Bid,
	}

	n.sendReply(msg, resp)
}

func (n *Node) addProvisionedResources(orchestratorDID string, p provider.Provider, server *provider.Server) error {
	return n.gatewayStore.Insert(&store.ProvisionedResources{
		ProvisionedVMPeerID: server.PeerID,
		Orchestrator:        orchestratorDID,
		ProviderName:        p.Name(),
		Resource:            *server,
		CreatedAt:           time.Now(),
	})
}
