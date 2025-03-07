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
	"gitlab.com/nunet/device-management-service/types"
)

func (a *Allocation) handleAllocationStart(msg actor.Envelope) {
	log.Infof("behavior allocation start invoked by: %+v", msg.From)
	defer msg.Discard()

	var req behaviors.AllocationStartRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Errorf("error unmarshalling allocation start request: %s", err)
		return
	}

	var resp behaviors.AllocationStartResponse
	// TODO: context should cancel when the actor is stopped to stop monitor
	if err := a.Run(context.TODO(), req.SubnetIP, req.GatewayIP, req.PortMapping); err != nil {
		err = fmt.Errorf("failed to run allocation: %w", err)
		log.Error(err)

		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	log.Info("Running allocation's job: ", a.ID)

	a.state.subnetIP = req.SubnetIP
	a.state.gatewayIP = req.GatewayIP
	a.state.portMapping = req.PortMapping

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
		log.Error(err)

		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
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
		exitCode, stdout, stderr, err := a.executor.Exec(context.TODO(), a.executionID, mf.Exec)

		log.Debugf("health check command: %s\nstdout: %s\nstderr: %s", mf.Exec, stdout, stderr)
		if err != nil {
			log.Warnf("health check command failed: %s", err)
			return fmt.Errorf("health check command failed: %w", err)
		}

		if exitCode != 0 {
			log.Warnf("health check command failed with exit code: %d", exitCode)
			return fmt.Errorf("health check command failed with exit code %d", exitCode)
		}

		if !strings.Contains(stdout+stderr, mf.Response.Value) {
			log.Warnf("health check command err: %s", stderr)
			return fmt.Errorf("unexpected health check command output: %s\nstderr: %s", stdout, stderr)
		}

		log.Debugf("health check command succeeded")
		return nil
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
		log.Warnf("failed to create healthcheck reply: %s", err)
		return
	}
	if err := a.Actor.Send(reply); err != nil {
		log.Warnf("failed to send healthcheck reply: %s", err)
	}
}
