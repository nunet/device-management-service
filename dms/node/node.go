package node

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: remove after resource manager MR is merged
type benchmarker interface {
	Benchmark(ctx context.Context) (*types.Capability, error)
}

// Node is the structure that holds the node's dependencies.
type Node struct {
	ID string

	actor        *dms.BasicActor
	network      network.Network
	actorFactory *dms.ActorFactory

	// TODO: fix when resource manager is merged to develop
	resourceManager resources.Manager

	benchmark   benchmarker
	allocations map[string]*jobs.Allocation
	mu          sync.RWMutex
}

// New creates a new node, attaches an actor to the node.
func New(_ context.Context, id string, net network.Network, benchmark benchmarker, resourceManager resources.Manager) (*Node, error) {
	if id == "" {
		return nil, errors.New("id is nil")
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

	actorFactory := dms.NewActorFactory(id, net)

	n := &Node{
		ID:              id,
		actorFactory:    actorFactory,
		network:         net,
		allocations:     make(map[string]*jobs.Allocation),
		benchmark:       benchmark,
		resourceManager: resourceManager,
	}

	err := n.createNodeActor()
	if err != nil {
		return nil, fmt.Errorf("failed to create node actor: %w", err)
	}

	return n, nil
}

// GetAllocation gets an allocation by id.
func (n *Node) GetAllocation(id string) (*jobs.Allocation, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	alloc, ok := n.allocations[id]
	if !ok {
		return nil, errors.New("allocation not found")
	}

	return alloc, nil
}

// GetAllocation gets an allocation by id.
func (n *Node) BenchmarkCapability(ctx context.Context) (*types.Capability, error) {
	return n.benchmark.Benchmark(ctx)
}

// CreateAllocation creates an allocation.
func (n *Node) CreateAllocation(_ context.Context, job jobs.Job) (*jobs.Allocation, error) {
	allocationActor, err := n.actor.CreateActor()
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation actor: %w", err)
	}

	allocation, err := jobs.NewAllocation(allocationActor, jobs.AllocationDetails{Job: job, NodeID: n.ID, SourceID: ""}, n.resourceManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation: %w", err)
	}

	n.mu.Lock()
	n.allocations[allocation.ID] = allocation
	n.mu.Unlock()

	return allocation, nil
}

// ProcessMessages processes actor messages.
func (n *Node) ProcessMessages() {
	for msg := range n.actor.Messages() {
		n.dispatchMethod(msg.Type, msg.Data)
	}
}

// SendMessage sends a message through the actor.
func (n *Node) SendMessage(ctx context.Context, destination *dms.ActorAddrInfo, m *dms.Message) error {
	return n.actor.SendMessage(ctx, destination, m)
}

// HandleHello will be called when a message type of `Hello` arrives to the messages queue.
// For this to work properly we should always append Handle in front of the function.
func (n *Node) HandleHello(payload []byte) {
	fmt.Println("hello from: ", string(payload))
}

func (n *Node) dispatchMethod(methodName string, args ...any) {
	handlerMethod := fmt.Sprintf("Handle%s", methodName)

	arguments := make([]reflect.Value, 0)
	for _, v := range args {
		arguments = append(arguments, reflect.ValueOf(v))
	}
	method := reflect.ValueOf(n).MethodByName(handlerMethod)
	if method.IsValid() {
		method.Call(arguments)
		return
	}

	// check if actor has the method
	actorMethod := reflect.ValueOf(n.actor).MethodByName(handlerMethod)
	if actorMethod.IsValid() {
		actorMethod.Call(arguments)
	}
}

// createNodeActor the root actor in this node instance.
func (n *Node) createNodeActor() error {
	actor, err := n.actorFactory.NewActor()
	if err != nil {
		return fmt.Errorf("failed to create actor: %w", err)
	}

	n.actor = actor
	err = n.actor.Start()
	if err != nil {
		return fmt.Errorf("failed to start node actor: %w", err)
	}

	return nil
}
