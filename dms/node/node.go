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
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/tokenomics"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	helloMinDelay = 10 * time.Second
	helloMaxDelay = 20 * time.Second
	helloTimeout  = 3 * time.Second
	helloAttempts = 3

	clearCommitsFrequency    = 60 * time.Second
	ensembleMonitorFrequency = 10 * time.Second
	grantAllocationCapsFreq  = 1 * time.Hour

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
	hostLocation geolocation.Geolocation
	peers        map[peer.ID]*peerState
	bids         map[string]*bidState
	answeredBids map[string][]uint64
	running      atomic.Bool

	// db state
	orchestratorRepo repositories.GenericRepository[jobtypes.OrchestratorView]

	// volume controller
	volumeController controller.GlusterControllerInterface
	volumeOwners     map[string]string // mapping volume name with did

	// utils
	orchestratorRegistry orchestrator.Registry
	dmsConfig            config.Config
	fs                   afero.Afero
	ctx                  context.Context
	cancel               func()

	// contract store
	contractStore  store.Store
	contractActors []*tokenomics.ContractActor
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
	orchestratorRepo repositories.GenericRepository[jobtypes.OrchestratorView],
	geoIP types.GeoIPLocator, hostLocation geolocation.Geolocation,
	portConfig PortConfig, vt *storage.VolumeTracker,
	volumeController controller.GlusterControllerInterface,
	contractStore *store.Store,
) (*Node, error) {
	if onboarding == nil {
		return nil, errors.New("onboarding is nil")
	}
	if rootCap == nil {
		return nil, errors.New("root capability context is nil")
	}
	if hostID == "" {
		return nil, errors.New("hostID is empty")
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
	if contractStore == nil {
		return nil, errors.New("contract store is nil")
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
		allocator:            newAllocator(vt, newPortAllocator(portConfig), resourceManager, hardware, net, fs, cfg.WorkDir, hostID),
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
		orchestratorRegistry: orchestrator.NewRegistry(),
		geoIP:                geoIP,
		hostLocation:         hostLocation,
		dmsConfig:            cfg,
		fs:                   fs,
		volumeController:     volumeController,
		volumeOwners:         make(map[string]string),
		contractStore:        *contractStore,
		contractActors:       make([]*tokenomics.ContractActor, 0),
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
		log.Errorw("restoring deployments failed",
			"labels", string(observability.LabelNode),
			"error", err)
	}

	return n, nil
}

func (n *Node) saveDeployments() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	var failed []string
	for id, o := range n.orchestratorRegistry.Orchestrators() {
		if err := n.saveDeployment(o); err != nil {
			log.Errorw("error saving active deployment",
				"labels", string(observability.LabelDeployment),
				"deploymentID", id,
				"error", err)
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
			log.Warnf("deployment %s was not previously committed; ignoring restoration", d.OrchestratorID)
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
			log.Warnf("deployment %s has unknown restorable status %d; ignoring", d.OrchestratorID, d.Status)
			continue
		}

		if time.Since(d.CreatedAt) > restoreDeadline {
			log.Warnf("deployment %s exceeded restore deadline; skipping", d.OrchestratorID)
			continue
		}

		// recreate actor given priv key
		pvkey, err := lcrypto.UnmarshalPrivateKey(d.PrivKey)
		if err != nil {
			log.Errorf("unmarshal orchestrator actor private key for %s: %v", d.OrchestratorID, err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		childActor, err := n.actor.CreateChild(
			d.OrchestratorID,
			d.Manifest.Orchestrator,
			actor.WithPrivKey(pvkey),
		)
		if err != nil {
			log.Errorf("restore actor creation error for %s: %v", d.OrchestratorID, err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		if err := childActor.Start(); err != nil {
			log.Errorf("start orchestrator actor for %s: %v", d.OrchestratorID, err)
			continue
		}

		orchestrator, err := n.orchestratorRegistry.RestoreDeployment(childActor, d.OrchestratorID, d.Cfg, d.Manifest, d.Status, d.DeploymentSnapshot)
		if err != nil {
			log.Errorf("restore orchestrator %s: %v", d.OrchestratorID, err)
			failedToRestore = append(failedToRestore, d.OrchestratorID)
			continue
		}

		log.Infow("restored deployment",
			"labels", string(observability.LabelDeployment),
			"deploymentID", orchestrator.ID())
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
			fn: n.handleBroadcastHelloBehavior,
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
		behaviors.DeploymentUpdateBehavior: {
			fn: n.handleDeploymentUpdate,
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
		behaviors.AllocationsListBehavior: {
			fn: n.handleAllocationsList,
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
		behaviors.DeploymentRevertBehavior: {
			fn: n.handleDeploymentRevert,
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
		behaviors.HardwareSpecBehavior: {
			fn: n.handleHardwareSpec,
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
		behaviors.VolumeCreateBehavior: {
			fn: n.handleCreateVolume,
		},
		behaviors.VolumeDeleteBehavior: {
			fn: n.handleDeleteVolume,
		},
		behaviors.VolumeStartBehavior: {
			fn: n.handleStartVolume,
		},
		behaviors.StatusDiscoveryBehavior: {
			fn: n.handleStatusDiscoveryBehavior,
		},
		behaviors.BroadcastStatusDiscoveryBehavior: {
			fn: n.handleStatusDiscoveryBehavior,
			opts: []actor.BehaviorOption{
				actor.WithBehaviorTopic(behaviors.BroadcastStatusDiscoveryTopic),
			},
		},
		// solution enabler
		behaviors.ContractCreateBehavior: {
			fn: n.handleNewContract,
		},
		// listerner by service provider and compute provider
		behaviors.ContractProposeBehavior: {
			fn: n.handleContractPropose,
		},
		// used by compute provider to accpet a contract
		behaviors.ContractApproveLocalBehavior: {
			fn: n.handleContractApprovalLocal,
		},
		// used by compute provider to list incoming contracts
		behaviors.ContractListIncomingBehavior: {
			fn: n.handleListIncomingContracts,
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

func (n *Node) geolocate() {
	log.Infow("geolocation_initiated",
		"labels", string(observability.LabelNode),
	)

	ip, err := n.network.HostPublicIP()
	if err != nil {
		log.Errorw("failed to get host public IP: %v", err)
		return
	}

	if ip == nil {
		log.Errorw("host public IP is nil")
		return
	}

	location, err := geolocation.Geolocate(ip, n.geoIP)
	if err != nil {
		log.Errorw("failed to geolocate host: %v", err)
		return
	}

	n.lock.Lock()
	n.hostLocation = geolocation.Geolocation{
		Continent: location.Continent,
		Country:   location.Country,
		City:      location.City,
	}
	n.lock.Unlock()

	log.Infow("geolocation_successful",
		"labels", string(observability.LabelNode),
		"continent", location.Continent,
		"country", location.Country,
		"city", location.City,
	)
}

// Start node
func (n *Node) Start() error {
	log.Infow("node_start_initiated",
		"labels", string(observability.LabelNode))

	if err := n.allocator.Run(); err != nil {
		return fmt.Errorf("start node allocator: %w", err)
	}

	if err := n.actor.Start(); err != nil {
		return fmt.Errorf("start node actor: %w", err)
	}

	if err := n.subscribe(
		behaviors.BroadcastHelloTopic,
		behaviors.BidRequestTopic,
		behaviors.BroadcastStatusDiscoveryTopic,
	); err != nil {
		_ = n.actor.Stop()
		return err
	}

	n.running.Store(true)
	go n.gcBidState()
	go n.geolocate()

	log.Infow("node_started_successfully",
		"labels", string(observability.LabelNode))
	return nil
}

// Stop node
func (n *Node) Stop() error {
	log.Infow("node_stop_initiated",
		"labels", string(observability.LabelNode))

	if err := n.allocator.Stop(context.Background()); err != nil {
		log.Errorf("stopping node allocator: %s", err)
	}

	if err := n.saveDeployments(); err != nil {
		log.Errorw("error saving active deployments during node stop",
			"labels", string(observability.LabelDeployment),
			"error", err)
	}

	n.cancel()
	// clear the broadcast app score
	n.network.SetBroadcastAppScore(nil)

	// stop the actor
	if err := n.actor.Stop(); err != nil {
		return fmt.Errorf("stop node actor: %w", err)
	}

	n.running.Store(false)

	log.Infow("node_stopped_successfully",
		"labels", string(observability.LabelNode))
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
) (orchestrator.Orchestrator, error) {
	if ensemble.V1 == nil {
		return nil, fmt.Errorf("empty ensemble config")
	}

	ensembleID, err := createEnsembleID(n.actor.Handle().Address.HostID)
	if err != nil {
		return nil, fmt.Errorf("generate ensemble id: %w", err)
	}

	childActor, err := n.actor.CreateChild(ensembleID, n.actor.Handle())
	if err != nil {
		return nil, fmt.Errorf("create child actor: %w", err)
	}

	log.Infow("deploying ensemble",
		"labels", string(observability.LabelDeployment),
		"ensembleID", ensembleID)

	err = childActor.Start()
	if err != nil {
		return nil, fmt.Errorf("start child actor: %w", err)
	}

	orchestrator, err := n.orchestratorRegistry.NewOrchestrator(
		ctx, n.fs, n.dmsConfig.WorkDir,
		ensembleID, childActor, ensemble,
	)
	if err != nil {
		return nil, fmt.Errorf("new orchestrator: %w", err)
	}

	// if orchestrator needs to join subnet, add the subnet behaviors under ensemble namespace
	// and grant the caps
	if ensemble.Subnet().Join {
		dmsBehaviors := map[string]struct {
			fn   func(actor.Envelope)
			opts []actor.BehaviorOption
		}{
			fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID): {
				fn: n.handleSubnetCreate,
			},
			fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID): {
				fn: n.handleSubnetDestroy,
			},
			fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID): {
				fn: n.handleSubnetJoin,
			},
		}
		for behavior, handler := range dmsBehaviors {
			if err := n.actor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
				return nil, fmt.Errorf("adding %s behavior for orch to join subnet: %w", behavior, err)
			}
		}

		err := n.actor.Security().Grant(childActor.Handle().DID, n.actor.Handle().DID, []ucan.Capability{
			ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
		}, time.Hour)
		if err != nil {
			return nil, fmt.Errorf("granting subnet caps to self orchestrator: %w", err)
		}
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

func (n *Node) addContractActor(a *tokenomics.ContractActor) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.contractActors = append(n.contractActors, a)
}

func (n *Node) invokeBehaviour(destination actor.Handle, behavior string, payload any, timeout time.Duration) (actor.Envelope, error) {
	msg, err := actor.Message(
		n.actor.Handle(),
		destination,
		behavior,
		payload,
		actor.WithMessageExpiry(actor.MakeExpiry(timeout)),
	)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("failed to create contract actor message: %w", err)
	}

	replyCh, err := n.actor.Invoke(msg)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("failed to invoke message: %w", err)
	}

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		return reply, nil

	case <-ticker.C:
		return actor.Envelope{}, errors.New("failed to receive reply due to timeout")
	}
}
