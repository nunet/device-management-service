package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	NewDeploymentBehavior = "/dms/node/deployment/new"

	// Minimum time for deployment
	MinDeploymentTime = time.Minute - time.Second

	bidStateGCInterval = time.Minute
	bidStateTimeout    = 5 * time.Minute
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

func (n *Node) createOrchestrator(ensemble jobs.EnsembleConfig) (*jobs.Orchestrator, error) {
	orch, err := jobs.NewOrchestrator(n.actor, n.network, ensemble)
	if err != nil {
		return nil, err
	}

	return orch, nil
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

func (n *Node) handleBidRequest(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.EnsembleBidRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		return
	}

	// randomize the order of bid request checks
	rand.Shuffle(len(request.Request), func(i, j int) {
		request.Request[i], request.Request[j] = request.Request[j], request.Request[i]
	})

	// find the first bid request that matches
	var toAnswer jobs.BidRequest
	var found bool
loop:
	for _, v := range request.Request {
		// check if it is a V1 request
		if v.V1 == nil {
			continue
		}

		// check if we are excluded
		hostID := n.actor.Handle().Address.HostID
		for _, p := range request.PeerExclusion {
			if p == hostID {
				continue loop
			}
		}

		// TODO allow static ports
		if len(v.V1.PublicPorts.Static) > 0 {
			continue loop
		}

		// if the desired executable is not found stop
		for _, e := range v.V1.Executors {
			_, err := n.getExecutor(e)
			if err != nil {
				continue loop
			}
		}

		comparisonResult, err := machineResources.Compare(v.V1.Resources)
		if err != nil {
			continue loop
		}

		if comparisonResult != types.Better {
			continue
		}

		found = true
		toAnswer = v
		break
	}

	if !found {
		return
	}

	// handle dynamic port allocs
	allocKey := request.ID
	ports, err := n.portAllocator.Allocate(allocKey, toAnswer.V1.PublicPorts.Dynamic)
	if err != nil {
		return
	}
	cleanup := func() {
		n.portAllocator.Release(allocKey)
	}

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		cleanup()
		return
	}

	allFields := bytes.Join([][]byte{
		[]byte(request.ID),
		[]byte(n.hostID),
		[]byte(n.hostLocation.HostContinent),
		[]byte(n.hostLocation.HostCountry),
		[]byte(n.hostLocation.HostCity),
	}, []byte{})

	sig, err := provider.Sign(allFields)
	if err != nil {
		cleanup()
		return
	}

	bid := jobs.Bid{
		V1: &jobs.BidV1{
			EnsembleID: request.ID,
			NodeID:     toAnswer.V1.NodeID,
			Peer:       n.hostID,
			Location: jobs.Location{
				Region:  n.hostLocation.HostContinent,
				Country: n.hostLocation.HostCountry,
				City:    n.hostLocation.HostCity,
			},
			Handle:    n.actor.Handle(),
			Signature: sig,
		},
	}

	n.sendReply(msg, bid)
	n.rememberBid(request.ID, toAnswer, ports)
}

func (n *Node) rememberBid(eid string, req jobs.BidRequest, ports []int) {
	n.mx.Lock()
	defer n.mx.Unlock()

	_, exists := n.bids[eid]
	if exists {
		// we have an older bid
		n.portAllocator.Release(eid)
	}

	n.bids[eid] = &bidState{
		expire:  time.Now().Add(bidStateTimeout),
		request: req,
		ports:   ports,
	}
}

func (n *Node) gcBidState() {
	ticker := time.NewTicker(bidStateGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.doGCBidState()

		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) doGCBidState() {
	now := time.Now()

	n.mx.Lock()
	defer n.mx.Unlock()

	for k, bs := range n.bids {
		if bs.expire.Before(now) {
			n.portAllocator.Release(k)
			delete(n.bids, k)
		}
	}
}
