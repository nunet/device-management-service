package actor

import (
	"fmt"
	"sync"
	"time"
)

type BasicSecurityContext struct {
	id    ID
	did   DID
	privk PrivKey

	mx    sync.Mutex
	nonce uint64
}

var _ SecurityContext = (*BasicSecurityContext)(nil)

func NewBasicSecurityContext(pubk PubKey, privk PrivKey, did DID) (*BasicSecurityContext, error) {
	sctx := &BasicSecurityContext{
		did:   did,
		privk: privk,
		nonce: uint64(time.Now().UnixNano()),
	}

	var err error
	sctx.id, err = IDFromPublicKey(pubk)
	if err != nil {
		return nil, fmt.Errorf("creating security context: %w", err)
	}

	return sctx, nil
}

func (s *BasicSecurityContext) ID() ID {
	return s.id
}

func (s *BasicSecurityContext) DID() DID {
	return s.did
}

func (s *BasicSecurityContext) Nonce() uint64 {
	s.mx.Lock()
	defer s.mx.Unlock()

	nonce := s.nonce
	s.nonce++

	return nonce
}

func (s *BasicSecurityContext) Require(_ Envelope, _ ...Capability) error {
	//
	// TODO check capability tokens for required capabilities
	//      we do nothing for now, will be implemented in follow up

	return nil
}

func (s *BasicSecurityContext) Provide(msg *Envelope, _ ...Capability) error {
	// TODO provide capability tokes for the required capabilities
	//      we do nothing for now, will be implemented in follow up

	return s.Sign(msg)
}

func (s *BasicSecurityContext) Verify(msg Envelope) error {
	if msg.Expired() {
		return ErrMessageExpired
	}

	pubk, err := PublicKeyFromID(msg.From.ID)
	if err != nil {
		return fmt.Errorf("public key from id: %w", err)
	}

	data, err := msg.signatureData()
	if err != nil {
		return fmt.Errorf("signature data: %w", err)
	}

	ok, err := pubk.Verify(data, msg.Signature)
	if err != nil {
		return fmt.Errorf("verify message signature: %w", err)
	}
	if !ok {
		return ErrSignatureVerification
	}

	return nil
}

func (s *BasicSecurityContext) Sign(msg *Envelope) error {
	data, err := msg.signatureData()
	if err != nil {
		return fmt.Errorf("signature data: %w", err)
	}

	sig, err := s.privk.Sign(data)
	if err != nil {
		return fmt.Errorf("signing message: %w", err)
	}

	msg.Signature = sig
	return nil
}
