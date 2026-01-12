// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ucan

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
)

func TestBasicUCAN(t *testing.T) {
	root := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root, Capability("/test/invoke"))
	actor2 := makeActorCapabilityContext(t, root, Capability("/test/invoke"))

	actor1ID := makeActorID(t)
	actor2ID := makeActorID(t)

	actorCap, err := actor1.Provide(
		actor2.DID(),
		actor1ID,
		actor2ID,
		makeExpiry(30*time.Second),
		[]Capability{Capability("/test/invoke")},
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor2.Require(
		actor2.DID(),
		actor1ID,
		actor2ID,
		[]Capability{Capability("/test/invoke")},
	)
	require.NoError(t, err, "require")

	actorCap, err = actor2.Provide(
		actor1.DID(),
		actor2ID,
		actor1ID,
		makeExpiry(20*time.Second),
		[]Capability{Capability("/test/reply")},
		nil,
	)
	require.NoError(t, err, "provide")

	err = actor1.Consume(actor2.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor1.Require(
		actor1.DID(),
		actor2ID,
		actor1ID,
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "require")
}

func TestTokenDiscard(t *testing.T) {
	root := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root, Capability("/test/invoke"))
	actor2 := makeActorCapabilityContext(t, root, Capability("/test/invoke"))

	actor1ID := makeActorID(t)
	actor2ID := makeActorID(t)

	actorCap, err := actor1.Provide(
		actor2.DID(),
		actor1ID,
		actor2ID,
		makeExpiry(30*time.Second),
		[]Capability{Capability("/test/invoke")},
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	actor2Ctx := actor2.(*BasicCapabilityContext)
	require.Greater(t, len(actor2Ctx.tokens), 0, "token store is not empty")

	actor2.Discard(actorCap)
	require.Equal(t, len(actor2Ctx.tokens), 0, "token store is empty")
}

func TestReciprocalUCAN(t *testing.T) {
	root1 := makeCapabilityContext(t)
	root2 := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root1, Capability("/test/invoke"))
	actor2 := makeActorCapabilityContext(t, root2, Capability("/test/invoke"))
	allowReciprocal(t, actor1, root1, root2, Capability("/test/invoke"))
	allowReciprocal(t, actor2, root2, root1, Capability("/test/invoke"))

	actor1ID := makeActorID(t)
	actor2ID := makeActorID(t)

	actorCap, err := actor1.Provide(
		actor2.DID(),
		actor1ID,
		actor2ID,
		makeExpiry(30*time.Second),
		[]Capability{Capability("/test/invoke")},
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor2.Require(
		actor2.DID(),
		actor1ID,
		actor2ID,
		[]Capability{Capability("/test/invoke")},
	)
	require.NoError(t, err, "require")

	actorCap, err = actor2.Provide(
		actor1.DID(),
		actor2ID,
		actor1ID,
		makeExpiry(20*time.Second),
		[]Capability{Capability("/test/reply")},
		nil,
	)
	require.NoError(t, err, "provide")

	err = actor1.Consume(actor2.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor1.Require(
		actor1.DID(),
		actor2ID,
		actor1ID,
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "require")
}

func TestReciprocalDistrust(t *testing.T) {
	root1 := makeCapabilityContext(t)
	root2 := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root1, Capability("/test/invoke"))
	actor2 := makeActorCapabilityContext(t, root2, Capability("/test/invoke"))

	actor1ID := makeActorID(t)
	actor2ID := makeActorID(t)

	actorCap, err := actor1.Provide(
		actor2.DID(),
		actor1ID,
		actor2ID,
		makeExpiry(30*time.Second),
		[]Capability{Capability("/test/invoke")},
		[]Capability{Capability("/test/reply")},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor2.Require(
		actor2.DID(),
		actor1ID,
		actor2ID,
		[]Capability{Capability("/test/invoke")},
	)
	require.Error(t, err, "require")
}

func TestBroadcastUCAN(t *testing.T) {
	topic := "test"
	capability := Capability("/test/broadcast")

	root1 := makeCapabilityContext(t)
	root2 := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root1)
	actor2 := makeActorCapabilityContext(t, root2)
	allowBroadcast(t, actor1, actor2, root1, root2, topic, capability)

	actor1ID := makeActorID(t)
	actorCap, err := actor1.ProvideBroadcast(
		actor1ID,
		topic,
		makeExpiry(30*time.Second),
		[]Capability{capability},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor2.RequireBroadcast(
		actor2.DID(),
		actor1ID,
		topic,
		[]Capability{capability},
	)
	require.NoError(t, err, "require")
}

func TestBroadcastDistrust(t *testing.T) {
	topic := "test"
	capability := Capability("/test/broadcast")

	root1 := makeCapabilityContext(t)
	root2 := makeCapabilityContext(t)
	actor1 := makeActorCapabilityContext(t, root1)
	actor2 := makeActorCapabilityContext(t, root2)

	tokens, err := root1.Grant(
		Delegate,
		actor1.DID(),
		did.DID{},
		[]string{topic},
		makeExpiry(120*time.Second),
		0,
		[]Capability{capability},
	)
	require.NoError(t, err, "granting broadcast capability")

	err = actor1.AddRoots(nil, TokenList{}, tokens, TokenList{})
	require.NoError(t, err, "add roots")

	actor1ID := makeActorID(t)
	actorCap, err := actor1.ProvideBroadcast(
		actor1ID,
		topic,
		makeExpiry(30*time.Second),
		[]Capability{capability},
	)
	require.NoError(t, err, "provide")

	err = actor2.Consume(actor1.DID(), actorCap)
	require.NoError(t, err, "consume")

	err = actor2.RequireBroadcast(
		actor2.DID(),
		actor1ID,
		topic,
		[]Capability{capability},
	)
	require.Error(t, err, "require")
}

func TestDelegationDepth(t *testing.T) {
	root1 := makeCapabilityContext(t)
	root2 := makeCapabilityContext(t)
	root3 := makeCapabilityContext(t)

	expiry := makeExpiry(120 * time.Second)
	capabilities := []Capability{Capability("/test")}
	topic := "/broadcast/test"
	topics := []string{topic}

	tokens, err := root1.Grant(
		Delegate,
		root2.DID(),
		did.DID{},
		topics,
		expiry,
		1,
		capabilities,
	)
	require.NoError(t, err, "grant")

	err = root2.AddRoots(nil, TokenList{}, tokens, TokenList{})
	require.NoError(t, err, "provide anchor")

	_, err = root2.DelegateInvocation(
		root3.DID(),
		root3.DID(),
		did.DID{},
		expiry,
		capabilities,
		SelfSignNo,
	)
	require.NoError(t, err, "delegate invocation")

	_, err = root2.DelegateBroadcast(
		root3.DID(),
		topic,
		expiry,
		capabilities,
		SelfSignNo,
	)
	require.NoError(t, err, "delegate broadcast")

	_, err = root2.Delegate(
		root3.DID(),
		did.DID{},
		topics,
		expiry,
		0,
		capabilities,
		SelfSignNo,
	)
	require.Error(t, err, "delegate")
}

func makeTrustContext(t *testing.T) (did.DID, did.TrustContext) {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	provider, err := did.ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")

	ctx := did.NewTrustContext()
	ctx.AddProvider(provider)

	return provider.DID(), ctx
}

func makeCapabilityContext(t *testing.T) CapabilityContext {
	rootDID, trustCtx := makeTrustContext(t)

	capCtx, err := NewCapabilityContext(trustCtx, rootDID, nil, TokenList{}, TokenList{}, TokenList{})
	require.NoError(t, err, "make capability context")

	return capCtx
}

func makeExpiry(d time.Duration) uint64 {
	return uint64(time.Now().Add(d).UnixNano())
}

func makeActorCapabilityContext(t *testing.T, rootCtx CapabilityContext, actorCap ...Capability) CapabilityContext {
	actorDID, actorTrustCtx := makeTrustContext(t)

	tokens, err := rootCtx.Grant(
		Delegate,
		actorDID,
		did.DID{},
		nil,
		makeExpiry(120*time.Second),
		0,
		actorCap,
	)
	require.NoError(t, err, "granting capabilities to actor")

	actorCtx, err := NewCapabilityContext(
		actorTrustCtx,
		actorDID,
		[]did.DID{rootCtx.DID()},
		TokenList{},
		tokens,
		TokenList{},
	)
	require.NoError(t, err, "adding roots for actor")

	return actorCtx
}

func allowReciprocal(t *testing.T, actor, root, otherRoot CapabilityContext, actorCap ...Capability) {
	tokens, err := root.Grant(
		Delegate,
		otherRoot.DID(),
		did.DID{},
		nil,
		makeExpiry(120*time.Second),
		0,
		actorCap)
	require.NoError(t, err, "granting reciprocal capabilities")

	err = actor.AddRoots(nil, tokens, TokenList{}, TokenList{})
	require.NoError(t, err, "add roots")
}

func allowBroadcast(t *testing.T, actor1, actor2, root1, root2 CapabilityContext, topic string, actorCap ...Capability) {
	tokens, err := root1.Grant(
		Delegate,
		actor1.DID(),
		did.DID{},
		[]string{topic},
		makeExpiry(120*time.Second),
		0,
		actorCap,
	)
	require.NoError(t, err, "granting broadcast capability")

	err = actor1.AddRoots(nil, TokenList{}, tokens, TokenList{})
	require.NoError(t, err, "add roots")

	tokens, err = root2.Grant(
		Delegate,
		root1.DID(),
		did.DID{},
		[]string{topic},
		makeExpiry(120*time.Second),
		0,
		actorCap,
	)
	require.NoError(t, err, "granting broadcast capability")

	err = actor2.AddRoots(nil, tokens, TokenList{}, TokenList{})
	require.NoError(t, err, "add roots")
}

func makeActorID(t *testing.T) crypto.ID {
	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	id, err := crypto.IDFromPublicKey(pubk)
	require.NoError(t, err, "id from public key")

	return id
}

func makeActorIDFromDID(t *testing.T, d did.DID) crypto.ID {
	pbkey, err := did.PublicKeyFromDID(d)
	require.NoError(t, err)

	id, err := crypto.IDFromPublicKey(pbkey)
	require.NoError(t, err)

	return id
}

func createToken(t *testing.T, issuer CapabilityContext,
	subjectDID, audienceDID did.DID, capability Capability, expiry uint64,
) *Token {
	nonce := make([]byte, nonceLength)
	_, err := rand.Read(nonce)
	require.NoError(t, err)

	token := &Token{
		DMS: &DMSToken{
			Issuer:     issuer.DID(),
			Subject:    subjectDID,
			Audience:   audienceDID,
			Action:     Delegate,
			Capability: []Capability{capability},
			Expire:     expiry,
			Nonce:      nonce,
		},
	}
	data, err := token.DMS.SignatureData()
	require.NoError(t, err)

	provider, err := issuer.Trust().GetProvider(issuer.DID())
	require.NoError(t, err)

	token.DMS.Signature, err = provider.Sign(data)
	require.NoError(t, err)

	return token
}

// TestMultiIdentityDelegationChain tests a delegation chain with multiple grantors:
//
// dms0 and dms1 grant to dms2
// dms2 delegates to dms3
// dms3 invokes on dms0 and dms1 (should succeed)
//
// This tests that dms3 can invoke capabilities on both dms0 and dms1
// through the delegation chain via dms2.
func TestMultiIdentityDelegationChain(t *testing.T) {
	t.Parallel()

	// Create 4 identities
	dms0 := makeCapabilityContext(t)
	dms1 := makeCapabilityContext(t)
	dms2 := makeCapabilityContext(t)
	dms3 := makeCapabilityContext(t)

	// Get IDs for invocation
	dms0ID := makeActorIDFromDID(t, dms0.DID())
	dms1ID := makeActorIDFromDID(t, dms1.DID())
	dms3ID := makeActorIDFromDID(t, dms3.DID())

	capability := Capability("/test/invoke")
	expiry := makeExpiry(120 * time.Second)

	// Step 1: dms0 grants to dms2
	dms0ToDms2Tokens, err := dms0.Grant(
		Delegate,
		dms2.DID(),
		did.DID{}, // Empty audience allows dms2 to delegate for any audience
		nil,
		expiry,
		0,
		[]Capability{capability},
	)
	require.NoError(t, err, "dms0 granting to dms2")

	// dms2 adds dms0's tokens as provide roots so it can delegate them
	err = dms2.AddRoots(nil, TokenList{}, dms0ToDms2Tokens, TokenList{})
	require.NoError(t, err, "dms2 adding dms0's tokens as provide roots")

	// Step 2: dms1 grants to dms2
	dms1ToDms2Tokens, err := dms1.Grant(
		Delegate,
		dms2.DID(),
		did.DID{}, // Empty audience allows dms2 to delegate for any audience
		nil,
		expiry,
		0,
		[]Capability{capability},
	)
	require.NoError(t, err, "dms1 granting to dms2")

	// dms2 adds dms1's tokens as provide roots so it can delegate them
	err = dms2.AddRoots(nil, TokenList{}, dms1ToDms2Tokens, TokenList{})
	require.NoError(t, err, "dms2 adding dms1's tokens as provide roots")

	// Step 3: dms2 delegates to dms3 (single delegation with empty audience)
	// This should create tokens chained through both dms0 and dms1
	dms2ToDms3Tokens, err := dms2.Delegate(
		dms3.DID(),
		did.DID{}, // Empty audience allows dms3 to invoke on any audience
		nil,
		expiry,
		0,
		[]Capability{capability},
		SelfSignNo, // Use the provide tokens from dms0 and dms1
	)
	require.NoError(t, err, "dms2 delegating to dms3")
	// dms2 should create tokens for both dms0 and dms1 anchors
	require.Equal(t, 2, len(dms2ToDms3Tokens.Tokens), "dms2 should create tokens chained through both dms0 and dms1")

	// dms3 adds dms2's tokens as provide roots
	err = dms3.AddRoots(nil, TokenList{}, dms2ToDms3Tokens, TokenList{})
	require.NoError(t, err, "dms3 adding dms2's tokens as provide roots")

	// Step 4: dms3 invokes on dms0
	// When dms3 provides invocation tokens for dms0, it should use the token chained through dms0
	dms3InvokeDms0Tokens, err := dms3.Provide(
		dms0.DID(),
		dms3ID,
		dms0ID,
		expiry,
		[]Capability{capability},
		nil,
	)
	require.NoError(t, err, "dms3 providing invoke tokens for dms0")

	// dms0 consumes dms3's tokens
	err = dms0.Consume(dms3.DID(), dms3InvokeDms0Tokens)
	require.NoError(t, err, "dms0 consuming dms3's tokens")

	// dms0 requires the capabilities from dms3
	// This should succeed because dms3 has capabilities delegated from dms2,
	// which were granted by dms0
	err = dms0.Require(
		dms0.DID(), // Anchor: dms0 (the grantor)
		dms3ID,     // Subject: dms3 (the invoker)
		dms0ID,     // Audience: dms0 (the target)
		[]Capability{capability},
	)
	require.NoError(t, err, "dms0 requiring capabilities from dms3 - should succeed")

	// Step 5: dms3 invokes on dms1
	dms3InvokeDms1Tokens, err := dms3.Provide(
		dms1.DID(),
		dms3ID,
		dms1ID,
		expiry,
		[]Capability{capability},
		nil,
	)
	require.NoError(t, err, "dms3 providing invoke tokens for dms1")

	// dms1 consumes dms3's tokens
	err = dms1.Consume(dms3.DID(), dms3InvokeDms1Tokens)
	require.NoError(t, err, "dms1 consuming dms3's tokens")

	// dms1 requires the capabilities from dms3
	// This should succeed because dms3 has capabilities delegated from dms2,
	// which were granted by dms1
	err = dms1.Require(
		dms1.DID(), // Anchor: dms1 (the grantor)
		dms3ID,     // Subject: dms3 (the invoker)
		dms1ID,     // Audience: dms1 (the target)
		[]Capability{capability},
	)
	require.NoError(t, err, "dms1 requiring capabilities from dms3 - should succeed")
}
