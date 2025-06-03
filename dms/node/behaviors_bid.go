package node

import (
	"encoding/json"
	"errors"
	"math/rand"
	"slices"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

const bidStateTimeout = 5 * time.Minute

func (n *Node) getExecutor(execType jobs.AllocationExecutor) (executorMetadata, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	e, ok := n.executors[string(execType)]
	if !ok {
		return executorMetadata{}, errors.New("executor not available")
	}

	return e, nil
}

func (n *Node) storeBid(eid string, nonce uint64, req jobtypes.BidRequest) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.bids[eid] = &bidState{
		expire:  time.Now().Add(bidStateTimeout),
		nonce:   nonce,
		request: req,
	}

	n.answeredBids[eid] = append(n.answeredBids[eid], nonce)
}

func (n *Node) getBid(eid string) (*bidState, bool) {
	n.lock.Lock()
	defer n.lock.Unlock()

	b, exists := n.bids[eid]

	return b, exists
}

func (n *Node) bidAnswered(eid string, nonce uint64) bool {
	n.lock.Lock()
	defer n.lock.Unlock()

	for e, n := range n.answeredBids {
		if e == eid && slices.Contains(n, nonce) {
			return true
		}
	}
	return false
}

func (n *Node) location() jobtypes.Location {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return jobtypes.Location{
		Continent: n.hostLocation.Continent,
		Country:   n.hostLocation.Country,
		City:      n.hostLocation.City,
	}
}

// TODO: ignore bid if our location is rejected or not included on accepted
func (n *Node) handleBidRequest(msg actor.Envelope) {
	defer msg.Discard()

	// ignore bid request from self if broadcast
	// only accept self bid if own peer specified on ensemble
	if msg.IsBroadcast() &&
		n.actor.Handle().Address.HostID == msg.From.Address.HostID {
		return
	}

	log.Debugw(
		"got a bid request from actor",
		"labels", string(observability.LabelDeployment),
		"from", msg.From.Address,
	)

	onboarded := n.onboarding.IsOnboarded()
	if !onboarded {
		log.Debugw(
			"node_not_onboarded_ignoring_bid_request",
			"labels", string(observability.LabelDeployment),
		)
		return
	}

	var request jobtypes.EnsembleBidRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw(
			"unmarshal_bid_request_error",
			"labels", string(observability.LabelDeployment),
			"error", err,
		)
		return
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		log.Debugw(
			"machine_resources_retrieval_error",
			"labels", string(observability.LabelDeployment),
			"error", err,
		)
		return
	}

	// randomize the order of bid request checks
	rand.Shuffle(len(request.Request), func(i, j int) {
		request.Request[i], request.Request[j] = request.Request[j], request.Request[i]
	})

	// find the first bid request that matches
	var toAnswer jobtypes.BidRequest
	var found bool

loop:
	for _, v := range request.Request {
		// check if it is a V1 request
		if v.V1 == nil {
			log.Debugw("bid_request_not_v1",
				"labels", string(observability.LabelDeployment))
			continue
		}

		answered := n.bidAnswered(request.ID, request.Nonce)
		if answered {
			log.Debugf("bid already answered: ensembleID: %s, nonce: %d", request.ID, request.Nonce)
			return
		}

		// check if we are excluded
		hostID := n.actor.Handle().Address.HostID
		for _, p := range request.PeerExclusion {
			if p == hostID {
				log.Debugw("bid_request_excluded_peer",
					"labels", string(observability.LabelDeployment),
					"hostID", hostID)
				continue loop
			}
		}

		// if the desired executable is not found stop
		for _, e := range v.V1.Executors {
			executor, err := n.getExecutor(e)
			if err != nil {
				log.Debugw("executor_unavailable",
					"labels", string(observability.LabelDeployment),
					"executor", e,
					"error", err)
				continue loop
			}

			if executor.executionType == jobtypes.ExecutorDocker {
				if v.V1.GeneralRequirements.PrivilegedDocker {
					if !n.dmsConfig.AllowPrivilegedDocker {
						log.Debugw("privileged_docker_not_allowed",
							"labels", string(observability.LabelDeployment))
						continue loop
					}
				}
			}
		}

		comparisonResult, err := machineResources.Compare(v.V1.Resources)
		if err != nil {
			log.Debugw("compare_machine_resources_error",
				"labels", string(observability.LabelDeployment),
				"error", err)
			continue loop
		}

		if comparisonResult != types.Better {
			log.Debugw("resource_not_better",
				"labels", string(observability.LabelDeployment),
				"comparisonResult", comparisonResult)
			continue
		}

		found = true
		toAnswer = v
		break
	}

	if !found {
		log.Debugw("bid_requirements_not_satisfied",
			"labels", string(observability.LabelDeployment))
		return
	}

	if err := n.allocator.CheckAvailability(toAnswer.V1.PublicPorts.Static, toAnswer.V1.PublicPorts.Dynamic, toAnswer.V1.Resources); err != nil {
		log.Debugw("no_resource_availability_for_bid",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}

	log.Debugw("signing_bid_with_node_identity",
		"labels", string(observability.LabelDeployment),
		"DID", n.actor.Security().DID())

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		log.Debugw("provider_retrieval_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}
	log.Debugw("signing_bid_with_provider",
		"labels", string(observability.LabelDeployment),
		"providerDID", provider.DID())

	bid := jobtypes.Bid{
		V1: &jobtypes.BidV1{
			EnsembleID: request.ID,
			NodeID:     toAnswer.V1.NodeID,
			Peer:       n.hostID,
			Location:   n.location(),
			Handle:     n.actor.Handle(),
		},
	}

	if err := bid.Sign(provider); err != nil {
		log.Debugw("bid_signing_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}

	n.sendReply(msg, bid)
	n.storeBid(request.ID, request.Nonce, toAnswer)
}
