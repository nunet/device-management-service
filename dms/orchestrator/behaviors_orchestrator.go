// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func (o *BasicOrchestrator) handleTaskTermination(msg actor.Envelope) {
	msg.Discard()

	var req behaviors.TaskTerminationNotification

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Debugf("unmarshalling task completion request: %s", err)
		return
	}

	log.Infow("task_terminated", "labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id, "allocationID", req.AllocationID, "status", req.Status)

	// Parse the allocation ID to get the manifest key
	allocID, err := types.ParseAllocationID(req.AllocationID)
	if err != nil {
		log.Debugf("failed to parse allocation ID %s: %v", req.AllocationID, err)
		return
	}

	manifestKey := allocID.ManifestKey()
	a, ok := o.manifest.Allocations[manifestKey]
	if !ok {
		log.Debugf("allocation %s not found on the manifest", req.AllocationID)
		return
	}

	// update allocation status
	o.lock.Lock()
	a.Status = jtypes.AllocationStatus(req.Status)
	o.manifest.Allocations[manifestKey] = a
	o.lock.Unlock()

	if req.Error.Err != "" {
		log.Errorf(
			"allocation task %s yielded error: %v",
			req.AllocationID, req.Error,
		)
		return
	}

	allocDir, err := o.WriteAllocationLogs(manifestKey, req.Stdout, req.Stderr)
	if err != nil {
		log.Errorf("failed to write logs for allocation %s: %v", manifestKey, err)
		return
	}

	log.Infow("allocation_logs_saved", "labels", []string{string(observability.LabelDeployment)},
		"manifest", manifestKey, "path", allocDir, "orchestratorID", o.id)
}

func (o *BasicOrchestrator) WriteAllocationLogs(
	allocName string, stdout, stderr []byte,
) (string, error) {
	ensembleDir := filepath.Join(
		o.workDir,
		"deployments",
		o.id,
	)
	allocDir := filepath.Join(ensembleDir, allocName)

	err := o.fs.MkdirAll(allocDir, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create allocation directory %s: %w", allocDir, err)
	}

	if len(stdout) > 0 {
		stdoutPath := filepath.Join(allocDir, "stdout.log")
		err = o.fs.WriteFile(stdoutPath, stdout, 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write stdout logs to %s: %w", stdoutPath, err)
		}
	}

	if len(stderr) > 0 {
		stderrPath := filepath.Join(allocDir, "stderr.log")
		err = o.fs.WriteFile(stderrPath, stderr, 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write stderr logs to %s: %w", stderrPath, err)
		}
	}

	return allocDir, nil
}

// handleAllocationLiveness passively records push heartbeats
// NOTE: This does NOT affect health decisions - supervisor's pull checks remain authoritative
func (o *BasicOrchestrator) handleAllocationLiveness(msg actor.Envelope) {
	defer msg.Discard()

	var notification behaviors.AllocationLivenessNotification

	if err := json.Unmarshal(msg.Message, &notification); err != nil {
		log.Debugw("unmarshalling_liveness_notification_failed",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		return
	}

	// Log for observability (passive collection only)
	log.Debugw("received_allocation_heartbeat",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", o.id,
		"allocationID", notification.AllocationID,
		"sequence", notification.SequenceNumber,
		"status", notification.Status,
		"healthy", notification.Health.Healthy,
		"check_type", notification.Health.CheckType)

	// Log warning if allocation self-reports unhealthy
	// (just for observability, doesn't change supervisor behavior)
	if !notification.Health.Healthy {
		log.Warnw("allocation_self_reported_unhealthy",
			"labels", []string{string(observability.LabelDeployment)},
			"allocationID", notification.AllocationID,
			"message", notification.Health.Message,
			"check_type", notification.Health.CheckType,
			"note", "supervisor pull checks remain authoritative")
	}

	// Log resource usage if provided
	if notification.ResourceUsage != nil {
		log.Debugw("allocation_resource_usage",
			"labels", []string{string(observability.LabelDeployment)},
			"allocationID", notification.AllocationID,
			"cpu_percent", notification.ResourceUsage.CPUUsagePercent,
			"memory_used_bytes", notification.ResourceUsage.MemoryUsedBytes,
			"memory_limit_bytes", notification.ResourceUsage.MemoryLimitBytes)
	}
}

// handleAllocationStatusUpdate receives immediate status change notifications
func (o *BasicOrchestrator) handleAllocationStatusUpdate(msg actor.Envelope) {
	defer msg.Discard()

	var update behaviors.AllocationStatusUpdate

	if err := json.Unmarshal(msg.Message, &update); err != nil {
		log.Debugw("unmarshalling_status_update_failed",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		return
	}

	// Log the status change (observability)
	log.Infow("allocation_status_changed",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", o.id,
		"allocationID", update.AllocationID,
		"old_status", update.OldStatus,
		"new_status", update.NewStatus,
		"reason", update.Reason)

	// Optionally update manifest for faster visibility
	// Supervisor's pull checks will validate and correct if needed
	allocID, err := types.ParseAllocationID(update.AllocationID)
	if err != nil {
		log.Debugf("failed to parse allocation ID %s: %v", update.AllocationID, err)
		return
	}

	manifestKey := allocID.ManifestKey()

	o.lock.Lock()
	if a, ok := o.manifest.Allocations[manifestKey]; ok {
		// Store push status as supplementary info
		oldManifestStatus := a.Status
		a.Status = jtypes.AllocationStatus(update.NewStatus)
		o.manifest.Allocations[manifestKey] = a

		log.Debugw("updated_manifest_from_push",
			"allocationID", update.AllocationID,
			"old_manifest_status", oldManifestStatus,
			"new_push_status", update.NewStatus,
			"note", "supervisor pull checks remain authoritative")
	}
	o.lock.Unlock()
}
