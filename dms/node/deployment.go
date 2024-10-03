package node

import (
	"encoding/json"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
)

const (
	NewDeploymentBehavior = "/dms/deployment/new"

	// Minimum time for deployment
	MinDeploymentTime = time.Minute - time.Second
)

type NewDeploymentRequest struct {
	Ensemble jobs.EnsembleConfig
}

type NewDeploymentResponse struct {
	Status   string
	Ensemble *jobs.EnsembleManifest `json:",omitempty"`
	Error    string                 `json:",omitempty"`
}

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

	orchestrator, err := n.createOrchestrator(request.Ensemble)
	if err != nil {
		log.Warnf("creating orchestrator: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	manifest, err := orchestrator.Deploy(msg.Expiry())
	if err != nil {
		orchestrator.Stop()
		log.Warnf("creating ensemble: %s", err)
		n.sendReply(msg, NewDeploymentResponse{
			Status: "ERROR",
			Error:  err.Error(),
		})
		return
	}

	n.mx.Lock()
	n.deployments[manifest.ID] = orchestrator
	n.mx.Unlock()

	log.Infof("created ensemble: %s", manifest.ID)
	n.sendReply(msg, NewDeploymentResponse{
		Status:   "OK",
		Ensemble: &manifest,
	})
}

func (n *Node) createOrchestrator(_ jobs.EnsembleConfig) (*jobs.Orchestrator, error) {
	// TODO
	return nil, ErrTODO
}

func (n *Node) saveDeployments() error {
	// TODO
	return nil
}

func (n *Node) restoreDeployments() error {
	// TODO
	return nil
}
