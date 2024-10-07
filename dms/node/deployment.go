package node

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
)

const (
	NewDeploymentBehavior = "/dms/node/deployment/new"

	// Minimum time for deployment
	MinDeploymentTime = time.Minute - time.Second
)

type NewDeploymentRequest struct {
	Ensemble jobs.EnsembleConfig
}

type NewDeploymentResponse struct {
	Status     string
	EnsembleID string `json:",omitempty"`
	Error      string `json:",omitempty"`
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
	if err := n.saveDeployment(orchestrator.ID()); err != nil {
		log.Errorf("error saving deployment %s: %s", orchestrator.ID(), err)
	}
	n.mx.Unlock()
}

func (n *Node) deploymentVerifyEdgeConstraint(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.VerifyEdgeConstraintRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Warnf("error unmarshalling constraint request: %s", err)
		n.sendReply(msg, jobs.VerifyEdgeConstraintResponse{
			OK:    false,
			Error: err.Error(),
		})
	}

	// TODO
}

func (n *Node) createOrchestrator(_ jobs.EnsembleConfig) (*jobs.Orchestrator, error) {
	// TODO
	return nil, ErrTODO
}

func (n *Node) saveDeployments() error {
	n.mx.Lock()
	defer n.mx.Unlock()

	var failed []string
	for oid := range n.deployments {
		if err := n.saveDeployment(oid); err != nil {
			log.Errorf("error saving deployment %s: %s", oid, err)
			failed = append(failed, oid)
		}
	}

	if len(failed) != 0 {
		return fmt.Errorf("failed to save deployments: %v", failed)
	}

	return nil
}

func (n *Node) saveDeployment(_ string) error {
	// TODO
	return nil
}

func (n *Node) restoreDeployments() error {
	// TODO
	return nil
}
