package actor

import (
	"context"
)

// ActorID is the encoding of the actor's public key
type ID struct{ PublicKey []byte }

// ActorDID is the actor's unique distributed identifier
type DID struct{ PublicKey []byte }

// ActorCapability is a capability identifier
type Capability string

// ActorHandle is a handle for naming an actor reachable in the network
type Handle struct {
	ID      ID
	DID     DID
	Address Address
}

// ActorAddress is a raw actor address representation
type Address struct {
	HostID       string
	InboxAddress string
}

// Envelope is the envelope for messages in the actor system
type Envelope struct {
	To         Handle
	Behavior   string
	From       Handle
	Nonce      uint64
	Options    EnvelopeOptions
	Message    []byte
	Capability []byte
	Signature  []byte
}

// EnvelopeOptions are sender specified options for processing an envelope
type EnvelopeOptions struct {
	Expire  uint64
	ReplyTo string
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
	Invoke(msg Envelope, opt ...BehaviorOption) (<-chan Envelope, error)

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

	// Require verifies the envelope and checks the capability token(s).
	// It succeeds if and only if
	// - the signature is valid
	// - the capability token(s) in the envelope grants the origin actor ID/DID
	//   any of the specified capabilities.
	Require(msg Envelope, cap ...Capability) error
	// Provide populates the envelope with necessary capability tokens and signs it.
	// the envelope is modified in place
	Provide(msg *Envelope, cap ...Capability) error

	// Verify verifies the message signature in an envelope
	Verify(msg Envelope) error
	// Sign signs an envelope; the envelope is modified in place.
	Sign(msg *Envelope) error
}

type Behavior func(msg Envelope)

type MessageOption func(msg *Envelope) error

type BehaviorOption func(opt *BehaviorOptions) error

type BehaviorOptions struct {
	Capability []Capability
	Expire     uint64
	OneShot    bool
}
