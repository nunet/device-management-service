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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	lcrypto "github.com/libp2p/go-libp2p/core/crypto"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	helloMinDelay         = 10 * time.Second
	helloMaxDelay         = 20 * time.Second
	helloTimeout          = 3 * time.Second
	helloAttempts         = 3
	clearCommitsFrequency = 60 * time.Second

	grantAllocationCapsFreq = 1 * time.Hour

	rootProto = "actor/root/messages/0.0.1"

	RestoreDeadlineCommitting   = 1 * time.Minute
	RestoreDeadlineProvisioning = 1 * time.Minute
	RestoreDeadlineRunning      = 5 * time.Minute
	bidStateGCInterval          = time.Minute
)

type peerState struct {
	numConnections                  int
	hasRoot                         bool
	helloIn, helloOut, helloPending bool
	helloAttempts                   int
}

type bidState struct {
	expire  time.Time
	nonce   uint64
	request jobtypes.BidRequest
}

type executorMetadata struct {
	executor      types.Executor
	executionType jobs.AllocationExecutor
}

type PortConfig struct {
	AvailableRangeFrom int
	AvailableRangeTo   int
}

// Node is the structure that holds the node's dependencies.
type Node struct {
	lock    sync.RWMutex
	rootCap ucan.CapabilityContext

	// dms modules
	allocator       Allocator
	actor           actor.Actor
	scheduler       *bt.Scheduler
	network         network.Network
	resourceManager types.ResourceManager
	hardware        types.HardwareManager
	onboarding      *onboarding.Onboarding
	executors       map[string]executorMetadata

	// in-memory state
	hostID       string
	geoIP        types.GeoIPLocator
	hostLocation Geolocation
	peers        map[peer.ID]*peerState
	bids         map[string]*bidState
	answeredBids map[string][]uint64
	running      atomic.Bool

	// db state
	orchestratorRepo repositories.OrchestratorView

	// utils
	orchestratorProvider jobs.OrchestratorProvider
	dmsConfig            config.Config
	fs                   afero.Afero
	ctx                  context.Context
	cancel               func()
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
	newActor, err := actor.New(supervisor, net, sctx, limiter, actor.BasicActorParams{}, self)
	if err != nil {
		return nil, fmt.Errorf("create actor: %w", err)
	}

	return newActor, nil
}

// New creates a new node, attaches an actor to the node.
func New(cfg config.Config, fs afero.Afero,
	onboarding *onboarding.Onboarding,
	rootCap ucan.CapabilityContext,
	hostID string, net network.Network,
	resourceManager types.ResourceManager,
	scheduler *bt.Scheduler,
	hardware types.HardwareManager,
	orchestratorRepo repositories.OrchestratorView,
	geoIP types.GeoIPLocator, hostLocation Geolocation, portConfig PortConfig,
) (*Node, error) {
	if onboarding == nil {
		return nil, errors.New("onboarding is nil")
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
	if geoIP == nil {
		return nil, errors.New("geoIP is nil")
	}

	rootDID := rootCap.DID()
	rootTrust := rootCap.Trust()
	anchor, err := rootTrust.GetAnchor(rootDID)
	if err != nil {
		return nil, fmt.Errorf("get root DID anchor: %w", err)
	}
	pubk := anchor.PublicKey()
	provider, err := rootTrust.GetProvider(rootDID)
	if err != nil {
		return nil, fmt.Errorf("get root DID provider: %w", err)
	}

	privk, err := provider.PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("get root private key: %w", err)
	}

	rootSec, err := actor.NewBasicSecurityContext(pubk, privk, rootCap)
	if err != nil {
		return nil, fmt.Errorf("create security context: %w", err)
	}

	nodeActor, err := createActor(rootSec, actor.NewRateLimiter(actor.DefaultRateLimiterConfig()), hostID, "root", net, actor.Handle{})
	if err != nil {
		return nil, fmt.Errorf("create node actor: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		allocator:            newAllocator(newPortAllocator(portConfig), resourceManager, hardware, net, fs, cfg.WorkDir, hostID),
		hostID:               hostID,
		network:              net,
		bids:                 make(map[string]*bidState),
		answeredBids:         make(map[string][]uint64),
		peers:                make(map[peer.ID]*peerState),
		resourceManager:      resourceManager,
		hardware:             hardware,
		actor:                nodeActor,
		rootCap:              rootCap,
		scheduler:            scheduler,
		onboarding:           onboarding,
		executors:            make(map[string]executorMetadata),
		ctx:                  ctx,
		cancel:               cancel,
		orchestratorRepo:     orchestratorRepo,
		orchestratorProvider: jobs.NewOrchestratorProvider(),
		geoIP:                geoIP,
		hostLocation:         hostLocation,
		dmsConfig:            cfg,
		fs:                   fs,
	}

	if err := n.initSupportedExecutors(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("new executor: %w", err)
	}

	dmsBehaviors := n.getDMSBehaviors()
	for behavior, handler := range dmsBehaviors {
		if err := nodeActor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return nil, fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}

	if err := n.restoreDeployments(); err != nil {
		log.Errorf("restoring deployments: %s", err)
	}

	return n, nil
}

func (n *Node) saveDeployments() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	var failed []string
	for id, o := range n.orchestratorProvider.Orchestrators() {
		if err := n.saveDeployment(o); err != nil {
			log.Errorf("error saving deployment %s: %s", id, err)
			failed = append(failed, id)
		}
	}

	if len(failed) != 0 {
		return fmt.Errorf("save deployments: %v", failed)
	}

	return nil
}

func (n *Node) restoreDeployments() error {
	query := n.orchestratorRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.LTE("Status", jobtypes.DeploymentStatusRunning),
	)

	// TODO: delete old orchestrator views
	orchestratorsViews, err := n.orchestratorRepo.FindAll(n.ctx, query)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("query the database for hanging deployments: %w", err)
	}

	var failedToRestore []string
	for _, d := range orchestratorsViews {
		if d.Status < jobtypes.DeploymentStatusCommitting {
			log.Warnf(
				"deployment %s will not be restaured because it was not previously committed",
				d.OrchestratorID,
			)
			continue
		}

		// Check restore deadline based on deployment status
		// TODO: on the compute provider side, if the deployer stops to answer
		// for more than the restore deadlines, they should free any resources alocated
		// and consider the deployment as canceled
		var restoreDeadline time.Duration
		switch d.Status {
		case jobtypes.DeploymentStatusCommitting:
			restoreDeadline = RestoreDeadlineCommitting
		case jobtypes.DeploymentStatusProvisioning:
			restoreDeadline = RestoreDeadlineProvisioning
		case jobtypes.DeploymentStatusRunning:
			restoreDeadline = RestoreDeadlineRunning
		default:
			log.Warnf("Unknown restorable deployment status for %s, skipping restoration", d.OrchestratorID)
			continue
		}

		if time.Since(d.CreatedAt) > restoreDeadline {
			log.Warnf("Deployment %s has exceeded its restore deadline, skipping restoration", d.OrchestratorID)
			continue
		}

		// recreate actor given priv key
		pvkey, err := lcrypto.UnmarshalPrivateKey(d.PrivKey)
		if err != nil {
			log.Errorf("unmarshall orchestrator's actor priv key: %v", err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		childActor, err := n.createChildActor(pvkey, d.OrchestratorID, d.Manifest.Orchestrator)
		if err != nil {
			log.Errorf("restore orchestrator actor of ensemble %s: %v", d.OrchestratorID, err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		err = childActor.Start()
		if err != nil {
			log.Errorf("start actor: %v", err)
			continue
		}

		orchestrator, err := n.orchestratorProvider.RestoreDeployment(childActor, d.OrchestratorID, d.Cfg, d.Manifest, d.Status, d.DeploymentSnapshot)
		if err != nil {
			log.Errorf("restore orchestrator of id %s; Error: %v", d.OrchestratorID, err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		log.Debugf("deployment %s restored!\n", orchestrator.ID())
	}

	if len(failedToRestore) > 0 {
		return fmt.Errorf("failed to restore the following deployment(s): %v", failedToRestore)
	}

	return nil
}

func (n *Node) getDMSBehaviors() map[string]struct {
	fn   func(actor.Envelope)
	opts []actor.BehaviorOption
} {
	dmsBehaviors := map[string]struct {
		fn   func(actor.Envelope)
		opts []actor.BehaviorOption
	}{
		behaviors.PublicHelloBehavior: {
			fn: n.publicHelloBehavior,
		},
		behaviors.PublicStatusBehavior: {
			fn: n.publicStatusBehavior,
		},
		behaviors.BroadcastHelloBehavior: {
			fn: n.broadcastHelloBehavior,
			opts: []actor.BehaviorOption{
				actor.WithBehaviorTopic(behaviors.BroadcastHelloTopic),
			},
		},
		behaviors.PeersListBehavior: {
			fn: n.handlePeersList,
		},
		behaviors.PeerAddrInfoBehavior: {
			fn: n.handlePeerAddrInfo,
		},
		behaviors.PeerPingBehavior: {
			fn: n.handlePeerPing,
		},
		behaviors.PeerDHTBehavior: {
			fn: n.handlePeerDHT,
		},
		behaviors.PeerConnectBehavior: {
			fn: n.handlePeerConnect,
		},
		behaviors.PeerScoreBehavior: {
			fn: n.handlePeerScore,
		},
		behaviors.OnboardBehavior: {
			fn: n.handleOnboard,
		},
		behaviors.OffboardBehavior: {
			fn: n.handleOffboard,
		},
		behaviors.OnboardStatusBehavior: {
			fn: n.handleOnboardStatus,
		},
		behaviors.NewDeploymentBehavior: {
			fn: n.handleNewDeployment,
		},
		behaviors.DeploymentListBehavior: {
			fn: n.handleDeploymentList,
		},
		behaviors.DeploymentLogsBehavior: {
			fn: n.handleDeploymentLogs,
		},
		behaviors.DeploymentStatusBehavior: {
			fn: n.handleDeploymentStatus,
		},
		behaviors.DeploymentManifestBehavior: {
			fn: n.handleDeploymentManifest,
		},
		behaviors.DeploymentShutdownBehavior: {
			fn: n.handleDeploymentShutdown,
		},
		behaviors.VerifyEdgeConstraintBehavior: {
			fn: n.handleVerifyEdgeConstraint,
		},
		behaviors.BidRequestBehavior: {
			fn: n.handleBidRequest,
			opts: []actor.BehaviorOption{
				actor.WithBehaviorTopic(behaviors.BidRequestTopic),
			},
		},
		behaviors.RevertDeploymentBehavior: {
			fn: n.handleRevertDeployment,
		},
		behaviors.SubnetCreateBehavior.Static: {
			fn: n.handleSubnetCreate,
		},
		behaviors.SubnetDestroyBehavior.Static: {
			fn: n.handleSubnetDestroy,
		},
		behaviors.ResourcesAllocatedBehavior: {
			fn: n.handleAllocatedResources,
		},
		behaviors.ResourcesFreeBehavior: {
			fn: n.handleFreeResources,
		},
		behaviors.ResourcesOnboardedBehavior: {
			fn: n.handleOnboardedResources,
		},
		behaviors.LoggerConfigBehavior: {
			fn: n.handleLoggerConfig,
		},
		behaviors.HardwareUsageBehavior: {
			fn: n.handleHardwareUsage,
		},
		behaviors.CapListBehavior: {
			fn: n.handleCapList,
		},
		behaviors.CapAnchorBehavior: {
			fn: n.handleCapAnchor,
		},
		behaviors.AllocationDeploymentBehavior: {
			fn: n.handleAllocationDeployment,
		},
		behaviors.CommitDeploymentBehavior: {
			fn: n.handleCommitDeployment,
		},
	}

	return dmsBehaviors
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

	n.lock.Lock()
	defer n.lock.Unlock()

	for k, bs := range n.bids {
		if bs.expire.Before(now) {
			delete(n.bids, k)
			delete(n.answeredBids, k)
		}
	}
}

// Start node
func (n *Node) Start() error {
	if err := n.allocator.Run(); err != nil {
		return fmt.Errorf("start node allocator: %w", err)
	}

	if err := n.actor.Start(); err != nil {
		return fmt.Errorf("start node actor: %w", err)
	}

	if err := n.subscribe(behaviors.BroadcastHelloTopic, behaviors.BidRequestTopic); err != nil {
		_ = n.actor.Stop()
		return err
	}

	n.running.Store(true)
	go n.gcBidState()
	return nil
}

// Stop node
func (n *Node) Stop() error {
	if err := n.allocator.Stop(context.Background()); err != nil {
		log.Errorf("stop node allocator: %s", err)
	}

	if err := n.saveDeployments(); err != nil {
		log.Errorf("saving active deployments: %s", err)
	}

	n.cancel()
	// clear the broadcast app score
	n.network.SetBroadcastAppScore(nil)

	// stop the actor
	if err := n.actor.Stop(); err != nil {
		return fmt.Errorf("stop node actor: %w", err)
	}

	n.running.Store(false)
	return nil
}

func createEnsembleID(peerID string) (string, error) {
	var id string

	suffixID, err := uuid.NewUUID()
	if err != nil {
		return id, fmt.Errorf("failed to generate uuid for allocation inbox: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(peerID + suffixID.String()))

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (n *Node) createOrchestrator(ctx context.Context,
	ensemble jobtypes.EnsembleConfig,
	actr actor.Actor,
) (jobs.OrchestratorAPI, error) {
	ensembleID, err := createEnsembleID(actr.Handle().Address.HostID)
	if err != nil {
		return nil, fmt.Errorf("generate uuid for ensemble: %w", err)
	}

	childActor, err := actr.CreateChild(actr.Handle(), actor.BasicActorParams{})
	if err != nil {
		return nil, fmt.Errorf("create child actor: %w", err)
	}

	orchestrator, err := n.orchestratorProvider.NewOrchestrator(ctx, ensembleID, childActor, ensemble)
	if err != nil {
		return nil, fmt.Errorf("new orchestrator: %w", err)
	}

	return orchestrator, nil
}

// TODO: make send reply a helper func from actor pkg
func (n *Node) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(n.actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		log.Debugf("creating reply: %s", err)
		return
	}

	if err := n.actor.Send(reply); err != nil {
		log.Debugf("sending reply: %s", err)
	}
}

// createChildActor creates a child actor using node's limiter, scheduler and network.
func (n *Node) createChildActor(priv crypto.PrivKey, inbox string, supervisor actor.Handle) (*actor.BasicActor, error) {
	security, err := actor.NewBasicSecurityContext(priv.GetPublic(), priv, n.rootCap)
	if err != nil {
		return nil, fmt.Errorf("create security context: %w", err)
	}

	childActor, err := createActor(security, n.actor.Limiter(), n.hostID, inbox, n.network, supervisor)
	if err != nil {
		return nil, fmt.Errorf("create child actor: %w", err)
	}

	return childActor, nil
}

// ...
// TODO: do we wanna maintain the below code that is only used for e2e tests? Could there be a better way to do this?
// ...

func (n *Node) ResourceManager() types.ResourceManager {
	return n.resourceManager
}

func (n *Node) Allocator() Allocator {
	return n.allocator
}

// GetBidRequests returns the bid requests for the node.
func (n *Node) GetBidRequests() []jobs.BidRequest {
	n.lock.Lock()
	defer n.lock.Unlock()

	reqs := make([]jobs.BidRequest, 0, len(n.bids))
	for _, v := range n.bids {
		reqs = append(reqs, v.request)
	}

	return reqs
}
