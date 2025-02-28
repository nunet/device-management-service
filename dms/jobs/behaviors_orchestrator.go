package jobs

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/nunet/device-management-service/actor"
)

type TaskTerminationNotification struct {
	AllocationID string
	Status       AllocationStatus

	Error TerminationError

	Stdout []byte
	Stderr []byte
}

// TerminationError holds information necessary to handle
// failure recovery given retry policies.
type TerminationError struct {
	Err      error
	ExitCode int
	// Killed is used to identify if the application was killed
	// by external means, rather than app exiting itself
	Killed bool
}

func (o *Orchestrator) handleTaskTermination(msg actor.Envelope) {
	msg.Discard()

	var req TaskTerminationNotification

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
	a.Status = req.Status
	o.manifest.Allocations[allocName] = a
	o.lock.Unlock()

	if req.Error.Err != nil {
		log.Errorf(
			"allocation task %s yielded error: %v",
			req.AllocationID, req.Error,
		)
	}

	allocDir, err := o.WriteAllocationLogs(allocName, req.Stdout, req.Stderr)
	if err != nil {
		log.Errorf("failed to write logs for allocation %s: %v", allocName, err)
	}

	log.Infof("allocation logs for %s written to %s (ensemble: %s)", allocName, allocDir, o.id)
}

func (o *Orchestrator) WriteAllocationLogs(
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
