// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestTrustContext(t *testing.T) {
	ctx := NewTrustContext()

	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	pubDID := FromPublicKey(pubk)
	anchor, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor.DID(), pubDID, "compare anchor DID with pubk DID")

	anchor2, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor2, anchor, "cached anchor must equal the initial")

	provider, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")
	ctx.AddProvider(provider)

	provider2, err := ctx.GetProvider(provider.DID())
	require.NoError(t, err, "get key provider")
	require.Equal(t, provider2, provider, "cached provider must equal the initial")

	require.Equal(t, ctx.Anchors(), []DID{pubDID}, "anchor list")
	require.Equal(t, ctx.Providers(), []DID{pubDID}, "provider list")
}

func TestTrustContextGcRemovesExpiredAnchors(t *testing.T) {
	ctx := NewTrustContext()

	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	did := FromPublicKey(pubk)
	anchor := NewAnchor(did, pubk)
	ctx.AddAnchor(anchor)

	// Force-expire the entry then invoke the internal GC helper.
	btc := ctx.(*BasicTrustContext)
	btc.mx.Lock()
	btc.anchors[did].expire = time.Now().Add(-time.Minute)
	btc.mx.Unlock()

	btc.gcAnchorEntries()

	require.Empty(t, ctx.Anchors(), "expired anchor should be purged")
}

func TestTrustContextGetProviderMissing(t *testing.T) {
	ctx := NewTrustContext()

	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	_, err = ctx.GetProvider(FromPublicKey(pubk))
	require.ErrorIs(t, err, ErrNoProvider)
}

func TestTrustContextConstructors(t *testing.T) {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	// WithPrivateKey constructor
	ctxWithKey, err := NewTrustContextWithPrivateKey(privk)
	require.NoError(t, err)
	require.Len(t, ctxWithKey.Providers(), 1)

	// WithProvider constructor
	prov, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err)

	ctxWithProv := NewTrustContextWithProvider(prov)
	require.Len(t, ctxWithProv.Providers(), 1)
}

//nolint:revive // 't' is required for proper test context even if not used directly
func TestTrustContextConcurrentAccess(t *testing.T) {
	ctx := NewTrustContext()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, pubk, _ := crypto.GenerateKeyPair(crypto.Ed25519)
			did := FromPublicKey(pubk)
			ctx.AddAnchor(NewAnchor(did, pubk))
			_ = ctx.Anchors()   // read
			_ = ctx.Providers() // read
		}()
	}
	wg.Wait()
}

func TestTrustContextStartAndAutoGc(t *testing.T) {
	ctx := NewTrustContext()

	// Create a disposable anchor and mark it expired.
	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	did := FromPublicKey(pubk)
	ctx.AddAnchor(NewAnchor(did, pubk))

	// Force-expire it now.
	btc := ctx.(*BasicTrustContext)
	btc.mx.Lock()
	btc.anchors[did].expire = time.Now().Add(-time.Minute)
	btc.mx.Unlock()

	// Start GC with a very short interval.
	ctx.Start(5 * time.Millisecond)
	defer ctx.Stop()

	// Give the goroutine a moment to tick.
	time.Sleep(20 * time.Millisecond)

	require.Empty(t, ctx.Anchors(), "background GC should purge expired anchor")
}

func TestTrustContextGetAnchorRefreshesExpiry(t *testing.T) {
	ctx := NewTrustContext()

	// Generate a new key-based DID (will be fetched via makeKeyAnchor).
	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	did := FromPublicKey(pubk)

	anchor, err := ctx.GetAnchor(did)
	require.NoError(t, err)

	// Manually age the entry far in the past.
	btc := ctx.(*BasicTrustContext)
	btc.mx.Lock()
	btc.anchors[did].expire = time.Now().Add(-time.Hour)
	btc.mx.Unlock()

	// Calling GetAnchor again should *refresh* expire, making it >= now+TTL/2.
	anchorAgain, err := ctx.GetAnchor(did)
	require.NoError(t, err)
	require.Equal(t, anchor, anchorAgain, "should return cached anchor")

	btc.mx.Lock()
	defer btc.mx.Unlock()
	require.True(t, btc.anchors[did].expire.After(time.Now()),
		"expiry timestamp should be pushed forward by GetAnchor")
}

// Allow some leeway due to runtime noise on goroutine count
func TestTrustContextRestartAndDoubleStop(t *testing.T) {
	ctx := NewTrustContext().(*BasicTrustContext)

	// Start first GC goroutine
	gBefore := runtime.NumGoroutine()
	ctx.Start(5 * time.Millisecond)
	firstStop := ctx.stop

	// Call Start again; it should cancel the first loop and start a new one.
	// We only check that stop func pointer is non-nil and Start doesn’t panic.
	require.NotNil(t, firstStop)
	require.NotPanics(t, func() { ctx.Start(5 * time.Millisecond) })

	// Stop once – should cancel the active loop and nil-out ctx.stop
	ctx.Stop()
	require.Nil(t, ctx.stop, "Stop should nil-out the stopper")

	// Calling Stop again should be a harmless no-op (no panic)
	require.NotPanics(t, ctx.Stop)

	// Allow goroutines to settle and ensure GC loop(s) exited.
	time.Sleep(10 * time.Millisecond)
	require.LessOrEqual(t, runtime.NumGoroutine(), gBefore+1,
		"GC goroutine should have exited")
}
