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
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/dms/behaviors"

	"github.com/libp2p/go-libp2p/core/crypto"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/docker"
	netutils "gitlab.com/nunet/device-management-service/network/utils"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	BidRequestTimeout           = 5 * time.Second
	VerifyEdgeConstraintTimeout = 5 * time.Second
	CommitDeploymentTimeout     = 3 * time.Second
	AllocationDeploymentTimeout = 5 * time.Second

	AllocationStartTimeout    = 5 * time.Second
	AllocationShutdownTimeout = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second

	SubnetCreateTimeout  = 2 * time.Minute
	SubnetDestroyTimeout = 30 * time.Second

	MaxBidMultiplier = 8
	MaxPermutations  = 1_000_000
)

var (
	ErrProvisioningFailed   = errors.New("failed to provision the ensemble")
	ErrDeploymentFailed     = errors.New("failed to create deployment")
	ErrOrchestratorExists   = errors.New("orchestrator with ID already exists")
	ErrOrchestratorNotFound = errors.New("orchestrator with ID not found")
)

// OrchestratorProvider is an interface which acts as a source for orchestrators
type OrchestratorProvider interface {
	// NewOrchestrator creates a new orchestrator
	NewOrchestrator(ctx context.Context, id string, actor actor.Actor, cfg EnsembleConfig) (OrchestratorAPI, error)
	// RestoreDeployment restores deployments where the status is either provisioning, committing or running
	RestoreDeployment(
		actr actor.Actor, id string, cfg EnsembleConfig, manifest EnsembleManifest, status DeploymentStatus, restoreInfo DeploymentSnapshot,
	) (OrchestratorAPI, error)
	// Orchestrators returns a map of all orchestrators
	Orchestrators() map[string]OrchestratorAPI
	// GetOrchestrator returns an orchestrator by ID
	GetOrchestrator(id string) (OrchestratorAPI, error)
	// DeleteOrchestrator deletes an orchestrator by ID
	DeleteOrchestrator(id string)
}

// orchestratorProvider the default implementation of OrchestratorProvider
type orchestratorProvider struct {
	lock          sync.RWMutex
	orchestrators map[string]OrchestratorAPI // map of orchestrators
}

var _ OrchestratorProvider = (*orchestratorProvider)(nil)

// NewOrchestratorProvider creates a new orchestrator provider
func NewOrchestratorProvider() OrchestratorProvider {
	return &orchestratorProvider{
		orchestrators: make(map[string]OrchestratorAPI),
	}
}

// NewOrchestrator creates a new orchestrator
func (f *orchestratorProvider) NewOrchestrator(
	ctx context.Context, id string, actor actor.Actor, cfg EnsembleConfig,
) (OrchestratorAPI, error) {
	// check if orchestrator already exists
	f.lock.RLock()
	if _, ok := f.orchestrators[id]; ok {
		f.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	f.lock.RUnlock()

	o, err := NewOrchestrator(ctx, id, actor, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	f.lock.Lock()
	defer f.lock.Unlock()
	f.orchestrators[id] = o

	return o, nil
}

// restoreDeployment restores deployments where the status is either provisioning, committing or running
func restoreDeployment(
	actr actor.Actor, id string,
	cfg EnsembleConfig, manifest EnsembleManifest,
	status DeploymentStatus, restoreInfo DeploymentSnapshot,
) (*Orchestrator, error) {
	o := &Orchestrator{
		id:                 id,
		actor:              actr,
		cfg:                cfg,
		manifest:           manifest,
		status:             status,
		deploymentSnapshot: restoreInfo,
	}

	if o.status == DeploymentStatusCommitting {
		log.Debug("reverting deployment of old candidates and restarting deployment from the beginning")
		for nodeID, bid := range restoreInfo.Candidates {
			o.revertDeployment(nodeID, bid.Handle())
		}

		return o, o.deploy(restoreInfo.Expiry)
	}

	if o.status == DeploymentStatusProvisioning {
		log.Debug("restoring deployment from manifest")
		if err := o.provision(o.manifest); err != nil {
			log.Errorf("failed to provision network: %s", err)
			o.revert(manifest)
			return o, o.deploy(restoreInfo.Expiry)
		}

		o.setStatus(DeploymentStatusRunning)
	}

	o.ctx, o.cancel = context.WithCancel(context.Background())

	allocations := make(map[string]actor.Handle, len(manifest.Allocations))
	for _, allocation := range manifest.Allocations {
		allocations[allocation.ID] = allocation.Handle
	}
	o.manifest = manifest
	go o.supervise()

	return o, nil
}

// RestoreDeployment creates an orchestrator and attempts to restore its deployment
func (f *orchestratorProvider) RestoreDeployment(
	actr actor.Actor, id string, cfg EnsembleConfig, manifest EnsembleManifest, status DeploymentStatus, restoreInfo DeploymentSnapshot,
) (OrchestratorAPI, error) {
	// check if orchestrator already exists
	f.lock.RLock()
	if _, ok := f.orchestrators[id]; ok {
		f.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	f.lock.RUnlock()

	o, err := restoreDeployment(actr, id, cfg, manifest, status, restoreInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to restore deployment: %w", err)
	}

	f.lock.Lock()
	defer f.lock.Unlock()
	f.orchestrators[id] = o

	return o, nil
}

// Orchestrators returns a map of all orchestrators
func (f *orchestratorProvider) Orchestrators() map[string]OrchestratorAPI {
	f.lock.RLock()
	defer f.lock.RUnlock()

	orchestrators := make(map[string]OrchestratorAPI, len(f.orchestrators))
	for id, o := range f.orchestrators {
		orchestrators[id] = o
	}

	return orchestrators
}

// GetOrchestrator returns an orchestrator by ID
func (f *orchestratorProvider) GetOrchestrator(id string) (OrchestratorAPI, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	o, ok := f.orchestrators[id]
	if !ok {
		return nil, ErrOrchestratorNotFound
	}

	return o, nil
}

// DeleteOrchestrator deletes an orchestrator by ID
func (f *orchestratorProvider) DeleteOrchestrator(id string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	delete(f.orchestrators, id)
}

// OrchestratorAPI is the interface for orchestrators
type OrchestratorAPI interface {
	Deploy(expiry time.Time) error
	Shutdown()
	Stop()
	Status() DeploymentStatus
	Manifest() EnsembleManifest
	Config() EnsembleConfig
	ID() string
	ActorPrivateKey() crypto.PrivKey
	DeploymentSnapshot() DeploymentSnapshot
	GetAllocationLogs(name string) (AllocationLogsResponse, error)
}

type Orchestrator struct {
	lock   sync.Mutex
	ctx    context.Context
	cancel func()

	actor actor.Actor
	geo   *GeoLocator

	id       string
	cfg      EnsembleConfig
	manifest EnsembleManifest
	status   DeploymentStatus

	deploymentSnapshot DeploymentSnapshot
}

var _ OrchestratorAPI = (*Orchestrator)(nil)

func NewOrchestrator(
	ctx context.Context,
	id string,
	actor actor.Actor,
	cfg EnsembleConfig,
) (*Orchestrator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate ensemble configuration: %w", err)
	}

	geo, err := NewGeoLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create geolocator: %w", err)
	}

	o := &Orchestrator{
		actor: actor,
		geo:   geo,
		id:    id,
		cfg:   cfg,
		ctx:   ctx,
	}

	return o, nil
}

func (o *Orchestrator) setStatus(status DeploymentStatus) {
	o.lock.Lock()
	defer o.lock.Unlock()

	log.Infof("setting status to: %s", jobtypes.DeploymentStatusString(status))
	o.status = status
}

func (o *Orchestrator) Deploy(expiry time.Time) error {
	defer func() {
		if o.status != DeploymentStatusRunning {
			o.setStatus(DeploymentStatusFailed)
		}
	}()
	o.setStatus(DeploymentStatusPreparing)

	return o.deploy(expiry)
}

func (o *Orchestrator) deploy(expiry time.Time) error {
	o.deploymentSnapshot.Expiry = expiry
	edgeConstraintCache := make(map[string]bool)

deploy:
	for time.Now().Before(expiry) {
		o.setStatus(DeploymentStatusPreparing)

		// delete old state of candidates if any
		for c := range o.deploymentSnapshot.Candidates {
			o.lock.Lock()
			delete(o.deploymentSnapshot.Candidates, c)
			o.lock.Unlock()
		}
		// 0. check if one of the ensemble nodes have peer specified
		// If bid request to peer specified fails, the entire deployment must fail
		nodeForTargetPeer := make(map[string]string)
		for nodeID, node := range o.cfg.Nodes() {
			if node.Peer != "" {
				nodeForTargetPeer[node.Peer] = nodeID
			}
		}

		// 1. Create bid requests for nodes
		log.Debugf("creating bid requests for nodes: %+v", o.cfg.Nodes())
		bidrq, err := o.makeInitialBidRequest()
		if err != nil {
			return fmt.Errorf("creating bid request: %w", err)
		}

		// 2. Collect bids
		bidMap := make(map[string][]Bid)
		peerExclusion := make(map[string]struct{})
		addBid := func(bid Bid) bool {
			// if peer is already specified on another node, ignore the bid
			if _, ok := nodeForTargetPeer[bid.Peer()]; ok {
				if nodeForTargetPeer[bid.Peer()] != bid.NodeID() {
					return false
				}
			}

			// check that the peer has not already submitted a bid
			peerID := bid.Peer()
			if _, exclude := peerExclusion[peerID]; exclude {
				log.Debugf("ignoring duplicate bid from peer %s", peerID)
				return false
			}

			err := bid.Validate()
			if err != nil {
				log.Debugf("failed to validate bid from peer %s: %s", peerID, err)
				return false
			}

			// verify that this is a node in the ensemble
			nodeID := bid.NodeID()
			if _, ok := o.cfg.Node(nodeID); !ok {
				log.Debugf("ignoring bid from peer %s for unknown node %s", peerID, nodeID)
				return false
			}

			// verify the location constraints of the node
			loc := bid.Location()
			if !o.acceptPeerLocation(nodeID, peerID, loc) {
				log.Debugf("ignoring out of location bid from peer %s for node %s", peerID, nodeID)
				return false
			}

			// don't bloat the permutation space
			if len(bidMap[nodeID]) >= MaxBidMultiplier {
				log.Debugf("ignore bid from peer %s for saturated node %s", peerID, nodeID)
				return false
			}

			log.Debugf("added bid to bitMap from peer %s for %s", peerID, nodeID)
			bidMap[nodeID] = append(bidMap[nodeID], bid)
			peerExclusion[peerID] = struct{}{}
			return true
		}

		log.Debugf("collecting bids")
		bidCh, bidDoneCh, bidExpiryTime, err := o.requestBids(bidrq, expiry)
		if err != nil {
			return fmt.Errorf("request bids: %w", err)
		}

		maxBids := MaxBidMultiplier * len(o.cfg.Nodes())
		o.collectBids(bidCh, bidDoneCh, bidExpiryTime, addBid, maxBids)

		// 3. Create a candidate deployment
		log.Debugf("creating candidate deployments")
		var (
			nextCandidate func() (map[string]Bid, bool)
			ok            bool
		)
		for time.Now().Before(expiry) {
			nextCandidate, ok = o.makeCandidateDeployments(bidMap)
			if ok {
				break
			}

			// we don't have bids for some of our nodes, so we don't have a candidate
			// we need to make a residual bid request for the remaining nodes
			// Note: in order to facilitate random selection, the residual bid requests
			//       can drop some of the original bids
			log.Debugf("creating residual bid request")
			bidrq, err := o.makeResidualBidRequest(bidMap, peerExclusion)
			if err != nil {
				return fmt.Errorf("creating residual bid request: %w", err)
			}

			bidCh, bidDoneCh, bidExpiryTime, err := o.requestBids(bidrq, expiry)
			if err != nil {
				return fmt.Errorf("collecting residual bids: %w", err)
			}

			maxBids := MaxBidMultiplier * (len(o.cfg.Nodes()) - len(bidMap))
			o.collectBids(bidCh, bidDoneCh, bidExpiryTime, addBid, maxBids)
		}

		if !ok {
			log.Debugf("failed to create candidate deployments - trying again")
			continue deploy
		}

		// 4. Iterate through the candidates trying to find one that satisfies the
		//    edge constraints
		o.setStatus(DeploymentStatusGenerating)

		log.Debugf("generating candidate deployment")
		var candidate map[string]Bid
		for time.Now().Before(expiry) {
			candidate, ok = nextCandidate()
			if !ok {
				log.Debugf("failed to find candidate that satisfies edge constraints")
				continue deploy
			}

			log.Debugf("candidate deployment: %+v", candidate)
			if ok := o.verifyEdgeConstraints(candidate, edgeConstraintCache); !ok {
				log.Debugf("candidate does not satisfy edge constraints")
				continue
			}

			break
		}

		// 5. Commit the deployment
		o.setStatus(DeploymentStatusCommitting)
		o.deploymentSnapshot.Candidates = candidate

		log.Info("committing candidate bids")
		manifest, err := o.commit(candidate)
		if err != nil {
			log.Warnf("failed to commit deployment: %s", err)
			continue deploy
		}

		o.lock.Lock()
		o.manifest = manifest
		o.lock.Unlock()

		// 6. provision the network and start the allocations
		o.setStatus(DeploymentStatusProvisioning)

		log.Info("provisioning network")
		if err := o.provision(o.manifest); err != nil {
			log.Errorf("failed to provision network: %s", err)
			o.revert(manifest)
			continue deploy
		}

		// We are done! start the supervisor return the manifest.
		o.lock.Lock()
		o.manifest = manifest
		o.ctx, o.cancel = context.WithCancel(context.Background())
		o.lock.Unlock()

		log.Infof("deployment successful, starting supervisor")
		o.setStatus(DeploymentStatusRunning)
		allocations := make(map[string]actor.Handle, len(manifest.Allocations))
		for _, allocation := range manifest.Allocations {
			allocations[allocation.ID] = allocation.Handle
		}
		go o.supervise()

		return nil
	}

	// we failed to create the deployment in time
	log.Errorf("failed to create the deployment in time")
	return ErrDeploymentFailed
}

type AllocationStopRequest struct {
	AllocationID string
}

type AllocationStopResponse struct {
	OK    bool
	Error string
}

type SubnetDestroyRequest struct {
	SubnetID string
}

type SubnetDestroyResponse struct {
	OK    bool
	Error string
}

func (o *Orchestrator) Shutdown() {
	nodes := o.manifest.Nodes
	o.setStatus(DeploymentStatusShuttingDown)
	o.lock.Lock()

	defer func() {
		o.lock.Unlock()
		o.setStatus(DeploymentStatusCompleted)
		if o.cancel != nil {
			o.cancel()
		}
	}()

	wg := sync.WaitGroup{}
	for _, node := range nodes {
		wg.Add(1)

		go func(h actor.Handle, id string) {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, o.manifest.ID),
				SubnetDestroyRequest{
					SubnetID: o.manifest.ID,
				},
				actor.WithMessageExpiry(actor.MakeExpiry(5*time.Second)),
			)
			if err != nil {
				log.Errorf("error creating stop message for %s/%s: %s", o.manifest.ID, id, err)
				return
			}

			// invoke the subnet destroy message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s/%s: %s", o.manifest.ID, id, err)
				return
			}

			var reply actor.Envelope
			// wait for the reply
			select {
			case reply = <-replyCh:
				defer reply.Discard()
				var resp SubnetDestroyResponse
				if err := json.Unmarshal(reply.Message, &resp); err != nil {
					log.Errorf("error unmarshalling subnet destroy response: %v", err)
					return
				}
				if !resp.OK {
					log.Errorf("failed to destroy subnet %s/%s: %v", o.manifest.ID, id, resp.Error)
					return
				}

			case <-time.After(SubnetDestroyTimeout):
				log.Errorf("timeout destroying subnet %s", o.manifest.ID)
				return
			}

			log.Infof("subnet %s destroyed", o.manifest.ID)
		}(node.Handle, node.ID)
	}

	wg.Wait()

	wg = sync.WaitGroup{}
	for _, alloc := range o.manifest.Allocations {
		wg.Add(1)
		go func(h actor.Handle, allocID string) {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				fmt.Sprintf(behaviors.AllocationShutdownBehavior, o.manifest.ID),
				AllocationStopRequest{
					AllocationID: allocID,
				},
				actor.WithMessageExpiry(actor.MakeExpiry(AllocationShutdownTimeout)),
			)
			if err != nil {
				log.Errorf("error creating stop message for alloc: %s: %v", allocID, err)
				return
			}

			// invoke the stop message
			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("error invoking stop message for %s: %v", allocID, err)
				return
			}

			// wait for the reply
			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()
				var resp AllocationStopResponse
				if err := json.Unmarshal(reply.Message, &resp); err != nil {
					log.Errorf("error unmarshalling stop allocation response: %s", err)
					return
				}
				if !resp.OK {
					log.Errorf("failed to stop allocation %s", allocID)
					return
				}
			case <-time.After(AllocationShutdownTimeout):
				log.Errorf("timeout stopping allocation %s", allocID)
				return
			}
			log.Infof("allocation %s stopped", allocID)
		}(o.manifest.Nodes[alloc.NodeID].Handle, alloc.ID)
	}

	wg.Wait()
}

func (o *Orchestrator) requestBids(bidRequest jobtypes.EnsembleBidRequest, expiry time.Time) (chan Bid, chan struct{}, time.Time, error) {
	log.Debugf("requesting bids: %+v", bidRequest)

	bidExpiryTime := time.Now().Add(BidRequestTimeout)
	if expiry.Before(bidExpiryTime) {
		return nil, nil, time.Time{}, fmt.Errorf("not enough time for deployment: %w", ErrDeploymentFailed)
	}

	bidExpiry := uint64(bidExpiryTime.UnixNano())

	// Split requests into direct peer requests and broadcast requests
	var directRequests []BidRequest
	var broadcastRequests []BidRequest

	for _, req := range bidRequest.Request {
		if req.V1 == nil {
			continue
		}

		nodeConfig, ok := o.cfg.Node(req.V1.NodeID)
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
		nodeConfig, _ := o.cfg.Node(req.V1.NodeID)
		targetedReq := EnsembleBidRequest{
			ID:            bidRequest.ID,
			Request:       []BidRequest{req},
			PeerExclusion: bidRequest.PeerExclusion,
		}
		err := o.requestBidPeer(targetedReq, nodeConfig, bidExpiry)
		if err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("requesting bid to targeted peer: %w", err)
		}
	}

	// create reply behavior for this specific ensemble bid request
	bidCh := make(chan Bid)
	bidDoneCh := make(chan struct{})
	if err := o.actor.AddBehavior(
		behaviors.BidReplyBehavior,
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
		broadcastReq := EnsembleBidRequest{
			ID:            bidRequest.ID,
			Request:       broadcastRequests,
			PeerExclusion: bidRequest.PeerExclusion,
		}
		err := o.broadcastBid(broadcastReq, bidExpiry)
		if err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("broadcasting bid request: %w", err)
		}
	}

	return bidCh, bidDoneCh, bidExpiryTime, nil
}

func (o *Orchestrator) broadcastBid(bidRequest EnsembleBidRequest, bidExpiry uint64) error {
	msg, err := actor.Message(
		o.actor.Handle(),
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

	if err := o.actor.Publish(msg); err != nil {
		return fmt.Errorf("publishing bid request: %w", err)
	}
	return nil
}

func (o *Orchestrator) requestBidPeer(targetedReq EnsembleBidRequest, nodeConfig NodeConfig, bidExpiry uint64) error {
	destHandle, err := actor.HandleFromPeerID(nodeConfig.Peer)
	if err != nil {
		return fmt.Errorf("getting handle of selected peer %s: %w", nodeConfig.Peer, err)
	}

	log.Infof("sending direct peer request to %s: %+v", nodeConfig.Peer, targetedReq)
	msg, err := actor.Message(
		o.actor.Handle(),
		destHandle,
		behaviors.BidRequestBehavior,
		targetedReq,
		actor.WithMessageReplyTo(behaviors.BidReplyBehavior),
		actor.WithMessageExpiry(bidExpiry),
	)
	if err != nil {
		return fmt.Errorf("creating targeted bid message: %w", err)
	}

	if err := o.actor.Send(msg); err != nil {
		return fmt.Errorf("sending targeted bid request: %w", err)
	}

	return nil
}

func (o *Orchestrator) collectBids(bidCh chan Bid, bidDoneCh chan struct{}, bidExpiryTime time.Time, addBid func(Bid) bool, maxBids int) {
	defer close(bidDoneCh)

	log.Debugf("collecting bids until: %v", bidExpiryTime)
	timer := time.NewTimer(time.Until(bidExpiryTime))
	defer timer.Stop()

	bidCount := 0
	for {
		select {
		case bid, ok := <-bidCh:
			if !ok {
				log.Debugf("bid channel closed")
				return
			}
			log.Debugf("received bid: %+v", bid)
			if err := bid.Validate(); err != nil {
				log.Debugf("got invalid bid: %s", err)
				continue
			}
			if bid.EnsembleID() != o.id {
				log.Debugf("got bid for unexpected ensemble ID: %s", bid.EnsembleID())
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

func (o *Orchestrator) makeCandidateDeployments(bids map[string][]Bid) (func() (map[string]Bid, bool), bool) {
	// immediate satisfaction check: we need a bid for every node
	if len(o.cfg.Nodes()) != len(bids) {
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
		return o.makeCandidateDeploymentBig(bids)
	}

	return o.makeCandidateDeploymentSmall(bids)
}

func (o *Orchestrator) makeCandidateDeploymentSmall(bids map[string][]Bid) (func() (map[string]Bid, bool), bool) {
	// fix the order of permutation
	type permutator struct {
		mod  int64
		node string
		bids []Bid
	}
	permutators := make([]permutator, 0, len(bids))
	modulus := int64(1)
	for n, blst := range bids {
		permutators = append(permutators, permutator{mod: modulus, node: n, bids: blst})
		modulus *= int64(len(blst))
	}

	// function to get a permutation by index
	getPermutation := func(index int64) map[string]Bid {
		result := make(map[string]Bid)
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
	if nperm > MaxPermutations {
		nperm = MaxPermutations
	}
	count := int64(0)
	return func() (map[string]Bid, bool) {
		for count < nperm {
			count++

			nextPerm := rand.Int63n(nperm)
			perm := getPermutation(nextPerm)

			if !o.checkPermutationEdgeConstraints(perm) {
				continue
			}

			return perm, true
		}

		return nil, false
	}, true
}

func (o *Orchestrator) makeCandidateDeploymentBig(bids map[string][]Bid) (func() (map[string]Bid, bool), bool) {
	// Note: this is the same as above with bignums

	// fix the order of permutation
	type permutator struct {
		mod  *big.Int
		node string
		bids []Bid
	}
	permutators := make([]permutator, 0, len(bids))
	modulus := big.NewInt(1)
	for n, blst := range bids {
		permutators = append(permutators, permutator{mod: modulus, node: n, bids: blst})
		modulus = new(big.Int).Mul(modulus, big.NewInt(int64(len(blst))))
	}

	// function to get a permutation by index
	getPermutation := func(index *big.Int) map[string]Bid {
		result := make(map[string]Bid)
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
	return func() (map[string]Bid, bool) {
		for count < nperm {
			count++

			if _, err := crand.Read(bytes); err != nil {
				log.Errorf("error reading random bytes: %s", err)
				return nil, false
			}

			nextPerm := new(big.Int).SetBytes(bytes)
			perm := getPermutation(nextPerm)

			if !o.checkPermutationEdgeConstraints(perm) {
				continue
			}

			return perm, true
		}

		return nil, false
	}, true
}

func (o *Orchestrator) checkPermutationEdgeConstraints(candidate map[string]Bid) bool {
	for _, cst := range o.cfg.EdgeConstraints() {
		if cst.RTT == 0 {
			continue
		}

		bidS := candidate[cst.S]
		bidT := candidate[cst.T]

		locS, err := o.geo.Coordinate(bidS.Location())
		if err != nil {
			log.Errorf("Failed to get location for bid %s: %v", bidS.NodeID(), err)
			continue
		}

		locT, err := o.geo.Coordinate(bidT.Location())
		if err != nil {
			log.Errorf("Failed to get location for bid %s: %v", bidT.NodeID(), err)
			continue
		}

		distance := computeGeodesic(locS, locT)

		// in milliseconds
		minRTT := (distance / lightSpeed) * 2 * 1000

		if minRTT > float64(cst.RTT) {
			log.Debugf("Edge constraint not satisfied: min RTT %.2f ms > constraint %d ms for %s -> %s", minRTT, cst.RTT, cst.S, cst.T)
			return false
		}

		// TODO: add bandwidth check when that information becomes available
	}

	return true
}

func (o *Orchestrator) verifyEdgeConstraints(candidate map[string]Bid, cache map[string]bool) bool {
	var mx sync.Mutex
	var wg sync.WaitGroup
	var toVerify []EdgeConstraint

	for _, cst := range o.cfg.EdgeConstraints() {
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
		go func(cst EdgeConstraint) {
			defer wg.Done()
			result := o.verifyEdgeConstraint(candidate, cst)
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

func (o *Orchestrator) verifyEdgeConstraint(candidate map[string]Bid, cst EdgeConstraint) bool {
	bidS := candidate[cst.S]
	bidT := candidate[cst.T]
	key := bidS.Peer() + ":" + bidT.Peer()
	log.Debugf("verify edge constraint %s %v", key, cst)

	handle := bidS.Handle()
	msg, err := actor.Message(
		o.actor.Handle(),
		handle,
		behaviors.VerifyEdgeConstraintBehavior,
		VerifyEdgeConstraintRequest{
			EnsembleID: o.id,
			S:          bidS.Peer(),
			T:          bidT.Peer(),
			RTT:        cst.RTT,
			BW:         cst.BW,
		},
		actor.WithMessageTimeout(VerifyEdgeConstraintTimeout),
	)
	if err != nil {
		log.Warnf("error creating constraint check message for %s: %s", key, err)
		return false
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		log.Warnf("error invoking constraint check for %s: %s", key, err)
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
		log.Warnf("error unmarshalling bid constraint response for %s: %s", key, err)
		return false
	}

	if response.Error != "" {
		log.Debugf("error verifying bid constraint for %s: %s", key, err)
	}

	return response.OK
}

func (o *Orchestrator) commit(candidate map[string]Bid) (EnsembleManifest, error) {
	// This is a two phase commit:
	// - first commit the resources in all the nodes to ensure the deployment is (still)
	//   feasible.
	// - then create all the allocations for provisioning
	// - if there are any failures, we need to revert this deployment and start anew

	var mx sync.Mutex

	// Phase 1: commit
	var wg1 sync.WaitGroup
	ok := true
	committed := make([]string, 0, len(candidate))
	wg1.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid Bid) {
			defer wg1.Done()
			err := o.commitDeployment(n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorf("error committing bid for %s: %s", n, err)
				ok = false
			} else {
				log.Debugf("committed resources for %s", n)
				committed = append(committed, n)
			}
			mx.Unlock()
		}(n, bid)
	}
	wg1.Wait()

	if !ok {
		for _, n := range committed {
			bid := candidate[n]
			o.revertDeployment(n, bid.Handle())
		}
		return EnsembleManifest{}, fmt.Errorf("failed to commit resources: %w", ErrDeploymentFailed)
	}

	// Phase 2: allocate
	var wg2 sync.WaitGroup
	allocations := make(map[string]actor.Handle)
	wg2.Add(len(candidate))
	for n, bid := range candidate {
		go func(n string, bid Bid) {
			defer wg2.Done()
			allocated, err := o.allocate(n, bid.Handle())
			mx.Lock()
			if err != nil {
				log.Errorf("error allocating deployment for %s: %s", n, err)
				ok = false
			} else {
				log.Debugf("allocating deployment for %s", n)
				for a, h := range allocated {
					allocations[a] = h
				}
			}
			mx.Unlock()
		}(n, bid)
	}
	wg2.Wait()

	if !ok {
		for n, bid := range candidate {
			o.revertDeployment(n, bid.Handle())
		}
		return EnsembleManifest{}, fmt.Errorf("failed to allocate resources: %w", ErrDeploymentFailed)
	}

	// We are done, create the (partial) manifest
	// There are certain details that are filled during provisioning, e.g. allocation
	// VPN addresses and public port mappings
	mf := EnsembleManifest{
		ID:           o.id,
		Orchestrator: o.actor.Handle(),
		Allocations:  make(map[string]AllocationManifest),
		Nodes:        make(map[string]NodeManifest),
	}

	allocationNodes := make(map[string]string)
	portsByAllocation := make(map[string][]jobtypes.PortConfig)
	for n, bid := range candidate {
		ncfg, _ := o.cfg.Node(n)
		nmf := NodeManifest{
			ID:          n,
			Peer:        bid.Peer(),
			Handle:      bid.Handle(),
			Location:    bid.Location(),
			Allocations: ncfg.Allocations,
		}
		for _, a := range nmf.Allocations {
			allocationNodes[a] = n
			// TODO: optimize the manifest format and how node/alloc data is
			//       being passed around. A bit messy at the moment. see #825
			for _, portMap := range ncfg.Ports {
				if portMap.Allocation == a {
					portsByAllocation[a] = append(portsByAllocation[a], portMap)
				}
			}
		}
		mf.Nodes[n] = nmf
	}

	for name, alloc := range o.cfg.Allocations() {
		allocPorts := make(map[int]int)
		if ports, ok := portsByAllocation[name]; ok {
			for _, pc := range ports {
				allocPorts[pc.Public] = pc.Private
			}
		}
		amf := AllocationManifest{
			ID:          o.id + "_" + name,
			NodeID:      allocationNodes[name],
			Handle:      allocations[name],
			DNSName:     alloc.DNSName + ".internal",
			Healthcheck: alloc.HealthCheck,
			Ports:       allocPorts,
		}
		mf.Allocations[name] = amf
	}

	return mf, nil
}

type CommitDeploymentRequest struct {
	EnsembleID     string
	AllocationName string
	NodeID         string
	Resources      types.Resources
	PortMapping    map[int]int
}

type CommitDeploymentResponse struct {
	OK    bool
	Error string
}

func (o *Orchestrator) commitDeployment(n string, h actor.Handle) error {
	ncfg, _ := o.cfg.Node(n)

	getAllocPortMapping := func(allocName string) map[int]int {
		ports := make(map[int]int)
		for _, pc := range ncfg.Ports {
			if pc.Allocation == allocName {
				ports[pc.Public] = pc.Private
				break
			}
		}
		return ports
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(ncfg.Allocations))
	for _, allocName := range ncfg.Allocations {
		wg.Add(1)
		go func(allocName string) {
			defer wg.Done()
			allocation, ok := o.cfg.V1.Allocations[allocName]
			if !ok {
				errCh <- fmt.Errorf("allocation %s not found: %w", allocName, ErrDeploymentFailed)
			}

			allocPorts := getAllocPortMapping(allocName)
			msg, err := actor.Message(
				o.actor.Handle(),
				h,
				behaviors.CommitDeploymentBehavior,
				CommitDeploymentRequest{
					EnsembleID:     o.id,
					AllocationName: allocName,
					NodeID:         n,
					Resources:      allocation.Resources,
					PortMapping:    allocPorts,
				},
				actor.WithMessageTimeout(CommitDeploymentTimeout),
			)
			if err != nil {
				errCh <- fmt.Errorf("failed to create commit message for %s: %w", n, err)
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("failed to invoke commit for %s: %w", n, err)
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
			case <-time.After(CommitDeploymentTimeout):
				errCh <- fmt.Errorf("timeout committing for %s: %w", n, ErrDeploymentFailed)
			}
			defer reply.Discard()

			var response CommitDeploymentResponse
			if err := json.Unmarshal(reply.Message, &response); err != nil {
				errCh <- fmt.Errorf("error unmarshalling commit response for %s: %w", n, err)
			}

			if !response.OK {
				errCh <- fmt.Errorf("error committing for %s: %s: %w", n, response.Error, ErrDeploymentFailed)
			}
		}(allocName)
	}

	wg.Wait()
	close(errCh)

	var aggErr error
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	return nil
}

type RevertDeploymentMessage struct {
	EnsembleID   string
	AllocsByName []string
}

func (o *Orchestrator) revertDeployment(n string, h actor.Handle) {
	ncfg, _ := o.cfg.Node(n)

	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.RevertDeploymentBehavior,
		RevertDeploymentMessage{
			EnsembleID:   o.id,
			AllocsByName: ncfg.Allocations,
		},
	)
	if err != nil {
		log.Debugf("failed to create revert message for %s: %s", n, err)
		return
	}

	if err := o.actor.Send(msg); err != nil {
		log.Debugf("failed to send revert message for %s: %s", n, err)
	}
}

type AllocationDeploymentRequest struct {
	EnsembleID  string
	NodeID      string
	Allocations map[string]AllocationDeploymentConfig
}

type AllocationDeploymentConfig struct {
	Executor         AllocationExecutor
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
}

type AllocationDeploymentResponse struct {
	OK          bool
	Error       string
	Allocations map[string]actor.Handle
}

func (o *Orchestrator) allocate(n string, h actor.Handle) (map[string]actor.Handle, error) {
	allocs := make(map[string]AllocationDeploymentConfig)
	ncfg, _ := o.cfg.Node(n)
	for _, a := range ncfg.Allocations {
		acfg, _ := o.cfg.Allocation(a)

		provisionScripts := make(map[string][]byte)
		for _, p := range acfg.Provision {
			provisionScripts[p] = o.cfg.V1.Scripts[p]
		}

		allocs[a] = AllocationDeploymentConfig{
			Executor:         acfg.Executor,
			Resources:        acfg.Resources,
			Execution:        acfg.Execution,
			ProvisionScripts: provisionScripts,
		}
	}

	msg, err := actor.Message(
		o.actor.Handle(),
		h,
		behaviors.AllocationDeploymentBehavior,
		AllocationDeploymentRequest{
			EnsembleID:  o.id,
			NodeID:      n,
			Allocations: allocs,
		},
		actor.WithMessageTimeout(AllocationDeploymentTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation message for %s: %w", n, err)
	}

	log.Debugf("Invoking allocation for node: %s", n)
	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke allocate for %s: %w", n, err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(AllocationDeploymentTimeout):
		return nil, fmt.Errorf("timeout in allocation for %s: %w", n, err)
	}
	defer reply.Discard()

	var response AllocationDeploymentResponse
	if err := json.Unmarshal(reply.Message, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling allocation response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("allocation for %s failed: %s: %w", n, response.Error, ErrDeploymentFailed)
	}

	// verify that the allocation map has all the allocations
	for a := range allocs {
		if _, ok := response.Allocations[a]; !ok {
			return nil, fmt.Errorf("missing allocation %s for %s: %w", a, n, ErrDeploymentFailed)
		}
	}

	log.Debugf("Allocation successful for node: %s", n)
	return response.Allocations, nil
}

type SubnetCreateRequest struct {
	SubnetID     string
	IP           string
	RoutingTable map[string]string
}

type SubnetCreateResponse struct {
	OK    bool
	Error string
}

func (o *Orchestrator) provision(em EnsembleManifest) error {
	log.Infof("provisioning ensemble manifest: %+v", em)

	// 1. create subnet
	// 1.a generate routing table
	cidr, err := netutils.GetRandomCIDRInRange(
		24,
		net.ParseIP("10.0.0.0"),
		net.ParseIP("10.255.255.255"),
		[]string{},
	)
	if err != nil {
		return fmt.Errorf("error getting random CIDR: %w", err)
	}

	parts := strings.Split(strings.Split(cidr, "/")[0], ".")
	gatewayIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "1")
	broadcastIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "255")
	usedIPs := map[string]bool{
		gatewayIP:   true,
		broadcastIP: true,
	}

	routingTable := make(map[string]string)
	indexRoutingTable := make(map[string]string)
	for allocationID, manifest := range em.Allocations {
		ip, err := netutils.GetNextIP(cidr, usedIPs)
		log.Debug("Generated IP", ip, "for alllocation", allocationID)
		if err != nil {
			return fmt.Errorf("error getting next IP: %w", err)
		}
		routingTable[ip.String()] = em.Nodes[manifest.NodeID].Peer
		indexRoutingTable[allocationID] = ip.String()
		usedIPs[ip.String()] = true
	}

	errCh := make(chan error, len(em.Allocations))
	wg := sync.WaitGroup{}
	for _, manifest := range em.Nodes {
		wg.Add(1)
		go func() {
			defer wg.Done()

			msg, err := actor.Message(
				o.actor.Handle(),
				manifest.Handle,
				fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, em.ID),
				SubnetCreateRequest{
					SubnetID:     em.ID,
					RoutingTable: routingTable,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetCreateResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error creating subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(SubnetCreateTimeout):
				errCh <- fmt.Errorf("timeout creating subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("subnet successfully created on peer", manifest.Handle)
		}()
	}

	wg.Wait()
	close(errCh)

	var aggErr error
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 1.b create and plug IPs
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(em.Allocations))
	for allocationID, manifest := range em.Allocations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				manifest.Handle,
				behaviors.SubnetAddPeerBehavior,
				SubnetAddPeerRequest{
					SubnetID: em.ID,
					IP:       indexRoutingTable[allocationID],
					PeerID:   em.Nodes[manifest.NodeID].Peer,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-peer message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-peer message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetAddPeerResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout adding peer to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("peer successfully added to subnet on peer", manifest.Handle)
		}()
	}

	wg.Wait()
	close(errCh)
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 1.c configure DNS
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(em.Allocations))
	dnsRecords := make(map[string]string)
	for allocationID, manifest := range em.Allocations {
		dnsRecords[manifest.DNSName] = indexRoutingTable[allocationID]
	}

	for _, manifest := range em.Allocations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				o.actor.Handle(),
				manifest.Handle,
				behaviors.SubnetDNSAddRecordsBehavior,
				SubnetDNSAddRecordsRequest{
					SubnetID: em.ID,
					Records:  dnsRecords,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-dns-records message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-dns-records message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetDNSAddRecordsResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error sending dns records to peer: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}

			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout sending dns records to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("DNS records successfully added to subnet on peer", manifest.Handle)
		}()
	}

	wg.Wait()
	close(errCh)
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 1.d configure port mapping
	wg = sync.WaitGroup{}
	errCh = make(chan error, len(em.Allocations))
	for allocationID, manifest := range em.Allocations {
		for srcPort, destPort := range manifest.Ports {
			wg.Add(1)
			go func() {
				defer wg.Done()
				msg, err := actor.Message(
					o.actor.Handle(),
					manifest.Handle,
					behaviors.SubnetMapPortBehavior,
					SubnetMapPortRequest{
						SubnetID:   em.ID,
						Protocol:   "TCP", // TODO: add support in AllocationManifest for protocol
						SourceIP:   "0.0.0.0",
						SourcePort: strconv.Itoa(srcPort),
						DestIP:     indexRoutingTable[allocationID],
						DestPort:   strconv.Itoa(destPort),
					},
					actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
				)
				if err != nil {
					errCh <- fmt.Errorf("error creating subnet MapPort message: %w", err)
					return
				}

				replyCh, err := o.actor.Invoke(msg)
				if err != nil {
					errCh <- fmt.Errorf("error invoking subnet MapPort message: %w", err)
					return
				}

				var reply actor.Envelope
				select {
				case reply = <-replyCh:
					defer reply.Discard()

					var response SubnetMapPortResponse
					if err := json.Unmarshal(reply.Message, &response); err != nil {
						errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
						return
					}

					if !response.OK {
						errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
						return
					}
				case <-time.After(2 * time.Minute):
					errCh <- fmt.Errorf("timeout mapping port for subnet: %w", ErrDeploymentFailed)
					return
				}

				log.Info("port mapping successfully added to subnet on peer", manifest.Handle)
			}()
		}
	}

	wg.Wait()
	close(errCh)
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	// 2. start the allocations
	errCh = make(chan error, len(em.Allocations))
	wg = sync.WaitGroup{}
	for allocName, manifest := range em.Allocations {
		wg.Add(1)
		go func(manifest AllocationManifest) {
			defer wg.Done()

			msg, err := actor.Message(
				o.actor.Handle(),
				manifest.Handle,
				behaviors.AllocationStartBehavior,
				AllocationStartRequest{
					SubnetIP:    indexRoutingTable[allocName],
					GatewayIP:   gatewayIP,
					PortMapping: manifest.Ports,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating allocation start message: %w", err)
				return
			}

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking allocation start: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response AllocationStartResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling allocation start response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error starting allocation: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout starting allocation: %w", ErrDeploymentFailed)
				return
			}

			log.Infof("allocation successfully started on peer %s for allocation %s", &manifest.Handle.DID, manifest.ID)
		}(manifest)
	}

	wg.Wait()
	close(errCh)
	aggErr = nil
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	if aggErr != nil {
		return aggErr
	}

	return nil
}

func (o *Orchestrator) revert(mf EnsembleManifest) {
	log.Infof("reverting ensemble manifest: %+v", mf)
	for n, nmf := range mf.Nodes {
		o.revertDeployment(n, nmf.Handle)
	}
}

func (o *Orchestrator) acceptPeerLocation(nodeID, peerID string, loc Location) bool {
	n, ok := o.cfg.Node(nodeID)
	if !ok {
		return false
	}

	// check explicit peer placement
	if n.Peer != "" {
		return n.Peer == peerID
	}

	// check acceptable locations
	if len(n.Location.Accept) > 0 {
		accept := false
		for _, acceptable := range n.Location.Accept {
			if acceptable.Equal(loc) {
				accept = true
				break
			}
		}
		if !accept {
			return false
		}
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

func (o *Orchestrator) makeInitialBidRequest() (jobtypes.EnsembleBidRequest, error) {
	return o.ensembleConfigToBidRequest(&o.cfg)
}

func (o *Orchestrator) makeResidualBidRequest(candidate map[string][]Bid, exclusion map[string]struct{}) (jobtypes.EnsembleBidRequest, error) {
	residualConfig := EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: make(map[string]jobtypes.AllocationConfig),
			Nodes:       make(map[string]jobtypes.NodeConfig),
		},
	}

	for n, ncfg := range o.cfg.V1.Nodes {
		if _, exclude := candidate[n]; exclude {
			continue
		}

		residualConfig.V1.Nodes[n] = ncfg
	}

	for id, ncfg := range residualConfig.V1.Nodes {
		log.Debugf("still looking for node %s", id)
		for _, a := range ncfg.Allocations {
			residualConfig.V1.Allocations[a] = o.cfg.V1.Allocations[a]
		}
	}

	result, err := o.ensembleConfigToBidRequest(&residualConfig)
	if err != nil {
		return result, err
	}

	for p := range exclusion {
		result.PeerExclusion = append(result.PeerExclusion, p)
	}

	return result, nil
}

func (o *Orchestrator) ensembleConfigToBidRequest(config *EnsembleConfig) (jobtypes.EnsembleBidRequest, error) {
	v1Config := config.V1

	ensembleBidRequest := jobtypes.EnsembleBidRequest{
		ID: o.id,
	}

	log.Infof("creating bid request for nodes: %+v", v1Config.Nodes)
	for nodeID, nodeConfig := range v1Config.Nodes {
		bidRequest := jobtypes.BidRequest{
			V1: &jobtypes.BidRequestV1{
				NodeID:   nodeID,
				Location: nodeConfig.Location,
			},
		}

		var aggregateResources types.Resources
		var executors []AllocationExecutor

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

			if allocationConfig.Executor == jobtypes.ExecutorDocker {
				// check if bid includes allocation requiring privileged docker
				dockerCfg, err := docker.DecodeSpec(&allocationConfig.Execution)
				if err != nil {
					return EnsembleBidRequest{}, fmt.Errorf("decoding docker spec: %w", err)
				}

				if dockerCfg.Privileged {
					bidRequest.V1.GeneralRequirements.PrivilegedDocker = true
				}
			}

			err := aggregateResources.Add(allocationConfig.Resources)
			if err != nil {
				return jobtypes.EnsembleBidRequest{}, err
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

func (o *Orchestrator) supervise() {
	log.Debugf("Starting supervision for allocations: %+v", o.manifest.Allocations)
	expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
	wg := sync.WaitGroup{}

	for allocName, allocation := range o.manifest.Allocations {
		// skip empty healthchecks
		if o.manifest.Allocations[allocName].Healthcheck.Type == "" {
			continue
		}

		msg, err := actor.Message(
			o.actor.Handle(),
			allocation.Handle,
			behaviors.RegisterHealthcheckBehavior,
			RegisterHealthcheckRequest{
				EnsembleID:  o.id,
				HealthCheck: o.manifest.Allocations[allocName].Healthcheck,
			},
			actor.WithMessageExpiry(expiry),
		)
		if err != nil {
			log.Errorf("failed to create supervisor message: %s", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("failed to invoke heartbeat on allocation: %s", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response RegisterHealthcheckResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					log.Errorf("error unmarshalling supervisor reply: %s", err)
					return
				}

				if !response.OK {
					log.Errorf("error registering healthcheck: %s", response.Error)
					return
				}
			case <-time.After(time.Second * 5):
				log.Errorf("timeout waiting for supervisor reply")
				return
			}

			log.Info("successfully registered healthcheck for allocation: %s", allocation.ID)
		}()
	}

	wg.Wait()

	ticker := time.NewTicker(actor.HealthCheckInterval)
	defer ticker.Stop()

	failures := map[string]int{}
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
			wg := sync.WaitGroup{}
			for _, allocation := range o.manifest.Allocations {
				msg, err := actor.Message(
					o.actor.Handle(),
					allocation.Handle,
					actor.HealthCheckBehavior,
					struct{}{},
					actor.WithMessageExpiry(expiry),
				)
				if err != nil {
					log.Errorf("failed to create supervisor message: %s", err)
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()

					replyCh, err := o.actor.Invoke(msg)
					if err != nil {
						log.Errorf("failed to invoke heartbeat on allocation: %s", err)
						return
					}

					ticker := time.NewTicker(time.Second * 5)
					defer ticker.Stop()

					var reply actor.Envelope
					select {
					case reply = <-replyCh:
						defer reply.Discard()

						var resp HealthCheckResponse
						if err := json.Unmarshal(reply.Message, &resp); err != nil {
							log.Errorf("error unmarshalling supervisor reply: %s", err)
							return
						}

						if !resp.OK {
							log.Errorf("error in healthcheck: %s", resp.Error)
							failures[allocation.Handle.ID.String()]++
							v := failures[allocation.Handle.ID.String()]
							if v >= 3 {
								if err := o.escalateFailure(allocation.Handle); err != nil {
									log.Errorf("failed to escalate failure: %s", err)
								} else {
									log.Debug("escalated failure, resetting healthcheck failures counter")
									delete(failures, allocation.Handle.ID.String())
								}
							}
							return
						} else {
							log.Infof("successfully healthchecked allocation %s", allocation.ID)
							delete(failures, allocation.Handle.ID.String())
							return
						}

					case <-ticker.C:
						log.Warnf("timeout waiting for supervisor reply")
						failures[allocation.Handle.ID.String()]++
						v := failures[allocation.Handle.ID.String()]
						if v >= 3 {
							if err := o.escalateFailure(allocation.Handle); err != nil {
								log.Errorf("failed to escalate failure: %s", err)
							} else {
								log.Debug("escalated failure, resetting healthcheck failures counter")
								delete(failures, allocation.Handle.ID.String())
							}
							return
						}
					}
				}()
			}

			wg.Wait()
		}
	}
}

func (o *Orchestrator) escalateFailure(allocHandle actor.Handle) error {
	// TODO we need to decide how to handle repeated failures and also correlated failures
	//      from a node.
	//      Also, we should not restart at first failure, but wait for a number of
	//      consecutive failures.
	//      See https://gitlab.com/nunet/device-management-service/-/issues/794
	log.Debugf("escalating failure for allocation %s", allocHandle.String())
	expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
	msg, err := actor.Message(
		o.actor.Handle(),
		allocHandle,
		behaviors.AllocationRestartBehavior,
		struct{}{},
		actor.WithMessageExpiry(expiry),
	)
	if err != nil {
		return err
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return err
	}

	select {
	case reply := <-replyCh:
		reply.Discard()
		return nil
	case <-time.After(time.Minute * 2):
		return fmt.Errorf("timeout waiting for supervisor reply")
	}
}

func (o *Orchestrator) Stop() {
	// TODO

	err := o.actor.Stop()
	if err != nil {
		log.Warnf("error stopping orchestrator's actor: %s", err)
	}
}

type AllocationLogsRequest struct {
	AllocName string
}

type AllocationLogsResponse struct {
	Stdout []byte
	Stderr []byte
	Error  string
}

func (o *Orchestrator) GetAllocationLogs(name string) (AllocationLogsResponse, error) {
	var allocNodeHandle actor.Handle
	var logsResp AllocationLogsResponse
	for _, n := range o.manifest.Nodes {
		if ok := utils.SliceContains(n.Allocations, name); ok {
			allocNodeHandle = n.Handle
			break
		}
	}

	if allocNodeHandle.Empty() {
		return logsResp,
			fmt.Errorf(
				"node not found for allocation %s of ensemble %s",
				name, o.id,
			)
	}

	msg, err := actor.Message(
		o.actor.Handle(),
		allocNodeHandle,
		fmt.Sprintf(behaviors.AllocationLogsBehavior, o.manifest.ID),
		AllocationLogsRequest{
			AllocName: name,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
	)
	if err != nil {
		return logsResp, fmt.Errorf("creating get logs message: %w", err)
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return logsResp, fmt.Errorf("invoking get logs message: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(2 * time.Minute):
		return logsResp, fmt.Errorf("timeout getting logs for %s: %w", name, ErrDeploymentFailed)
	}

	defer reply.Discard()

	if err := json.Unmarshal(reply.Message, &logsResp); err != nil {
		return logsResp, fmt.Errorf("unmarshalling get logs response: %w", err)
	}

	if logsResp.Error != "" {
		return logsResp, fmt.Errorf("replied with error getting logs for %s: %s", name, logsResp.Error)
	}

	return logsResp, nil
}

func containsExecutor(executors []AllocationExecutor, executor AllocationExecutor) bool {
	for _, e := range executors {
		if e == executor {
			return true
		}
	}
	return false
}

func (o *Orchestrator) Status() DeploymentStatus {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.status
}

func (o *Orchestrator) Manifest() EnsembleManifest {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.manifest.Clone()
}

func (o *Orchestrator) Config() EnsembleConfig {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.cfg.Clone()
}

func (o *Orchestrator) ID() string {
	return o.id
}

func (o *Orchestrator) ActorPrivateKey() crypto.PrivKey {
	return o.actor.Security().PrivKey()
}

func (o *Orchestrator) DeploymentSnapshot() DeploymentSnapshot {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.deploymentSnapshot
}
