package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

type BasicActor struct {
	dispatch  *Dispatch
	scheduler *bt.Scheduler
	registry  *registry
	network   network.Network
	security  SecurityContext

	params BasicActorParams
	self   Handle

	addedTaskID int
}

type BasicActorParams struct {
	Heartbeat struct {
		Interval time.Duration
		Jitter   float64
	}
}

var _ Actor = (*BasicActor)(nil)

// New creates a new basic actor.
func New(dispatch *Dispatch, scheduler *bt.Scheduler, net network.Network, security *BasicSecurityContext, params BasicActorParams, self Handle) (*BasicActor, error) {
	if dispatch == nil {
		return nil, errors.New("dispatch is nil")
	}

	if scheduler == nil {
		return nil, errors.New("scheduler is nil")
	}

	if net == nil {
		return nil, errors.New("network is nil")
	}

	if security == nil {
		return nil, errors.New("security is nil")
	}

	actor := &BasicActor{
		dispatch:  dispatch,
		scheduler: scheduler,
		registry:  &registry{},
		network:   net,
		security:  security,
		params:    params,
		self:      self,
	}

	return actor, nil
}

func (a *BasicActor) Start() error {
	// Network messages
	if err := a.network.HandleMessage(
		fmt.Sprintf("actor/%s/messages/0.0.1", a.self.Address.InboxAddress),
		a.handleMessage,
	); err != nil {
		return fmt.Errorf("starting actor: %s: %w", a.self.ID, err)
	}

	// Heartbeat
	err := a.dispatch.AddBehavior(heartbeatBehavior, a.handleHeartbeat)
	if err != nil {
		return fmt.Errorf("failed to add heartbeat behaviour: %w", err)
	}

	if parent, ok := a.registry.GetParent(a.self); ok {
		task := &bt.Task{
			Name:        "actor heartbeat",
			Description: "send heartbeat to parent actor",
			Triggers: []bt.Trigger{
				&bt.PeriodicTriggerWithJitter{
					Interval: a.params.Heartbeat.Interval,
					Jitter: func() time.Duration {
						return jitter(
							a.params.Heartbeat.Interval,
							a.params.Heartbeat.Jitter,
						)
					},
				},
			},
			Function: func(_ interface{}) error {
				return a.sendHeartbeat(*parent)
			},
		}
		addedTask := a.scheduler.AddTask(task)
		a.addedTaskID = addedTask.ID
	}

	// and start the internal goroutines
	a.dispatch.Start()
	a.scheduler.Start()
	return nil
}

func (a *BasicActor) handleMessage(data []byte) {
	var msg Envelope
	if err := json.Unmarshal(data, &msg); err != nil {
		// TODO log debug
		return
	}

	if !a.self.ID.Equal(msg.To.ID) {
		// TODO log warn
		return
	}

	_ = a.dispatch.Receive(msg)
}

func (a *BasicActor) handleHeartbeat(_ Envelope) {
	// TODO
}

func (a *BasicActor) sendHeartbeat(parent Handle) error {
	msg, err := Message(
		a.self,
		parent,
		heartbeatBehavior,
		HeartbeatMessage{},
	)
	if err != nil {
		return fmt.Errorf("constructing heartbeat message: %w", err)
	}

	return a.Send(msg)
}

func (a *BasicActor) Context() context.Context {
	return a.dispatch.Context()
}

func (a *BasicActor) Handle() Handle {
	return a.self
}

func (a *BasicActor) Security() SecurityContext {
	return a.security
}

func (a *BasicActor) AddBehavior(behavior string, continuation Behavior, opt ...BehaviorOption) error {
	return a.dispatch.AddBehavior(behavior, continuation, opt...)
}

func (a *BasicActor) RemoveBehavior(behavior string) {
	a.dispatch.RemoveBehavior(behavior)
}

func (a *BasicActor) Receive(msg Envelope) error {
	if !a.self.ID.Equal(msg.To.ID) {
		return fmt.Errorf("bad receiver: %w", ErrInvalidMessage)
	}

	return a.dispatch.Receive(msg)
}

func (a *BasicActor) Send(msg Envelope) error {
	if msg.To.ID.Equal(a.self.ID) {
		return a.Receive(msg)
	}

	if msg.Signature == nil {
		if msg.Nonce == 0 {
			msg.Nonce = a.security.Nonce()
		}

		invoke := []Capability{Capability(msg.Behavior)}
		var delegate []Capability
		if msg.Options.ReplyTo != "" {
			delegate = append(delegate, Capability(msg.Options.ReplyTo))
		}
		if err := a.security.Provide(&msg, invoke, delegate); err != nil {
			return fmt.Errorf("providing behavior capability for %s: %w", msg.Behavior, err)
		}
	}

	addrs, err := a.network.ResolveAddress(
		a.Context(),
		msg.To.Address.HostID,
	)
	if err != nil {
		return fmt.Errorf("resolving address for %s: %w", msg.To.ID, err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	err = a.network.SendMessage(
		a.Context(),
		addrs,
		types.MessageEnvelope{
			Type: types.MessageType(
				fmt.Sprintf("actor/%s/messages/0.0.1", msg.To.Address.InboxAddress),
			),
			Data: data,
		})
	if err != nil {
		return fmt.Errorf("sending message to %s: %w", msg.To.ID, err)
	}

	return nil
}

func (a *BasicActor) Invoke(msg Envelope, opt ...BehaviorOption) (<-chan Envelope, error) {
	if msg.Options.ReplyTo == "" {
		msg.Options.ReplyTo = fmt.Sprintf("/dms/actor/replyto/%d", a.security.Nonce())
	}

	result := make(chan Envelope, 1)

	opt = append([]BehaviorOption{
		WithBehaviorExpiry(msg.Options.Expire),
		WithBehaviorOneShot(true),
	}, opt...)
	if err := a.dispatch.AddBehavior(
		msg.Options.ReplyTo,
		func(reply Envelope) {
			result <- reply
			close(result)
		},
		opt...,
	); err != nil {
		return nil, fmt.Errorf("adding reply behavior: %w", err)
	}

	if err := a.Send(msg); err != nil {
		a.dispatch.RemoveBehavior(msg.Options.ReplyTo)
		return nil, fmt.Errorf("sending message: %w", err)
	}

	return result, nil
}

func (a *BasicActor) Stop() error {
	a.dispatch.close()
	a.scheduler.RemoveTask(a.addedTaskID)
	return nil
}
