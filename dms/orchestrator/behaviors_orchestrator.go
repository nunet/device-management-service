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
	"strings"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

func (o *BasicOrchestrator) handleTaskTermination(msg actor.Envelope) {
	msg.Discard()

	var req behaviors.TaskTerminationNotification

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Debugf("unmarshalling task completion request: %s", err)
		return
	}

	log.Infof("notification: task %s terminated with status: %s", req.AllocationID, req.Status)

	allocName := allocNameFromID(req.AllocationID)
	a, ok := o.manifest.Allocations[allocName]
	if !ok {
		log.Debugf("allocation %s not found on the manifest", req.AllocationID)
		return
	}

	// update allocation status
	o.lock.Lock()
	a.Status = jtypes.AllocationStatus(req.Status)
	o.manifest.Allocations[allocName] = a
	o.lock.Unlock()

	if req.Error.Err != "" {
		log.Errorf(
			"allocation task %s yielded error: %v",
			req.AllocationID, req.Error,
		)
		return
	}

	allocDir, err := o.WriteAllocationLogs(allocName, req.Stdout, req.Stderr)
	if err != nil {
		log.Errorf("failed to write logs for allocation %s: %v", allocName, err)
		return
	}

	log.Infof("allocation logs for %s written to %s (ensemble: %s)", allocName, allocDir, o.id)
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

	err := o.fs.MkdirAll(allocDir, 0o744)
	if err != nil {
		return "", fmt.Errorf("failed to create allocation directory %s: %w", allocDir, err)
	}

	if len(stdout) > 0 {
		stdoutPath := filepath.Join(allocDir, "stdout.logs")
		err = o.fs.WriteFile(stdoutPath, stdout, 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write stdout logs to %s: %w", stdoutPath, err)
		}
	}

	if len(stderr) > 0 {
		stderrPath := filepath.Join(allocDir, "stderr.logs")
		err = o.fs.WriteFile(stderrPath, stderr, 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write stderr logs to %s: %w", stderrPath, err)
		}
	}

	return allocDir, nil
}

func allocNameFromID(id string) string {
	parts := strings.Split(id, "_")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
