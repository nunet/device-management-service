package dms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

// Actor defines the functionalities of actor.
type Actor interface {
	// Address of actor.
	Address() *ActorAddrInfo
	// SendMessage to another actor.
	SendMessage(destination *ActorAddrInfo, m *Message)
	// CreateActor creates a new actor.
	CreateActor() (*BasicActor, error)
	// Messages gets the actor messages
	Messages() <-chan Message
}

// ActorAddrInfo encapsulates the data required to address an actor.
type ActorAddrInfo struct {
	HostID       string
	InboxAddress string
}

// Valid checks if an actor is valid.
func (a *ActorAddrInfo) Valid() bool {
	return a.HostID != "" && a.InboxAddress != ""
}

// Message is passed between actors.
type Message struct {
	Type   string
	Sender string
	Data   []byte
}

// ActorFactory is an actor factory.
type ActorFactory struct {
	hostID  string
	network network.Network
	params  *ActorParams
}

// Actor represents an actor.
type BasicActor struct {
	hostID                 string
	address                string
	network                network.Network
	messages               chan Message
	actorRegistry          *ActorRegistry
	factory                *ActorFactory
	scheduler              *bt.Scheduler
	heartbeatTracker       map[string]*heartbeatConfig
	heartbeatInterval      time.Duration
	heartbeatCheckInterval time.Duration
}

type ActorParams struct {
	HeartbeatInterval      time.Duration
	HeartbeatCheckInterval time.Duration
	Threshold              int
	Action                 func()
}

type heartbeatConfig struct {
	threshold       int
	missed          int
	lastHeartbeatMS int64
	interval        time.Duration
	action          func()
	cancel          context.CancelFunc
}

// NewActorFactory holds the dependencies to create and manage actors.
func NewActorFactory(hostID string, network network.Network, params *ActorParams) *ActorFactory {
	return &ActorFactory{
		hostID:  hostID,
		network: network,
		params:  params,
	}
}

// NewActor allows the factory to create a new actor.
func (f *ActorFactory) NewActor() (*BasicActor, error) {
	return f.newActor(nil)
}

func (f *ActorFactory) newActor(parentActorAddress *ActorAddrInfo) (*BasicActor, error) {
	return newActor(parentActorAddress, f.hostID, f.network, f, f.params)
}

// newActor returns a new actor based on the given arguments.
func newActor(
	parentActorAddress *ActorAddrInfo,
	hostID string,
	net network.Network,
	factory *ActorFactory,
	params *ActorParams,
) (*BasicActor, error) {
	if hostID == "" {
		return nil, errors.New("host id is empty")
	}

	if net == nil {
		return nil, fmt.Errorf("network is nil")
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid: %w", err)
	}

	createdActor := &BasicActor{
		hostID:                 hostID,
		network:                net,
		address:                id.String(),
		actorRegistry:          NewActorRegistry(),
		factory:                factory,
		messages:               make(chan Message, 100),
		scheduler:              bt.NewScheduler(5),
		heartbeatTracker:       make(map[string]*heartbeatConfig),
		heartbeatInterval:      params.HeartbeatInterval,
		heartbeatCheckInterval: params.HeartbeatCheckInterval,
	}

	createdActor.actorRegistry.AddActorAddress(createdActor.Address())

	if parentActorAddress != nil {
		createdActor.actorRegistry.SetParentAddress(createdActor.address, parentActorAddress)
		createdActor.actorRegistry.AddChild(parentActorAddress.InboxAddress, createdActor.Address())
	}

	return createdActor, nil
}

// CreateActor creates another actor.
func (a *BasicActor) CreateActor() (*BasicActor, error) {
	newActor, err := a.factory.newActor(a.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to create new actor: %w", err)
	}

	a.trackHeartbeats(
		newActor.Address().InboxAddress,
		a.factory.params.HeartbeatCheckInterval,
		a.factory.params.Threshold,
		a.factory.params.Action,
	)

	return newActor, nil
}

// Address returns the address of an actor.
func (a *BasicActor) Address() *ActorAddrInfo {
	return &ActorAddrInfo{
		HostID:       a.hostID,
		InboxAddress: a.address,
	}
}

// Start registers the message handlers and starts an actor.
func (a *BasicActor) Start() error {
	err := a.network.HandleMessage(fmt.Sprintf("actor/%s/messages/0.0.1", a.address), func(data []byte) {
		var msg Message
		err := json.Unmarshal(data, &msg)
		if err != nil {
			log.Warn("error while handling message", err)
			return
		}
		a.messages <- msg
	})
	if err != nil {
		return fmt.Errorf("failed to start actor %s: %w", a.address, err)
	}

	err = a.network.HandleMessage(fmt.Sprintf("/actor/%s/heartbeat/0.0.1", a.address), a.handleHeartbeat)
	if err != nil {
		return fmt.Errorf("failed to start actor %s: %w", a.address, err)
	}

	parent, ok := a.actorRegistry.GetParentAddress(a.address)
	if ok && parent != nil {
		heartbeatTask := &bt.Task{
			Name:        "Heartbeat parent actors",
			Description: "Send periodic heartbeat to parent actors",
			Triggers: []bt.Trigger{
				&bt.PeriodicTriggerWithJitter{
					Interval: a.heartbeatInterval,
					Jitter: func() time.Duration {
						return jitter(a.heartbeatInterval, 0.1) // 10% additional jitter
					},
				},
			},
			Function: func(_ interface{}) error {
				err = a.SendHeartbeat(context.Background(), parent, &Message{
					Type:   "heartbeat",
					Sender: a.address,
					Data:   []byte("heartbeat"),
				})
				if err != nil {
					return err
				}
				return nil
			},
		}

		a.scheduler.AddTask(heartbeatTask)
		a.scheduler.Start()
	}

	return nil
}

func (a *BasicActor) Stop() error {
	a.scheduler.Stop()
	for _, hbt := range a.heartbeatTracker {
		hbt.cancel()
	}
	return nil
}

// SendMessage sends a message to another actor.
func (a *BasicActor) SendMessage(ctx context.Context, destination *ActorAddrInfo, m *Message) error {
	if !destination.Valid() {
		return errors.New("destination actor addr info is invalid")
	}

	if m == nil {
		return errors.New("message is invalid")
	}

	// get the multiaddress of a host by resolving the hostid
	addresses, err := a.network.ResolveAddress(ctx, destination.HostID)
	if err != nil {
		return fmt.Errorf("failed to send message to actor %s: %v", destination.HostID, err)
	}

	// TODO: convert to proto message and marshal with protobuf
	actorMessge, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal actor message")
	}

	err = a.network.SendMessage(ctx, []string{addresses[0]}, types.MessageEnvelope{
		Type: types.MessageType(fmt.Sprintf("actor/%s/messages/0.0.1", destination.InboxAddress)),
		Data: actorMessge,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to remote actor %s: %v", destination.InboxAddress, err)
	}

	return nil
}

func (a *BasicActor) HandleGenericAction1(payload []byte) {
	fmt.Println("This is generic method for actor: ", string(payload))
}

func (a *BasicActor) SendHeartbeat(ctx context.Context, destination *ActorAddrInfo, m *Message) error {
	if !destination.Valid() {
		return errors.New("destination actor addr info is invalid")
	}

	if m == nil {
		return errors.New("message is invalid")
	}

	// get the multiaddress of a host by resolving the hostid
	addresses, err := a.network.ResolveAddress(ctx, destination.HostID)
	if err != nil {
		return fmt.Errorf("failed to send message to actor %s: %v", destination.HostID, err)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	err = a.network.SendMessage(ctx, []string{addresses[0]}, types.MessageEnvelope{
		Type: types.MessageType(fmt.Sprintf("/actor/%s/heartbeat/0.0.1", destination.InboxAddress)),
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to remote actor %s: %v", destination.InboxAddress, err)
	}

	return nil
}

// ProcessMessages reads messages from the incoming messages channel.
func (a *BasicActor) ProcessMessages() {
	for msg := range a.messages {
		fmt.Printf("received message from %s", msg.Sender)
	}
}

// Messages returns the messages channel to be consumed
func (a *BasicActor) Messages() <-chan Message {
	return a.messages
}

// Hello behaviour
func (a *BasicActor) Hello(ctx context.Context, destination *ActorAddrInfo, m *Message) {
	_ = a.SendMessage(ctx, destination, m)
}

// nolint:unused
func (a *BasicActor) handleHello(m Message) {
	fmt.Println("handled hello message", m)
}

func (a *BasicActor) handleHeartbeat(data []byte) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Warn("error while handling message", err)
		return
	}

	heartbeatTracker, ok := a.heartbeatTracker[msg.Sender]
	if !ok {
		log.Debug("unknown/untracked actor heartbeating")
		return
	}

	fmt.Println("heartbeat received from", msg.Sender)
	heartbeatTracker.lastHeartbeatMS = time.Now().UnixMilli()
}

func (a *BasicActor) trackHeartbeats(address string, interval time.Duration, threshold int, action func()) {
	ctx, cancel := context.WithCancel(context.Background())
	heartbeatCfg := &heartbeatConfig{
		threshold:       threshold,
		missed:          0,
		lastHeartbeatMS: 0,
		interval:        interval,
		action:          action,
		cancel:          cancel,
	}
	a.heartbeatTracker[address] = heartbeatCfg

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if time.Now().UnixMilli()-heartbeatCfg.lastHeartbeatMS > heartbeatCfg.interval.Milliseconds() {
					heartbeatCfg.missed++
				}
				if heartbeatCfg.missed > heartbeatCfg.threshold {
					heartbeatCfg.action()
					heartbeatCfg.cancel()
				}
			}
		}
	}()
}

func jitter(val time.Duration, percentage float64) time.Duration {
	return time.Duration(float64(val) * percentage)
}
