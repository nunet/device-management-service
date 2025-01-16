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
	"path/filepath"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

type NewDeploymentRequest struct {
	Ensemble job_types.EnsembleConfig
}

type NewDeploymentResponse struct {
	Status     string
	EnsembleID string `json:",omitempty"`
	Error      string `json:",omitempty"`
}

// TODO: rename -> handleNewDeployment
func (n *Node) newDeployment(msg actor.Envelope) {
	defer msg.Discard()

	if time.Until(msg.Expiry()) < MinDeploymentTime {
		log.Debugf("deployment time too short")
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  "requested deployment time too short",
		})
		return
	}

	var request NewDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("unmarshalling deployment request: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	childCtx := context.WithoutCancel(n.ctx)
	orchestrator, err := n.createOrchestrator(childCtx, request.Ensemble)
	if err != nil {
		log.Warnf("creating orchestrator: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	n.mx.Lock()
	n.deployments[orchestrator.ID()] = orchestrator
	n.mx.Unlock()

	log.Infof("deploying ensemble: %s", orchestrator.ID())
	n.sendReply(msg, NewDeploymentResponse{
		Status:     "OK",
		EnsembleID: orchestrator.ID(),
	})

	if err := orchestrator.Deploy(msg.Expiry().Add(-jobs.MinEnsembleDeploymentTime)); err != nil {
		orchestrator.Stop()
		log.Errorf("error creating ensemble: %s", err)
		n.mx.Lock()
		delete(n.deployments, orchestrator.ID())
		n.mx.Unlock()

		return
	}

	// save the deployment
	n.mx.Lock()
	if err := n.saveDeployment(orchestrator); err != nil {
		log.Errorf("error saving deployment %s: %s", orchestrator.ID(), err)
	}
	n.mx.Unlock()
}

type DeploymentListResponse struct {
	Deployments map[string]string
}

func (n *Node) handleDeploymentList(msg actor.Envelope) {
	defer msg.Discard()

	var resp DeploymentListResponse

	resp.Deployments = make(map[string]string)
	for ID, dep := range n.deployments {
		resp.Deployments[ID] = job_types.DeploymentStatusString(dep.Status())
	}

	n.sendReply(msg, resp)
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

	var request DeploymentLogsRequest
	var resp DeploymentLogsResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.EnsembleID]
	n.mx.Unlock()
	if !ok {
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	data, err := d.GetAllocationLogs(request.AllocationName)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	ensembleDir := filepath.Join(
		n.dmsConfig.WorkDir,
		"deployments",
		request.EnsembleID,
	)
	allocDir := filepath.Join(ensembleDir, request.AllocationName)

	err = n.writeAllocationLogsTo(allocDir, data.Stdout, data.Stderr)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
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

	var request DeploymentStatusRequest
	var resp DeploymentStatusResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.ID]
	n.mx.Unlock()
	if !ok {
		// TODO: check database for persisted deployments data
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Status = job_types.DeploymentStatusString(d.Status())
	n.sendReply(msg, resp)
}

type DeploymentManifestRequest struct {
	ID string
}

type DeploymentManifestResponse struct {
	Manifest jobs.EnsembleManifest
	Error    string
}

func (n *Node) handleDeploymentManifest(msg actor.Envelope) {
	defer msg.Discard()

	var request DeploymentManifestRequest
	var resp DeploymentManifestResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	n.mx.Lock()
	d, ok := n.deployments[request.ID]
	n.mx.Unlock()
	if !ok {
		// TODO: check database for persisted deployments data
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Manifest = d.Manifest()
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

	var request DeploymentShutdownRequest
	var resp DeploymentShutdownResponse

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	d, ok := n.deployments[request.ID]
	if !ok {
		resp.Error = ErrDeploymentNotFound.Error()
		n.sendReply(msg, resp)
		return
	}

	if d.Status() != jobs.DeploymentStatusRunning {
		log.Debugf("deployment %q is not running(status=%q), cannot shutdown", request.ID, d.Status())
		// maybe-TODO: if it's still provisioning/committing,
		// we should stop the deployment process anyway
		resp.Error = ErrDeploymentNotRunning.Error()
		n.sendReply(msg, resp)
		return
	}

	d.Shutdown()
	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleCommitDeployment(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.CommitDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.CommitDeploymentResponse{}
	allocationID := n.constructAllocationID(request.EnsembleID, request.AllocationName)
	err := n.commitDeployment(request.EnsembleID, allocationID, request.Resources, request.PortMapping)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}
