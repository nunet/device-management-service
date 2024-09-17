package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/executor"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
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
	executor        executor.Executor

	mx          sync.Mutex
	peers       map[peer.ID]*peerState
	allocations map[string]*jobs.Allocation
	running     int32
}

type peerState struct {
	conns             int
	helloIn, helloOut bool
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

	privk, err := provider.PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get root private key: %w", err)
	}

	rootSec, err := actor.NewBasicSecurityContext(pubk, privk, rootCap)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context: %w", err)
	}

	nodeActor, err := createActor(rootSec, actor.NewRateLimiter(actor.DefaultRateLimiterConfig()), hostID, "root", net, scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create node actor: %w", err)
	}

	executor, err := NewExecutor(ctx)
	if err != nil {
		return nil, fmt.Errorf("new executor: %w", err)
	}

	n := &Node{
		hostID:          hostID,
		network:         net,
		allocations:     make(map[string]*jobs.Allocation),
		peers:           make(map[peer.ID]*peerState),
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

	if err := nodeActor.AddBehavior(PeerScoreBehavior, n.handlePeerScore); err != nil {
		return nil, fmt.Errorf("adding peer score behavior: %w", err)
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
	if !atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		return nil
	}

	if err := n.actor.Start(); err != nil {
		return fmt.Errorf("failed to start node actor: %w", err)
	}

	if err := n.subscribe(BroadcastHelloTopic); err != nil {
		_ = n.actor.Stop()
		return err
	}

	return nil
}

func (n *Node) subscribe(topics ...string) error {
	for _, topic := range topics {
		if err := n.actor.Subscribe(topic, n.setupBroadcast); err != nil {
			return fmt.Errorf("error subscribing to %s: %w", topic, err)
		}
	}

	n.network.SetBroadcastAppScore(n.broadcastScore)
	if err := n.network.Notify(n.actor.Context(), n.peerConnected, n.peerDisconnected); err != nil {
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
	if ok && st.helloIn && st.helloOut {
		return 0.01
	}

	return -100
}

func (n *Node) peerConnected(p peer.ID) {
	n.mx.Lock()
	defer n.mx.Unlock()

	st, ok := n.peers[p]
	if !ok {
		st = &peerState{}
		n.peers[p] = st
	}

	if !st.helloOut {
		go n.sayHello(p)
	}
	st.conns++
}

func (n *Node) peerDisconnected(p peer.ID) {
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

	msg, err := actor.Message(
		n.actor.Handle(),
		handle,
		PublicHelloBehavior,
		nil,
		actor.WithMessageTimeout(time.Second),
	)
	if err != nil {
		log.Debugf("failed to construct hello message: %s", err)
		return
	}

	replyCh, err := n.actor.Invoke(msg)
	if err != nil {
		log.Debugf("error invoking hello: %s", err)
		return
	}

	select {
	case reply := <-replyCh:
		reply.Discard()
		n.mx.Lock()
		if st, ok := n.peers[p]; ok {
			st.helloOut = true
		} else if n.network.PeerConnected(p) {
			// rance with connected notification
			st = &peerState{helloOut: true}
			n.peers[p] = st
		}
		n.mx.Unlock()
		log.Infof("got hello from %s", handle)

	case <-time.After(time.Until(msg.Expiry())):
		log.Debugf("hello timeout for %s", handle)
	}
}

// Stop node
func (n *Node) Stop() error {
	n.mx.Lock()
	defer n.mx.Unlock()

	if !atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		return nil
	}

	// stop all allocations
	for k, alloc := range n.allocations {
		if err := alloc.Stop(context.Background()); err != nil {
			log.Warnf("error stopping allocation %s: %w", k, err)
		}
	}

	// clear the broadcast app score
	n.network.SetBroadcastAppScore(nil)

	// stop the actor
	if err := n.actor.Stop(); err != nil {
		return fmt.Errorf("failed to stop node actor: %w", err)
	}

	return nil
}

func (n *Node) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(n.actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
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
