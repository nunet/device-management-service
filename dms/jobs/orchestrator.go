package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/network"
)

type Orchestrator struct {
	actor   actor.Actor
	network network.Network //nolint

	id       string
	cfg      EnsembleConfig //nolint
	manifest EnsembleManifest

	running bool
	ctx     context.Context
	cancel  func()
}

func (o *Orchestrator) Deploy(expiry time.Time) (EnsembleManifest, error) {
deploy:
	for time.Now().Before(expiry) {
		// 1. Create bid requests for nodes
		bidrq, err := o.makeInitialBidRequest()
		if err != nil {
			return EnsembleManifest{}, fmt.Errorf("creating bid request: %w", err)
		}

		// 2. Collect bids
		bidMap := make(map[string][]Bid)
		peerExclusion := make(map[string]struct{})
		addBid := func(bid Bid) {
			// check that the peer has not already submitted a bid
			peerID := bid.Peer()
			if _, exclude := peerExclusion[peerID]; exclude {
				log.Debugf("ignoring duplicate bid from peer %s", peerID)
				return
			}

			// verify the location constraints of the node
			nodeID := bid.NodeID()
			loc := bid.Location()
			if !o.acceptPeerLocation(nodeID, loc) {
				log.Debugf("ignoring out of location bid from peer %s for node %s", peerID, nodeID)
				return
			}

			bidMap[nodeID] = append(bidMap[nodeID], bid)
			peerExclusion[peerID] = struct{}{}
		}

		bidCh, bidExpiryTime, err := o.requestBids(bidrq, expiry)
		if err != nil {
			return EnsembleManifest{}, fmt.Errorf("collecting bids: %w", err)
		}

		o.collectBids(bidCh, bidExpiryTime, addBid)

		// 3. Create a candidate deployment
		var candidate map[string]Bid
		var ok bool
		for time.Now().Before(expiry) {
			candidate, ok = o.makeCandidateDeployment(bidMap)
			if ok {
				break
			}

			// we don't have bids for some of our nodes so we don't have a candidate
			// we need to make a residual bid request for the remaining nodes
			// Note: in order to facilitate random selection, the residual bid requests
			//       can drop some of the original bids
			bidrq, err := o.makeResidualBidRequest(bidMap, peerExclusion)
			if err != nil {
				return EnsembleManifest{}, fmt.Errorf("creating residual bid request: %w", err)
			}

			bidCh, bidExpiryTime, err := o.requestBids(bidrq, expiry)
			if err != nil {
				return EnsembleManifest{}, fmt.Errorf("collecting residual bids: %w", err)
			}

			o.collectBids(bidCh, bidExpiryTime, addBid)
		}

		if !ok {
			log.Debugf("failed to create candidate deployment")
			continue deploy
		}

		// 5. Check the edge constraints
		if err := o.verifyEdgeConstraints(candidate); err != nil {
			log.Debugf("failed to verify edge constraints: %s", err)
			continue deploy
		}

		// 6. Commit the deployment
		manifest, err := o.commitDeployment(candidate)
		if err != nil {
			log.Warnf("failed to commit deployment: %s", err)
			continue deploy
		}

		// 7. provision the network
		if err := o.provision(manifest); err != nil {
			log.Errorf("failed to privision network: %s", err)
			o.revertDeployment(manifest)
			continue deploy
		}

		// 8. start the deployment
		if err := o.start(manifest); err != nil {
			log.Errorf("failed to start the deployment: %s", err)
			o.revertDeployment(manifest)
			continue deploy
		}

		// We are done! start the supervisor return the manifest.
		o.manifest = manifest
		o.running = true
		o.ctx, o.cancel = context.WithCancel(context.Background())
		go o.supervise()

		return manifest, nil
	}

	// we failed to create the deployment in time
	return EnsembleManifest{}, ErrDeploymentFailed
}

func (o *Orchestrator) requestBids(bidrq EnsembleBidRequest, expiry time.Time) (chan Bid, time.Time, error) {
	log.Debugf("requesting bids: %+v", bidrq)

	bidExpiryTime := time.Now().Add(BidRequestTimeout)
	if expiry.Before(bidExpiryTime) {
		return nil, time.Time{}, fmt.Errorf("not enough time for deployment: %w", ErrDeploymentFailed)
	}

	bidExpiry := uint64(bidExpiryTime.UnixNano())
	msg, err := actor.Message(
		o.actor.Handle(),
		actor.Handle{},
		BidRequestBehavior,
		bidrq,
		actor.WithMessageTopic(BidRequestTopic),
		actor.WithMessageReplyTo(BidReplyBehavior),
		actor.WithMessageExpiry(bidExpiry),
	)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("creating bid request message: %w", err)
	}

	bidCh := make(chan Bid)
	if err := o.actor.AddBehavior(
		BidReplyBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var bid Bid
			if err := json.Unmarshal(msg.Message, &bid); err != nil {
				log.Debugf("failed to unmarshal bid from %s: %s", msg.From, err)
				return
			}

			timer := time.NewTimer(time.Until(bidExpiryTime))
			defer timer.Stop()

			select {
			case bidCh <- bid:
			case <-timer.C:
			}
		},
		actor.WithBehaviorExpiry(bidExpiry),
	); err != nil {
		return nil, time.Time{}, fmt.Errorf("adding bid behavior: %w", err)
	}

	if err := o.actor.Publish(msg); err != nil {
		return nil, time.Time{}, fmt.Errorf("publishing bid request: %w", err)
	}

	return bidCh, bidExpiryTime, nil
}

func (o *Orchestrator) collectBids(bidCh chan Bid, bidExpiryTime time.Time, addBid func(Bid)) {
	timer := time.NewTimer(time.Until(bidExpiryTime))
	defer timer.Stop()

	for {
		select {
		case bid := <-bidCh:
			if err := bid.Validate(); err != nil {
				log.Debugf("got invalid bid: %s", err)
				continue
			}
			if bid.EnsembleID() != o.id {
				log.Debugf("got bid for unexpected ensemble ID: %s", bid.EnsembleID())
				continue
			}
			addBid(bid)
		case <-timer.C:
			return
		}
	}
}

func (o *Orchestrator) makeCandidateDeployment(_ map[string][]Bid) (map[string]Bid, bool) {
	// TODO
	return nil, false
}

func (o *Orchestrator) verifyEdgeConstraints(_ map[string]Bid) error {
	// TODO
	return ErrTODO
}

func (o *Orchestrator) commitDeployment(_ map[string]Bid) (EnsembleManifest, error) {
	// TODO
	return EnsembleManifest{}, ErrTODO
}

func (o *Orchestrator) provision(_ EnsembleManifest) error {
	// TODO
	return ErrTODO
}

func (o *Orchestrator) start(_ EnsembleManifest) error {
	// TODO
	return ErrTODO
}

func (o *Orchestrator) revertDeployment(_ EnsembleManifest) {
	// TODO
}

func (o *Orchestrator) acceptPeerLocation(_ string, _ Location) bool {
	// TODO
	return true
}

func (o *Orchestrator) makeInitialBidRequest() (EnsembleBidRequest, error) {
	// TODO
	return EnsembleBidRequest{}, ErrTODO
}

func (o *Orchestrator) makeResidualBidRequest(_ map[string][]Bid, _ map[string]struct{}) (EnsembleBidRequest, error) {
	// TODO
	return EnsembleBidRequest{}, ErrTODO
}

func (o *Orchestrator) supervise() {
	// TODO
}

func (o *Orchestrator) Stop() {
	// TODO
}
