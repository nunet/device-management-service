package dms

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/network"
)

// ActorInterface defines the functionalities of actor.
type ActorInterface interface {
	// Address of actor.
	Address() *ActorAddrInfo
	// SendMessage to another actor.
	SendMessage(destination *ActorAddrInfo, m *Message)
	// CreateActor creates a new actor.
	CreateActor() (*ActorAddrInfo, error)
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
	msgType string
	sender  string
	data    []byte
}

// ActorFactory is an actor factory.
type ActorFactory struct {
	hostID        string
	network       network.Network
	actorRegistry *ActorRegistry
}

// Actor represents an actor.
type Actor struct {
	hostID        string
	address       string
	network       network.Network
	messages      chan Message
	actorRegistry *ActorRegistry
	factory       *ActorFactory
}

// NewActorFactory holds the dependencies to create and manage actors.
func NewActorFactory(hostID string, network network.Network, actorRegistry *ActorRegistry) *ActorFactory {
	return &ActorFactory{
		hostID:        hostID,
		network:       network,
		actorRegistry: actorRegistry,
	}
}

// NewActor allows the factory to create a new actor.
func (f *ActorFactory) NewActor() (*Actor, error) {
	return f.newActor(nil)
}

func (f *ActorFactory) newActor(parentActorAddress *ActorAddrInfo) (*Actor, error) {
	return newActor(parentActorAddress, f.hostID, f.network, f.actorRegistry, f)
}

// newActor returns a new actor based on the given arguments.
func newActor(parentActorAddress *ActorAddrInfo, hostID string, net network.Network, actorRegistry *ActorRegistry, factory *ActorFactory) (*Actor, error) {
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

	createdActor := &Actor{
		hostID:        hostID,
		network:       net,
		address:       id.String(),
		actorRegistry: actorRegistry,
		factory:       factory,
		messages:      make(chan Message, 100),
	}

	actorRegistry.AddActorAddress(createdActor.Address())

	if parentActorAddress != nil {
		actorRegistry.SetParentAddress(createdActor.address, parentActorAddress)
		actorRegistry.AddChild(parentActorAddress.InboxAddress, createdActor.Address())
	}

	return createdActor, nil
}

// Address returns the address of an actor.
func (a *Actor) Address() *ActorAddrInfo {
	return &ActorAddrInfo{
		HostID:       a.hostID,
		InboxAddress: a.address,
	}
}

// SendMessage sends a message to another actor.
func (a *Actor) SendMessage(ctx context.Context, destination *ActorAddrInfo, m *Message) error {
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

	err = a.network.SendMessage(ctx, []string{addresses[0]}, models.MessageEnvelope{
		Type: models.MessageType(fmt.Sprintf("actor/%s/messages/0.0.1", destination.InboxAddress)),
		Data: m.data,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to remote actor %s: %v", destination.InboxAddress, err)
	}

	return nil
}

// Start registers the message handlers and starts an actor.
func (a *Actor) Start() error {
	err := a.network.HandleMessage(fmt.Sprintf("actor/%s/messages/0.0.1", a.address), func(data []byte) {
		a.messages <- Message{
			sender: "sender",
			data:   data,
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start actor %s: %w", a.address, err)
	}

	return nil
}

// CreateActor creates another actor.
func (a *Actor) CreateActor() (*ActorAddrInfo, error) {
	newActor, err := a.factory.newActor(a.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to create new actor: %w", err)
	}
	if err := newActor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start new actor: %w", err)
	}

	return newActor.Address(), nil
}

// ProcessMessages reads messages from the incoming messages channel.
func (a *Actor) ProcessMessages() {
	for msg := range a.messages {
		fmt.Printf("received message from %s", msg.sender)
		// switch msg.msgType {
		// case "hello":
		// 	{
		// 		a.handleHello(msg)
		// 	}
		// default:
		// 	fmt.Printf("unhandled message type: %s\n", msg.msgType)
		// }
	}
}

// Hello behaviour
func (a *Actor) Hello(ctx context.Context, destination *ActorAddrInfo, m *Message) {
	m.msgType = "hello"
	a.SendMessage(ctx, destination, m)
}

func (a *Actor) handleHello(m Message) {
	fmt.Println("handled hello message", m)
}
