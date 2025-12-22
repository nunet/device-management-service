// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func (a *Allocation) handleAllocationStart(msg actor.Envelope) {
	log.Infow("behavior_allocation_start_invoked",
		"labels", string(observability.LabelAllocation),
		"from", msg.From)
	defer msg.Discard()

	var req behaviors.AllocationStartRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Errorw("allocation_start_request_unmarshal_error",
			"labels", string(observability.LabelAllocation),
			"error", err)
		return
	}

	var resp behaviors.AllocationStartResponse

	// Store state regardless of whether we're running or in standby
	a.state.subnetIP = req.SubnetIP
	a.state.gatewayIP = req.GatewayIP
	a.state.portMapping = req.PortMapping

	// TODO: context should cancel when the actor is stopped to stop monitor
	if err := a.Run(context.TODO(), req.SubnetIP, req.GatewayIP, req.PortMapping); err != nil {
		err = fmt.Errorf("failed to run allocation: %w", err)
		log.Errorw("allocation_start_run_failure",
			"labels", string(observability.LabelAllocation),
			"error", err)
		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	log.Infow("allocation_start_success",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID)

	resp.OK = true
	a.sendReply(msg, resp)
}

type AllocationRestartResponse struct {
	OK    bool
	Error string
}

func (a *Allocation) handleAllocationRestart(msg actor.Envelope) {
	defer msg.Discard()

	resp := behaviors.AllocationRestartResponse{}
	if err := a.Restart(context.TODO()); err != nil { // TODO: fix context.TODO()
		err = fmt.Errorf("failed to restart allocation: %w", err)
		log.Errorw("allocation_restart_failure",
			"labels", string(observability.LabelAllocation),
			"error", err)
		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	log.Infow("allocation_restart_success",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID)
	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleAllocationStats(msg actor.Envelope) {
	defer msg.Discard()

	var resp behaviors.AllocationStatsResponse
	if len(msg.Message) > 0 {
		var req behaviors.AllocationStatsRequest
		if err := json.Unmarshal(msg.Message, &req); err != nil {
			resp.Error = err.Error()
			a.sendReply(msg, resp)
			return
		}
	}

	// zero resource usage if allocation not running
	if a.Status().Status != jobtypes.AllocationRunning {
		resp.Stats = &types.ExecutorStats{}
		resp.OK = true
		a.sendReply(msg, resp)
		return
	}

	if a.executor == nil {
		err := fmt.Errorf("allocation executor not initialized")
		log.Errorw("allocation_stats_executor_nil",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID,
			"error", err,
		)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	stats, err := a.executor.Stats(context.TODO(), a.executionID) // TODO: fix context.TODO()
	if err != nil {
		err = fmt.Errorf("failed to retrieve allocation stats: %w", err)
		log.Errorw("allocation_stats_failure",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID,
			"executionID", a.executionID,
			"error", err,
		)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	resp.Stats = stats
	a.sendReply(msg, resp)
}

func (a *Allocation) handleRegisterHealthcheck(msg actor.Envelope) {
	defer msg.Discard()

	var request behaviors.RegisterHealthcheckRequest
	resp := behaviors.RegisterHealthcheckResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	healthcheck, err := types.NewHealthCheck(request.HealthCheck, func(mf types.HealthCheckManifest) error {
		return a.execHealthCheck(mf)
	})
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	a.SetHealthCheck(healthcheck)
	resp.OK = true
	a.sendReply(msg, resp)
}

type HealthCheckResponse struct {
	OK    bool
	Error string
}

func (a *Allocation) handleHealthcheck(msg actor.Envelope) {
	defer msg.Discard()

	a.lock.Lock()
	healthcheck := a.healthcheck
	a.lock.Unlock()

	var resp HealthCheckResponse
	if healthcheck != nil {
		if err := healthcheck(); err != nil {
			resp.Error = err.Error()
		} else {
			resp.OK = true
		}
	} else {
		resp.OK = true
	}

	reply, err := actor.ReplyTo(msg, resp)
	if err != nil {
		log.Warnw("allocation_healthcheck_reply_creation_failure",
			"labels", string(observability.LabelAllocation),
			"error", err)
		return
	}
	if err := a.Actor.Send(reply); err != nil {
		log.Warnw("allocation_healthcheck_reply_send_failure",
			"labels", string(observability.LabelAllocation),
			"error", err)
	}
}

func (a *Allocation) execHealthCheck(mf types.HealthCheckManifest) error {
	exitCode, stdout, stderr, err := a.executor.Exec(context.TODO(), a.executionID, mf.Exec)

	log.Debugw("health_check_command_output",
		"labels", string(observability.LabelAllocation),
		"command", mf.Exec,
		"stdout", stdout,
		"stderr", stderr)
	if err != nil {
		log.Warnw("health_check_command_exec_failure",
			"labels", string(observability.LabelAllocation),
			"error", err)
		return fmt.Errorf("health check command failed: %w", err)
	}

	if exitCode != 0 {
		log.Warnw("health_check_command_exitcode_failure",
			"labels", string(observability.LabelAllocation),
			"exitCode", exitCode)
		return fmt.Errorf("health check command failed with exit code %d", exitCode)
	}

	if !strings.Contains(stdout+stderr, mf.Response.Value) {
		log.Warnw("health_check_command_unexpected_output",
			"labels", string(observability.LabelAllocation),
			"stderr", stderr,
			"expectedValue", mf.Response.Value)
		return fmt.Errorf("unexpected health check command output: %s\nstderr: %s", stdout, stderr)
	}

	log.Debugw("health_check_command_succeeded",
		"labels", string(observability.LabelAllocation))
	return nil
}
