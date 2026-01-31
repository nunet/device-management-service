// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/ed25519"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

// PRISMResolverConfig holds configuration for PRISM DID resolution
type PRISMResolverConfig struct {
	// ResolverURL is the base URL of the PRISM DID resolver
	// Example: "https://prism-agent.example.com"
	ResolverURL string

	// HTTPClient is the HTTP client to use for resolution
	// If nil, a default client with 30s timeout will be used
	HTTPClient *http.Client

	// PreferredVerificationMethod specifies which verification method to prefer
	// when multiple are available. Options: "authentication", "assertionMethod", "capabilityInvocation"
	// If empty, defaults to "authentication"
	PreferredVerificationMethod string
}

var defaultPRISMResolverConfig = PRISMResolverConfig{
	ResolverURL:                 "https://prism-agent.example.com",
	PreferredVerificationMethod: "authentication",
}

// DIDDocument represents a W3C DID Document
type DIDDocument struct { //nolint:revive
	Context              interface{}             `json:"@context"`
	ID                   string                  `json:"id"`
	VerificationMethod   []VerificationMethod    `json:"verificationMethod,omitempty"`
	Authentication       []VerificationMethodRef `json:"authentication,omitempty"`
	AssertionMethod      []VerificationMethodRef `json:"assertionMethod,omitempty"`
	KeyAgreement         []VerificationMethodRef `json:"keyAgreement,omitempty"`
	CapabilityInvocation []VerificationMethodRef `json:"capabilityInvocation,omitempty"`
	CapabilityDelegation []VerificationMethodRef `json:"capabilityDelegation,omitempty"`
	Service              []Service               `json:"service,omitempty"`
}

// VerificationMethod represents a public key in a DID Document
type VerificationMethod struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Controller   string          `json:"controller"`
	PublicKeyJWK json.RawMessage `json:"publicKeyJwk,omitempty"`
}

// VerificationMethodRef can be either a string (ID reference) or a VerificationMethod object
type VerificationMethodRef struct {
	ID           string          `json:"id,omitempty"`
	Type         string          `json:"type,omitempty"`
	Controller   string          `json:"controller,omitempty"`
	PublicKeyJWK json.RawMessage `json:"publicKeyJwk,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for VerificationMethodRef
// It handles both string references (just the ID) and full objects
func (v *VerificationMethodRef) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// It's a string reference, just set the ID
		v.ID = str
		return nil
	}

	// If not a string, unmarshal as an object
	type Alias VerificationMethodRef
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*v = VerificationMethodRef(alias)
	return nil
}

// Service represents a service endpoint in a DID Document
type Service struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`         // Key type: "EC" or "OKP"
	Crv string `json:"crv"`         // Curve: "Ed25519", "secp256k1", "X25519"
	X   string `json:"x"`           // X coordinate (base64url)
	Y   string `json:"y,omitempty"` // Y coordinate (base64url, only for EC keys)
}

// resolvePRISMDID resolves a PRISM DID to its DID document
func resolvePRISMDID(ctx context.Context, did DID, config PRISMResolverConfig) (*DIDDocument, error) {
	// Build resolution URL
	// PRISM resolvers typically use: https://resolver-url/did/{did}
	// OpenPrismNode uses: https://resolver-url/api/v1/identifiers/{did}
	resolverURL := config.ResolverURL
	if resolverURL == "" {
		resolverURL = defaultPRISMResolverConfig.ResolverURL
	}

	// Detect resolver format and build appropriate URL
	// Note: DIDs in URL paths need to be URL-encoded (e.g., : becomes %3A)
	encodedDID := url.PathEscape(did.URI)

	// NeoPRISM format - use W3C-compliant DID resolution endpoint
	// /api/dids/{did} returns W3C-compliant JSON DID resolution result
	resolutionURL := fmt.Sprintf("%s/api/dids/%s", strings.TrimSuffix(resolverURL, "/"), encodedDID)

	// Create HTTP client if not provided
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolutionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/did+json,application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resolve DID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DID resolution failed with status %d", resp.StatusCode)
	}

	// Parse response
	var doc DIDDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse DID document: %w", err)
	}

	// Validate DID matches
	if doc.ID != did.URI {
		return nil, fmt.Errorf("DID document ID mismatch: expected %s, got %s", did.URI, doc.ID)
	}

	return &doc, nil
}

// extractPublicKeyFromDIDDocument extracts a public key from a PRISM DID document
func extractPublicKeyFromDIDDocument(doc *DIDDocument, config PRISMResolverConfig) (crypto.PubKey, error) {
	// Determine which verification methods to check
	var methodRefs []VerificationMethodRef
	preferred := config.PreferredVerificationMethod
	if preferred == "" {
		preferred = defaultPRISMResolverConfig.PreferredVerificationMethod
	}

	switch preferred {
	case "authentication":
		methodRefs = doc.Authentication
	case "assertionMethod":
		methodRefs = doc.AssertionMethod
	case "capabilityInvocation":
		methodRefs = doc.CapabilityInvocation
	default:
		// Fallback to authentication
		methodRefs = doc.Authentication
	}

	// Try to find a verification method in the preferred relationship
	for _, ref := range methodRefs {
		var vm *VerificationMethod

		// Handle case where ref might be embedded (has PublicKeyJWK)
		if ref.PublicKeyJWK != nil {
			// Embedded verification method
			vm = &VerificationMethod{
				ID:           ref.ID,
				Type:         ref.Type,
				Controller:   ref.Controller,
				PublicKeyJWK: ref.PublicKeyJWK,
			}
		} else if ref.ID != "" {
			// Look up by ID in verificationMethod array
			for i := range doc.VerificationMethod {
				if doc.VerificationMethod[i].ID == ref.ID {
					vm = &doc.VerificationMethod[i]
					break
				}
			}
		}

		if vm != nil {
			if pubk, err := extractPublicKeyFromVerificationMethod(vm); err == nil {
				return pubk, nil
			}
		}
	}

	// Fallback: try all verification methods
	for i := range doc.VerificationMethod {
		if pubk, err := extractPublicKeyFromVerificationMethod(&doc.VerificationMethod[i]); err == nil {
			return pubk, nil
		}
	}

	return nil, fmt.Errorf("no supported verification method found in DID document")
}

// extractPublicKeyFromVerificationMethod extracts a public key from a verification method
func extractPublicKeyFromVerificationMethod(vm *VerificationMethod) (crypto.PubKey, error) {
	// Only support JsonWebKey2020 type
	if vm.Type != "JsonWebKey2020" {
		return nil, fmt.Errorf("unsupported verification method type: %s", vm.Type)
	}

	if vm.PublicKeyJWK == nil {
		return nil, fmt.Errorf("missing publicKeyJwk in verification method")
	}

	// Parse JWK
	var jwk JWK
	if err := json.Unmarshal(vm.PublicKeyJWK, &jwk); err != nil {
		return nil, fmt.Errorf("parse JWK: %w", err)
	}

	// Convert JWK to crypto.PubKey based on curve
	switch jwk.Crv {
	case "Ed25519": //nolint:goconst
		return extractEd25519Key(jwk)
	case "secp256k1": //nolint:goconst
		return extractSecp256k1Key(jwk)
	case "X25519":
		// X25519 is for key agreement, not signing
		// We'll skip it for now as we need signing keys
		return nil, fmt.Errorf("X25519 keys are for key agreement, not signing")
	default:
		return nil, fmt.Errorf("unsupported curve: %s", jwk.Crv)
	}
}

// extractEd25519Key extracts an Ed25519 public key from JWK
func extractEd25519Key(jwk JWK) (crypto.PubKey, error) {
	if jwk.Kty != "OKP" {
		return nil, fmt.Errorf("Ed25519 key must have kty=OKP, got %s", jwk.Kty)
	}

	// Decode base64url X coordinate
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("decode Ed25519 X coordinate: %w", err)
	}

	if len(xBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 key size: expected %d, got %d", ed25519.PublicKeySize, len(xBytes))
	}

	// Convert to libp2p Ed25519 public key
	pubKey := ed25519.PublicKey(xBytes)
	return libp2p_crypto.UnmarshalEd25519PublicKey(pubKey)
}

// extractSecp256k1Key extracts a secp256k1 public key from JWK
func extractSecp256k1Key(jwk JWK) (crypto.PubKey, error) {
	if jwk.Kty != "EC" {
		return nil, fmt.Errorf("secp256k1 key must have kty=EC, got %s", jwk.Kty)
	}

	if jwk.Y == "" {
		return nil, fmt.Errorf("secp256k1 key missing Y coordinate")
	}

	// Decode base64url coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("decode secp256k1 X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("decode secp256k1 Y coordinate: %w", err)
	}

	// libp2p's UnmarshalSecp256k1PublicKey expects the uncompressed public key format:
	// 0x04 || X (32 bytes) || Y (32 bytes) = 65 bytes total
	if len(xBytes) != 32 || len(yBytes) != 32 {
		return nil, fmt.Errorf("invalid secp256k1 key size: X=%d bytes, Y=%d bytes (expected 32 each)", len(xBytes), len(yBytes))
	}

	pubKeyBytes := make([]byte, 0, 1+len(xBytes)+len(yBytes))
	pubKeyBytes = append(pubKeyBytes, 0x04) // uncompressed point prefix
	pubKeyBytes = append(pubKeyBytes, xBytes...)
	pubKeyBytes = append(pubKeyBytes, yBytes...)

	// Use libp2p's secp256k1 unmarshaler
	return libp2p_crypto.UnmarshalSecp256k1PublicKey(pubKeyBytes)
}

// makePrismAnchor creates an Anchor for a PRISM DID
func makePrismAnchor(did DID) (Anchor, error) {
	// Use global config
	config := globalPRISMConfig

	// Resolve DID document
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	doc, err := resolvePRISMDID(ctx, did, config)
	if err != nil {
		return nil, fmt.Errorf("resolve PRISM DID: %w", err)
	}

	// Extract public key from DID document
	pubk, err := extractPublicKeyFromDIDDocument(doc, config)
	if err != nil {
		return nil, fmt.Errorf("extract public key: %w", err)
	}

	// Create anchor
	return NewAnchor(did, pubk), nil
}

// SetPRISMResolverConfig sets the global PRISM resolver configuration
// This allows users to configure the resolver URL and other options
var globalPRISMConfig = defaultPRISMResolverConfig

// SetPRISMResolverConfig updates the global PRISM resolver configuration
func SetPRISMResolverConfig(config PRISMResolverConfig) {
	globalPRISMConfig = config
}

// GetPRISMResolverConfig returns the current global PRISM resolver configuration
func GetPRISMResolverConfig() PRISMResolverConfig {
	return globalPRISMConfig
}

// ProviderFromPRISMPrivateKey creates a Provider from a PRISM DID and private key
// This allows using PRISM DIDs for signing UCAN tokens
// Note: The private key is not verified against the DID document at creation time.
// Verification happens when the provider is used (e.g., when signing tokens).
func ProviderFromPRISMPrivateKey(prismDID DID, privk crypto.PrivKey) (Provider, error) {
	if prismDID.Method() != "prism" {
		return nil, fmt.Errorf("expected PRISM DID, got %s", prismDID.Method())
	}

	return NewProvider(prismDID, privk), nil
}

// ImportPRISMPrivateKeyFromJWK imports a PRISM private key from JWK format
// Returns the private key and the corresponding PRISM DID
func ImportPRISMPrivateKeyFromJWK(jwkData []byte, _ DID) (crypto.PrivKey, error) {
	var jwk JWK
	if err := json.Unmarshal(jwkData, &jwk); err != nil {
		return nil, fmt.Errorf("parse JWK: %w", err)
	}

	switch jwk.Crv {
	case "Ed25519":
		return importEd25519PrivateKeyFromJWK(jwk)
	case "secp256k1":
		return importSecp256k1PrivateKeyFromJWK(jwk)
	default:
		return nil, fmt.Errorf("unsupported curve: %s", jwk.Crv)
	}
}

// importEd25519PrivateKeyFromJWK imports an Ed25519 private key from JWK
func importEd25519PrivateKeyFromJWK(jwk JWK) (crypto.PrivKey, error) {
	if jwk.Kty != "OKP" {
		return nil, fmt.Errorf("Ed25519 key must have kty=OKP, got %s", jwk.Kty)
	}

	// JWK for Ed25519 private keys typically has both 'd' (private key) and 'x' (public key)
	// But we might only have 'd' (the seed) or the full private key
	// For Ed25519, the private key is 32 bytes (seed) or 64 bytes (seed + public key)

	// Try to get 'd' (private key material)
	// Note: JWK private keys have a 'd' field, but our JWK struct doesn't include it yet
	// For now, we'll need to parse it from the raw JSON
	return nil, fmt.Errorf("JWK private key import not yet implemented - use raw key format")
}

// importSecp256k1PrivateKeyFromJWK imports a secp256k1 private key from JWK
func importSecp256k1PrivateKeyFromJWK(_ JWK) (crypto.PrivKey, error) {
	return nil, fmt.Errorf("JWK private key import not yet implemented - use raw key format")
}
