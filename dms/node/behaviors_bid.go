package node

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
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

func (n *Node) storeBid(eid string, req jobtypes.BidRequest) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.bids[eid] = &bidState{
		expire:  time.Now().Add(bidStateTimeout),
		request: req,
	}
}

func (n *Node) getBid(eid string) (*bidState, bool) {
	n.lock.Lock()
	defer n.lock.Unlock()

	b, exists := n.bids[eid]

	return b, exists
}

func (n *Node) handleBidRequest(msg actor.Envelope) {
	defer msg.Discard()

	// ignore bid request from self if broadcast
	// only accept self bid if own peer specified on ensemble
	if msg.IsBroadcast() &&
		n.actor.Handle().Address.HostID == msg.From.Address.HostID {
		return
	}

	log.Debugf("got a bid request from: %+v", &msg.From.Address)

	onboarded, err := n.onboarding.IsOnboarded()
	if err != nil {
		log.Debugf("checking onboarding: %v", err)
		return
	}
	if !onboarded {
		log.Debugf("node not onboarded. ignoring bid request...")
		return
	}

	var request jobtypes.EnsembleBidRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("unmarshalling bid request: %s", err)
		return
	}

	_, bidExists := n.getBid(request.ID)
	if bidExists {
		log.Debugf("bid already sent for request: %s", request.ID)
		return
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		log.Debugf("get machine resources")
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
			log.Debug("bid request not v1")
			continue
		}

		// check if we are excluded
		hostID := n.actor.Handle().Address.HostID
		for _, p := range request.PeerExclusion {
			if p == hostID {
				log.Debug("bid request has exclusion")
				continue loop
			}
		}

		// if the desired executable is not found stop
		for _, e := range v.V1.Executors {
			executor, err := n.getExecutor(e)
			if err != nil {
				log.Debugf("failed to get executor: %+v", e)
				continue loop
			}
			if executor.executionType == jobtypes.ExecutorDocker {
				if v.V1.GeneralRequirements.PrivilegedDocker {
					if !n.dmsConfig.AllowPrivilegedDocker {
						log.Debugf("node does not allow privileged docker")
						continue loop
					}
				}
			}
		}

		comparisonResult, err := machineResources.Compare(v.V1.Resources)
		if err != nil {
			log.Debugf("compare machine resources")
			continue loop
		}

		if comparisonResult != types.Better {
			log.Debugf("resource comparison - not better - result: %+v", comparisonResult)
			continue
		}

		found = true
		toAnswer = v
		break
	}

	if !found {
		log.Debugf("bid requirements were not satisfied")
		return
	}

	if err := n.allocator.CheckAvailability(toAnswer.V1.PublicPorts.Static, toAnswer.V1.PublicPorts.Dynamic, toAnswer.V1.Resources); err != nil {
		log.Debugf("no resource availability: %v", err)
		return
	}

	log.Debugf("signing bid with did: %+v", n.actor.Security().DID())
	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		log.Debugf("bid request: failed to get provider: %w", err)
		return
	}
	log.Debugf("signing bid with provider: %v", provider.DID())

	var location jobtypes.Location
	if n.hostLocation.Empty() {
		loc, err := n.geolocate()
		if err != nil {
			log.Debugf("bid request: failed to geolocate node: %w", err)
		}
		location = loc
	} else {
		location = jobtypes.Location{
			City:      n.hostLocation.City,
			Country:   n.hostLocation.Country,
			Continent: n.hostLocation.Continent,
		}
	}

	bid := jobtypes.Bid{
		V1: &jobtypes.BidV1{
			EnsembleID: request.ID,
			NodeID:     toAnswer.V1.NodeID,
			Peer:       n.hostID,
			Location:   location,
			Handle:     n.actor.Handle(),
		},
	}

	err = bid.Sign(provider)
	if err != nil {
		log.Debugf("bid request: failed to sign bid: %w", err)
		return
	}

	n.sendReply(msg, bid)
	n.storeBid(request.ID, toAnswer)
}
