// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ucan

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
)

// createMockPRISMResolverForIntegration creates a mock PRISM resolver
// that simulates a Cardano testnet PRISM agent
func createMockPRISMResolverForIntegration(t *testing.T, prismDID string, pubKeyBytes []byte) *httptest.Server {
	// Encode public key as base64url for JWK
	pubKeyB64 := base64.RawURLEncoding.EncodeToString(pubKeyBytes)

	// Create a mock DID document matching PRISM format
	didDoc := did.DIDDocument{
		Context: []interface{}{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/jws2020/v1",
			"https://didcomm.org/messaging/contexts/v2",
		},
		ID: prismDID,
		VerificationMethod: []did.VerificationMethod{
			{
				ID:         fmt.Sprintf("%s#authentication0", prismDID),
				Type:       "JsonWebKey2020",
				Controller: prismDID,
				PublicKeyJWK: json.RawMessage(fmt.Sprintf(`{
	                                       "kty": "OKP",
	                                       "crv": "Ed25519",
	                                       "x": "%s"
	                               }`, pubKeyB64)),
			},
		},
		Authentication: []did.VerificationMethodRef{
			{
				ID: fmt.Sprintf("%s#authentication0", prismDID),
			},
		},
	}

	// Create HTTP server that mocks PRISM agent DID resolution endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// PRISM agent resolution endpoint: GET /api/did/{did}
		expectedPath := fmt.Sprintf("/api/dids/%s", prismDID)
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf("DID not found: %s", r.URL.Path))) //nolint:staticcheck
			return
		}

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Return DID document with proper content type
		w.Header().Set("ContentType", "application/did+json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(didDoc)
		require.NoError(t, err, "encode DID document")
	}))

	return server
}

// Note: createMockPRISMResolverForIntegration is defined in prism_integration_test.go

// TestMixedDIDMethods demonstrates that PRISM and did:key identities can interact seamlessly
// This test shows:
// 1. A PRISM identity (issuer) can delegate to a did:key identity (subject)
// 2. A did:key identity (issuer) can delegate to a PRISM identity (subject)
// 3. Token chains can mix methods freely
func TestMixedDIDMethods(t *testing.T) {
	// Setup: Create both a PRISM and a did:key identity
	prismPrivKey, prismPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	keyPrivKey, keyPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	// Create PRISM DID
	prismDIDStr := "did:prism:mixedtest123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prismDID, err := did.FromString(prismDIDStr)
	require.NoError(t, err)

	// Create did:key DID from the public key
	keyDID := did.FromPublicKey(keyPubKey)

	// Setup mock PRISM resolver
	prismPubKeyBytes, err := prismPubKey.Raw()
	require.NoError(t, err)
	// Use the helper from prism_integration_test.go (same package, so accessible)
	mockResolver := createMockPRISMResolverForIntegration(t, prismDIDStr, prismPubKeyBytes)
	defer mockResolver.Close()

	originalConfig := did.GetPRISMResolverConfig()
	defer did.SetPRISMResolverConfig(originalConfig)

	did.SetPRISMResolverConfig(did.PRISMResolverConfig{
		ResolverURL:                 mockResolver.URL,
		PreferredVerificationMethod: "authentication",
	})

	// Create TrustContext with both identities
	trustCtx := did.NewTrustContext()

	// Add PRISM provider and anchor
	prismProvider, err := did.ProviderFromPRISMPrivateKey(prismDID, prismPrivKey)
	require.NoError(t, err)
	prismAnchor, err := did.GetAnchorForDID(prismDID)
	require.NoError(t, err)
	trustCtx.AddProvider(prismProvider)
	trustCtx.AddAnchor(prismAnchor)

	// Add did:key provider and anchor
	keyProvider, err := did.ProviderFromPrivateKey(keyPrivKey)
	require.NoError(t, err)
	keyAnchor, err := did.GetAnchorForDID(keyDID)
	require.NoError(t, err)
	trustCtx.AddProvider(keyProvider)
	trustCtx.AddAnchor(keyAnchor)

	t.Run("PRISM delegates to did:key", func(t *testing.T) {
		// Scenario: PRISM identity (issuer) delegates capability to did:key identity (subject)
		prismCapCtx, err := NewCapabilityContext(
			trustCtx,
			prismDID,
			[]did.DID{prismDID}, // roots
			TokenList{},         // require
			TokenList{},         // provide
			TokenList{},         // revoke
		)
		require.NoError(t, err)

		capb := Capability("/test/resource")
		expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())

		// PRISM signs a token delegating to did:key
		tokenList, err := prismCapCtx.Delegate(
			keyDID, // Subject: did:key identity
			keyDID, // Audience: did:key identity
			nil,    // topics
			expiry,
			0,                  // depth
			[]Capability{capb}, // capabilities
			SelfSignOnly,       // self-sign since we're the root
		)
		require.NoError(t, err, "PRISM should be able to delegate to did:key")
		require.NotEmpty(t, tokenList.Tokens, "should have tokens")

		token := tokenList.Tokens[0]

		// Verify the token
		revokeSet := &RevocationSet{revoked: make(map[string]*Token)}
		now := uint64(time.Now().UnixNano())
		err = token.Verify(trustCtx, now, revokeSet)
		require.NoError(t, err, "Token signed by PRISM should verify correctly")

		// Verify issuer is PRISM and subject is did:key
		require.True(t, token.Issuer().Equal(prismDID), "Issuer should be PRISM DID")
		require.True(t, token.Subject().Equal(keyDID), "Subject should be did:key DID")
	})

	t.Run("did:key delegates to PRISM", func(t *testing.T) {
		// Scenario: did:key identity (issuer) delegates capability to PRISM identity (subject)
		keyCapCtx, err := NewCapabilityContext(
			trustCtx,
			keyDID,
			[]did.DID{keyDID}, // roots
			TokenList{},       // require
			TokenList{},       // provide
			TokenList{},       // revoke
		)
		require.NoError(t, err)

		capb := Capability("/test/resource")
		expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())

		// did:key signs a token delegating to PRISM
		tokenList, err := keyCapCtx.Delegate(
			prismDID, // Subject: PRISM identity
			prismDID, // Audience: PRISM identity
			nil,      // topics
			expiry,
			0,                  // depth
			[]Capability{capb}, // capabilities
			SelfSignOnly,       // self-sign since we're the root
		)
		require.NoError(t, err, "did:key should be able to delegate to PRISM")
		require.NotEmpty(t, tokenList.Tokens, "should have tokens")

		token := tokenList.Tokens[0]

		// Verify the token
		revokeSet := &RevocationSet{revoked: make(map[string]*Token)}
		now := uint64(time.Now().UnixNano())
		err = token.Verify(trustCtx, now, revokeSet)
		require.NoError(t, err, "Token signed by did:key should verify correctly")

		// Verify issuer is did:key and subject is PRISM
		require.True(t, token.Issuer().Equal(keyDID), "Issuer should be did:key DID")
		require.True(t, token.Subject().Equal(prismDID), "Subject should be PRISM DID")
	})

	t.Run("Mixed method token chain", func(t *testing.T) {
		// Scenario: Create a token chain where each token uses a different DID method
		// Chain: PRISM -> did:key (delegation chain)

		// Step 1: PRISM grants capability to did:key (self-signed, no chain needed)
		prismCapCtx, err := NewCapabilityContext(
			trustCtx,
			prismDID,
			[]did.DID{prismDID}, // roots
			TokenList{},         // require
			TokenList{},         // provide
			TokenList{},         // revoke
		)
		require.NoError(t, err)

		capb := Capability("/test/resource")
		expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())

		// PRISM grants to did:key (creates a token with PRISM as issuer, did:key as subject)
		prismToKeyTokens, err := prismCapCtx.Grant(
			Delegate,
			keyDID,
			prismDID,
			nil, // topics
			expiry,
			0,                  // depth
			[]Capability{capb}, // capabilities
		)
		require.NoError(t, err, "PRISM should grant to did:key")
		require.NotEmpty(t, prismToKeyTokens.Tokens, "should have tokens")

		// Step 2: did:key adds PRISM's tokens as provide roots, then delegates
		keyCapCtx, err := NewCapabilityContext(
			trustCtx,
			keyDID,
			[]did.DID{keyDID}, // roots
			TokenList{},       // require
			TokenList{},       // provide
			TokenList{},       // revoke
		)
		require.NoError(t, err)

		// Add PRISM's tokens as provide tokens so did:key can delegate through them
		err = keyCapCtx.AddRoots(
			[]did.DID{prismDID},
			TokenList{},
			prismToKeyTokens, // provide tokens from PRISM
			TokenList{},
		)
		require.NoError(t, err, "did:key should add PRISM tokens as provide")

		// Create a third identity to delegate to (to show the chain works)
		thirdPrivKey, thirdPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)
		thirdDID := did.FromPublicKey(thirdPubKey)
		thirdProvider, err := did.ProviderFromPrivateKey(thirdPrivKey)
		require.NoError(t, err)
		thirdAnchor, err := did.GetAnchorForDID(thirdDID)
		require.NoError(t, err)
		trustCtx.AddProvider(thirdProvider)
		trustCtx.AddAnchor(thirdAnchor)

		// did:key delegates to third identity, chaining through PRISM's token
		keyToThirdTokens, err := keyCapCtx.Delegate(
			thirdDID,
			prismDID, // audience is PRISM (the original grantor)
			nil,      // topics
			expiry,
			0,                  // depth
			[]Capability{capb}, // capabilities
			SelfSignNo,         // use provided tokens (chain through PRISM's token)
		)
		require.NoError(t, err, "did:key should delegate through PRISM token")
		require.NotEmpty(t, keyToThirdTokens.Tokens, "should have delegated tokens")

		// Verify the chain token
		chainToken := keyToThirdTokens.Tokens[0]
		revokeSet := &RevocationSet{revoked: make(map[string]*Token)}
		now := uint64(time.Now().UnixNano())
		err = chainToken.Verify(trustCtx, now, revokeSet)
		require.NoError(t, err, "Mixed method token chain should verify correctly")

		// Verify chain structure: did:key -> third, chaining through PRISM -> did:key
		require.True(t, chainToken.Issuer().Equal(keyDID), "Chain issuer should be did:key")
		require.True(t, chainToken.Subject().Equal(thirdDID), "Chain subject should be third DID")
		require.NotNil(t, chainToken.DMS, "Token should have DMS")
		require.NotNil(t, chainToken.DMS.Chain, "Token should have a chain")
		require.True(t, chainToken.DMS.Chain.Issuer().Equal(prismDID), "Chain issuer should be PRISM")
		require.True(t, chainToken.DMS.Chain.Subject().Equal(keyDID), "Chain subject should be did:key")
	})
}
