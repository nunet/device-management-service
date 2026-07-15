// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/utils"
)

type AllocationLogsJobStatus string

const (
	AllocationLogsJobRunning  AllocationLogsJobStatus = "running"
	AllocationLogsJobComplete AllocationLogsJobStatus = "complete"
	AllocationLogsJobStopped  AllocationLogsJobStatus = "stopped"
	AllocationLogsJobFailed   AllocationLogsJobStatus = "failed"

	// DefaultLogsFollowInterval is used when Follow is set and no interval is given.
	DefaultLogsFollowInterval = 10 * time.Second
	minLogsFollowInterval     = time.Second
	maxLogsFollowInterval     = 5 * time.Minute
)

// AllocationLogsFetchOpts configures StartFetchAllocationLogs.
type AllocationLogsFetchOpts struct {
	// Follow keeps the fetch open after catching up to EOF, polling for new
	// data until StopFetchAllocationLogs or orchestrator shutdown.
	Follow bool
	// FollowInterval is how long to wait between polls when Follow is true.
	// Zero uses DefaultLogsFollowInterval; values outside [1s, 5m] are clamped.
	FollowInterval time.Duration
}

// AllocationLogsJob tracks a singleton async log fetch for one
// (requester DID, allocation) pair on this orchestrator/ensemble.
type AllocationLogsJob struct {
	RequesterDID   string
	AllocName      string
	LogsWrittenTo  string
	Status         AllocationLogsJobStatus
	Error          string
	BytesWritten   int64
	Follow         bool
	FollowInterval time.Duration
}

type allocationLogsJobState struct {
	AllocationLogsJob
	cancel context.CancelFunc
}

func logsJobKey(requesterDID, allocName string) string {
	return requesterDID + "\x00" + allocName
}

func normalizeFollowInterval(d time.Duration) time.Duration {
	if d <= 0 {
		return DefaultLogsFollowInterval
	}
	if d < minLogsFollowInterval {
		return minLogsFollowInterval
	}
	if d > maxLogsFollowInterval {
		return maxLogsFollowInterval
	}
	return d
}

// StartFetchAllocationLogs starts (or returns) the singleton async fetch
// for requesterDID + allocName. A second start while running is a no-op
// that returns the in-flight job. A completed/failed/stopped job can be restarted.
func (o *BasicOrchestrator) StartFetchAllocationLogs(
	requesterDID, allocName string,
	opts AllocationLogsFetchOpts,
) (AllocationLogsJob, error) {
	if requesterDID == "" {
		return AllocationLogsJob{}, fmt.Errorf("requester DID is required")
	}
	if allocName == "" {
		return AllocationLogsJob{}, fmt.Errorf("allocation name is required")
	}

	key := logsJobKey(requesterDID, allocName)
	interval := normalizeFollowInterval(opts.FollowInterval)

	o.logsJobsLock.Lock()
	if o.logsJobs == nil {
		o.logsJobs = make(map[string]*allocationLogsJobState)
	}
	if existing, ok := o.logsJobs[key]; ok && existing.Status == AllocationLogsJobRunning {
		job := existing.AllocationLogsJob
		o.logsJobsLock.Unlock()
		return job, nil
	}

	allocDir, err := o.allocationLogsDir(allocName)
	if err != nil {
		o.logsJobsLock.Unlock()
		return AllocationLogsJob{}, err
	}

	ctx, cancel := context.WithCancel(o.ctx)
	job := &allocationLogsJobState{
		AllocationLogsJob: AllocationLogsJob{
			RequesterDID:   requesterDID,
			AllocName:      allocName,
			LogsWrittenTo:  allocDir,
			Status:         AllocationLogsJobRunning,
			Follow:         opts.Follow,
			FollowInterval: interval,
		},
		cancel: cancel,
	}
	o.logsJobs[key] = job
	o.logsJobsLock.Unlock()

	go o.runFetchAllocationLogs(ctx, key)

	return job.AllocationLogsJob, nil
}

// GetFetchAllocationLogsJob returns the singleton job for requesterDID + allocName.
func (o *BasicOrchestrator) GetFetchAllocationLogsJob(requesterDID, allocName string) (AllocationLogsJob, bool) {
	o.logsJobsLock.RLock()
	defer o.logsJobsLock.RUnlock()

	job, ok := o.logsJobs[logsJobKey(requesterDID, allocName)]
	if !ok {
		return AllocationLogsJob{}, false
	}
	return job.AllocationLogsJob, true
}

// StopFetchAllocationLogs stops a running (typically Follow) fetch for this
// requester and allocation. Idempotent if already finished.
func (o *BasicOrchestrator) StopFetchAllocationLogs(requesterDID, allocName string) (AllocationLogsJob, error) {
	key := logsJobKey(requesterDID, allocName)

	o.logsJobsLock.Lock()
	defer o.logsJobsLock.Unlock()

	job, ok := o.logsJobs[key]
	if !ok {
		return AllocationLogsJob{}, fmt.Errorf(
			"no log fetch for allocation %s by requester %s",
			allocName, requesterDID,
		)
	}

	if job.Status == AllocationLogsJobRunning {
		if job.cancel != nil {
			job.cancel()
		}
		job.Status = AllocationLogsJobStopped
	}

	return job.AllocationLogsJob, nil
}

func (o *BasicOrchestrator) runFetchAllocationLogs(ctx context.Context, key string) {
	o.logsJobsLock.RLock()
	job, ok := o.logsJobs[key]
	if !ok {
		o.logsJobsLock.RUnlock()
		return
	}
	allocName := job.AllocName
	logsDir := job.LogsWrittenTo
	follow := job.Follow
	interval := job.FollowInterval
	o.logsJobsLock.RUnlock()

	stdoutPath := filepath.Join(logsDir, "stdout.log")
	stderrPath := filepath.Join(logsDir, "stderr.log")

	stdoutFile, err := o.fs.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		o.setLogsJobFailed(key, fmt.Errorf("open stdout log: %w", err))
		return
	}
	defer stdoutFile.Close()

	stderrFile, err := o.fs.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		o.setLogsJobFailed(key, fmt.Errorf("open stderr log: %w", err))
		return
	}
	defer stderrFile.Close()

	var stdoutOff, stderrOff int64
	sawAny := false

	for {
		if err := ctx.Err(); err != nil {
			o.setLogsJobStopped(key)
			return
		}

		n, next, err := o.drainLogStreamToFile(ctx, allocName, behaviors.LogStreamStdout, stdoutFile, stdoutOff, key)
		if err != nil {
			if ctx.Err() != nil {
				o.setLogsJobStopped(key)
				return
			}
			o.setLogsJobFailed(key, err)
			return
		}
		stdoutOff = next
		if n > 0 {
			sawAny = true
		}

		n, next, err = o.drainLogStreamToFile(ctx, allocName, behaviors.LogStreamStderr, stderrFile, stderrOff, key)
		if err != nil {
			if ctx.Err() != nil {
				o.setLogsJobStopped(key)
				return
			}
			o.setLogsJobFailed(key, err)
			return
		}
		stderrOff = next
		if n > 0 {
			sawAny = true
		}

		if !follow {
			if !sawAny {
				o.setLogsJobFailed(key,
					fmt.Errorf("stdout and stderr for allocation %s are empty (ensemble: %s)", allocName, o.id))
				return
			}
			o.setLogsJobComplete(key)
			return
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			o.setLogsJobStopped(key)
			return
		case <-timer.C:
		}
	}
}

func (o *BasicOrchestrator) drainLogStreamToFile(
	ctx context.Context,
	allocName string,
	stream behaviors.LogStream,
	f aferoFile,
	offset int64,
	key string,
) (int64, int64, error) {
	allocNodeHandle, behavior, err := o.allocationLogsTarget(allocName)
	if err != nil {
		return 0, offset, err
	}

	var bytesThisPass int64
	ack := offset

	for {
		if err := ctx.Err(); err != nil {
			return bytesThisPass, offset, err
		}

		resp, err := o.invokeAllocationLogs(allocNodeHandle, behavior, AllocationLogsRequest{
			AllocName: allocName,
			Stream:    stream,
			ChunkedTransferRequest: behaviors.ChunkedTransferRequest{
				Ack:      ack,
				Offset:   offset,
				MaxBytes: behaviors.DefaultLogChunkSize,
			},
		})
		if err != nil {
			return bytesThisPass, offset, err
		}
		if resp.Error != "" {
			return bytesThisPass, offset, fmt.Errorf("replied with error getting %s logs for %s: %s", stream, allocName, resp.Error)
		}

		if len(resp.Data) > 0 {
			n, err := f.Write(resp.Data)
			if err != nil {
				return bytesThisPass, offset, fmt.Errorf("write %s log: %w", stream, err)
			}
			bytesThisPass += int64(n)
			o.addLogsJobBytesWritten(key, int64(n))
		}

		ack = resp.NextOffset
		offset = resp.NextOffset
		if resp.EOF {
			break
		}
	}

	return bytesThisPass, offset, nil
}

// aferoFile is the subset of afero.File used by the drain loop.
type aferoFile interface {
	Write(p []byte) (n int, err error)
	Close() error
}

func (o *BasicOrchestrator) allocationLogsDir(allocName string) (string, error) {
	allocDir := filepath.Join(o.workDir, "deployments", o.id, allocName)
	if err := o.fs.MkdirAll(allocDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create allocation directory %s: %w", allocDir, err)
	}
	return allocDir, nil
}

func (o *BasicOrchestrator) allocationLogsTarget(allocName string) (actor.Handle, string, error) {
	var allocNodeHandle actor.Handle
	for _, n := range o.manifest.Nodes {
		if ok := utils.SliceContains(n.Allocations, allocName); ok {
			allocNodeHandle = n.Handle
			break
		}
	}

	if allocNodeHandle.Empty() {
		return actor.Handle{}, "", fmt.Errorf(
			"node not found for allocation %s of ensemble %s",
			allocName, o.id,
		)
	}

	behavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, o.manifest.ID)
	return allocNodeHandle, behavior, nil
}

func (o *BasicOrchestrator) addLogsJobBytesWritten(key string, n int64) {
	o.logsJobsLock.Lock()
	defer o.logsJobsLock.Unlock()

	if job, ok := o.logsJobs[key]; ok {
		job.BytesWritten += n
	}
}

func (o *BasicOrchestrator) setLogsJobComplete(key string) {
	o.logsJobsLock.Lock()
	defer o.logsJobsLock.Unlock()

	if job, ok := o.logsJobs[key]; ok {
		job.Status = AllocationLogsJobComplete
	}
}

func (o *BasicOrchestrator) setLogsJobStopped(key string) {
	o.logsJobsLock.Lock()
	defer o.logsJobsLock.Unlock()

	if job, ok := o.logsJobs[key]; ok {
		job.Status = AllocationLogsJobStopped
	}
}

func (o *BasicOrchestrator) setLogsJobFailed(key string, err error) {
	o.logsJobsLock.Lock()
	defer o.logsJobsLock.Unlock()

	if job, ok := o.logsJobs[key]; ok {
		job.Status = AllocationLogsJobFailed
		job.Error = err.Error()
	}
}
