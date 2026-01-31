// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	prismpb "gitlab.com/nunet/device-management-service/proto/generated/prism"
	"google.golang.org/protobuf/proto"
)

// PRISMCreateDIDOperation represents a PRISM CreateDID operation
type PRISMCreateDIDOperation struct {
	PublicKeys []PRISMPublicKey
	Services   []PRISMService
	Context    []string
}

// PRISMPublicKey represents a public key in PRISM format
type PRISMPublicKey struct {
	ID    string // e.g., "master-0"
	Usage string // "MASTER_KEY", "AUTHENTICATION_KEY", etc.
	Key   []byte // Public key bytes (Ed25519: 32 bytes, Secp256k1: 33 bytes compressed)
	Curve string // "Ed25519" or "secp256k1" (optional, auto-detected if empty)
}

// PRISMService represents a service in PRISM format
type PRISMService struct {
	ID              string
	Type            string
	ServiceEndpoint string
}

// CreateSignedPRISMOperation creates a signed PRISM CreateDID operation using protobuf encoding
func CreateSignedPRISMOperation(
	privKey crypto.PrivKey,
	_ crypto.PubKey,
	keyID string,
	publicKeys []PRISMPublicKey,
	services []PRISMService,
	context []string,
) (string, error) {
	// Step 1: Create the ProtoCreateDID operation
	createDID, err := buildCreateDIDOperation(publicKeys, services, context)
	if err != nil {
		return "", fmt.Errorf("build create DID operation: %w", err)
	}

	// Step 2: Create PrismOperation with create_did field
	prismOp := &prismpb.PrismOperation{
		Operation: &prismpb.PrismOperation_CreateDid{
			CreateDid: createDID,
		},
	}

	// Step 3: Encode PrismOperation to bytes
	operationBytes, err := proto.Marshal(prismOp)
	if err != nil {
		return "", fmt.Errorf("marshal prism operation: %w", err)
	}

	// Step 4: Sign the encoded operation bytes
	signature, err := privKey.Sign(operationBytes)
	if err != nil {
		return "", fmt.Errorf("sign operation: %w", err)
	}

	// Step 5: Create SignedPrismOperation
	signedOp := &prismpb.SignedPrismOperation{
		SignedWith: keyID,
		Signature:  signature,
		Operation:  prismOp,
	}

	// Step 6: Encode SignedPrismOperation to bytes
	signedOpBytes, err := proto.Marshal(signedOp)
	if err != nil {
		return "", fmt.Errorf("marshal signed operation: %w", err)
	}

	// Step 7: Encode to hex
	return hex.EncodeToString(signedOpBytes), nil
}

// buildCreateDIDOperation builds a ProtoCreateDID message from the provided keys, services, and context
func buildCreateDIDOperation(
	publicKeys []PRISMPublicKey,
	services []PRISMService,
	context []string,
) (*prismpb.ProtoCreateDID, error) {
	// Convert PRISMPublicKey to protobuf PublicKey
	pbPublicKeys := make([]*prismpb.PublicKey, 0, len(publicKeys))
	for _, pk := range publicKeys {
		pbKey, err := convertToPRISMPublicKey(pk)
		if err != nil {
			return nil, fmt.Errorf("convert public key %s: %w", pk.ID, err)
		}
		pbPublicKeys = append(pbPublicKeys, pbKey)
	}

	// Convert PRISMService to protobuf Service
	pbServices := make([]*prismpb.Service, 0, len(services))
	for _, svc := range services {
		pbServices = append(pbServices, &prismpb.Service{
			Id:              svc.ID,
			Type:            svc.Type,
			ServiceEndpoint: svc.ServiceEndpoint,
		})
	}

	// Build DIDCreationData
	didData := &prismpb.ProtoCreateDID_DIDCreationData{
		PublicKeys: pbPublicKeys,
		Services:   pbServices,
		Context:    context,
	}

	// Build ProtoCreateDID
	createDID := &prismpb.ProtoCreateDID{
		DidData: didData,
	}

	return createDID, nil
}

// convertToPRISMPublicKey converts a PRISMPublicKey to a protobuf PublicKey
func convertToPRISMPublicKey(pk PRISMPublicKey) (*prismpb.PublicKey, error) {
	// Convert usage string to KeyUsage enum
	usage, err := parseKeyUsage(pk.Usage)
	if err != nil {
		return nil, fmt.Errorf("parse key usage: %w", err)
	}

	var ecKeyData *prismpb.ECKeyData

	// Auto-detect curve if not specified
	curve := pk.Curve
	if curve == "" {
		// Detect from key size: Ed25519 is 32 bytes, Secp256k1 compressed is 33 bytes
		if len(pk.Key) == ed25519.PublicKeySize { //nolint:gocritic
			curve = "Ed25519"
		} else if len(pk.Key) == 33 {
			curve = "secp256k1"
		} else {
			return nil, fmt.Errorf("cannot auto-detect curve: key size %d bytes (expected 32 for Ed25519 or 33 for Secp256k1)", len(pk.Key))
		}
	}

	switch curve {
	case "Ed25519":
		// For Ed25519, we use ECKeyData with the full key in the x field
		// (Ed25519 doesn't have separate x/y coordinates like secp256k1)
		if len(pk.Key) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("Ed25519 key must be 32 bytes, got %d", len(pk.Key))
		}
		ecKeyData = &prismpb.ECKeyData{
			Curve: "Ed25519",
			X:     pk.Key, // For Ed25519, the full 32-byte key goes in X
			Y:     nil,    // Ed25519 doesn't use Y coordinate
		}
	case "secp256k1":
		// For Secp256k1, we need to extract X and Y coordinates from compressed key
		// Standard approach: Parse compressed key, then extract coordinates
		xBytes, yBytes, err := extractSecp256k1Coordinates(pk.Key)
		if err != nil {
			return nil, fmt.Errorf("extract secp256k1 coordinates: %w", err)
		}
		ecKeyData = &prismpb.ECKeyData{
			Curve: "secp256k1",
			X:     xBytes,
			Y:     yBytes,
		}
	default:
		return nil, fmt.Errorf("unsupported curve: %s (supported: Ed25519, secp256k1)", curve)
	}

	// Create PublicKey with ECKeyData
	pbKey := &prismpb.PublicKey{
		Id:    pk.ID,
		Usage: usage,
		KeyData: &prismpb.PublicKey_EcKeyData{
			EcKeyData: ecKeyData,
		},
	}

	return pbKey, nil
}

// extractSecp256k1Coordinates extracts X and Y coordinates from a compressed Secp256k1 public key.
// The input should be a 33-byte compressed public key.
// Returns X and Y coordinates as 32-byte slices.
// This uses the standard secp256k1 library approach: parse compressed key, serialize uncompressed, extract coordinates.
func extractSecp256k1Coordinates(compressedKey []byte) (xBytes, yBytes []byte, err error) {
	// Validate input: compressed Secp256k1 keys are 33 bytes
	if len(compressedKey) != 33 {
		return nil, nil, fmt.Errorf("compressed secp256k1 key must be 33 bytes, got %d", len(compressedKey))
	}

	// Parse compressed public key using standard secp256k1 library
	pubKey, err := secp256k1.ParsePubKey(compressedKey)
	if err != nil {
		return nil, nil, fmt.Errorf("parse compressed public key: %w", err)
	}

	// Serialize uncompressed format: 0x04 || X (32 bytes) || Y (32 bytes) = 65 bytes total
	// This is the standard SEC 1 uncompressed point format
	uncompressed := pubKey.SerializeUncompressed()
	if len(uncompressed) != 65 {
		return nil, nil, fmt.Errorf("unexpected uncompressed key size: %d (expected 65)", len(uncompressed))
	}

	// Validate prefix byte (should be 0x04 for uncompressed)
	if uncompressed[0] != 0x04 {
		return nil, nil, fmt.Errorf("invalid uncompressed key prefix: expected 0x04, got 0x%02x", uncompressed[0])
	}

	// Extract X and Y coordinates (skip first byte which is 0x04)
	xBytes = make([]byte, 32)
	yBytes = make([]byte, 32)
	copy(xBytes, uncompressed[1:33])  // X coordinate (32 bytes)
	copy(yBytes, uncompressed[33:65]) // Y coordinate (32 bytes)

	return xBytes, yBytes, nil
}

// parseKeyUsage converts a string to KeyUsage enum
func parseKeyUsage(usage string) (prismpb.KeyUsage, error) {
	switch usage {
	case "MASTER_KEY", "master_key":
		return prismpb.KeyUsage_MASTER_KEY, nil
	case "ISSUING_KEY", "issuing_key":
		return prismpb.KeyUsage_ISSUING_KEY, nil
	case "KEY_AGREEMENT_KEY", "key_agreement_key":
		return prismpb.KeyUsage_KEY_AGREEMENT_KEY, nil
	case "AUTHENTICATION_KEY", "authentication_key":
		return prismpb.KeyUsage_AUTHENTICATION_KEY, nil
	case "REVOCATION_KEY", "revocation_key":
		return prismpb.KeyUsage_REVOCATION_KEY, nil
	case "CAPABILITY_INVOCATION_KEY", "capability_invocation_key":
		return prismpb.KeyUsage_CAPABILITY_INVOCATION_KEY, nil
	case "CAPABILITY_DELEGATION_KEY", "capability_delegation_key":
		return prismpb.KeyUsage_CAPABILITY_DELEGATION_KEY, nil
	case "VDR_KEY", "vdr_key":
		return prismpb.KeyUsage_VDR_KEY, nil
	default:
		return prismpb.KeyUsage_UNKNOWN_KEY, fmt.Errorf("unknown key usage: %s", usage)
	}
}

// CreateSignedPRISMOperationSimple is a convenience function that creates a signed PRISM operation
// with a single master key from the provided private/public key pair
// Note: NeoPRISM requires Secp256k1 keys for master keys, so this function will use Secp256k1 if available
func CreateSignedPRISMOperationSimple(
	privKey crypto.PrivKey,
	pubKey crypto.PubKey,
	keyID string,
) (string, error) {
	// Get public key bytes
	pubKeyBytes, err := pubKey.Raw()
	if err != nil {
		return "", fmt.Errorf("get public key bytes: %w", err)
	}

	// Detect key type and set curve
	var curve string
	var context []string
	switch pubKey.Type() {
	case crypto.Ed25519:
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return "", fmt.Errorf("expected Ed25519 public key (32 bytes), got %d bytes", len(pubKeyBytes))
		}
		curve = "Ed25519"
		context = []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		}
	case crypto.Secp256k1:
		if len(pubKeyBytes) != 33 {
			return "", fmt.Errorf("expected Secp256k1 compressed public key (33 bytes), got %d bytes", len(pubKeyBytes))
		}
		curve = "secp256k1"
		context = []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/secp256k1-2019/v1",
		}
	default:
		return "", fmt.Errorf("unsupported key type: %d (supported: Ed25519, Secp256k1)", pubKey.Type())
	}

	// Create keys: master key (required) and authentication key (for DID document)
	// Note: Master keys don't appear in verificationMethod, so we add an authentication key
	// to ensure the DID document has verification methods
	publicKeys := []PRISMPublicKey{
		{
			ID:    keyID,
			Usage: "MASTER_KEY",
			Key:   pubKeyBytes,
			Curve: curve,
		},
		{
			ID:    "auth-0",
			Usage: "AUTHENTICATION_KEY",
			Key:   pubKeyBytes, // Use the same key for authentication
			Curve: curve,
		},
	}

	return CreateSignedPRISMOperation(privKey, pubKey, keyID, publicKeys, nil, context)
}
