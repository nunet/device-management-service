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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/executor"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	helloMinDelay                   = 10 * time.Second
	helloMaxDelay                   = 20 * time.Second
	helloTimeout                    = 3 * time.Second
	helloAttempts                   = 3
	clearCommitedResourcesFrequency = 60 * time.Second

	grantAllocationCapsFreq = 1 * time.Hour

	rootProto = "actor/root/messages/0.0.1"
)

// Node is the structure that holds the node's dependencies.
type Node struct {
	rootCap         ucan.CapabilityContext
	actor           actor.Actor
	scheduler       *bt.Scheduler
	network         network.Network
	resourceManager types.ResourceManager
	hardware        types.HardwareManager
	hostID          string
	onboarder       *onboarding.Onboarding
	executors       map[string]executorMetadata
	rumutex         sync.RWMutex

	ctx    context.Context
	cancel func()

	mx          sync.Mutex
	allocmx     sync.Mutex
	peers       map[peer.ID]*peerState
	bids        map[string]*bidState
	deployments map[string]*jobs.Orchestrator
	allocations map[string]*jobs.Allocation
	running     int32

	orchestratorRepo  repositories.OrchestratorView
	geoip             types.GeoIPLocator
	hostLocation      HostGeolocation
	portConfig        PortConfig
	portAllocator     *PortAllocator
	commitedResources map[string]*bidState

	dmsConfig config.Config
	fs        afero.Afero

	contractRepo repositories.Contract
}

type peerState struct {
	conns                           int
	hasRoot                         bool
	helloIn, helloOut, helloPending bool
	helloAttempts                   int
}

type bidState struct {
	expire  time.Time
	request job_types.BidRequest
	ports   []int
}

type executorMetadata struct {
	executor      executor.Executor
	executionType jobs.AllocationExecutor
}

type HostGeolocation struct {
	HostContinent string
	HostCountry   string
	HostCity      string
}

type PortConfig struct {
	AvailableRangeFrom int
	AvailableRangeTo   int
}

// New creates a new node, attaches an actor to the node.
func New(cfg config.Config, fs afero.Afero,
	onboarder *onboarding.Onboarding,
	rootCap ucan.CapabilityContext,
	hostID string, net network.Network,
	resourceManager types.ResourceManager,
	scheduler *bt.Scheduler,
	hardware types.HardwareManager,
	orchestratorRepo repositories.OrchestratorView,
	geoip types.GeoIPLocator, hostLocation HostGeolocation, portConfig PortConfig,
	contractRepo repositories.Contract,
) (*Node, error) {
	if onboarder == nil {
		return nil, errors.New("onboarder is nil")
	}
	if rootCap == nil {
		return nil, errors.New("root capability context is nil")
	}

	if hostID == "" {
		return nil, errors.New("host id is nil")
	}

	if net == nil {
		return nil, errors.New("network is nil")
	}

	if resourceManager == nil {
		return nil, errors.New("resource manager is nil")
	}

	if scheduler == nil {
		return nil, errors.New("scheduler is nil")
	}

	if geoip == nil {
		return nil, errors.New("geoip is nil")
	}

	rootDID := rootCap.DID()
	rootTrust := rootCap.Trust()

	anchor, err := rootTrust.GetAnchor(rootDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root DID anchor: %w", err)
	}
	pubk := anchor.PublicKey()

	provider, err := rootTrust.GetProvider(rootDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root DID provider: %w", err)
	}

	privk, err := provider.PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get root private key: %w", err)
	}

	rootSec, err := actor.NewBasicSecurityContext(pubk, privk, rootCap)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context: %w", err)
	}

	nodeActor, err := createActor(rootSec, actor.NewRateLimiter(actor.DefaultRateLimiterConfig()), hostID, "root", net, actor.Handle{})
	if err != nil {
		return nil, fmt.Errorf("failed to create node actor: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	n := &Node{
		hostID:            hostID,
		network:           net,
		bids:              make(map[string]*bidState),
		deployments:       make(map[string]*jobs.Orchestrator),
		allocations:       make(map[string]*jobs.Allocation),
		peers:             make(map[peer.ID]*peerState),
		resourceManager:   resourceManager,
		hardware:          hardware,
		actor:             nodeActor,
		rootCap:           rootCap,
		scheduler:         scheduler,
		onboarder:         onboarder,
		executors:         make(map[string]executorMetadata),
		ctx:               ctx,
		cancel:            cancel,
		orchestratorRepo:  orchestratorRepo,
		geoip:             geoip,
		hostLocation:      hostLocation,
		portConfig:        portConfig,
		portAllocator:     NewPortAllocator(portConfig),
		commitedResources: make(map[string]*bidState),
		dmsConfig:         cfg,
		fs:                fs,
		contractRepo:      contractRepo,
	}

	if err := n.initSupportedExecutors(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("new executor: %w", err)
	}

	dmsBehaviors := map[string]struct {
		fn   func(actor.Envelope)
		opts []actor.BehaviorOption
	}{
		PublicHelloBehavior: {
			fn: n.publicHelloBehavior,
		},
		PublicStatusBehavior: {
			fn: n.publicStatusBehavior,
		},
		BroadcastHelloBehavior: {
			fn: n.broadcastHelloBehavior,
			opts: []actor.BehaviorOption{
				actor.WithBehaviorTopic(BroadcastHelloTopic),
			},
		},
		PeersListBehavior: {
			fn: n.handlePeersList,
		},
		PeerAddrInfoBehavior: {
			fn: n.handlePeerAddrInfo,
		},
		PeerPingBehavior: {
			fn: n.handlePeerPing,
		},
		PeerDHTBehavior: {
			fn: n.handlePeerDHT,
		},
		PeerConnectBehavior: {
			fn: n.handlePeerConnect,
		},
		PeerScoreBehavior: {
			fn: n.handlePeerScore,
		},
		OnboardBehavior: {
			fn: n.handleOnboard,
		},
		OffboardBehavior: {
			fn: n.handleOffboard,
		},
		OnboardStatusBehavior: {
			fn: n.handleOnboardStatus,
		},
		VMStartBehavior: {
			fn: n.handleVMContainerStart,
		},
		VMStopBehavior: {
			fn: n.handleVMContainerStop,
		},
		VMListBehavior: {
			fn: n.handleVMContainerList,
		},
		ContainerStartBehavior: {
			fn: n.handleVMContainerStart,
		},
		ContainerStopBehavior: {
			fn: n.handleVMContainerStop,
		},
		ContainerListBehavior: {
			fn: n.handleVMContainerList,
		},
		NewDeploymentBehavior: {
			fn: n.newDeployment,
		},
		DeploymentListBehavior: {
			fn: n.handleDeploymentList,
		},
		DeploymentLogsBehavior: {
			fn: n.handleDeploymentLogs,
		},
		DeploymentStatusBehavior: {
			fn: n.handleDeploymentStatus,
		},
		DeploymentManifestBehavior: {
			fn: n.handleDeploymentManifest,
		},
		DeploymentShutdownBehavior: {
			fn: n.handleDeploymentShutdown,
		},
		jobs.VerifyEdgeConstraintBehavior: {
			fn: n.deploymentVerifyEdgeConstraint,
		},
		jobs.BidRequestBehavior: {
			fn: n.handleBidRequest,
			opts: []actor.BehaviorOption{
				actor.WithBehaviorTopic(jobs.BidRequestTopic),
			},
		},
		jobs.RevertDeploymentBehavior: {
			fn: n.handleRevertDeployment,
		},
		jobs.SubnetCreateBehavior.Static: {
			fn: n.handleSubnetCreate,
		},
		jobs.SubnetDestroyBehavior.Static: {
			fn: n.handleSubnetDestroy,
		},
		ResourcesAllocatedBehavior: {
			fn: n.handleAllocatedResources,
		},
		ResourcesFreeBehavior: {
			fn: n.handleFreeResources,
		},
		ResourcesOnboardedBehavior: {
			fn: n.handleOnboardedResources,
		},
		LoggerConfigBehavior: {
			fn: n.handleLoggerConfig,
		},
		HardwareUsageBehavior: {
			fn: n.handleHardwareUsage,
		},
		CapListBehavior: {
			fn: n.handleCapList,
		},
		CapAnchorBehavior: {
			fn: n.handleCapAnchor,
		},
		jobs.AllocationDeploymentBehavior: {
			fn: n.handleAllocationDeployment,
		},
		jobs.CommitDeploymentBehavior: {
			fn: n.handleCommitDeployment,
		},
	}
	for behavior, handler := range dmsBehaviors {
		if err := nodeActor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return nil, fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}

	if err := n.restoreDeployments(); err != nil {
		log.Errorf("restoring deployments: %s", err)
	}

	ticker := time.NewTicker(clearCommitedResourcesFrequency)
	go func() {
		for range ticker.C {
			n.clearCommitedResources()
		}
	}()

	return n, nil
}

// GetBidRequests returns the bid requests for the node.
func (n *Node) GetBidRequests() []jobs.BidRequest {
	n.mx.Lock()
	defer n.mx.Unlock()

	reqs := []jobs.BidRequest{}
	for _, v := range n.bids {
		reqs = append(reqs, v.request)
	}

	return reqs
}

// GetAllocation gets an allocation by id.
func (n *Node) GetAllocation(id string) (*jobs.Allocation, error) {
	n.allocmx.Lock()
	defer n.allocmx.Unlock()

	alloc, ok := n.allocations[id]
	if !ok {
		return nil, errors.New("allocation not found")
	}

	return alloc, nil
}

// GetAllocations returns a list of allocations in the node.
func (n *Node) GetAllocations() []*jobs.Allocation {
	n.allocmx.Lock()
	defer n.allocmx.Unlock()

	allAllocs := []*jobs.Allocation{}

	for _, v := range n.allocations {
		allAllocs = append(allAllocs, v)
	}

	return allAllocs
}

// Start node
func (n *Node) Start() error {
	if !atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		return nil
	}

	if err := n.actor.Start(); err != nil {
		return fmt.Errorf("failed to start node actor: %w", err)
	}

	if err := n.subscribe(BroadcastHelloTopic, jobs.BidRequestTopic); err != nil {
		_ = n.actor.Stop()
		return err
	}

	go n.gcBidState()

	return nil
}

// ExecutorAvailable returns the availability of a specific executor.
func (n *Node) ExecutorAvailable(execType jobs.AllocationExecutor) bool {
	n.rumutex.RLock()
	defer n.rumutex.RUnlock()

	_, ok := n.executors[string(execType)]
	return ok
}

func (n *Node) subscribe(topics ...string) error {
	for _, topic := range topics {
		if err := n.actor.Subscribe(topic, n.setupBroadcast); err != nil {
			return fmt.Errorf("error subscribing to %s: %w", topic, err)
		}
	}

	n.network.SetBroadcastAppScore(n.broadcastScore)
	if err := n.network.Notify(n.actor.Context(), n.peerPreConnected, n.peerConnected, n.peerDisconnected, n.peerIdentified, n.peerIdentified); err != nil {
		return fmt.Errorf("error setting up peer notifications: %w", err)
	}

	return nil
}

func (n *Node) setupBroadcast(topic string) error {
	return n.network.SetupBroadcastTopic(topic, func(t *network.Topic) error {
		return t.SetScoreParams(&pubsub.TopicScoreParams{
			SkipAtomicValidation:           true,
			TopicWeight:                    1.0,
			TimeInMeshWeight:               0.00027, // ~1/3600
			TimeInMeshQuantum:              time.Second,
			TimeInMeshCap:                  1.0,
			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		})
	})
}

func (n *Node) broadcastScore(p peer.ID) float64 {
	n.mx.Lock()
	defer n.mx.Unlock()

	st, ok := n.peers[p]
	if !ok {
		return 0
	}

	if st.helloIn && st.helloOut {
		return 5
	}

	if st.hasRoot {
		return 1
	}

	return 0
}

func (n *Node) peerConnected(p peer.ID) {
	logConns.Debugf("peer connected: %s", p)
	n.mx.Lock()
	defer n.mx.Unlock()

	st, ok := n.peers[p]
	if !ok {
		st = &peerState{}
		n.peers[p] = st
	}

	st.conns++
}

func (n *Node) peerPreConnected(p peer.ID, protos []protocol.ID, conns int) {
	logConns.Debugf("peer preconnected: %s %s (%d)", p, protos, conns)
	n.mx.Lock()
	defer n.mx.Unlock()

	st := &peerState{conns: conns}
	n.peers[p] = st

	if includesRootProtocol(protos) {
		st.hasRoot = true
		st.helloPending = true
		st.helloAttempts = 1
		go n.sayHello(p)
	}
}

func (n *Node) peerIdentified(p peer.ID, protos []protocol.ID) {
	logConns.Debugf("peer identified: %s %s", p, protos)
	n.mx.Lock()
	defer n.mx.Unlock()

	st, ok := n.peers[p]
	if !ok {
		st = &peerState{}
		n.peers[p] = st
	}

	if includesRootProtocol(protos) {
		st.hasRoot = true
		if !st.helloOut && !st.helloPending {
			st.helloPending = true
			st.helloAttempts++
			go n.sayHello(p)
		}
	}
}

func (n *Node) peerDisconnected(p peer.ID) {
	logConns.Debugf("peer disconnected: %s", p)
	n.mx.Lock()
	defer n.mx.Unlock()

	st, ok := n.peers[p]
	if !ok {
		return
	}
	st.conns--

	if st.conns <= 0 {
		delete(n.peers, p)
	}
}

func (n *Node) sayHello(p peer.ID) {
	pubk, err := p.ExtractPublicKey()
	if err != nil {
		log.Debugf("failed to extract public key: %s", err)
		return
	}

	if !crypto.AllowedKey(int(pubk.Type())) {
		log.Debugf("unexpected key type: %d", pubk.Type())
		return
	}

	actorID, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		log.Debugf("failed to extract actor ID: %s", err)
		return
	}

	actorDID := did.FromPublicKey(pubk)
	handle := actor.Handle{
		ID:  actorID,
		DID: actorDID,
		Address: actor.Address{
			HostID:       p.String(),
			InboxAddress: "root",
		},
	}

	wait := helloMinDelay + time.Duration(rand.Int63n(int64(helloMaxDelay-helloMinDelay)))
	time.Sleep(wait)

	n.mx.Lock()
	st, ok := n.peers[p]
	if !ok {
		n.mx.Unlock()
		return
	}

	if !n.network.PeerConnected(p) {
		st.helloPending = false
		n.mx.Unlock()
		return
	}
	n.mx.Unlock()

	msg, err := actor.Message(
		n.actor.Handle(),
		handle,
		PublicHelloBehavior,
		nil,
		actor.WithMessageTimeout(helloTimeout),
	)
	if err != nil {
		log.Debugf("failed to construct hello message: %s", err)
		return
	}

	logConns.Debugf("saying hello to %s", handle.Address.HostID)
	replyCh, err := n.actor.Invoke(msg)
	if err != nil {
		n.mx.Lock()
		if st, ok = n.peers[p]; ok {
			if st.helloAttempts < helloAttempts {
				st.helloAttempts++
				go n.sayHello(p)
			} else {
				st.helloPending = false
			}
		}
		n.mx.Unlock()
		logConns.Debugf("error invoking hello: %s", err)
		return
	}

	select {
	case reply := <-replyCh:
		reply.Discard()
		n.mx.Lock()
		if st, ok = n.peers[p]; ok {
			st.helloOut = true
			st.helloPending = false
		} else if n.network.PeerConnected(p) {
			// race with connected notification
			st = &peerState{helloOut: true}
			n.peers[p] = st
		}
		n.mx.Unlock()
		log.Infof("got hello response from %s", handle.Address.HostID)

	case <-time.After(time.Until(msg.Expiry())):
		n.mx.Lock()
		if st, ok = n.peers[p]; ok {
			if st.helloAttempts < helloAttempts {
				st.helloAttempts++
				go n.sayHello(p)
			} else {
				st.helloPending = false
			}
		}
		n.mx.Unlock()
		logConns.Debugf("hello timeout for %s", handle.Address.HostID)
	}
}

// Stop node
func (n *Node) Stop() error {
	if !atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		return nil
	}

	n.mx.Lock()
	// stop all allocations
	for k, alloc := range n.allocations {
		if err := alloc.Stop(n.ctx); err != nil {
			log.Warnf("error stopping allocation %s: %err", k, err)
		}
	}
	n.mx.Unlock()

	if err := n.saveDeployments(); err != nil {
		log.Errorf("error saving active deployments: %s", err)
	}

	n.cancel()
	// clear the broadcast app score
	n.network.SetBroadcastAppScore(nil)

	// stop the actor
	if err := n.actor.Stop(); err != nil {
		return fmt.Errorf("failed to stop node actor: %w", err)
	}

	return nil
}

// TODO: make send reply a helper func from actor pkg
func (n *Node) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(n.actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		log.Debugf("error creating reply: %s", err)
		return
	}

	if err := n.actor.Send(reply); err != nil {
		log.Debugf("error sending  reply: %s", err)
	}
}

func (n *Node) getExecutor(execType jobs.AllocationExecutor) (executorMetadata, error) {
	n.rumutex.RLock()
	defer n.rumutex.RUnlock()

	e, ok := n.executors[string(execType)]
	if !ok {
		return executorMetadata{}, errors.New("executor not available")
	}

	return e, nil
}

func (n *Node) registerDynamicBehaviors(ensembleID string) error {
	dmsBehaviors := map[string]struct {
		fn   func(actor.Envelope)
		opts []actor.BehaviorOption
	}{
		fmt.Sprintf(jobs.SubnetCreateBehavior.DynamicTemplate, ensembleID): {
			fn: n.handleSubnetCreate,
		},
		fmt.Sprintf(jobs.SubnetDestroyBehavior.DynamicTemplate, ensembleID): {
			fn: n.handleSubnetDestroy,
		},
	}
	for behavior, handler := range dmsBehaviors {
		if err := n.actor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}
	return nil
}

func (n *Node) createAllocations(
	orchestrator did.DID,
	ensembleID string,
	allocations map[string]jobs.AllocationDeploymentConfig,
	supervisor actor.Handle,
) (map[string]actor.Handle, error) {
	allocHandles := make(map[string]actor.Handle, len(allocations))
	for allocationID, config := range allocations {
		if _, ok := n.allocations[allocationID]; ok {
			continue
		}

		allocation, err := n.createAllocation(jobs.Job{
			ID:               ensembleID,
			AllocationID:     allocationID,
			Resources:        config.Resources,
			Execution:        config.Execution,
			ProvisionScripts: config.ProvisionScripts,
		}, supervisor)
		if err != nil {
			return nil, fmt.Errorf("failed to create allocation %s: %w", allocationID, err)
		}

		allocHandles[allocationID] = allocation.Actor.Handle()

		// node grants subnet create/destroy caps to the orchestrator
		if err := n.grantCaps(orchestrator, n.actor.Handle().DID, []ucan.Capability{
			ucan.Capability(fmt.Sprintf(jobs.EnsembleNamespace, ensembleID)),
		}); err != nil {
			return nil, fmt.Errorf("failed to grant node caps: %w", err)
		}

		// allocation grants subnet manage caps to the orchestrator
		allocDID, err := did.FromID(allocation.Actor.Handle().ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get did from id: %w", err)
		}

		if err := n.grantCaps(orchestrator, allocDID, []ucan.Capability{
			ucan.Capability(jobs.AllocationNamespace),
		}); err != nil {
			return nil, fmt.Errorf("failed to grant allocation caps: %w", err)
		}

		// refresh allocation caps grants periodically
		go func() {
			ticker := time.NewTicker(grantAllocationCapsFreq)
			defer ticker.Stop()

			for allocation.Status(context.TODO()).Status != jobs.AllocationStatus("stopped") {
				select {
				case <-n.ctx.Done():
					return
				case <-ticker.C:
					// node grants subnet create/destroy caps to the orchestrator
					if err := n.grantCaps(orchestrator, n.actor.Handle().DID, []ucan.Capability{
						ucan.Capability(fmt.Sprintf(jobs.EnsembleNamespace, ensembleID)),
					}); err != nil {
						log.Warnf("failed to grant node caps: %w", err)
					}

					// allocation grants subnet manage caps to the orchestrator
					if err := n.grantCaps(orchestrator, allocDID, []ucan.Capability{
						ucan.Capability(jobs.AllocationNamespace),
					}); err != nil {
						log.Warnf("failed to grant allocation caps: %w", err)
					}
				}
			}
		}()
	}

	log.Infof("Finished createAllocations for ensembleID: %s", ensembleID)
	return allocHandles, nil
}

// createAllocation creates an allocation
func (n *Node) createAllocation(job jobs.Job, supervisor actor.Handle) (*jobs.Allocation, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair for allocation job %s: %w", job.ID, err)
	}

	allocationInbox, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid for allocation inbox: %w", err)
	}

	allocActor, err := n.createChildActor(priv, allocationInbox.String(), supervisor)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation actor: %w", err)
	}

	n.mx.Lock()
	_, alreadyCommited := n.commitedResources[job.AllocationID]

	if !alreadyCommited {
		n.mx.Unlock()
		return nil, fmt.Errorf("no committed resources for ensemble id: %s", job.AllocationID)
	}

	if err := n.resourceManager.UncommitResources(context.Background(), job.AllocationID); err != nil {
		log.Errorf("failed to uncommit resources for ensemble id: %s: %w", job.ID, err)
	}

	resourceAllocation := types.ResourceAllocation{
		JobID:        job.ID,
		AllocationID: job.AllocationID,
		Resources:    job.Resources,
	}
	err = n.resourceManager.AllocateResources(n.ctx, resourceAllocation)
	if err != nil {
		n.mx.Unlock()
		return nil, fmt.Errorf("failed to allocate resources: %w", err)
	}

	delete(n.commitedResources, job.AllocationID)
	n.mx.Unlock()

	allocation, err := jobs.NewAllocation(
		n.fs,
		n.dmsConfig,
		allocActor,
		jobs.AllocationDetails{Job: job, NodeID: n.hostID},
		n.resourceManager,
		n.network,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation: %w", err)
	}

	err = allocation.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start the allocation: %w", err)
	}

	n.updateAllocations(allocation)

	return allocation, nil
}

func (n *Node) grantCaps(orchestrator did.DID, aud did.DID, caps []ucan.Capability) error {
	tokens, err := n.rootCap.Grant(
		ucan.Delegate,
		orchestrator,
		aud,
		[]string{},
		actor.MakeExpiry(grantAllocationCapsFreq),
		1,
		caps,
	)
	if err != nil {
		return fmt.Errorf("failed to create granting token for audience %s caps: %w", aud, err)
	}

	err = n.rootCap.AddRoots([]did.DID{}, tokens, ucan.TokenList{}, ucan.TokenList{})
	if err != nil {
		return fmt.Errorf("failed to add roots for audience %s: %w", aud, err)
	}

	return nil
}

func (n *Node) updateAllocations(alloc *jobs.Allocation) {
	n.allocmx.Lock()
	n.allocations[alloc.ID] = alloc
	n.allocmx.Unlock()
}

func (n *Node) commitDeployment(ensembleID, allocationID string, resources types.Resources) error {
	n.mx.Lock()
	defer n.mx.Unlock()

	bidState, ok := n.bids[ensembleID]
	if !ok {
		return fmt.Errorf("no bid requests for ensemble id: %s", ensembleID)
	}

	if bidState.expire.Before(time.Now()) {
		return fmt.Errorf("bid request for ensemble id: %s has expired", ensembleID)
	}

	_, alreadyCommited := n.commitedResources[allocationID]
	if alreadyCommited {
		return nil
	}

	if err := n.resourceManager.CommitResources(context.TODO(), types.CommittedResources{
		AllocationID: allocationID,
		Resources:    resources,
	}); err != nil {
		return fmt.Errorf("failed to preallocate resources for ensemble id: %s: %w", allocationID, err)
	}

	n.commitedResources[allocationID] = bidState

	return nil
}

func (n *Node) clearCommitedResources() {
	n.mx.Lock()
	defer n.mx.Unlock()

	for allocationID, v := range n.commitedResources {
		// if allocation not found for this commitment and bid is expired release resources
		_, allocFound := n.allocations[allocationID]
		if !allocFound && time.Now().After(v.expire) {
			if err := n.resourceManager.UncommitResources(context.Background(), allocationID); err != nil {
				log.Errorf("failed to preallocate resources for ensemble id: %s: %w", allocationID, err)
			}
			delete(n.bids, allocationID)
			delete(n.commitedResources, allocationID)
		}
	}
}

// createChildActor creates a child actor using node's limiter, scheduler and network.
func (n *Node) createChildActor(pvkey crypto.PrivKey, inbox string, supervisor actor.Handle) (*actor.BasicActor, error) {
	security, err := actor.NewBasicSecurityContext(pvkey.GetPublic(), pvkey, n.rootCap)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context: %w", err)
	}

	childActor, err := createActor(security, n.actor.Limiter(), n.hostID, inbox, n.network, supervisor)
	if err != nil {
		return nil, fmt.Errorf("failed to create child actor: %w", err)
	}

	return childActor, nil
}

// createActor creates an actor.
func createActor(
	sctx *actor.BasicSecurityContext,
	limiter actor.RateLimiter,
	hostID, inboxAddress string,
	net network.Network,
	supervisor actor.Handle,
) (*actor.BasicActor, error) {
	self := actor.Handle{
		ID:  sctx.ID(),
		DID: sctx.DID(),
		Address: actor.Address{
			HostID:       hostID,
			InboxAddress: inboxAddress,
		},
	}
	actor, err := actor.New(supervisor, net, sctx, limiter, actor.BasicActorParams{}, self)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor: %w", err)
	}

	return actor, nil
}

func includesRootProtocol(protos []protocol.ID) bool {
	for _, proto := range protos {
		if proto == rootProto {
			return true
		}
	}

	return false
}
