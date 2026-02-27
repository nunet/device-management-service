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
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (o *BasicOrchestrator) handleTaskTermination(msg actor.Envelope) {
	msg.Discard()

	var req behaviors.TaskTerminationNotification

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Debugf("unmarshalling task completion request: %s", err)
		return
	}

	log.Infow("task_terminated",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id,
		"allocationID", req.AllocationID,
		"status", req.Status)

	// Parse the allocation ID to get the manifest key
	allocID, err := types.ParseAllocationID(req.AllocationID)
	if err != nil {
		log.Debugf("failed to parse allocation ID %s: %v", req.AllocationID, err)
		return
	}

	manifestKey := allocID.ManifestKey()

	updateMan := o.Manifest()

	a, ok := updateMan.Allocations[manifestKey]
	if !ok {
		log.Debugf("allocation %s not found on the manifest", req.AllocationID)
		return
	}

	// update allocation status
	a.Status = jtypes.AllocationStatus(req.Status)
	updateMan.Allocations[manifestKey] = a
	o.updateManifest(updateMan)

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

	var notification jtypes.AllocationLivenessNotification

	if err := json.Unmarshal(msg.Message, &notification); err != nil {
		log.Debugw("unmarshalling_liveness_notification_failed",
			"labels", []string{string(observability.LabelDeployment)},
			"error", err)
		return
	}

	o.lock.Lock()
	defer o.lock.Unlock()
	if _, ok := o.allocs[notification.AllocationID]; !ok {
		log.Debugw("liveness_notification_for_unknown_allocation",
			"labels", []string{string(observability.LabelDeployment)},
			"allocationID", notification.AllocationID)
		return
	}

	log.Debugw("received_allocation_heartbeat",
		"labels", []string{string(observability.LabelDeployment)},
		"ensembleID", o.id,
		"allocationID", notification.AllocationID,
		"sequence", notification.SequenceNumber,
		"status", notification.Status,
		"healthy", notification.Health.Healthy,
		"check_type", notification.Health.CheckType)

	nInfo := o.allocs[notification.AllocationID]
	nInfo.HeartbeatSeq = notification.SequenceNumber
	if nInfo.Status != jtypes.AllocationStatus(notification.Status) {
		if m := observability.AllocationStatus; m != nil {
			m.Record(o.ctx, 1, metric.WithAttributes(
				observability.AttrDID,
				attribute.String("orchestratorID", o.id),
				attribute.String("allocationID", notification.AllocationID),
				attribute.String("status", notification.Status),
			))
		}
	}
	nInfo.Status = jtypes.AllocationStatus(notification.Status)
	if notification.ResourceUsage != nil {
		nInfo.ResourceUsage.CPUUsagePercent = notification.ResourceUsage.CPUUsagePercent
		nInfo.ResourceUsage.MemoryUsedBytes = notification.ResourceUsage.MemoryUsedBytes
		nInfo.ResourceUsage.MemoryLimitBytes = notification.ResourceUsage.MemoryLimitBytes
		nInfo.ResourceUsage.NetworkRxBytes = notification.ResourceUsage.NetworkRxBytes
		nInfo.ResourceUsage.NetworkTxBytes = notification.ResourceUsage.NetworkTxBytes
	}
	if o.allocs[notification.AllocationID].HasHealthCheck {
		if notification.Health.Healthy {
			nInfo.Health = "Healthy"
		} else {
			log.Warnw("allocation_self_reported_unhealthy",
				"labels", []string{string(observability.LabelDeployment)},
				"allocationID", notification.AllocationID,
				"message", notification.Health.Message,
				"check_type", notification.Health.CheckType,
				// TODO metric in supervisor for unhealthy
				"note", "supervisor pull checks remain authoritative")
			nInfo.Health = "Unhealthy: " + notification.Health.Message
		}
	}

	nInfo.Timestamp = time.Now().Unix()
	o.allocs[notification.AllocationID] = nInfo

	// Log resource usage if provided
	if notification.ResourceUsage != nil {
		log.Debugw("allocation_resource_usage",
			"labels", []string{string(observability.LabelDeployment)},
			"allocationID", notification.AllocationID,
			"cpu_percent", notification.ResourceUsage.CPUUsagePercent,
			"memory_used_bytes", notification.ResourceUsage.MemoryUsedBytes,
			"memory_limit_bytes", notification.ResourceUsage.MemoryLimitBytes)
	}

	// metrics
	if observability.AllocationHeartbeat != nil {
		u := nInfo.ResourceUsage
		observability.AllocationHeartbeat.Add(o.ctx, 1, metric.WithAttributes(
			observability.AttrDID,
			attribute.String("orchestratorID", o.id),
			attribute.String("allocationID", notification.AllocationID),
			attribute.String("status", notification.Status),
		))

		allocAttrs := metric.WithAttributes(
			observability.AttrDID,
			attribute.String("orchestratorID", o.id),
			attribute.String("allocationID", notification.AllocationID),
		)
		observability.AllocCPUUsage.Record(o.ctx, u.CPUUsagePercent, allocAttrs)
		observability.AllocMemUsed.Record(o.ctx, int64(u.MemoryUsedBytes), allocAttrs)
		observability.AllocMemLimit.Record(o.ctx, int64(u.MemoryLimitBytes), allocAttrs)
		observability.AllocNetRx.Record(o.ctx, int64(u.NetworkRxBytes), allocAttrs)
		observability.AllocNetTx.Record(o.ctx, int64(u.NetworkTxBytes), allocAttrs)
	}
}

// handleAllocationStatusUpdate receives immediate status change notifications
func (o *BasicOrchestrator) handleAllocationStatusUpdate(msg actor.Envelope) {
	defer msg.Discard()

	var update jtypes.AllocationStatusUpdate

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
	manifest := o.Manifest()
	a, ok := manifest.Allocations[manifestKey]
	if !ok {
		log.Debugf("allocation %s not found on the manifest", update.AllocationID)
		return
	}

	// update allocation status
	a.Status = jtypes.AllocationStatus(update.NewStatus)
	manifest.Allocations[manifestKey] = a
	o.updateManifest(manifest)
}
