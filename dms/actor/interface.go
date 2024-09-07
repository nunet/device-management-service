package actor

import (
	"context"

	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

type (
	ID         = crypto.ID
	DID        = did.DID
	Capability = ucan.Capability
)

// ActorHandle is a handle for naming an actor reachable in the network
type Handle struct {
	ID      ID      `json:"id"`
	DID     DID     `json:"did"`
	Address Address `json:"addr"`
}

// ActorAddress is a raw actor address representation
type Address struct {
	HostID       string `json:"host,omitempty"`
	InboxAddress string `json:"inbox,omitempty"`
}

// Envelope is the envelope for messages in the actor system
type Envelope struct {
	To         Handle          `json:"to"`
	Behavior   string          `json:"be"`
	From       Handle          `json:"from"`
	Nonce      uint64          `json:"nonce"`
	Options    EnvelopeOptions `json:"opt"`
	Message    []byte          `json:"msg"`
	Capability []byte          `json:"cap,omitempty"`
	Signature  []byte          `json:"sig,omitempty"`

	Discard func() `json:"-"`
}

// EnvelopeOptions are sender specified options for processing an envelope
type EnvelopeOptions struct {
	Expire  uint64 `json:"exp"`
	ReplyTo string `json:"cont,omitempty"`
	Topic   string `json:"topic,omitempty"`
}

// Actor is the local interface to the actor system
type Actor interface {
	Context() context.Context
	Handle() Handle
	Security() SecurityContext

	AddBehavior(behavior string, continuation Behavior, opt ...BehaviorOption) error
	RemoveBehavior(behavior string)

	Receive(msg Envelope) error
	Send(msg Envelope) error
	Invoke(msg Envelope) (<-chan Envelope, error)

	Publish(msg Envelope) error
	Subscribe(topic string) error

	Stop() error
}

// ActorSecurityContext provides a context for which to perform cryptographic operations
// for an actor.
// This includes:
// - signing messages
// - verifying message signatures
// - requiring capabilities
// - granting capabilities
type SecurityContext interface {
	ID() ID
	DID() DID
	Nonce() uint64

	// Require checks the capability token(s).
	// It succeeds if and only if
	// - the signature is valid
	// - the capability token(s) in the envelope grants the origin actor ID/DID
	//   any of the specified capabilities.
	Require(msg Envelope, invoke []Capability) error

	// Provide populates the envelope with necessary capability tokens and signs it.
	// the envelope is modified in place
	Provide(msg *Envelope, invoke []Capability, delegate []Capability) error

	// Require verifies the envelope and checks the capability tokens
	// for a broadcast topic
	RequireBroadcast(msg Envelope, topic string, broadcast []Capability) error

	// ProvideBroadcast populates the envelope with the necessary capability tokens
	// for broadcast in the topic and signs it
	ProvideBroadcast(msg *Envelope, topic string, broadcast []Capability) error

	// Verify verifies the message signature in an envelope
	Verify(msg Envelope) error
	// Sign signs an envelope; the envelope is modified in place.
	Sign(msg *Envelope) error

	// Disparcrd discards unwanted tokens from a consumed envelope
	Discard(msg Envelope)
}

type Behavior func(msg Envelope)

type MessageOption func(msg *Envelope) error

type BehaviorOption func(opt *BehaviorOptions) error

type BehaviorOptions struct {
	Capability []Capability
	Expire     uint64
	OneShot    bool
	Topic      string
}
