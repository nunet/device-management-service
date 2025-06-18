// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

var ErrCandidateNotFound = errors.New("candidate not found")

// BidCoordinator handles the bidding process for ensemble deployment
type BidCoordinator struct {
	eid   string // ensembleID
	actor actor.Actor
	geo   geolocation.LocationProvider
	nonce uint64
}

// NewBidCoordinator creates a new BidCoordinator instance given a ensemble config copy
func NewBidCoordinator(
	eid string, actor actor.Actor,
) (*BidCoordinator, error) {
	geo, err := geolocation.NewGeoLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create geolocator: %w", err)
	}

	return &BidCoordinator{
		eid:   eid,
		actor: actor,
		geo:   geo,
	}, nil
}

// bid handles the bid process from beginning to end
//
// TODO: update deployment status when Generating
func (b *BidCoordinator) bid(cfgReader jtypes.EnsembleCfgReader, expiry time.Time) (map[string]jtypes.Bid, error) {
	cfg := cfgReader.Read() // read cfg copy

	candidate := make(map[string]jtypes.Bid)
	edgeConstraintCache := make(map[string]bool)

	// 0. check if one of the ensemble nodes have peer specified
	// If bid request to peer specified fails, the entire deployment must fail
	nodeForTargetPeer := make(map[string]string)
	for nodeID, node := range cfg.Nodes() {
		if node.Peer != "" {
			nodeForTargetPeer[node.Peer] = nodeID
		}
	}

	// 1. Create bid requests for nodes
	log.Debugw("creating initial bid request",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", b.eid,
		"nodes: ", cfg.Nodes())
	bidrq, err := b.makeInitialBidRequest(cfg)
	if err != nil {
		return candidate, fmt.Errorf("creating bid request: %w", err)
	}

	// 2. Collect bids
	log.Debugw("collecting bids",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", b.eid)

	bidMap := make(map[string][]jtypes.Bid)
	peerExclusion := make(map[string]struct{})
	addBid := func(bid jtypes.Bid) bool {
		// if peer is already specified on another node, ignore the bid
		if _, ok := nodeForTargetPeer[bid.Peer()]; ok {
			if nodeForTargetPeer[bid.Peer()] != bid.NodeID() {
				return false
			}
		}

		// check that the peer has not already submitted a bid
		peerID := bid.Peer()
		if _, exclude := peerExclusion[peerID]; exclude {
			log.Debugw("ignoring duplicate bid from peer",
				"labels", []string{string(observability.LabelDeployment)},
				"peerID", peerID)
			return false
		}

		err := bid.Validate()
		if err != nil {
			log.Debugw("failed to validate bid",
				"labels", []string{string(observability.LabelDeployment)},
				"peerID", peerID,
				"error", err)
			return false
		}

		// verify that this is a node in the ensemble
		nodeID := bid.NodeID()
		if _, ok := cfg.Node(nodeID); !ok {
			log.Debugw("ignoring bid for unknown node",
				"labels", []string{string(observability.LabelDeployment)},
				"peerID", peerID,
				"nodeID", nodeID)
			return false
		}

		// verify the location constraints of the node
		loc := bid.Location()
		if !acceptPeerLocation(cfg, nodeID, peerID, loc) {
			log.Debugw("ignoring out-of-location bid",
				"labels", []string{string(observability.LabelDeployment)},
				"peerID", peerID,
				"nodeID", nodeID,
				"location", loc,
			)
			return false
		}

		// don't bloat the permutation space
		if len(bidMap[nodeID]) >= MaxBidMultiplier {
			log.Debugw("node is saturated, ignoring new bid",
				"labels", []string{string(observability.LabelDeployment)},
				"peerID", peerID,
				"nodeID", nodeID)
			return false
		}

		log.Debugf("added bid to bitMap from peer %s for %s", peerID, nodeID)
		bidMap[nodeID] = append(bidMap[nodeID], bid)
		peerExclusion[peerID] = struct{}{}
		return true
	}

	// remove bid from bidMap and peerExclusion
	rmBid := func(bid jtypes.Bid) {
		peerID := bid.Peer()
		delete(peerExclusion, peerID)
		nodeID := bid.NodeID()
		bids := bidMap[nodeID]
		for i, b := range bids {
			if b.Peer() == peerID {
				bidMap[nodeID] = append(bids[:i], bids[i+1:]...)
				break
			}
		}
	}

	bidCh, bidDoneCh, bidExpiryTime, err := b.requestBids(cfg, bidrq, expiry)
	if err != nil {
		return candidate, fmt.Errorf("request bids: %w", err)
	}

	maxBids := MaxBidMultiplier * len(cfg.Nodes())
	b.collectBids(bidCh, bidDoneCh, bidExpiryTime, addBid, maxBids)

	// 3. Create a candidate deployment
	log.Debugw("creating candidate deployments",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", b.eid)
	var (
		nextCandidate func() (map[string]jtypes.Bid, bool)
		ok            bool
	)
	for time.Now().Before(expiry) {
		nextCandidate, ok = b.makeCandidateDeployments(cfg, bidMap)
		if ok {
			break
		}

		// we don't have bids for some of our nodes, so we don't have a candidate
		// we need to make a residual bid request for the remaining nodes
		// Note: in order to facilitate random selection, the residual bid requests
		//       can drop some of the original bids
		log.Debugw("not enough bids for all nodes, making residual request",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", b.eid)
		bidrq, err := b.makeResidualBidRequest(cfg, bidMap, rmBid)
		if err != nil {
			return candidate, fmt.Errorf("creating residual bid request: %w", err)
		}

		bidCh, bidDoneCh, bidExpiryTime, err := b.requestBids(cfg, bidrq, expiry)
		if err != nil {
			return candidate, fmt.Errorf("collecting residual bids: %w", err)
		}

		maxBids := MaxBidMultiplier * (len(cfg.Nodes()) - len(bidMap))
		b.collectBids(bidCh, bidDoneCh, bidExpiryTime, addBid, maxBids)
	}

	if !ok {
		log.Debugw("failed to create candidate deployments, retrying",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", b.eid)
		return candidate,
			fmt.Errorf("%w: failed to create candidate deployments - trying again",
				ErrCandidateNotFound)
	}

	for n, bids := range bidMap {
		log.Infof("node %s has %d bids", n, len(bids))
		for _, bid := range bids {
			log.Infof("    bid from %s", bid.Peer())
		}
	}

	// 4. Iterate through the candidates trying to find one that satisfies the
	//    edge constraints
	log.Debugf("generating candidate deployment")
	for time.Now().Before(expiry) {
		candidate, ok = nextCandidate()
		if !ok {
			return candidate,
				fmt.Errorf("%w: failed to find candidate that satisfies edge constraints",
					ErrCandidateNotFound)
		}

		log.Debugf("candidate deployment: %+v", candidate)
		if ok := b.verifyEdgeConstraints(cfg, candidate, edgeConstraintCache); !ok {
			log.Debugf("candidate does not satisfy edge constraints")
			continue
		}

		break
	}

	return candidate, nil
}

func (b *BidCoordinator) requestBids(
	cfg jtypes.EnsembleConfig,
	bidRequest jtypes.EnsembleBidRequest, expiry time.Time,
) (chan jtypes.Bid, chan struct{}, time.Time, error) {
	log.Debugf("requesting bids: %+v", bidRequest)

	bidExpiryTime := time.Now().Add(BidRequestTimeout)
	if expiry.Before(bidExpiryTime) {
		return nil, nil, time.Time{}, fmt.Errorf("not enough time for deployment: %w", ErrDeploymentFailed)
	}

	bidExpiry := uint64(bidExpiryTime.UnixNano())

	// Split requests into direct peer requests and broadcast requests
	var directRequests []jtypes.BidRequest
	var broadcastRequests []jtypes.BidRequest

	for _, req := range bidRequest.Request {
		if req.V1 == nil {
			continue
		}

		nodeConfig, ok := cfg.Node(req.V1.NodeID)
		if !ok {
			continue
		}

		if nodeConfig.Peer != "" {
			// This node has a specific peer target
			directRequests = append(directRequests, req)
		} else {
			// This node needs broadcast
			broadcastRequests = append(broadcastRequests, req)
		}
	}

	// Send direct peer requests
	log.Debugf("sending direct peer requests: %+v", directRequests)
	for _, req := range directRequests {
		nodeConfig, _ := cfg.Node(req.V1.NodeID)
		targetedReq := jtypes.EnsembleBidRequest{
			ID:            bidRequest.ID,
			Nonce:         bidRequest.Nonce,
			Request:       []jtypes.BidRequest{req},
			PeerExclusion: bidRequest.PeerExclusion,
		}
		err := b.requestBidPeer(targetedReq, nodeConfig, bidExpiry)
		if err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("requesting bid to targeted peer: %w", err)
		}
	}

	// create reply behavior for this specific ensemble bid request
	bidCh := make(chan jtypes.Bid)
	bidDoneCh := make(chan struct{})
	if err := b.actor.AddBehavior(
		behaviors.BidReplyBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var bid jtypes.Bid
			if err := json.Unmarshal(msg.Message, &bid); err != nil {
				log.Debugw("failed to unmarshal bid",
					"labels", []string{string(observability.LabelDeployment)},
					"from", msg.From,
					"error", err)
				return
			}

			timer := time.NewTimer(time.Until(bidExpiryTime))
			defer timer.Stop()

			select {
			case bidCh <- bid:
			case <-timer.C:
			case <-bidDoneCh:
			}
		},
		actor.WithBehaviorExpiry(bidExpiry),
	); err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("adding bid behavior: %w", err)
	}
	// Send broadcast
	log.Debugf("sending broadcast requests: %+v", broadcastRequests)
	if len(broadcastRequests) > 0 {
		broadcastReq := jtypes.EnsembleBidRequest{
			ID:            bidRequest.ID,
			Nonce:         bidRequest.Nonce,
			Request:       broadcastRequests,
			PeerExclusion: bidRequest.PeerExclusion,
		}
		err := b.broadcastBid(broadcastReq, bidExpiry)
		if err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("broadcasting bid request: %w", err)
		}
	}

	return bidCh, bidDoneCh, bidExpiryTime, nil
}

func (b *BidCoordinator) broadcastBid(bidRequest jtypes.EnsembleBidRequest, bidExpiry uint64) error {
	msg, err := actor.Message(
		b.actor.Handle(),
		actor.Handle{},
		behaviors.BidRequestBehavior,
		bidRequest,
		actor.WithMessageTopic(behaviors.BidRequestTopic),
		actor.WithMessageReplyTo(behaviors.BidReplyBehavior),
		actor.WithMessageExpiry(bidExpiry),
	)
	if err != nil {
		return fmt.Errorf("creating broadcast bid message: %w", err)
	}

	if err := b.actor.Publish(msg); err != nil {
		return fmt.Errorf("publishing broadcast bid request: %w", err)
	}

	return nil
}

func (b *BidCoordinator) requestBidPeer(
	targetedReq jtypes.EnsembleBidRequest, nodeConfig jtypes.NodeConfig, bidExpiry uint64,
) error {
	destHandle, err := actor.HandleFromPeerID(nodeConfig.Peer)
	if err != nil {
		return fmt.Errorf("getting handle of selected peer %s: %w", nodeConfig.Peer, err)
	}

	log.Infof("sending direct peer request to %s: %+v", nodeConfig.Peer, targetedReq)
	msg, err := actor.Message(
		b.actor.Handle(),
		destHandle,
		behaviors.BidRequestBehavior,
		targetedReq,
		actor.WithMessageReplyTo(behaviors.BidReplyBehavior),
		actor.WithMessageExpiry(bidExpiry),
	)
	if err != nil {
		return fmt.Errorf("creating targeted bid message: %w", err)
	}

	log.Infow("requesting bid from targeted peer",
		"labels", []string{string(observability.LabelDeployment)},
		"peerID", nodeConfig.Peer,
		"orchestratorID", b.eid)
	if err := b.actor.Send(msg); err != nil {
		return fmt.Errorf("sending targeted bid request: %w", err)
	}

	return nil
}

func (b *BidCoordinator) collectBids(
	bidCh chan jtypes.Bid, bidDoneCh chan struct{}, bidExpiryTime time.Time,
	addBid func(jtypes.Bid) bool, maxBids int,
) {
	defer close(bidDoneCh)

	log.Debugf("collecting bids until: %v", bidExpiryTime)
	timer := time.NewTimer(time.Until(bidExpiryTime))
	defer timer.Stop()

	bidCount := 0
	for {
		select {
		case bid, ok := <-bidCh:
			if !ok {
				log.Debugw("bid channel closed",
					"labels", []string{string(observability.LabelDeployment)})
				return
			}
			log.Debugw("received bid",
				"labels", []string{string(observability.LabelDeployment)},
				"ensembleID", bid.EnsembleID(),
				"peerID", bid.Peer(),
				"nodeID", bid.NodeID())
			if err := bid.Validate(); err != nil {
				log.Debugw("invalid bid",
					"labels", []string{string(observability.LabelDeployment)},
					"error", err)
				continue
			}
			if bid.EnsembleID() != b.eid {
				log.Debugw("bid for unexpected ensemble id",
					"labels", []string{string(observability.LabelDeployment)},
					"expectedID", b.eid,
					"gotID", bid.EnsembleID())
				continue
			}
			if addBid(bid) {
				bidCount++
				if bidCount >= maxBids {
					return
				}
			}
		case <-timer.C:
			return
		}
	}
}

func (b *BidCoordinator) makeCandidateDeployments(
	cfg jtypes.EnsembleConfig, bids map[string][]jtypes.Bid,
) (func() (map[string]jtypes.Bid, bool), bool) {
	// immediate satisfaction check: we need a bid for every node
	if len(cfg.Nodes()) != len(bids) {
		return nil, false
	}

	// first shuffle all the bids to seed the permutation generator
	for _, blst := range bids {
		rand.Shuffle(len(blst), func(i, j int) {
			blst[i], blst[j] = blst[j], blst[i]
		})
	}

	// count the bits in the permutation space; if it is more than 63, we need to use
	// a bignum based permutation generator, or it will overflow.
	bits := 0
	for _, blst := range bids {
		bits += int(math.Ceil(math.Log2(float64(len(blst)))))
	}

	if bits > 63 {
		return b.makeCandidateDeploymentBig(cfg, bids)
	}

	return b.makeCandidateDeploymentSmall(cfg, bids)
}

func (b *BidCoordinator) makeCandidateDeploymentSmall(
	cfg jtypes.EnsembleConfig, bids map[string][]jtypes.Bid,
) (func() (map[string]jtypes.Bid, bool), bool) {
	// fix the order of permutation
	type permutator struct {
		mod  int64
		node string
		bids []jtypes.Bid
	}
	permutators := make([]permutator, 0, len(bids))
	modulus := int64(1)
	for n, blst := range bids {
		permutators = append(permutators, permutator{mod: modulus, node: n, bids: blst})
		modulus *= int64(len(blst))
	}

	// function to get a permutation by index
	getPermutation := func(index int64) map[string]jtypes.Bid {
		result := make(map[string]jtypes.Bid)
		for _, permutator := range permutators {
			selection := (index / permutator.mod) % int64(len(permutator.bids))
			result[permutator.node] = permutator.bids[selection]
		}

		return result
	}

	// and return a function that gets a random next permutation
	// note that we cache the constraint results, so potential duplication is ok.
	// also note that the permutation space is large enough so that it's ok to skip
	// some permutations.
	// final note: Obviously we can deterministically generate all permutations in order
	// (and we were doing that initially) but this has the problem that we are not
	// modifying the network structure enough to get meaningful variance in a reasonable
	// time.
	nperm := modulus
	if nperm > int64(MaxPermutations) {
		nperm = int64(MaxPermutations)
	}
	count := int64(0)
	return func() (map[string]jtypes.Bid, bool) {
		for count < nperm {
			count++

			nextPerm := rand.Int63n(nperm)
			perm := getPermutation(nextPerm)

			if !b.checkPermutationEdgeConstraints(cfg, perm) {
				continue
			}

			return perm, true
		}

		return nil, false
	}, true
}

func (b *BidCoordinator) makeCandidateDeploymentBig(
	cfg jtypes.EnsembleConfig, bids map[string][]jtypes.Bid,
) (func() (map[string]jtypes.Bid, bool), bool) {
	// Note: this is the same as above with bignums

	// fix the order of permutation
	type permutator struct {
		mod  *big.Int
		node string
		bids []jtypes.Bid
	}
	permutators := make([]permutator, 0, len(bids))
	modulus := big.NewInt(1)
	for n, blst := range bids {
		permutators = append(permutators, permutator{mod: modulus, node: n, bids: blst})
		modulus = new(big.Int).Mul(modulus, big.NewInt(int64(len(blst))))
	}

	// function to get a permutation by index
	getPermutation := func(index *big.Int) map[string]jtypes.Bid {
		result := make(map[string]jtypes.Bid)
		for _, permutator := range permutators {
			selection := int(
				new(big.Int).Mod(
					new(big.Int).Div(index, permutator.mod),
					big.NewInt(int64(len(permutator.bids))),
				).Int64(),
			)
			result[permutator.node] = permutator.bids[selection]
		}

		return result
	}

	// and return a function that gets a random next permutation
	// note that we cache the constraint results, so potential duplication is ok.
	// also note that the permutation space is large enough so that it's ok to skip
	// some permutations.
	// final note: Obviously we can deterministically generate all permutations in order
	// (and we were doing that initially) but this has the problem that we are not
	// modifying the network structure enough to get meaningful variance in a reasonable
	// time.
	nperm := MaxPermutations
	count := 0
	bytes := make([]byte, (modulus.BitLen()+7)/8)
	return func() (map[string]jtypes.Bid, bool) {
		for count < nperm {
			count++

			if _, err := crand.Read(bytes); err != nil {
				log.Errorw("random_bytes_read_error",
					"labels", []string{string(observability.LabelDeployment)},
					"error", err)
				return nil, false
			}

			nextPerm := new(big.Int).SetBytes(bytes)
			perm := getPermutation(nextPerm)

			if !b.checkPermutationEdgeConstraints(cfg, perm) {
				continue
			}

			return perm, true
		}

		return nil, false
	}, true
}

func (b *BidCoordinator) checkPermutationEdgeConstraints(
	cfg jtypes.EnsembleConfig, candidate map[string]jtypes.Bid,
) bool {
	for _, cst := range cfg.EdgeConstraints() {
		if cst.RTT == 0 {
			continue
		}

		bidS, ok := candidate[cst.S]
		if !ok {
			log.Errorf("Bid %s not found in candidate", cst.S)
			return false
		}
		bidT, ok := candidate[cst.T]
		if !ok {
			log.Errorf("Bid %s not found in candidate", cst.T)
			return false
		}

		locS, err := b.geo.Coordinate(bidS.Location())
		if err != nil {
			log.Errorf("Failed to get location for bid %s: %v", bidS.NodeID(), err)
			continue
		}

		locT, err := b.geo.Coordinate(bidT.Location())
		if err != nil {
			log.Errorf("Failed to get location for bid %s: %v", bidT.NodeID(), err)
			continue
		}

		distance := geolocation.ComputeGeodesic(locS, locT)

		// in milliseconds
		minRTT := (distance / geolocation.LightSpeed) * 2 * 1000
		if minRTT > float64(cst.RTT) {
			log.Debugw("edge constraint not satisfied",
				"labels", []string{string(observability.LabelDeployment)},
				"minRTT", minRTT,
				"constraint", cst.RTT,
				"from", cst.S,
				"to", cst.T)
			return false
		}

		// TODO: add bandwidth check when that information becomes available
	}

	return true
}

func (b *BidCoordinator) verifyEdgeConstraints(
	cfg jtypes.EnsembleConfig, candidate map[string]jtypes.Bid, cache map[string]bool,
) bool {
	var mx sync.Mutex
	var wg sync.WaitGroup
	var toVerify []jtypes.EdgeConstraint

	for _, cst := range cfg.EdgeConstraints() {
		bidS := candidate[cst.S]
		bidT := candidate[cst.T]
		key := bidS.Peer() + ":" + bidT.Peer()
		accept, ok := cache[key]
		if !ok {
			toVerify = append(toVerify, cst)
			continue
		}
		if !accept {
			return false
		}
	}

	if len(toVerify) == 0 {
		return true
	}

	accept := true
	wg.Add(len(toVerify))
	for _, cst := range toVerify {
		go func(cst jtypes.EdgeConstraint) {
			defer wg.Done()
			result := b.verifyEdgeConstraint(candidate, cst)
			bidS := candidate[cst.S]
			bidT := candidate[cst.T]
			key := bidS.Peer() + ":" + bidT.Peer()
			mx.Lock()
			cache[key] = result
			accept = accept && result
			mx.Unlock()
		}(cst)
	}

	wg.Wait()
	return accept
}

type VerifyEdgeConstraintRequest struct {
	EnsembleID string // the ensemble identifier
	S, T       string // the peer IDs of the edge S->T
	RTT        uint   //  maximum RTT in ms (if > 0)
	BW         uint   // minim BW in Kbps
}

type VerifyEdgeConstraintResponse struct {
	OK    bool
	Error string
}

func (b *BidCoordinator) verifyEdgeConstraint(candidate map[string]jtypes.Bid, cst jtypes.EdgeConstraint) bool {
	bidS := candidate[cst.S]
	bidT := candidate[cst.T]
	key := bidS.Peer() + ":" + bidT.Peer()
	log.Debugw("verifying edge constraint",
		"labels", []string{string(observability.LabelDeployment)},
		"peerS", bidS.Peer(),
		"peerT", bidT.Peer(),
		"constraint", cst)

	handle := bidS.Handle()
	msg, err := actor.Message(
		b.actor.Handle(),
		handle,
		behaviors.VerifyEdgeConstraintBehavior,
		VerifyEdgeConstraintRequest{
			EnsembleID: b.eid,
			S:          bidS.Peer(),
			T:          bidT.Peer(),
			RTT:        cst.RTT,
			BW:         cst.BW,
		},
		actor.WithMessageTimeout(VerifyEdgeConstraintTimeout),
	)
	if err != nil {
		log.Warnw("creating constraint check message error",
			"labels", []string{string(observability.LabelDeployment)},
			"edgeKey", key,
			"error", err)
		return false
	}

	replyCh, err := b.actor.Invoke(msg)
	if err != nil {
		log.Warnw("invoke constraint check error",
			"labels", []string{string(observability.LabelDeployment)},
			"edgeKey", key,
			"error", err)
		return false
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(VerifyEdgeConstraintTimeout):
		return false
	}
	defer reply.Discard()

	var response VerifyEdgeConstraintResponse
	if err := json.Unmarshal(reply.Message, &response); err != nil {
		log.Warnw("unmarshal bid constraint response error",
			"labels", []string{string(observability.LabelDeployment)},
			"edgeKey", key,
			"error", err)
		return false
	}

	if response.Error != "" {
		log.Debugw("verify bid constraint not satisfied",
			"labels", []string{string(observability.LabelDeployment)},
			"edgeKey", key,
			"error", response.Error)
	}

	return response.OK
}

func acceptPeerLocation(
	cfg jtypes.EnsembleConfig, nodeID, peerID string, loc jtypes.Location,
) bool {
	n, ok := cfg.Node(nodeID)
	if !ok {
		return false
	}

	// check explicit peer placement
	if n.Peer != "" {
		return n.Peer == peerID
	}

	// check acceptable locations
	// if acceptable locations are specified, then reject locations are ignored
	// (since the user probably wants only specified locations)
	if len(n.Location.Accept) > 0 {
		accept := false
		for _, acceptable := range n.Location.Accept {
			if acceptable.Equal(loc) {
				accept = true
				break
			}
		}

		return accept
	}

	// check unacceptable locations
	if len(n.Location.Reject) > 0 {
		reject := false
		for _, unacceptable := range n.Location.Reject {
			if unacceptable.Equal(loc) {
				reject = true
				break
			}
		}
		if reject {
			return false
		}
	}

	return true
}

func (b *BidCoordinator) makeInitialBidRequest(cfg jtypes.EnsembleConfig) (jtypes.EnsembleBidRequest, error) {
	return b.ensembleConfigToBidRequest(&cfg)
}

func (b *BidCoordinator) makeResidualBidRequest(
	cfg jtypes.EnsembleConfig,
	candidate map[string][]jtypes.Bid, rmbid func(jtypes.Bid),
) (jtypes.EnsembleBidRequest, error) {
	residualConfig := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Allocations: make(map[string]jtypes.AllocationConfig),
			Nodes:       make(map[string]jtypes.NodeConfig),
		},
	}

	// randomly drop some of the candidate bids and exclusion
	newCandidates := make(map[string][]jtypes.Bid)
	newExclusion := make(map[string]struct{})

	// drop half of the bids and delete from candidate and exclusion
	for n, bids := range candidate {
		newBids := make([]jtypes.Bid, 0, len(bids))
		desiredSize := int(math.Floor(float64(rand.Intn(len(bids))) / 2))
		for i, bid := range bids {
			if i > desiredSize {
				log.Infof("dropping bid from %s (%s) from candidate ", bid.Peer(), bid.V1.Handle.DID)
				rmbid(bid)
				continue
			}
			log.Infof("keeping bid from %s (%s) for candidate", bid.Peer(), bid.V1.Handle.DID)
			newBids = append(newBids, bid)
			newExclusion[bid.Peer()] = struct{}{}
		}
		if len(newBids) > 0 {
			newCandidates[n] = newBids
		}
	}

	for n, ncfg := range cfg.V1.Nodes {
		if _, exclude := newCandidates[n]; exclude {
			log.Debugw(
				fmt.Sprintf("node %s is in candidate, skipping", n),
				"labels", []string{string(observability.LabelDeployment)},
				"nodeID", n,
			)
			continue
		}

		residualConfig.V1.Nodes[n] = ncfg
	}

	for id, ncfg := range residualConfig.V1.Nodes {
		log.Debugw(
			fmt.Sprintf("still looking for node %s", id),
			"labels", []string{string(observability.LabelDeployment)},
			"nodeID", id,
		)
		for _, a := range ncfg.Allocations {
			residualConfig.V1.Allocations[a] = cfg.V1.Allocations[a]
		}
	}

	result, err := b.ensembleConfigToBidRequest(&residualConfig)
	if err != nil {
		return result, err
	}

	for p := range newExclusion {
		result.PeerExclusion = append(result.PeerExclusion, p)
	}

	return result, nil
}

func (b *BidCoordinator) ensembleConfigToBidRequest(config *jtypes.EnsembleConfig) (jtypes.EnsembleBidRequest, error) {
	v1Config := config.V1

	ensembleBidRequest := jtypes.EnsembleBidRequest{
		ID:    b.eid,
		Nonce: b.getNonce(),
	}

	log.Infow("generating bid request",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", b.eid,
		"nodes", v1Config.Nodes)
	for nodeID, nodeConfig := range v1Config.Nodes {
		bidRequest := jtypes.BidRequest{
			V1: &jtypes.BidRequestV1{
				NodeID:   nodeID,
				Location: nodeConfig.Location,
			},
		}

		var aggregateResources types.Resources
		var executors []jtypes.AllocationExecutor

		var staticPorts []int
		dynamicPortsCount := 0

		for _, allocationName := range nodeConfig.Allocations {
			allocationConfig, ok := v1Config.Allocations[allocationName]
			if !ok {
				continue
			}

			if !containsExecutor(executors, allocationConfig.Executor) {
				executors = append(executors, allocationConfig.Executor)
			}

			if allocationConfig.Executor == jtypes.ExecutorDocker {
				// check if bid includes allocation requiring privileged docker
				dockerCfg, err := docker.DecodeSpec(&allocationConfig.Execution)
				if err != nil {
					return jtypes.EnsembleBidRequest{}, fmt.Errorf("decoding docker spec: %w", err)
				}

				if dockerCfg.Privileged {
					bidRequest.V1.GeneralRequirements.PrivilegedDocker = true
				}
			}

			err := aggregateResources.Add(allocationConfig.Resources)
			if err != nil {
				return jtypes.EnsembleBidRequest{}, err
			}

			for _, portConfig := range nodeConfig.Ports {
				if portConfig.Allocation == allocationName {
					if portConfig.Public == 0 {
						dynamicPortsCount++
					} else {
						staticPorts = append(staticPorts, portConfig.Public)
					}
				}
			}
		}

		bidRequest.V1.Executors = executors
		bidRequest.V1.Resources = aggregateResources

		bidRequest.V1.PublicPorts.Static = staticPorts
		bidRequest.V1.PublicPorts.Dynamic = dynamicPortsCount

		ensembleBidRequest.Request = append(ensembleBidRequest.Request, bidRequest)
	}

	return ensembleBidRequest, nil
}

func (b *BidCoordinator) getNonce() uint64 {
	atomic.AddUint64(&b.nonce, 1)
	return b.nonce
}

func containsExecutor(executors []jtypes.AllocationExecutor, executor jtypes.AllocationExecutor) bool {
	for _, e := range executors {
		if e == executor {
			return true
		}
	}
	return false
}
