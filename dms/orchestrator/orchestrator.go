// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	BidRequestTimeout           = 5 * time.Second
	VerifyEdgeConstraintTimeout = 5 * time.Second
	CommitDeploymentTimeout     = 3 * time.Second
	AllocationDeploymentTimeout = 5 * time.Second

	// Setting a big timeout as the user might have to
	// download large execution images
	AllocationStartTimeout    = 5 * time.Minute
	AllocationShutdownTimeout = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second

	SubnetCreateTimeout  = 2 * time.Minute
	SubnetDestroyTimeout = 30 * time.Second

	MaxBidMultiplier = 8
	MaxPermutations  = 1_000_000

	grantOrchestratorCapsFrequency = 5 * time.Minute

	orchSubnetName = "orchestrator"
)

var (
	ErrProvisioningFailed   = errors.New("failed to provision the ensemble")
	ErrDeploymentFailed     = errors.New("failed to create deployment")
	ErrOrchestratorExists   = errors.New("orchestrator with ID already exists")
	ErrOrchestratorNotFound = errors.New("orchestrator with ID not found")
)

// Orchestrator manages the lifecycle of an ensemble deployment
type Orchestrator interface {
	Deploy(expiry time.Time) error
	Shutdown()
	Stop()
	Status() jtypes.DeploymentStatus
	Manifest() jtypes.EnsembleManifest
	Config() jtypes.EnsembleConfig
	ID() string
	ActorPrivateKey() crypto.PrivKey
	DeploymentSnapshot() jtypes.DeploymentSnapshot
	GetAllocationLogs(name string) (AllocationLogsResponse, error)
	WriteAllocationLogs(name string, stdout, stderr []byte) (string, error)
}

type BasicOrchestrator struct {
	lock   sync.Mutex
	ctx    context.Context
	cancel func()

	fs      afero.Afero
	workDir string
	actor   actor.Actor
	geo     *geolocation.GeoLocator

	id       string
	cfg      jtypes.EnsembleConfig
	manifest jtypes.EnsembleManifest
	status   jtypes.DeploymentStatus

	deploymentSnapshot jtypes.DeploymentSnapshot

	nonce uint64
}

var _ Orchestrator = (*BasicOrchestrator)(nil)

func NewOrchestrator(
	ctx context.Context,
	fs afero.Afero,
	workDir string,
	id string,
	oActor actor.Actor,
	cfg jtypes.EnsembleConfig,
) (*BasicOrchestrator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate ensemble configuration: %w", err)
	}

	geo, err := geolocation.NewGeoLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create geolocator: %w", err)
	}

	o := &BasicOrchestrator{
		actor:   oActor,
		geo:     geo,
		id:      id,
		cfg:     cfg,
		ctx:     ctx,
		fs:      fs,
		workDir: workDir,
	}

	orchestratorBehaviors := map[string]func(actor.Envelope){
		behaviors.NotifyTaskTerminationBehavior: o.handleTaskTermination,
	}

	for b, handler := range orchestratorBehaviors {
		err := o.actor.AddBehavior(b, handler)
		if err != nil {
			return nil, fmt.Errorf("add allocation start behavior to allocation actor: %w", err)
		}
	}

	return o, nil
}

func (o *BasicOrchestrator) setStatus(status jtypes.DeploymentStatus) {
	o.lock.Lock()
	defer o.lock.Unlock()

	log.Infof("setting status to: %s", jtypes.DeploymentStatusString(status))
	o.status = status
}

func (o *BasicOrchestrator) Deploy(expiry time.Time) error {
	defer func() {
		if o.status != jtypes.DeploymentStatusRunning {
			o.setStatus(jtypes.DeploymentStatusFailed)
		}
	}()
	o.setStatus(jtypes.DeploymentStatusPreparing)

	return o.deploy(expiry)
}

func (o *BasicOrchestrator) initializeManifest() {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.manifest = jtypes.EnsembleManifest{
		ID:           o.id,
		Orchestrator: o.actor.Handle(),
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Nodes:        make(map[string]jtypes.NodeManifest),
	}

	for name, alloc := range o.cfg.Allocations() {
		amf := jtypes.AllocationManifest{
			ID:          types.ConstructAllocationID(o.id, name),
			DNSName:     alloc.DNSName + ".internal",
			Healthcheck: alloc.HealthCheck,
			Status:      jtypes.AllocationPending,
			Ports:       make(map[int]int),
			Type:        alloc.Type,
		}
		o.manifest.Allocations[name] = amf
	}
	for name, node := range o.cfg.Nodes() {
		nmf := jtypes.NodeManifest{
			Allocations: node.Allocations,
			Peer:        node.Peer,
		}
		o.manifest.Nodes[name] = nmf
	}

	o.manifest.Subnet = o.cfg.V1.Subnet
}

func (o *BasicOrchestrator) deploy(expiry time.Time) error {
	o.deploymentSnapshot.Expiry = expiry
	edgeConstraintCache := make(map[string]bool)

deploy:
	for time.Now().Before(expiry) {
		o.setStatus(jtypes.DeploymentStatusPreparing)

		log.Debugf("manifest being initialized...")
		o.initializeManifest()

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
			nextCandidate func() (map[string]jtypes.Bid, bool)
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
			bidrq, err := o.makeResidualBidRequest(bidMap, rmBid)
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

		for n, bids := range bidMap {
			log.Infof("node %s has %d bids", n, len(bids))
			for _, bid := range bids {
				log.Infof("    bid from %s", bid.Peer())
			}
		}

		// 4. Iterate through the candidates trying to find one that satisfies the
		//    edge constraints
		o.setStatus(jtypes.DeploymentStatusGenerating)

		log.Debugf("generating candidate deployment")
		var candidate map[string]jtypes.Bid
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
		o.setStatus(jtypes.DeploymentStatusCommitting)
		o.deploymentSnapshot.Candidates = candidate

		log.Info("committing candidate bids")
		err = o.commit(candidate)
		if err != nil {
			log.Warnf("failed to commit deployment: %s", err)
			continue deploy
		}

		// 6. provision the network and start the allocations
		o.setStatus(jtypes.DeploymentStatusProvisioning)

		log.Info("provisioning network")
		if err := o.provision(); err != nil {
			log.Errorf("failed to provision network: %s", err)

			o.lock.Lock()
			o.revert(o.manifest)
			o.lock.Unlock()
			continue deploy
		}

		// We are done! start the supervisor return the manifest.
		o.lock.Lock()
		o.ctx, o.cancel = context.WithCancel(context.Background())
		o.lock.Unlock()

		log.Infof("deployment successful, starting supervisor")
		o.setStatus(jtypes.DeploymentStatusRunning)
		allocations := make(map[string]actor.Handle, len(o.manifest.Allocations))
		for _, allocation := range o.manifest.Allocations {
			allocations[allocation.ID] = allocation.Handle
		}
		go o.supervise()

		return nil
	}

	// we failed to create the deployment in time
	log.Errorf("failed to create the deployment in time")
	return ErrDeploymentFailed
}

func (o *BasicOrchestrator) Stop() {
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

func (o *BasicOrchestrator) GetAllocationLogs(name string) (AllocationLogsResponse, error) {
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

func (o *BasicOrchestrator) Status() jtypes.DeploymentStatus {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.status
}

func (o *BasicOrchestrator) Manifest() jtypes.EnsembleManifest {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.manifest.Clone()
}

func (o *BasicOrchestrator) Config() jtypes.EnsembleConfig {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.cfg.Clone()
}

func (o *BasicOrchestrator) ID() string {
	return o.id
}

func (o *BasicOrchestrator) ActorPrivateKey() crypto.PrivKey {
	return o.actor.Security().PrivKey()
}

func (o *BasicOrchestrator) DeploymentSnapshot() jtypes.DeploymentSnapshot {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.deploymentSnapshot
}
