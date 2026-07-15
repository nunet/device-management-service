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
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
)

func (o *BasicOrchestrator) GetAllocationLogs(name string) (AllocationLogsResponse, error) {
	var logsResp AllocationLogsResponse

	allocNodeHandle, behavior, err := o.allocationLogsTarget(name)
	if err != nil {
		return logsResp, err
	}

	stdout, err := o.fetchAllocationLogStream(name, allocNodeHandle, behavior, behaviors.LogStreamStdout)
	if err != nil {
		return logsResp, err
	}

	stderr, err := o.fetchAllocationLogStream(name, allocNodeHandle, behavior, behaviors.LogStreamStderr)
	if err != nil {
		return logsResp, err
	}

	if len(stdout) == 0 && len(stderr) == 0 {
		return logsResp,
			fmt.Errorf("stdout and stderr for allocation %s are empty (ensemble: %s)", name, o.id)
	}

	logsResp.Stdout = stdout
	logsResp.Stderr = stderr
	return logsResp, nil
}

func (o *BasicOrchestrator) fetchAllocationLogStream(
	name string,
	allocNodeHandle actor.Handle,
	behavior string,
	stream behaviors.LogStream,
) ([]byte, error) {
	var buf []byte
	offset := int64(0)
	ack := int64(0)

	for {
		resp, err := o.invokeAllocationLogs(allocNodeHandle, behavior, AllocationLogsRequest{
			AllocName: name,
			Stream:    stream,
			ChunkedTransferRequest: behaviors.ChunkedTransferRequest{
				Ack:      ack,
				Offset:   offset,
				MaxBytes: behaviors.DefaultLogChunkSize,
			},
		})
		if err != nil {
			return nil, err
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("replied with error getting %s logs for %s: %s", stream, name, resp.Error)
		}

		buf = append(buf, resp.Data...)
		ack = resp.NextOffset
		offset = resp.NextOffset
		if resp.EOF {
			break
		}
	}

	return buf, nil
}

func (o *BasicOrchestrator) invokeAllocationLogs(
	allocNodeHandle actor.Handle,
	behavior string,
	req AllocationLogsRequest,
) (AllocationLogsResponse, error) {
	var logsResp AllocationLogsResponse

	msg, err := actor.Message(
		o.actor.Handle(),
		allocNodeHandle,
		behavior,
		req,
		actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
	)
	if err != nil {
		return logsResp, fmt.Errorf("creating get logs message: %w", err)
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return logsResp, fmt.Errorf("invoking get logs message: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(2 * time.Minute):
		return logsResp, fmt.Errorf("timeout getting logs for %s: %w", req.AllocName, ErrDeploymentFailed)
	}

	defer reply.Discard()

	if err := json.Unmarshal(reply.Message, &logsResp); err != nil {
		return logsResp, fmt.Errorf("unmarshalling get logs response: %w", err)
	}

	return logsResp, nil
}
