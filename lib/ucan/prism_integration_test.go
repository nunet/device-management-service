// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build integration
// +build integration

package ucan

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	prismpb "gitlab.com/nunet/device-management-service/proto/generated/prism"
)

// TestPrismIntegrationFullFlow tests the complete PRISM identity flow:
// 1. Create a PRISM identity (with mock resolver simulating Cardano testnet)
// 2. Import it
// 3. Sign UCAN tokens with it
//
// To test against a real Cardano testnet PRISM agent:
// 1. Set PRISM_TESTNET_URL environment variable
// 2. Create a DID using PRISM agent API
// 3. Export the private key
// 4. Update the test to use real DID and key
func TestPrismIntegrationFullFlow(t *testing.T) {
	// Step 1: Generate a key pair for our PRISM identity
	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key pair")

	pubKeyBytes, err := pubKey.Raw()
	require.NoError(t, err, "get public key bytes")

	// Create a PRISM DID
	// In a real scenario, this would be created on Cardano testnet via PRISM agent
	prismDIDStr := "did:prism:integrationtest123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prismDID, err := did.FromString(prismDIDStr)
	require.NoError(t, err, "parse PRISM DID")

	// Step 2: Create a mock PRISM resolver (simulating Cardano testnet resolver)
	mockResolver := createMockPRISMResolverForIntegration(t, prismDIDStr, pubKeyBytes)
	defer mockResolver.Close()

	// Configure PRISM resolver
	originalConfig := did.GetPRISMResolverConfig()
	defer did.SetPRISMResolverConfig(originalConfig)

	did.SetPRISMResolverConfig(did.PRISMResolverConfig{
		ResolverURL:                 mockResolver.URL,
		PreferredVerificationMethod: "authentication",
	})

	// Step 3: Resolve the PRISM DID to get an anchor
	anchor, err := did.GetAnchorForDID(prismDID)
	require.NoError(t, err, "resolve PRISM DID")
	require.Equal(t, prismDID, anchor.DID(), "anchor DID should match PRISM DID")

	// Step 4: Create a Provider from the PRISM private key
	provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
	require.NoError(t, err, "create PRISM provider")
	require.Equal(t, prismDID, provider.DID(), "provider DID should match PRISM DID")

	// Step 5: Create TrustContext and CapabilityContext for UCAN operations
	trustCtx := did.NewTrustContext()
	trustCtx.AddProvider(provider)
	trustCtx.AddAnchor(anchor)

	// Create capability context with PRISM identity
	capCtx, err := NewCapabilityContext(
		trustCtx,
		provider.DID(),
		[]did.DID{provider.DID()}, // roots
		TokenList{},               // require
		TokenList{},               // provide
		TokenList{},               // revoke
	)
	require.NoError(t, err, "create capability context")
	require.Equal(t, provider.DID(), capCtx.DID(), "capability context DID should match")

	// Step 6: Grant a capability token using PRISM identity
	subjectDID := provider.DID()
	audienceDID := provider.DID()
	expire := uint64(time.Now().Add(1 * time.Hour).UnixNano())
	capability := Capability("/test/prism/capability")

	tokens, err := capCtx.Grant(
		Delegate,
		subjectDID,
		audienceDID,
		nil, // topics
		expire,
		0, // depth
		[]Capability{capability},
	)
	require.NoError(t, err, "grant capability with PRISM identity")
	require.NotEmpty(t, tokens.Tokens, "should have tokens")

	// Step 7: Verify the token was signed by PRISM identity
	token := tokens.Tokens[0]
	require.Equal(t, provider.DID(), token.Issuer(), "token issuer should be PRISM DID")
	require.Equal(t, subjectDID, token.Subject(), "token subject should match")

	// Step 8: Verify the token signature
	now := uint64(time.Now().UnixNano())
	revokeSet := &RevocationSet{revoked: make(map[string]*Token)}
	err = token.Verify(trustCtx, now, revokeSet)
	require.NoError(t, err, "verify UCAN token signed by PRISM identity")

	// Step 9: Test delegation - delegate capabilities to another identity
	// Create a second identity (simulating another party)
	_, otherPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate other key pair")
	otherDID := did.FromPublicKey(otherPubKey)

	// Delegate capabilities to the other identity
	// Use SelfSignOnly since we're self-granting (no external provide tokens)
	delegatedTokens, err := capCtx.Delegate(
		otherDID,
		provider.DID(), // audience
		nil,            // topics
		expire,
		0, // depth
		[]Capability{capability},
		SelfSignOnly, // Self-sign since we're the root authority
	)
	require.NoError(t, err, "delegate capabilities")
	require.NotEmpty(t, delegatedTokens.Tokens, "should have delegated tokens")

	// Verify delegated token
	delegatedToken := delegatedTokens.Tokens[0]
	require.Equal(t, provider.DID(), delegatedToken.Issuer(), "delegated token issuer should be PRISM DID")
	require.Equal(t, otherDID, delegatedToken.Subject(), "delegated token subject should be other DID")

	err = delegatedToken.Verify(trustCtx, now, revokeSet)
	require.NoError(t, err, "verify delegated token")

	t.Logf("Successfully created PRISM identity: %s", prismDIDStr)
	t.Logf("Successfully signed UCAN tokens with PRISM identity")
	t.Logf("Successfully delegated capabilities using PRISM identity")
}

// TestPRISMNeoPRISMIntegration tests the complete PRISM integration with NeoPRISM:
// 1. Creates a DID document via NeoPRISM
// 2. Resolves the document from NeoPRISM after creation
// 3. Tests the rest of UCAN PRISM integration
//
// Prerequisites:
//
//  1. Start NeoPRISM standalone (from compose-testnet.yml):
//     cd neoprism/docker/blockfrost-neoprism-demo
//     docker compose -f compose-testnet.yml up -d
//
//  2. Set environment variables:
//     export PRISM_TESTNET_URL=http://localhost:8080
//
//  3. Ensure NeoPRISM is running and connected to cardano-testnet
func TestPRISMNeoPRISMIntegration(t *testing.T) {
	testnetURL := os.Getenv("PRISM_TESTNET_URL")
	if testnetURL == "" {
		t.Skip("Skipping NeoPRISM integration test. Set PRISM_TESTNET_URL environment variable to enable.")
		return
	}

	// Detect if this is NeoPRISM (port 8080) or OpenPrismNode
	isNeoPRISM := strings.Contains(testnetURL, "8080") || strings.Contains(testnetURL, "neoprism") || strings.Contains(testnetURL, "localhost:8080")

	if !isNeoPRISM {
		t.Skip("This test is for NeoPRISM. Use TestPRISMRealTestnetWithTatumRPC for OpenPrismNode.")
		return
	}

	t.Logf("Testing against NeoPRISM standalone at: %s", testnetURL)

	// Configure resolver for NeoPRISM
	originalConfig := did.GetPRISMResolverConfig()
	defer did.SetPRISMResolverConfig(originalConfig)

	did.SetPRISMResolverConfig(did.PRISMResolverConfig{
		ResolverURL:                 testnetURL,
		PreferredVerificationMethod: "authentication",
	})

	// Step 1: Generate key pair locally
	// Note: NeoPRISM requires Secp256k1 keys for master keys
	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Secp256k1)
	require.NoError(t, err, "generate Secp256k1 key pair")

	t.Logf("✅ Generated Secp256k1 key pair")

	// Step 2: Create a signed PRISM operation
	t.Logf("Creating signed PRISM operation...")
	signedOpHex, err := did.CreateSignedPRISMOperationSimple(privKey, pubKey, "master-0")
	require.NoError(t, err, "create signed PRISM operation")
	require.NotEmpty(t, signedOpHex, "signed operation should not be empty")

	t.Logf("✅ Created signed PRISM operation (hex length: %d)", len(signedOpHex))

	// Step 3: Submit operation to NeoPRISM
	t.Logf("Submitting operation to NeoPRISM...")
	txID, operationIDs, err := submitPRISMOperationToNeoPRISM(testnetURL, signedOpHex)
	require.NoError(t, err, "submit operation to NeoPRISM")
	require.NotEmpty(t, txID, "transaction ID should not be empty")
	require.NotEmpty(t, operationIDs, "operation IDs should not be empty")

	t.Logf("✅ Submitted operation to NeoPRISM")
	t.Logf("   Transaction ID: %s", txID)
	t.Logf("   Operation IDs: %v", operationIDs)

	// Step 4: Extract DID from operation
	// The DID suffix is computed from the operation hash using hexadecimal encoding
	// NeoPRISM expects canonical PRISM DIDs: did:prism:{64-char-hex}
	prismDID, err := extractDIDFromSignedOperation(signedOpHex)
	require.NoError(t, err, "extract DID from operation")
	require.NotEmpty(t, prismDID, "DID should not be empty")

	t.Logf("✅ Extracted PRISM DID: %s", prismDID)

	// Step 5: Wait for transaction to be confirmed and DID to be indexed
	t.Logf("Waiting for transaction to be confirmed on blockchain...")
	// Give the transaction some time to be included in a block
	time.Sleep(3 * time.Second)

	t.Logf("Waiting for DID to be indexed and resolving...")
	prismDIDObj, err := did.FromString(prismDID)
	require.NoError(t, err, "parse PRISM DID")

	var anchor did.Anchor
	maxRetries := 1200 // Increased retries for blockchain indexing
	retryDelay := 1 * time.Second
	initialDelay := 30 * time.Second // Wait a bit longer initially

	// First, wait a bit for the transaction to propagate
	time.Sleep(initialDelay)

	// Try to resolve the DID using our resolver
	for i := 0; i < maxRetries; i++ {
		// Try to resolve using GetAnchorForDID which uses our resolver
		anchor, err = did.GetAnchorForDID(prismDIDObj)
		if err == nil {
			t.Logf("✅ Successfully resolved PRISM DID from NeoPRISM (attempt %d/%d)", i+1, maxRetries)
			require.Equal(t, prismDIDObj, anchor.DID(), "resolved DID should match")

			// Verify the anchor has a valid public key
			anchorPubKey := anchor.PublicKey()
			require.NotNil(t, anchorPubKey, "anchor should have a public key")

			// Verify the public key matches what we used to create the operation
			anchorPubKeyBytes, err := anchorPubKey.Raw()
			require.NoError(t, err, "get anchor public key bytes")

			originalPubKeyBytes, err := pubKey.Raw()
			require.NoError(t, err, "get original public key bytes")
			require.Equal(t, originalPubKeyBytes, anchorPubKeyBytes, "anchor public key should match the key used to create the operation")

			break
		}

		if i == 0 {
			t.Logf("Note: Error resolving PRISM DID (will retry): %v", err)
			t.Logf("   Resolution URL: %s/api/dids/%s", testnetURL, prismDID)
		}

		if i < maxRetries-1 {
			if i%5 == 0 { // Log every 5th attempt
				t.Logf("Waiting for DID to be indexed (attempt %d/%d, error: %v)...", i+1, maxRetries, err)
			}
			time.Sleep(retryDelay)
		} else {
			// Last attempt failed - provide detailed troubleshooting info
			t.Logf("❌ Failed to resolve DID after %d attempts", maxRetries)
			t.Logf("   Last error: %v", err)
			t.Logf("   Transaction ID: %s", txID)
			t.Logf("   Operation IDs: %v", operationIDs)
			t.Logf("   Expected DID: %s", prismDID)
			t.Logf("   Resolution URL: %s/api/dids/%s", testnetURL, prismDID)
			t.Fatalf("Failed to resolve DID after %d attempts: %v\n\n"+
				"Troubleshooting:\n"+
				"1. Ensure NeoPRISM is running: docker ps | grep neoprism\n"+
				"2. Check NeoPRISM logs: docker logs neoprism --tail 50\n"+
				"3. Verify transaction was confirmed: check cardano-testnet logs\n"+
				"4. Try manual resolution: curl -v %s/api/dids/%s\n"+
				"5. Check if DID was indexed (may take time after blockchain confirmation)\n"+
				"6. Ensure cardano-wallet has funds (NeoPRISM needs UTXOs to submit transactions)",
				maxRetries, err, testnetURL, prismDID)
		}
	}

	// Step 6: Create Provider from private key
	// prismDIDObj is already parsed above

	provider, err := did.ProviderFromPRISMPrivateKey(prismDIDObj, privKey)
	require.NoError(t, err, "create PRISM provider")
	require.Equal(t, prismDIDObj, provider.DID(), "provider DID should match")

	t.Logf("✅ Created Provider from PRISM private key")

	// Step 7: Test signing
	testMessage := []byte("test message for NeoPRISM integration")
	signature, err := provider.Sign(testMessage)
	require.NoError(t, err, "sign message with PRISM identity")

	err = anchor.Verify(testMessage, signature)
	require.NoError(t, err, "verify signature")

	t.Logf("✅ Successfully signed and verified message with PRISM identity")

	// Step 8: Create TrustContext and sign UCAN tokens
	trustCtx := did.NewTrustContext()
	trustCtx.AddProvider(provider)
	trustCtx.AddAnchor(anchor)

	capCtx, err := NewCapabilityContext(
		trustCtx,
		provider.DID(),
		[]did.DID{provider.DID()},
		TokenList{},
		TokenList{},
		TokenList{},
	)
	require.NoError(t, err, "create capability context")

	// Step 9: Grant a capability token using the PRISM identity
	expire := uint64(time.Now().Add(1 * time.Hour).UnixNano())
	capability := Capability("/test/prism/neoprism/integration")

	tokens, err := capCtx.Grant(
		Delegate,
		provider.DID(),
		provider.DID(),
		nil,
		expire,
		0,
		[]Capability{capability},
	)
	require.NoError(t, err, "grant capability with PRISM identity")
	require.NotEmpty(t, tokens.Tokens, "should have tokens")

	// Step 10: Verify the token
	token := tokens.Tokens[0]
	require.Equal(t, provider.DID(), token.Issuer(), "token should be issued by PRISM DID")

	now := uint64(time.Now().UnixNano())
	revokeSet := &RevocationSet{revoked: make(map[string]*Token)}
	err = token.Verify(trustCtx, now, revokeSet)
	require.NoError(t, err, "verify UCAN token signed by PRISM identity")

	t.Logf("✅ Successfully created PRISM identity with NeoPRISM: %s", prismDID)
	t.Logf("✅ Successfully signed and verified UCAN tokens with PRISM identity")

	// Step 11: Test delegation - delegate capabilities to another identity
	_, otherPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate other key pair")
	otherDID := did.FromPublicKey(otherPubKey)

	// Delegate capabilities to the other identity
	delegatedTokens, err := capCtx.Delegate(
		otherDID,
		provider.DID(), // audience
		nil,            // topics
		expire,
		0, // depth
		[]Capability{capability},
		SelfSignOnly, // Self-sign since we're the root authority
	)
	require.NoError(t, err, "delegate capabilities")
	require.NotEmpty(t, delegatedTokens.Tokens, "should have delegated tokens")

	// Verify delegated token
	delegatedToken := delegatedTokens.Tokens[0]
	require.Equal(t, provider.DID(), delegatedToken.Issuer(), "delegated token issuer should be PRISM DID")
	require.Equal(t, otherDID, delegatedToken.Subject(), "delegated token subject should be other DID")

	err = delegatedToken.Verify(trustCtx, now, revokeSet)
	require.NoError(t, err, "verify delegated token")

	t.Logf("✅ Successfully delegated capabilities using PRISM identity")
}

// extractDIDFromSignedOperation extracts the PRISM DID from a signed operation
// The DID suffix is the hexadecimal-encoded SHA256 hash of the operation bytes
// NeoPRISM expects canonical PRISM DIDs in the format: did:prism:{64-char-hex}
func extractDIDFromSignedOperation(signedOpHex string) (string, error) {
	// Decode hex
	signedOpBytes, err := hex.DecodeString(signedOpHex)
	if err != nil {
		return "", fmt.Errorf("decode hex: %w", err)
	}

	// Parse the SignedPrismOperation to get the operation
	var signedOp prismpb.SignedPrismOperation
	if err := proto.Unmarshal(signedOpBytes, &signedOp); err != nil {
		return "", fmt.Errorf("unmarshal signed operation: %w", err)
	}

	if signedOp.Operation == nil {
		return "", fmt.Errorf("operation is nil")
	}

	// Encode the operation to bytes
	operationBytes, err := proto.Marshal(signedOp.Operation)
	if err != nil {
		return "", fmt.Errorf("marshal operation: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(operationBytes)

	// NeoPRISM expects canonical PRISM DIDs with hexadecimal suffix (64 chars)
	// Format: did:prism:{64-char-hex}
	didSuffix := hex.EncodeToString(hash[:])
	return fmt.Sprintf("did:prism:%s", didSuffix), nil
}

// createMockPRISMResolverForIntegration creates a mock PRISM resolver
// that simulates a Cardano testnet PRISM agent
func createMockPRISMResolverForIntegration(t *testing.T, prismDID string, pubKeyBytes []byte) *httptest.Server {
	// Encode public key as base64url for JWK
	pubKeyB64 := base64.RawURLEncoding.EncodeToString(pubKeyBytes)

	// Create a mock DID document matching PRISM format
	didDoc := did.DIDDocument{
		Context: []interface{}{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/jws-2020/v1",
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
		// PRISM agent resolution endpoint: GET /did/{did}
		expectedPath := fmt.Sprintf("/did/%s", prismDID)
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf("DID not found: %s", r.URL.Path)))
			return
		}

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Return DID document with proper content type
		w.Header().Set("Content-Type", "application/did+json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(didDoc)
		require.NoError(t, err, "encode DID document")
	}))

	return server
}

// extractShortFormDID extracts the short-form DID from a long-form DID
// Long-form: did:prism:hash:encoded-state
// Short-form: did:prism:hash
func extractShortFormDID(didStr string) string {
	parts := strings.Split(didStr, ":")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], ":")
	}
	return didStr
}

// Note: createPRISMSignedOperation has been replaced by did.CreateSignedPRISMOperationSimple
// which uses proper protobuf encoding. The old placeholder function is no longer needed.

// submitPRISMOperationToNeoPRISM submits a signed PRISM operation to NeoPRISM
func submitPRISMOperationToNeoPRISM(neoprismURL string, signedOpHex string) (string, []string, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// NeoPRISM API format
	submitURL := fmt.Sprintf("%s/api/signed-operation-submissions", strings.TrimSuffix(neoprismURL, "/"))

	reqBody := map[string]interface{}{
		"signed_operations": []string{signedOpHex},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("submit operation: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("submission failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse NeoPRISM response
	var submitResponse struct {
		TxID         string   `json:"tx_id"`
		OperationIDs []string `json:"operation_ids"`
	}

	if err := json.Unmarshal(respBody, &submitResponse); err != nil {
		return "", nil, fmt.Errorf("parse response: %w", err)
	}

	return submitResponse.TxID, submitResponse.OperationIDs, nil
}
