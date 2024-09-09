package node

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/resources"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: remove after resource manager MR is merged
type benchmarker interface {
	Benchmark(ctx context.Context) (*types.Capability, error)
}

// Node is the structure that holds the node's dependencies.
type Node struct {
	rootCap         ucan.CapabilityContext
	actor           actor.Actor
	benchmark       benchmarker
	scheduler       *bt.Scheduler
	network         network.Network
	resourceManager resources.Manager
	hostID          string

	mx          sync.Mutex
	allocations map[string]*jobs.Allocation
	running     bool
}

// New creates a new node, attaches an actor to the node.
func New(rootCap ucan.CapabilityContext, hostID string, net network.Network, benchmark benchmarker, resourceManager resources.Manager, scheduler *bt.Scheduler) (*Node, error) {
	if rootCap == nil {
		return nil, errors.New("root capability context is nil")
	}

	if hostID == "" {
		return nil, errors.New("host id is nil")
	}

	if net == nil {
		return nil, errors.New("network is nil")
	}

	if benchmark == nil {
		return nil, errors.New("benchmarker is nil")
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

	actor, err := createActor(rootSec, hostID, "root", net, scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create node actor: %w", err)
	}

	n := &Node{
		hostID:          hostID,
		network:         net,
		allocations:     make(map[string]*jobs.Allocation),
		benchmark:       benchmark,
		resourceManager: resourceManager,
		actor:           actor,
		rootCap:         rootCap,
		scheduler:       scheduler,
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

	actor, err := createActor(security, n.hostID, allocationInbox.String(), n.network, n.scheduler)
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

// GetAllocation gets an allocation by id.
func (n *Node) BenchmarkCapability(ctx context.Context) (*types.Capability, error) {
	return n.benchmark.Benchmark(ctx)
}

// createActor creates an actor.
func createActor(sctx *actor.BasicSecurityContext, hostID, inboxAddress string, net network.Network, scheduler *bt.Scheduler) (*actor.BasicActor, error) {
	self := actor.Handle{
		ID:  sctx.ID(),
		DID: sctx.DID(),
		Address: actor.Address{
			HostID:       hostID,
			InboxAddress: inboxAddress,
		},
	}
	actor, err := actor.New(actor.NewDispatch(sctx), scheduler, net, sctx, actor.BasicActorParams{}, self)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor: %w", err)
	}

	return actor, nil
}
