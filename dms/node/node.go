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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	lcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"
	gatewastore "gitlab.com/nunet/device-management-service/gateway/store"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/gateway/provider"
	"gitlab.com/nunet/device-management-service/internal"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/tokenomics"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts/processors"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/payment"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
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

	// TODO: We should consider a restoration deadline down the line (see code at restoreDeployments)
	// RestoreDeadlineCommitting   = 1 * time.Minute
	// RestoreDeadlineProvisioning = 1 * time.Minute
	// RestoreDeadlineRunning      = 5 * time.Minute
	bidStateGCInterval             = time.Minute
	provisionedResourcesGCInterval = time.Minute

	// contract event handler config
	eventHandlerWorkers   = 2
	eventHandlerQueueSize = 200
	eventHandlerBaseDelay = 5 * time.Second
	eventHandlerMaxDelay  = 15 * time.Second
)

// TODO issue #1154 - better handle transient allocations
// temporary subnet status handling - 1 = active , 0 = destroyed
var (
	subnetStatusMx sync.Mutex
	subnetStatus   map[string]int
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
	publicIP     net.IP
	peers        map[peer.ID]*peerState
	bids         map[string]*bidState
	answeredBids map[string][]uint64
	running      atomic.Bool

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
	contractStore    *store.Store
	paymentStore     *payment.Store
	usageStore       *usage.Store
	contractActors   map[string]*tokenomics.ContractActor // DID URI -> Actor (optimized lookup)
	billingScheduler *tokenomics.ContractBillingScheduler
	transactionStore *transaction.Store

	// contract event handler
	contractEventHandler *eventhandler.EventHandler

	// serverProviderRegistry registory
	serverProviderRegistry *provider.Registry

	// payment processor
	paymentProcessor PaymentProcessor
	gatewayStore     *gatewastore.Store
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
	geoIP types.GeoIPLocator, hostLocation geolocation.Geolocation,
	portConfig PortConfig, vt *storage.VolumeTracker,
	volumeController controller.GlusterControllerInterface,
	contractStore *store.Store,
	paymentStore *payment.Store,
	usageStore *usage.Store,
	transactionStore *transaction.Store,
	deploymentStore orchestrator.DeploymentStore,
	providerRegistry *provider.Registry,
	gatewayStore *gatewastore.Store,
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
	if paymentStore == nil {
		return nil, errors.New("payment store is nil")
	}
	if usageStore == nil {
		return nil, errors.New("usage store is nil")
	}
	if transactionStore == nil {
		return nil, errors.New("transaction store is nil")
	}
	if deploymentStore == nil {
		return nil, errors.New("deployment store is nil")
	}

	subnetStatus = make(map[string]int)

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

	allocator := newAllocator(vt, newPortAllocator(portConfig), resourceManager, hardware, net, fs, cfg.WorkDir, hostID)
	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		allocator:              allocator,
		hostID:                 hostID,
		network:                net,
		bids:                   make(map[string]*bidState),
		answeredBids:           make(map[string][]uint64),
		peers:                  make(map[peer.ID]*peerState),
		resourceManager:        resourceManager,
		hardware:               hardware,
		actor:                  nodeActor,
		rootCap:                rootCap,
		scheduler:              scheduler,
		onboarding:             onboarding,
		executors:              make(map[string]executorMetadata),
		ctx:                    ctx,
		cancel:                 cancel,
		orchestratorRegistry:   orchestrator.NewRegistry(deploymentStore),
		geoIP:                  geoIP,
		hostLocation:           hostLocation,
		dmsConfig:              cfg,
		fs:                     fs,
		volumeController:       volumeController,
		volumeOwners:           make(map[string]string),
		contractStore:          contractStore,
		paymentStore:           paymentStore,
		usageStore:             usageStore,
		transactionStore:       transactionStore,
		contractActors:         make(map[string]*tokenomics.ContractActor),
		serverProviderRegistry: providerRegistry,
		gatewayStore:           gatewayStore,
	}

	// Create payment processor with invokeBehaviour function
	invokeBehaviourFunc := func(destination actor.Handle, behavior string, req interface{}, timeout time.Duration) (actor.Envelope, error) {
		msg, err := actor.Message(
			nodeActor.Handle(),
			destination,
			behavior,
			req,
			actor.WithMessageExpiry(actor.MakeExpiry(timeout)),
		)
		if err != nil {
			return actor.Envelope{}, fmt.Errorf("failed to create message: %w", err)
		}

		replyCh, err := nodeActor.Invoke(msg)
		if err != nil {
			return actor.Envelope{}, fmt.Errorf("failed to invoke message: %w", err)
		}

		ticker := time.NewTicker(timeout)
		defer ticker.Stop()

		select {
		case reply := <-replyCh:
			defer reply.Discard()
			return reply, nil
		case <-ticker.C:
			return actor.Envelope{}, errors.New("failed to receive reply due to timeout")
		}
	}

	n.paymentProcessor = NewPaymentProcessor(paymentStore, net, nodeActor.Handle(), invokeBehaviourFunc)

	// set up the flight recorder
	observability.FlightrecInit()

	// setup contract event handler
	n.contractEventHandler = eventhandler.New(ctx, eventHandlerWorkers, eventHandlerQueueSize, eventHandlerBaseDelay, eventHandlerMaxDelay, n.handleContractEvents)

	// Initialize payment model processors
	// This must be called before using GetPaymentModelProcessor
	processors.InitPaymentModelProcessors(usageStore)

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

	// Create billing scheduler with billing function
	// Note: billingFunc closure references n, so it must be created after n
	billingFunc := func(contractDID did.DID) error {
		return n.executeBillingForContract(contractDID)
	}

	billingScheduler, err := tokenomics.NewContractBillingScheduler(
		contractStore,
		usageStore,
		billingFunc,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create billing scheduler: %w", err)
	}

	// Set billing scheduler on node
	n.billingScheduler = billingScheduler

	// NOTE: Do NOT start billing scheduler here
	// It will be started in Node.Start() method

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
	// Get all deployments from store (source of truth)
	allDeployments, err := n.orchestratorRegistry.GetAllDeployments()
	if err != nil {
		return fmt.Errorf("failed to get all deployments: %w", err)
	}

	var failedToRestore []string
	for _, d := range allDeployments {
		// Only restore deployments in restorable states
		if !isRestorableStatus(d.Status) {
			log.Infof("deployment %s has non-restorable status %s skipping", d.OrchestratorID, d.Status.String())
			continue
		}

		// Check if deployment is still valid (not based on time)
		if !isDeploymentStillValid(d) {
			log.Warnf("deployment %s is no longer valid; skipping", d.OrchestratorID)
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
			n.actor.Handle(),
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

		if d.Manifest.Subnet.Join {
			if err := n.addOrchestratorBehaviors(childActor, d.OrchestratorID); err != nil {
				return fmt.Errorf("adding behaviors for orch to join subnet: %w", err)
			}
		}

		orchestrator, err := n.
			orchestratorRegistry.
			RestoreDeployment(
				n.ctx,
				n.fs,
				childActor,
				d.OrchestratorID,
				d.Cfg,
				d.Manifest,
				d.Status,
				d.DeploymentSnapshot,
				d.SubnetManifest,
				types.NewDefaultAllocationIDGenerator(),
			)
		if err != nil {
			log.Errorf("restoring deployment %s failed: %v", d.OrchestratorID, err)
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
		behaviors.DebugFlightrecBehavior: {
			fn: n.handleFlightrec,
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
		behaviors.DeploymentPruneBehavior: {
			fn: n.handleDeploymentPrune,
		},
		behaviors.DeploymentDeleteBehavior: {
			fn: n.handleDeploymentDelete,
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
		behaviors.ContractUsagesCalculateBehavior: {
			fn: n.handleContractUsagesCalculate,
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
		behaviors.ContractListBehavior: {
			fn: n.handleListIncomingContracts,
		},

		// used by payment validator
		behaviors.ContractUsageBehavior: {
			fn: n.handleIncomingContractUsage,
		},

		// used by SP and CP
		behaviors.ContractTransactionBehavior: {
			fn: n.handleIncomingTransaction,
		},
		behaviors.ContractPaymentStatusBehavior: {
			fn: n.handlePaymentStatus,
		},
		// used by payment validator to validate payment
		behaviors.ContractPaymentValidationRequestBehavior: {
			fn: n.handleContractPaymentValidationRequestFromContractHost,
		},
		behaviors.ContractListLocalTransactionsBehavior: {
			fn: n.handleListLocalTransactions,
		},

		behaviors.ContractConfirmLocalTransactionBehavior: {
			fn: n.handleConfirmLocalTransaction,
		},

		// gateway
		behaviors.PromiseBidToBidBehavior: {
			fn: n.handlePromiseBid,
		},

		// provisioned server
		behaviors.PromiseBidSigningBehavior: {
			fn: n.handleBidSigning,
		},
	}

	return dmsBehaviors
}

func (n *Node) addOrchestratorBehaviors(actr actor.Actor, ensembleID string) error {
	orchBehaviors := map[string]struct {
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

	for behavior, handler := range orchBehaviors {
		if err := n.actor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}
	err := n.actor.Security().Grant(actr.Handle().DID, n.actor.Handle().DID, []ucan.Capability{
		ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
	}, time.Hour)
	if err != nil {
		return fmt.Errorf("granting subnet caps to self orchestrator: %w", err)
	}
	return nil
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

func (n *Node) shutdownUnusedProvisionedResources() {
	ticker := time.NewTicker(provisionedResourcesGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.deleteProvionedResource()

		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) deleteProvionedResource() {
	all, err := n.gatewayStore.All()
	if err != nil {
		log.Errorf("failed to get provisioned resources from store: %v", err)
		return
	}

	for _, v := range all {
		destination, err := actor.HandleFromPeerID(v.ProvisionedVMPeerID)
		if err != nil {
			log.Errorf("failed to get handle of provisioned dms")
			continue
		}
		envelope, err := n.invokeBehaviour(destination, behaviors.AllocationsListBehavior, nil, invokeMessageTimeout)
		if envelope.Message == nil || err != nil {
			log.Errorf("failed to get allocation list from new dms: %v", err)
			continue
		}

		var allocs AllocationsListResponse
		err = json.Unmarshal(envelope.Message, &allocs)
		if err != nil {
			log.Errorf("failed to unmarshal allocation list from new dms: %v", err)
			continue
		}

		killVM := true
		for _, alloc := range allocs.Allocations {
			if alloc.Status == "pending" || alloc.Status == "running" || alloc.Status == "stopped" {
				killVM = false
				break
			}
		}

		if killVM {
			serverProvider, err := n.serverProviderRegistry.Get(v.ProviderName)
			if err != nil {
				log.Errorf("failed to get server provider for deleting unused resource: %v", err)
				continue
			}

			err = serverProvider.DeleteServer(context.Background(), v.Resource.ID)
			if err != nil {
				log.Errorf("failed to delete provisioned resource: %v", err)
				continue
			}

			err = n.gatewayStore.Delete(v.Resource.ID)
			if err != nil {
				log.Errorf("failed to delete provisioned resource from local store: %v", err)
			}
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

	n.lock.Lock()
	n.publicIP = ip
	n.lock.Unlock()

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

	go func() {
		if err := n.restoreDeployments(); err != nil {
			log.Errorw("restoring deployments failed",
				"labels", string(observability.LabelNode),
				"error", err)
		}
	}()

	// Start billing scheduler
	if n.billingScheduler != nil {
		n.billingScheduler.Start()
		log.Infow("billing scheduler started",
			"labels", string(observability.LabelNode))
	}

	n.running.Store(true)
	go n.gcBidState()
	go n.geolocate()
	if n.dmsConfig.General.ComputeGateway {
		go n.shutdownUnusedProvisionedResources()
	}

	log.Infow("node_started_successfully",
		"labels", string(observability.LabelNode))
	return nil
}

// Stop node
func (n *Node) Stop() error {
	log.Infow("node_stop_initiated",
		"labels", string(observability.LabelNode))

	// Stop billing scheduler first (before stopping other components)
	if n.billingScheduler != nil {
		n.billingScheduler.Stop()
		log.Infow("billing scheduler stopped",
			"labels", string(observability.LabelNode))
	}

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

// ListenForCapabilityContextsUpdates reloads all capability contexts from disk
func (n *Node) ListenForCapabilityContextsUpdates() error {
	for {
		select {
		case <-internal.ReloadChan:
			log.Infow("Received SIGUSR1, reloading capability contexts...")

			// Reload the capability context while holding the lock
			capCtx, err := func() (ucan.CapabilityContext, error) {
				n.lock.Lock()
				defer n.lock.Unlock()

				// Reload the DMS capability context
				// Note: Capability contexts are stored in UserDir, not WorkDir
				capCtx, err := LoadCapabilityContext(n.rootCap.Trust(), n.fs, n.rootCap.Name(), n.dmsConfig.General.UserDir)
				if err != nil {
					return nil, fmt.Errorf("failed to reload DMS capability context: %w", err)
				}

				n.rootCap = capCtx
				return capCtx, nil
			}()
			if err != nil {
				log.Errorw("Failed to reload capability context", "error", err)
				continue
			}

			// Create a new security context from the updated rootCap (same logic as in New())
			rootDID := capCtx.DID()
			rootTrust := capCtx.Trust()
			anchor, err := rootTrust.GetAnchor(rootDID)
			if err != nil {
				log.Errorw("Failed to get root DID anchor", "error", err)
				continue
			}
			pubk := anchor.PublicKey()
			provider, err := rootTrust.GetProvider(rootDID)
			if err != nil {
				log.Errorw("Failed to get root DID provider", "error", err)
				continue
			}
			privk, err := provider.PrivateKey()
			if err != nil {
				log.Errorw("Failed to get root private key", "error", err)
				continue
			}

			newSecurity, err := actor.NewBasicSecurityContext(pubk, privk, capCtx)
			if err != nil {
				log.Errorw("Failed to create new security context", "error", err)
				continue
			}

			// Update the actor's security context
			if err := n.actor.UpdateSecurityContext(newSecurity); err != nil {
				log.Errorw("Failed to update actor security context", "error", err)
				continue
			}
			log.Infow("Capability contexts reloaded successfully from disk")
		case <-n.ctx.Done():
			log.Infow("Node context done, stopping reload loop")
			return nil
		}
	}
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

func (n *Node) createOrchestrator(
	ctx context.Context,
	ensemble jobtypes.EnsembleConfig,
	contracts map[string]types.ContractConfig,
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

	orch, err := n.orchestratorRegistry.NewOrchestrator(
		ctx, n.fs, n.dmsConfig.WorkDir,
		ensembleID, childActor, ensemble,
		types.NewDefaultNodeIDGenerator(),
		types.NewDefaultAllocationIDGenerator(),
		n.contractEventHandler,
		contracts,
	)
	if err != nil {
		return nil, fmt.Errorf("new orchestrator: %w", err)
	}

	// if orchestrator needs to join subnet, add the subnet behaviors under ensemble namespace
	// and grant the caps
	if ensemble.Subnet().Join {
		if err := n.addOrchestratorBehaviors(childActor, ensembleID); err != nil {
			return nil, fmt.Errorf("adding behaviors for orch to join subnet: %w", err)
		}
	}

	return orch, nil
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

	n.contractActors[a.ContractDID.URI] = a
}

// executeBillingForContract executes billing for a contract
// This is called by the scheduler
func (n *Node) executeBillingForContract(contractDID did.DID) error {
	// Get contract to verify it exists
	contract, err := n.contractStore.GetContract(contractDID.URI)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	// Find the contract actor using map lookup (O(1))
	contractActor, exists := n.contractActors[contractDID.URI]
	if !exists {
		return fmt.Errorf("contract actor not found for %s", contractDID.URI)
	}

	// Check if contract should stop billing
	if contract.CurrentState == contracts.ContractTerminated ||
		contract.CurrentState == contracts.ContractCompleted {
		// For FixedRental and Periodic payment models, generate pro-rated final invoice before unregistering
		if contract.PaymentDetails.PaymentModel == contracts.FixedRental ||
			contract.PaymentDetails.PaymentModel == contracts.Periodic {
			// Execute billing to generate pro-rated final invoice for terminated contract
			// This will call handleTerminatedContractInvoice() which generates the pro-rated invoice
			err = contractActor.CheckAndGenerateInvoice()
			if err != nil {
				// Log error but still unregister - the invoice generation may have failed
				// but we don't want to keep retrying for a terminated contract
				log.Warnw("failed to generate final invoice for terminated contract",
					"labels", string(observability.LabelContract),
					"contract_did", contractDID.URI,
					"payment_model", contract.PaymentDetails.PaymentModel,
					"error", err)
			}
		}
		// Unregister from scheduler after attempting to generate final invoice
		n.billingScheduler.UnregisterContract(contractDID)
		return fmt.Errorf("contract %s is %s, stopping billing", contractDID.URI, contract.CurrentState)
	}

	// Execute billing
	err = contractActor.CheckAndGenerateInvoice()
	if err != nil {
		return fmt.Errorf("billing execution failed: %w", err)
	}

	// Update billing schedule after successful invoice
	// This updates the trigger's lastInvoiceAt to use actual invoice time
	if err = n.billingScheduler.UpdateContract(contractDID); err != nil {
		return fmt.Errorf("failed to update billing schedule: %w", err)
	}

	return nil
}

func (n *Node) collectUsagesAndForwardToPaymentProviders(req contracts.CollectUsagesAndForwardToPaymentProvidersRequest) contracts.CollectUsagesAndForwardToPaymentProvidersReponse {
	resp := contracts.CollectUsagesAndForwardToPaymentProvidersReponse{
		Results: make([]contracts.ContractUsageResult, 0),
	}
	now := time.Now()

	// Get contracts to process
	var contractsToProcess []*contracts.Contract
	var err error

	if req.ContractDID != "" {
		// Process specific contract
		contract, err := n.contractStore.GetContract(req.ContractDID)
		if err != nil {
			resp.Error = fmt.Sprintf("failed to get contract %s: %v", req.ContractDID, err)
			return resp
		}
		contractsToProcess = []*contracts.Contract{contract}
	} else {
		// Process all contracts
		contractsToProcess, err = n.contractStore.GetAllContracts()
		if err != nil {
			resp.Error = fmt.Sprintf("failed to get contracts: %v", err)
			return resp
		}
	}

	type paymentForwardToProviderRequest struct {
		Usages              int                                 // For backward compatibility
		TimeUtilization     *contracts.TimeUtilizationUsage     // For pay_per_time_utilization
		ResourceUtilization *contracts.ResourceUtilizationUsage // For pay_per_resource_utilization
		FixedRentalDetails  *contracts.FixedRentalUsage         // For fixed_rental
		Contract            contracts.Contract
	}

	allPayments := make([]paymentForwardToProviderRequest, 0)

	// Collect usage for each contract based on its payment model
	for _, contract := range contractsToProcess {
		// Get contract-specific last processed timestamp
		lastProcessedAt, _ := n.usageStore.GetLastProcessedAt(contract.ContractDID)

		// Get processor for this payment model
		processor, err := contracts.GetPaymentModelProcessor(contract.PaymentDetails.PaymentModel)
		if err != nil {
			result := contracts.ContractUsageResult{
				ContractDID:  contract.ContractDID,
				PaymentModel: contract.PaymentDetails.PaymentModel,
				Error:        fmt.Sprintf("unsupported payment model: %v", err),
			}
			resp.Results = append(resp.Results, result)
			log.Warnf("unsupported payment model for contract %s: %v", contract.ContractDID, err)
			continue
		}

		// Check if this model supports manual billing
		if !processor.SupportsManualBilling() {
			result := contracts.ContractUsageResult{
				ContractDID:  contract.ContractDID,
				PaymentModel: contract.PaymentDetails.PaymentModel,
				Error:        fmt.Sprintf("%s payment model uses automatic periodic billing and cannot be manually triggered. Invoices are generated automatically by the contract actor.", contract.PaymentDetails.PaymentModel),
			}
			resp.Results = append(resp.Results, result)
			log.Warnf("manual invoice generation attempted for %s contract %s", contract.PaymentDetails.PaymentModel, contract.ContractDID)
			continue
		}

		// Collect usage using processor
		usageData, err := processor.CollectUsage(contract.ContractDID, lastProcessedAt, now)
		if err != nil {
			result := contracts.ContractUsageResult{
				ContractDID:  contract.ContractDID,
				PaymentModel: contract.PaymentDetails.PaymentModel,
				Error:        fmt.Sprintf("failed to collect usage: %v", err),
			}
			resp.Results = append(resp.Results, result)
			log.Warnf("failed to collect usage for contract %s: %v", contract.ContractDID, err)
			continue
		}

		// Convert usage data to result format
		result := n.convertUsageDataToResult(usageData)
		if result.Error != "" {
			resp.Results = append(resp.Results, result)
			log.Warnf("failed to convert usage data for contract %s: %s", contract.ContractDID, result.Error)
			continue
		}

		// Save contract-specific last processed timestamp
		err = n.usageStore.SaveLastProcessedAt(contract.ContractDID, now)
		if err != nil {
			result.Error = fmt.Sprintf("failed to save last processed timestamp: %v", err)
			resp.Results = append(resp.Results, result)
			log.Warnf("failed to save last processed usage for contract %s: %v", contract.ContractDID, err)
			continue
		}

		// Add result
		resp.Results = append(resp.Results, result)

		// Add to payments if there are usages
		if result.Usages > 0 || result.TimeUtilization != nil || result.ResourceUtilization != nil {
			paymentReq := paymentForwardToProviderRequest{
				Contract: *contract,
				Usages:   result.Usages,
			}
			if result.TimeUtilization != nil {
				paymentReq.TimeUtilization = result.TimeUtilization
			}
			if result.ResourceUtilization != nil {
				paymentReq.ResourceUtilization = result.ResourceUtilization
			}
			allPayments = append(allPayments, paymentReq)
		}
	}

	for _, v := range allPayments {
		req := contracts.ContractUsageRequest{
			UniqueID:            uuid.NewString(),
			Contract:            v.Contract,
			Usages:              v.Usages,
			TimeUtilization:     v.TimeUtilization,     // May contain multiple deployments for pay_per_time_utilization
			ResourceUtilization: v.ResourceUtilization, // May contain multiple deployments for pay_per_resource_utilization
		}

		// construct destination address
		destination, err := actor.HandleFromDID(v.Contract.PaymentValidatorDID.URI)
		if err != nil {
			log.Errorf("failed to get handle of payment provider for contract %s: %v", v.Contract.ContractDID, err)
			continue
		}
		envelope, err := n.invokeBehaviour(destination, behaviors.ContractUsageBehavior, req, invokeMessageTimeout)
		if envelope.Message == nil || err != nil {
			log.Errorf("failed to update payment status of contract host for contract %s: %v", v.Contract.ContractDID, err)
			continue
		}
		log.Infof("Successfully sent ContractUsageRequest for contract %s to payment validator", v.Contract.ContractDID)
		resp.TotalUsages++
	}

	return resp
}

// convertUsageDataToResult converts UsageData from processor to ContractUsageResult
func (n *Node) convertUsageDataToResult(usageData *contracts.UsageData) contracts.ContractUsageResult {
	result := contracts.ContractUsageResult{
		ContractDID:  usageData.ContractDID,
		PaymentModel: usageData.PaymentModel,
	}

	switch usageData.PaymentModel {
	case contracts.PayPerAllocation:
		usageCount, ok := usageData.Data.(int)
		if !ok {
			result.Error = "invalid usage data type for pay_per_allocation"
			return result
		}
		result.Usages = usageCount

	case contracts.PayPerDeployment:
		usageCount, ok := usageData.Data.(int)
		if !ok {
			result.Error = "invalid usage data type for pay_per_deployment"
			return result
		}
		result.Usages = usageCount

	case contracts.PayPerTimeUtilization:
		timeUtil, ok := usageData.Data.(*contracts.TimeUtilizationUsage)
		if !ok {
			result.Error = "invalid usage data type for pay_per_time_utilization"
			return result
		}
		result.TimeUtilization = timeUtil
		result.Usages = len(timeUtil.Deployments) // For backward compatibility

	case contracts.PayPerResourceUtilization:
		resourceUtil, ok := usageData.Data.(*contracts.ResourceUtilizationUsage)
		if !ok {
			result.Error = "invalid usage data type for pay_per_resource_utilization"
			return result
		}
		result.ResourceUtilization = resourceUtil
		result.Usages = len(resourceUtil.Deployments) // For backward compatibility

	default:
		result.Error = fmt.Sprintf("unsupported payment model: %s", usageData.PaymentModel)
	}

	return result
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

func (n *Node) handleContractEvents(event eventhandler.Event) error {
	hostDID, err := did.FromString(event.ContractHostDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts host did: %w", err)
	}
	pubKey, err := did.PublicKeyFromDID(hostDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts host public key from did: %w", err)
	}

	pid, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to get peer id: %w", err)
	}

	// get actor public key
	contractActorDID, err := did.FromString(event.ContractDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts actor did: %w", err)
	}
	pubKeyContractActor, err := did.PublicKeyFromDID(contractActorDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts actor public key from did: %w", err)
	}

	destination, err := actor.HandleFromPublicKeyWithInboxAddress(pubKeyContractActor, event.ContractDID, pid.String())
	if err != nil {
		return fmt.Errorf("failed to get contracts host handle: %w", err)
	}

	bts, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event object: %w", err)
	}

	req := contracts.ContractEventRequest{
		Payload: bts,
	}
	reply, err := n.invokeBehaviour(destination, behaviors.ContractEventsBehavior, req, invokeMessageTimeout)
	if err != nil {
		return fmt.Errorf("failed to send message to contract host: %w", err)
	}

	var respEnvelope contracts.ContractEventResponse
	err = json.Unmarshal(reply.Message, &respEnvelope)
	if err != nil {
		return fmt.Errorf("failed to unmarshal contract hosts response payload: %w", err)
	}

	if respEnvelope.Error != "" {
		return fmt.Errorf("failed to process contract event: %s", respEnvelope.Error)
	}

	return nil
}

// isRestorableStatus checks if a deployment status is restorable
// TODO: This will be implemented later with more sophisticated logic
// For now, all deployments with status <= Running are considered restorable
func isRestorableStatus(status jobtypes.DeploymentStatus) bool {
	// TODO: Implement more sophisticated restorable status logic
	// This should consider:
	// - Deployment lifecycle state
	// - Resource allocation status
	// - Compute provider availability
	// - Network connectivity requirements
	// For now we will keep it as true until this logic has been designed.
	return status <= jobtypes.DeploymentStatusRunning
}

// isDeploymentStillValid checks if a deployment is still valid for restoration
// TODO: This will be implemented later with comprehensive validation
// For now, all deployments are considered valid
func isDeploymentStillValid(_ *jobtypes.OrchestratorView) bool {
	// TODO: Implement comprehensive deployment validation
	// This should check:
	// - Deployment configuration is still valid
	// - Required resources are still available
	// - Network configuration is still accessible
	// - Compute provider is still responsive
	// - Deployment hasn't been explicitly cancelled
	// - Resource quotas haven't been exceeded
	// - Security policies haven't changed
	//
	// Note: Compute providers should also implement similar validation
	// on their side to ensure resources are still available and
	// haven't been reclaimed due to extended downtime
	return true
}
