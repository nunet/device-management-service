package did

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

const anchorEntryTTL = time.Hour

// Anchor is a DID anchor that encapsulates a public key that can be used
// for verification of signatures.
type Anchor interface {
	DID() DID
	Verify(data []byte, sig []byte) error
	PublicKey() crypto.PubKey
}

// Provider holds the private key material necessary to sign statements for
// a DID.
type Provider interface {
	DID() DID
	Sign(data []byte) ([]byte, error)
	Anchor() Anchor
	PrivateKey() crypto.PrivKey
}

type TrustContext interface {
	Anchors() []DID
	Providers() []DID
	GetAnchor(did DID) (Anchor, error)
	GetProvider(did DID) (Provider, error)
	AddAnchor(anchor Anchor)
	AddProvider(provider Provider)

	Start(gcInterval time.Duration)
	Stop()
}

type anchorEntry struct {
	anchor Anchor
	expire time.Time
}

type BasicTrustContext struct {
	mx        sync.Mutex
	anchors   map[DID]*anchorEntry
	providers map[DID]Provider

	stop func()
}

var _ TrustContext = (*BasicTrustContext)(nil)

func NewTrustContext() TrustContext {
	return &BasicTrustContext{
		anchors:   make(map[DID]*anchorEntry),
		providers: make(map[DID]Provider),
	}
}

func (ctx *BasicTrustContext) Anchors() []DID {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	result := make([]DID, 0, len(ctx.anchors))
	for anchor := range ctx.anchors {
		result = append(result, anchor)
	}

	return result
}

func (ctx *BasicTrustContext) Providers() []DID {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	result := make([]DID, 0, len(ctx.providers))
	for provider := range ctx.providers {
		result = append(result, provider)
	}

	return result
}

func (ctx *BasicTrustContext) GetAnchor(did DID) (Anchor, error) {
	anchor, ok := ctx.getAnchor(did)
	if ok {
		return anchor, nil
	}

	anchor, err := GetAnchorForDID(did)
	if err != nil {
		return nil, fmt.Errorf("get anchor for did: %w", err)
	}

	ctx.AddAnchor(anchor)
	return anchor, nil
}

func (ctx *BasicTrustContext) getAnchor(did DID) (Anchor, bool) {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	entry, ok := ctx.anchors[did]
	if ok {
		entry.expire = time.Now().Add(anchorEntryTTL)
		return entry.anchor, true
	}

	return nil, false
}

func (ctx *BasicTrustContext) GetProvider(did DID) (Provider, error) {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	provider, ok := ctx.providers[did]
	if !ok {
		return nil, ErrNoProvider
	}

	return provider, nil
}

func (ctx *BasicTrustContext) AddAnchor(anchor Anchor) {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	ctx.anchors[anchor.DID()] = &anchorEntry{
		anchor: anchor,
		expire: time.Now().Add(anchorEntryTTL),
	}
}

func (ctx *BasicTrustContext) AddProvider(provider Provider) {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	ctx.providers[provider.DID()] = provider
}

func (ctx *BasicTrustContext) Start(gcInterval time.Duration) {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	if ctx.stop != nil {
		ctx.stop()
	}

	gcCtx, stop := context.WithCancel(context.Background())
	ctx.stop = stop
	go ctx.gc(gcCtx, gcInterval)
}

func (ctx *BasicTrustContext) Stop() {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	if ctx.stop != nil {
		ctx.stop()
		ctx.stop = nil
	}
}

func (ctx *BasicTrustContext) gc(gcCtx context.Context, gcInterval time.Duration) {
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx.gcAnchorEntries()
		case <-gcCtx.Done():
			return
		}
	}
}

func (ctx *BasicTrustContext) gcAnchorEntries() {
	ctx.mx.Lock()
	defer ctx.mx.Unlock()

	now := time.Now()
	for k, e := range ctx.anchors {
		if e.expire.Before(now) {
			delete(ctx.anchors, k)
		}
	}
}
