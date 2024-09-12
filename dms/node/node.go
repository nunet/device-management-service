package node

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/google/uuid"

	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
)

// Node is the structure that holds the node's dependencies.
type Node struct {
	rootCap         ucan.CapabilityContext
	actor           actor.Actor
	scheduler       *bt.Scheduler
	network         network.Network
	resourceManager types.ResourceManager
	hostID          string
	onboarder       *onboarding.Onboarding
	executor        *firecracker.Executor

	mx          sync.Mutex
	allocations map[string]*jobs.Allocation
	running     bool
}

// New creates a new node, attaches an actor to the node.
func New(ctx context.Context, onboarder *onboarding.Onboarding, rootCap ucan.CapabilityContext, hostID string, net network.Network, resourceManager types.ResourceManager, scheduler *bt.Scheduler) (*Node, error) {
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
	privk := provider.PrivateKey()

	rootSec, err := actor.NewBasicSecurityContext(pubk, privk, rootCap)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context: %w", err)
	}

	nodeActor, err := createActor(rootSec, actor.NewRateLimiter(actor.DefaultRateLimiterConfig()), hostID, "root", net, scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create node actor: %w", err)
	}

	executor, err := firecracker.NewExecutor(ctx, "root")
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	n := &Node{
		hostID:          hostID,
		network:         net,
		allocations:     make(map[string]*jobs.Allocation),
		resourceManager: resourceManager,
		actor:           nodeActor,
		rootCap:         rootCap,
		scheduler:       scheduler,
		onboarder:       onboarder,
		executor:        executor,
	}

	if err := nodeActor.AddBehavior(PublicHelloBehavior, n.publicHelloBehavior); err != nil {
		return nil, fmt.Errorf("adding public hello behavior: %w", err)
	}
	if err := nodeActor.AddBehavior(PublicStatusBehavior, n.publicStatusBehavior); err != nil {
		return nil, fmt.Errorf("adding public status behavior: %w", err)
	}
	if err := nodeActor.AddBehavior(BroadcastHelloBehavior, n.broadcastHelloBehavior, actor.WithBehaviorTopic(BroadcastHelloTopic)); err != nil {
		return nil, fmt.Errorf("adding broadcast status behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(PeersListBehavior, n.handlePeersList); err != nil {
		return nil, fmt.Errorf("adding peers list behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(PeerAddrInfoBehavior, n.handlePeerAddrInfo); err != nil {
		return nil, fmt.Errorf("adding peers addr info behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(PeerPingBehavior, n.handlePeerPing); err != nil {
		return nil, fmt.Errorf("adding peer ping behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(PeerDHTBehavior, n.handlePeerDHT); err != nil {
		return nil, fmt.Errorf("adding peer dht behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(PeerConnectBehavior, n.handlePeerConnect); err != nil {
		return nil, fmt.Errorf("adding peer connect behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(OnboardBehaviour, n.handleOnboard); err != nil {
		return nil, fmt.Errorf("adding onboard behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(OffboardBehaviour, n.handleOffboard); err != nil {
		return nil, fmt.Errorf("adding offboard behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(OnboardStatusBehaviour, n.handleOnboardStatus); err != nil {
		return nil, fmt.Errorf("adding onboard status behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(OnboardResourceBehaviour, n.handleOnboardResource); err != nil {
		return nil, fmt.Errorf("adding onboard resource behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(CustomVMStart, n.handleCustomVMStart); err != nil {
		return nil, fmt.Errorf("adding custom vm start behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(VMStop, n.handleVMStop); err != nil {
		return nil, fmt.Errorf("adding vm stop behavior: %w", err)
	}

	if err := nodeActor.AddBehavior(VMList, n.handleListVM); err != nil {
		return nil, fmt.Errorf("adding vm list behavior: %w", err)
	}

	return n, nil
}

// CreateAllocation creates an allocation
func (n *Node) CreateAllocation(job jobs.Job) (*jobs.Allocation, error) {
	// generate random keypair
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random keypair for allocation job %s: %w", job.ID, err)
	}

	security, err := actor.NewBasicSecurityContext(pub, priv, n.rootCap)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context: %w", err)
	}

	allocationInbox, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid for allocation inbox: %w", err)
	}

	actor, err := createActor(security, n.actor.Limiter(), n.hostID, allocationInbox.String(), n.network, n.scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation actor: %w", err)
	}

	allocation, err := jobs.NewAllocation(actor, jobs.AllocationDetails{Job: job, NodeID: n.hostID}, n.resourceManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation actor: %w", err)
	}

	err = allocation.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start the allocation: %w", err)
	}

	n.mx.Lock()
	n.allocations[allocation.ID] = allocation
	n.mx.Unlock()

	return allocation, nil
}

// GetAllocation gets an allocation by id.
func (n *Node) GetAllocation(id string) (*jobs.Allocation, error) {
	n.mx.Lock()
	defer n.mx.Unlock()

	alloc, ok := n.allocations[id]
	if !ok {
		return nil, errors.New("allocation not found")
	}

	return alloc, nil
}

// Start node
func (n *Node) Start() error {
	n.mx.Lock()
	defer n.mx.Unlock()
	if n.running {
		return nil
	}

	err := n.actor.Start()
	if err != nil {
		return fmt.Errorf("failed to start node actor: %w", err)
	}

	n.running = true
	return nil
}

// Stop node
func (n *Node) Stop() error {
	n.mx.Lock()
	defer n.mx.Unlock()

	if !n.running {
		return nil
	}

	// stop all allocations
	for k, alloc := range n.allocations {
		if err := alloc.Stop(context.Background()); err != nil {
			log.Warnf("error stopping allocation %s: %w", k, err)
		}
	}

	if err := n.actor.Stop(); err != nil {
		return fmt.Errorf("failed to stop node actor: %w", err)
	}

	n.running = false
	return nil
}

func (n *Node) sendReply(envelope actor.Envelope, payload interface{}) {
	reply, err := actor.ReplyTo(envelope, payload)
	if err != nil {
		log.Debugf("error creating peers list reply: %s", err)
		return
	}

	if err := n.actor.Send(reply); err != nil {
		log.Debugf("error sending peers list reply: %s", err)
	}
}

// createActor creates an actor.
func createActor(sctx *actor.BasicSecurityContext, limiter actor.RateLimiter, hostID, inboxAddress string, net network.Network, scheduler *bt.Scheduler) (*actor.BasicActor, error) {
	self := actor.Handle{
		ID:  sctx.ID(),
		DID: sctx.DID(),
		Address: actor.Address{
			HostID:       hostID,
			InboxAddress: inboxAddress,
		},
	}
	actor, err := actor.New(scheduler, net, sctx, limiter, actor.BasicActorParams{}, self)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor: %w", err)
	}

	return actor, nil
}
