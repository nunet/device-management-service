# PRISM Identity with UCAN - Documentation

This document provides a comprehensive guide on using PRISM (Decentralized Identifier) identities with UCAN (User Controlled Authorization Network) tokens in the Device Management Service.

## Table of Contents

1. [Overview](#overview)
2. [Hyperledger Identus Compatibility](#hyperledger-identus-compatibility)
3. [PRISM DID Basics](#prism-did-basics)
4. [Importing PRISM Keys](#importing-prism-keys)
5. [Using PRISM Identity with UCAN](#using-prism-identity-with-ucan)
6. [Running Tests](#running-tests)
7. [Examples](#examples)
8. [Troubleshooting](#troubleshooting)
9. [Running NeoPRISM Locally](#running-neoprism-locally)
10. [Additional Resources](#additional-resources)
11. [API Reference](#api-reference)

## Overview

PRISM is a DID method that runs on the Cardano blockchain, providing decentralized identity management. This system integrates PRISM DIDs with UCAN tokens, allowing you to:

- Create and manage PRISM identities
- Sign UCAN tokens with PRISM DIDs
- Verify UCAN tokens signed by PRISM identities
- Interoperate between `did:prism` and `did:key` identities

We specifically offer interoperability between the neoprism implementation (Hyperledger Identus implementation for PRISM) and UCAN. This integration has been tested against `hyperledgeridentus/identus-neoprism:0.9.1` image.

### Key Components

- **PRISM DID**: A decentralized identifier following the format `did:prism:<hash>`
- **Provider**: An interface for signing with a PRISM DID's private key
- **Anchor**: A public key extracted from a PRISM DID document for verification
- **TrustContext**: Manages providers and anchors for UCAN operations

## Hyperledger Identus Compatibility

This implementation is fully compatible with **Hyperledger Identus NeoPRISM**, the Hyperledger Identus implementation for PRISM DIDs. The compatibility has been tested and verified against the `hyperledgeridentus/identus-neoprism:0.9.1` Docker image.

### Implementation Details

The following implementation details ensure compatibility with Hyperledger Identus NeoPRISM:

#### 1. W3C-Compliant DID Resolution

The implementation uses NeoPRISM's W3C-compliant DID resolution endpoint:

- **Endpoint**: `/api/dids/{did}`
- **Format**: Returns W3C-compliant JSON DID resolution result
- **Accept Header**: `application/did+json,application/json`
- **URL Encoding**: DIDs are properly URL-encoded in the path (e.g., `:` becomes `%3A`)

This endpoint is used instead of alternative formats (like OpenPrismNode's `/api/v1/identifiers/{did}`) to ensure compatibility with NeoPRISM's API structure.

#### 2. Protobuf Operation Encoding

PRISM operations are encoded using Protocol Buffers (protobuf) as specified by the PRISM protocol:

- **Operation Format**: Uses `PrismOperation` and `SignedPrismOperation` protobuf messages
- **Encoding**: Operations are serialized to bytes, signed, and hex-encoded for submission
- **Key Types**: Supports both Ed25519 and secp256k1 keys in protobuf format
- **Key Usage**: Properly maps key usage types (MASTER_KEY, AUTHENTICATION_KEY, etc.) to protobuf enums

#### 3. Operation Submission API

Operations are submitted to NeoPRISM using the standard submission endpoint:

- **Endpoint**: `/api/signed-operation-submissions`
- **Method**: POST
- **Content-Type**: `application/json`
- **Request Format**: `{"signed_operations": ["<hex-encoded-signed-operation>"]}`
- **Response Format**: `{"tx_id": "<transaction-id>", "operation_ids": ["<operation-id>"]}`

#### 4. Key Type Requirements

NeoPRISM has specific requirements for key types:

- **Master Keys**: NeoPRISM requires **Secp256k1 keys** for master keys (not Ed25519)
- **Authentication Keys**: Both Ed25519 and secp256k1 are supported
- **Key Format**: Master keys must be provided in the correct protobuf format with proper curve specification

The implementation automatically handles key type detection and conversion, ensuring master keys use Secp256k1 when creating DIDs via NeoPRISM.

#### 5. DID Format Compliance

The implementation ensures DIDs follow NeoPRISM's expected format:

- **Format**: `did:prism:{64-character-hexadecimal-string}`
- **Validation**: DIDs are validated to ensure they match the canonical PRISM format
- **Resolution**: DIDs are properly URL-encoded when used in API paths

#### 6. Verification Method Support

DID documents are parsed with support for NeoPRISM's verification method structure:

- **Type**: `JsonWebKey2020` verification methods
- **Key Curves**: Ed25519 and secp256k1
- **Relationships**: Supports `authentication`, `assertionMethod`, and `capabilityInvocation` relationships
- **JWK Parsing**: Properly extracts public keys from `publicKeyJwk` fields in base64url encoding

#### 7. DID Document Structure

The implementation correctly handles W3C DID document structure as returned by NeoPRISM:

- **Context**: Supports multiple `@context` values
- **Verification Methods**: Handles both embedded and referenced verification methods
- **Service Endpoints**: Supports service definitions in DID documents
- **ID Matching**: Validates that resolved DID document ID matches the requested DID

### Testing and Validation

Compatibility is validated through integration tests that:

1. Create DIDs via NeoPRISM's submission API
2. Resolve DID documents from NeoPRISM's resolution endpoint
3. Extract public keys from resolved documents
4. Sign and verify UCAN tokens using PRISM identities
5. Test interoperability with `did:key` identities

See the [Running Tests](#running-tests) section for instructions on running tests against NeoPRISM.

### Version Compatibility

- **Tested Version**: `hyperledgeridentus/identus-neoprism:0.9.1`
- **API Compatibility**: Compatible with NeoPRISM API v1
- **Protocol Compatibility**: Follows PRISM protocol specification for operation encoding

## PRISM DID Basics

### DID Format

PRISM DIDs follow the format:
```
did:prism:<method-specific-identifier>
```

Example:
```
did:prism:4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b
```

### DID Resolution

PRISM DIDs are resolved to DID documents via HTTP requests to PRISM resolver endpoints. The system supports:

- **Default resolver**: `https://prism-agent.example.com`
- **OpenPrismNode format**: `/api/v1/identifiers/{did}`
- **Custom resolvers**: Configurable via `PRISMResolverConfig`

### Supported Key Types

- **Ed25519**: Most common, recommended for new identities
- **secp256k1**: Alternative for compatibility

## Importing PRISM Keys

There are several ways to import PRISM private keys depending on your use case.

### Method 1: Import from JWK Format

If you have a PRISM private key in JSON Web Key (JWK) format (e.g., from a PRISM agent API response):

**Note**: The JWK import function may require manual parsing of the `d` field. See Method 4 for a complete example of extracting keys from PRISM agent responses.

```go
import (
    "encoding/json"
    "gitlab.com/nunet/device-management-service/lib/did"
)

// JWK data from PRISM agent (typically from didState.secret.verificationMethod[0].privateKeyJwk)
jwkData := []byte(`{
    "kty": "OKP",
    "crv": "Ed25519",
    "d": "base64url-encoded-private-key",
    "x": "base64url-encoded-public-key"
}`)

prismDID, err := did.FromString("did:prism:...")
if err != nil {
    return err
}

// Import the private key
// Note: This may require the JWK struct to include the 'd' field
privKey, err := did.ImportPRISMPrivateKeyFromJWK(jwkData, prismDID)
if err != nil {
    return err
}

// Create a provider
provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
if err != nil {
    return err
}
```

### Method 2: Import from Hex-Encoded Key (CLI)

Using the CLI command to import a PRISM identity:

```bash
nunet key import-prism <name> <prism-did> <private-key-hex>
```

**Supported formats:**
- libp2p protobuf format (hex encoded)
- Raw Ed25519 seed (32 bytes, hex encoded)
- Raw Ed25519 private key (64 bytes, hex encoded)
- Raw secp256k1 private key (32 bytes, hex encoded)

**Example:**
```bash
nunet key import-prism myprism \
    did:prism:9b5118411248d9663b6ab15128fba8106511230ff654e7514cdcc4ce919bde9b \
    08011240...
```

### Method 3: Generate Keys Locally

For testing or when you control key generation:

```go
import (
    "gitlab.com/nunet/device-management-service/lib/crypto"
    "gitlab.com/nunet/device-management-service/lib/did"
)

// Generate a key pair
privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
if err != nil {
    return err
}

// Create a PRISM DID (in real scenario, this would be created on Cardano)
prismDID, err := did.FromString("did:prism:...")
if err != nil {
    return err
}

// Create provider
provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
if err != nil {
    return err
}
```

### Method 4: Extract from PRISM Agent API Response

When creating a DID via PRISM agent API (e.g., OpenPrismNode), extract the private key from the response:

```go
// After creating DID via PRISM agent API
var createDIDResponse struct {
    DIDState struct {
        DID    string                 `json:"did"`
        Secret map[string]interface{} `json:"secret,omitempty"`
    } `json:"didState"`
}

// Extract private key from response
secret := createDIDResponse.DIDState.Secret
vmList := secret["verificationMethod"].([]interface{})
vmData := vmList[0].(map[string]interface{})
privateKeyJWK := vmData["privateKeyJwk"].(map[string]interface{})

// Convert to JSON
jwkBytes, err := json.Marshal(privateKeyJWK)
if err != nil {
    return err
}

// Import
prismDID, _ := did.FromString(createDIDResponse.DIDState.DID)
privKey, err := did.ImportPRISMPrivateKeyFromJWK(jwkBytes, prismDID)
if err != nil {
    return err
}

provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
```

## Using PRISM Identity with UCAN

### Step 1: Configure PRISM Resolver

Set up the PRISM resolver configuration:

```go
import "gitlab.com/nunet/device-management-service/lib/did"

// Configure resolver
originalConfig := did.GetPRISMResolverConfig()
defer did.SetPRISMResolverConfig(originalConfig)

did.SetPRISMResolverConfig(did.PRISMResolverConfig{
    ResolverURL:                 "https://opn.preprod.blocktrust.dev",
    PreferredVerificationMethod: "authentication",
})
```

### Step 2: Resolve PRISM DID to Get Anchor

An anchor is needed for verification:

```go
prismDID, err := did.FromString("did:prism:...")
if err != nil {
    return err
}

// Resolve DID to get anchor (public key)
anchor, err := did.GetAnchorForDID(prismDID)
if err != nil {
    return err
}
```

### Step 3: Create Provider

Create a provider from your PRISM private key:

```go
provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
if err != nil {
    return err
}
```

### Step 4: Set Up Trust Context

Create a trust context and add your provider and anchor:

```go
import "gitlab.com/nunet/device-management-service/lib/ucan"

trustCtx := did.NewTrustContext()
trustCtx.AddProvider(provider)
trustCtx.AddAnchor(anchor)
```

### Step 5: Create Capability Context

Create a capability context for UCAN operations:

```go
capCtx, err := ucan.NewCapabilityContext(
    trustCtx,
    provider.DID(),              // issuer DID
    []did.DID{provider.DID()},  // roots
    ucan.TokenList{},            // require
    ucan.TokenList{},            // provide
    ucan.TokenList{},            // revoke
)
if err != nil {
    return err
}
```

### Step 6: Grant Capabilities

Grant capabilities using your PRISM identity:

```go
subjectDID := provider.DID()
audienceDID := provider.DID()
expire := uint64(time.Now().Add(1 * time.Hour).UnixNano())
capability := ucan.Capability("/test/prism/capability")

tokens, err := capCtx.Grant(
    ucan.Delegate,
    subjectDID,
    audienceDID,
    nil,              // topics
    expire,
    0,                 // depth
    []ucan.Capability{capability},
)
if err != nil {
    return err
}

token := tokens.Tokens[0]
```

### Step 7: Verify Tokens

Verify UCAN tokens signed by PRISM identities:

```go
now := uint64(time.Now().UnixNano())
revokeSet := &ucan.RevocationSet{Revoked: make(map[string]*ucan.Token)}
err = token.Verify(trustCtx, now, revokeSet)
if err != nil {
    return err
}
```

### Interoperability with did:key

PRISM identities can seamlessly interact with `did:key` identities:

```go
// PRISM identity can delegate to did:key
prismProvider, _ := did.ProviderFromPRISMPrivateKey(prismDID, prismPrivKey)
keyProvider, _ := did.ProviderFromPrivateKey(keyPrivKey)

// Both can be used in the same TrustContext
trustCtx.AddProvider(prismProvider)
trustCtx.AddProvider(keyProvider)
trustCtx.AddAnchor(prismAnchor)
trustCtx.AddAnchor(keyAnchor)

// PRISM can delegate capabilities to did:key and vice versa
```

## Running Tests

### Prerequisites

1. **Go 1.21+** installed
2. **Integration test tag**: Tests are marked with `//go:build integration`

### Running Integration Tests

#### Test with Mock Resolver

The basic integration test uses a mock PRISM resolver:

```bash
go test -v -tags=integration ./lib/ucan -run TestPrismIntegrationFullFlow
```

#### Test with Real PRISM Agent (OpenPrismNode)

To test against a real PRISM agent on Cardano testnet:

**Option 1: Quick Start with Hosted Instance** (Recommended for quick testing)

```bash
PRISM_TESTNET_URL=https://opn.preprod.blocktrust.dev \
PRISM_API_KEY=kfUpMnvUf32KLi73KLifhahfQ! \
PRISM_WALLET_ID=beb041bbbc689c6762f7fb743735e9c39df25ad5 \
go test -v -tags=integration ./lib/ucan -run TestPRISMRealTestnetWithTatumRPC
```

**Option 2: Quick Start with Local PRISM Agent (Tatum RPC)**

For a quick local test without full setup:

```bash
# Start OpenPrismNode connected to Tatum's free Cardano preprod testnet
docker run -d \
  --name prism-agent \
  -p 8080:8080 \
  -e CARDANO_NETWORK=preprod \
  -e CARDANO_RPC_URL=https://cardano-preprod.gateway.tatum.io \
  openprismnode/opn:latest

# Wait for agent to start (check logs)
docker logs -f prism-agent
# Look for "Server started" or similar message

# Run test
export PRISM_TESTNET_URL=http://localhost:8080
go test -v -tags=integration ./lib/ucan -run TestPRISMRealTestnetWithTatumRPC
```

**Note**: Tatum free tier has 5 requests/minute limit. For production testing, use Option 3 or 4.

**Option 3: Run Local OpenPrismNode with Blockfrost** (Read-only, no wallet)

1. **Get a Blockfrost API Key** (free tier available):
   - Sign up at https://blockfrost.io/
   - Create a project for Cardano Preprod testnet
   - Copy your API key

2. **Start OpenPrismNode locally**:
   ```bash
   # Set your Blockfrost API key (optional, defaults to placeholder)
   export BLOCKFROST_API_KEY=your-blockfrost-api-key-here
   
   # Start OpenPrismNode using docker-compose
   docker compose -f docker-compose.opn.yml up -d
   ```

3. **Wait for sync to start** (may take a few minutes):
   - Check logs: `docker logs -f openprismnode`
   - Access UI: http://localhost:5001
   - Swagger: http://localhost:5001/swagger/index.html

4. **Run tests** (DID resolution only, no creation):
   ```bash
   PRISM_TESTNET_URL=http://localhost:5001 \
   go test -v -tags=integration ./lib/ucan -run TestPrismIntegrationFullFlow
   ```

**Option 4: Full Local Setup with Wallet Support** (Complete local environment)

See [LOCAL_SETUP_WITH_WALLET.md](./LOCAL_SETUP_WITH_WALLET.md) for complete instructions.

After setting up:
1. Create a Cardano wallet (see "Creating a Cardano Wallet" section above)
2. Fund the wallet with testnet ADA
3. Register wallet with OpenPrismNode
4. Run tests:
   ```bash
   PRISM_TESTNET_URL=http://localhost:5001 \
   PRISM_API_KEY=pwUser! \
   PRISM_WALLET_ID=your-wallet-id \
   go test -v -tags=integration ./lib/ucan -run TestPRISMRealTestnetWithTatumRPC
   ```

**Environment Variables:**
- `PRISM_TESTNET_URL`: URL of the PRISM agent (default: mock resolver)
  - For local: `http://localhost:5001`
  - For hosted: `https://opn.preprod.blocktrust.dev`
- `PRISM_API_KEY`: API key for authentication
  - For local: `pwUser!` (default user key)
  - For hosted: Use provided shared key or your own
- `PRISM_WALLET_ID`: Wallet ID for wallet-scoped operations (optional, but required for DID creation)

#### Test Mixed DID Methods

Test interoperability between PRISM and did:key:

```bash
go test -v -tags=integration ./lib/ucan -run TestMixedDIDMethods
```

#### Run All UCAN Tests

```bash
go test -v -tags=integration ./lib/ucan
```

### Test Structure

Tests are located in:
- `lib/ucan/prism_integration_test.go`: Main PRISM integration tests
- `lib/ucan/mixed_did_test.go`: Tests for PRISM/did:key interoperability
- `lib/did/prism_test.go`: Unit tests for PRISM DID resolution

### Mock PRISM Resolver

For local testing, a mock PRISM resolver is available:

```go
// In test files
mockResolver := createMockPRISMResolverForIntegration(t, prismDIDStr, pubKeyBytes)
defer mockResolver.Close()

did.SetPRISMResolverConfig(did.PRISMResolverConfig{
    ResolverURL: mockResolver.URL,
    PreferredVerificationMethod: "authentication",
})
```

## Examples

### Complete Example: Create and Use PRISM Identity

```go
package main

import (
    "fmt"
    "time"
    
    "gitlab.com/nunet/device-management-service/lib/crypto"
    "gitlab.com/nunet/device-management-service/lib/did"
    "gitlab.com/nunet/device-management-service/lib/ucan"
)

func main() {
    // 1. Generate key pair
    privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
    if err != nil {
        panic(err)
    }
    
    // 2. Create PRISM DID (in production, create via PRISM agent)
    prismDID, err := did.FromString("did:prism:test123...")
    if err != nil {
        panic(err)
    }
    
    // 3. Configure resolver (use mock for testing)
    did.SetPRISMResolverConfig(did.PRISMResolverConfig{
        ResolverURL: "https://mock-resolver.example.com",
        PreferredVerificationMethod: "authentication",
    })
    
    // 4. Resolve DID to get anchor
    anchor, err := did.GetAnchorForDID(prismDID)
    if err != nil {
        panic(err)
    }
    
    // 5. Create provider
    provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
    if err != nil {
        panic(err)
    }
    
    // 6. Set up trust context
    trustCtx := did.NewTrustContext()
    trustCtx.AddProvider(provider)
    trustCtx.AddAnchor(anchor)
    
    // 7. Create capability context
    capCtx, err := ucan.NewCapabilityContext(
        trustCtx,
        provider.DID(),
        []did.DID{provider.DID()},
        ucan.TokenList{},
        ucan.TokenList{},
        ucan.TokenList{},
    )
    if err != nil {
        panic(err)
    }
    
    // 8. Grant capability
    expire := uint64(time.Now().Add(1 * time.Hour).UnixNano())
    tokens, err := capCtx.Grant(
        ucan.Delegate,
        provider.DID(),
        provider.DID(),
        nil,
        expire,
        0,
        []ucan.Capability{ucan.Capability("/test/capability")},
    )
    if err != nil {
        panic(err)
    }
    
    // 9. Verify token
    token := tokens.Tokens[0]
    now := uint64(time.Now().UnixNano())
    revokeSet := &ucan.RevocationSet{Revoked: make(map[string]*ucan.Token)}
    err = token.Verify(trustCtx, now, revokeSet)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("✅ Successfully created and verified UCAN token with PRISM identity: %s\n", token.Issuer())
}
```

### Example: Import from PRISM Agent Response

```go
// After creating DID via PRISM agent API
func importFromAgentResponse(responseBody []byte) (*did.Provider, error) {
    var resp struct {
        DIDState struct {
            DID    string                 `json:"did"`
            Secret map[string]interface{} `json:"secret"`
        } `json:"didState"`
    }
    
    if err := json.Unmarshal(responseBody, &resp); err != nil {
        return nil, err
    }
    
    // Extract private key JWK
    secret := resp.DIDState.Secret
    vmList := secret["verificationMethod"].([]interface{})
    vmData := vmList[0].(map[string]interface{})
    privateKeyJWK := vmData["privateKeyJwk"].(map[string]interface{})
    
    // Convert to JSON
    jwkBytes, err := json.Marshal(privateKeyJWK)
    if err != nil {
        return nil, err
    }
    
    // Import
    prismDID, err := did.FromString(resp.DIDState.DID)
    if err != nil {
        return nil, err
    }
    
    privKey, err := did.ImportPRISMPrivateKeyFromJWK(jwkBytes, prismDID)
    if err != nil {
        return nil, err
    }
    
    return did.ProviderFromPRISMPrivateKey(prismDID, privKey)
}
```

## Troubleshooting

### Common Issues

#### 1. "Failed to resolve PRISM DID"

**Problem**: Cannot resolve PRISM DID to DID document.

**Solutions**:
- Check resolver URL is correct
- Verify network connectivity
- Ensure DID exists on the blockchain/resolver
- Check resolver endpoint format (OpenPrismNode uses `/api/v1/identifiers/{did}`)

#### 2. "No supported verification method found"

**Problem**: Cannot extract public key from DID document.

**Solutions**:
- Verify DID document has `verificationMethod` or `authentication` fields
- Check that verification method uses supported key type (Ed25519 or secp256k1)
- Ensure `publicKeyJwk` is present and valid
- Try different `PreferredVerificationMethod` in config

#### 3. "Signature verification failed"

**Problem**: Token signature doesn't match public key.

**Solutions**:
- Verify private key matches the DID's public key
- Ensure provider was created with correct private key
- Check that anchor was resolved from the same DID
- Verify key format (Ed25519 vs secp256k1)

#### 4. "Failed to create signed Atala operation"

**Problem**: Error when creating DID via PRISM agent API.

**Solutions**:
- Verify wallet exists and is accessible
- Check wallet has sufficient funds (ADA for transaction fees)
- Ensure API key authentication is correct
- Verify request format matches API specification
- Check wallet state (may need initialization)

#### 5. Import Errors

**Problem**: Cannot import private key.

**Solutions**:
- Verify key format (JWK, hex, etc.)
- Check key type matches DID's key type
- Ensure key is complete (not truncated)
- For JWK, verify `kty`, `crv`, and `d` fields are present

### Debug Tips

1. **Enable verbose logging**: Use `t.Logf()` in tests to see detailed information
2. **Check DID document**: Resolve DID manually and inspect the document
3. **Verify key format**: Print key bytes to verify format
4. **Test with mock**: Use mock resolver to isolate network issues
5. **Check resolver config**: Verify `PRISMResolverConfig` is set correctly

### Getting Help

- Check test files for working examples: `lib/ucan/prism_integration_test.go`
- Review API documentation: PRISM agent Swagger docs
- Check error messages: They often indicate the specific issue
- Verify environment variables: Ensure all required vars are set

## Running NeoPRISM Locally

For development and testing, you can run your own NeoPRISM instance locally using Docker Compose.

Prerequisites: Cabal, Nix and Docker Compose installed.

1. First clone the cardano-node repository and use the devShell to build the cardano-testnet binary.
```bash
git clone https://github.com/IntersectMBO/cardano-node.git
cd cardano-node
nix develop --extra-experimental-features "nix-command flakes"
cabal update
cabal build cardano-testnet
cabal build cardano-node
```
2. Export variables for the binaries and guarantee they are printing the versions successfully.
```bash
export CARDANO_NODE=$(cabal list-bin cardano-node)
export CARDANO_CLI=$(which cardano-cli) # use the Nix binary
$CARDANO_NODE --version
$CARDANO_CLI --version
cabal list-bin cardano-testnet
```

You should see outputs like these ones:
```bash
cardano-node$ $CARDANO_NODE --version
cardano-node 10.6.1 - linux-x86_64 - ghc-9.6
git rev 0ffa5d589668ad899d8c69ea769896e5bc96ad43
cardano-node$ $CARDANO_CLI --version
cardano-cli 10.13.1.0 - linux-x86_64 - ghc-9.6
git rev 0000000000000000000000000000000000000000
cardano-node$ cabal list-bin cardano-testnet
<absolute path>/cardano-node/dist-newstyle/build/x86_64-linux/ghc-9.6.7/cardano-testnet-10.0.1/x/cardano-testnet/build/cardano-testnet/cardano-testnet
```

3. Link the output of the testnet environment to be generated to the `builds/docker/testnet-env/` directory of the DMS (device-management-service).
```bash
# Note: If the cardano-node/testnet directory exists, remove it since it will be generated again.
mkdir ~<relative DMS path>/builds/docker/testnet-env
ln -s ~<relative DMS path>/builds/docker/testnet-env testnet
```

4. Run your local cardano testnet by running the cardano-testnet binary and put it on background.
```bash
$(cabal list-bin cardano-testnet) cardano \
     --testnet-magic 42 \
     --active-slots-coeff 0.1 \
     --epoch-length 60 \
     --num-pool-nodes 3 \
     --output-dir ~<relative DMS path>/builds/docker/testnet-env &
```

5. Start the NeoPRISM services by running the following command:
```bash
# Note: If the cardano-node/testnet directory exists, remove it since it will be generated by the next commands.
cd ~<relative DMS path>/builds/docker
docker compose -f prism-cardano-testnet.yml up -d
```

6. Create a wallet for NeoPRISM to use for submitting transactions:
```bash
./create-neoprism-wallet.sh neoprism-wallet <wallet_passphrase>
```

7. Fund the wallet using UTXO keys from `builds/docker/testnet-env/utxo-keys/`:
```bash
./fund-wallet.sh ${PAYMENT_ADDR} 10000
```

8. Add wallet id, passphrase and address to docker compose file for neoprism and restart neoprism.
```bash
docker compose -f prism-cardano-testnet.yml down neoprism
vi prism-cardano-testnet.yml
      # Edit these parameters using the output of the step 6:
      - NPRISM_CARDANO_WALLET_WALLET_ID=${WALLET_ID:-<wallet_id>}
      - NPRISM_CARDANO_WALLET_PASSPHRASE=${WALLET_PASSPHRASE:-<wallet_passphrase>}
      - NPRISM_CARDANO_WALLET_PAYMENT_ADDR=${PAYMENT_ADDRESS:-<payment_address>}
docker compose -f prism-cardano-testnet.yml up -d neoprism
```

9. Set the environment variable and run the test:
```bash
export PRISM_TESTNET_URL=http://localhost:8080
go test -v -tags=integration ./lib/ucan -run TestPRISMNeoPRISMIntegration
```

10. Test creation of keys using an already built DMS binary:
```bash
nunet key create-prism --prism-url http://localhost:8080 --key-type secp256k1 --wait-confirmation --timeout 20m test-prism
```

### Demonstration Video

A demonstration video was recorded following the steps described in this section. The video serves as a visual reference to complement the written documentation and can be accessed at https://drive.google.com/file/d/1Orw9PNknwp6YViToY47HTqV3_eqKrLay/view?usp=sharing.

## Additional Resources

- **PRISM DID Method Specification**: [PRISM Documentation](https://github.com/input-output-hk/prism-did)
- **UCAN Specification**: [UCAN Spec](https://ucan.xyz/)
- **Cardano Testnet**: [Testnet Information](https://developers.cardano.org/docs/get-started/testnets/)

## API Reference

### Key Functions

- `did.ProviderFromPRISMPrivateKey(prismDID DID, privk crypto.PrivKey) (Provider, error)`
- `did.ImportPRISMPrivateKeyFromJWK(jwkData []byte, prismDID DID) (crypto.PrivKey, error)`
- `did.GetAnchorForDID(did DID) (Anchor, error)`
- `did.SetPRISMResolverConfig(config PRISMResolverConfig)`
- `did.GetPRISMResolverConfig() PRISMResolverConfig`

### Types

- `PRISMResolverConfig`: Configuration for PRISM DID resolution
- `Provider`: Interface for signing with a DID's private key
- `Anchor`: Public key for verification
- `TrustContext`: Manages providers and anchors
- `CapabilityContext`: Context for UCAN operations

---

**Last Updated**: 2025-01-02
**Version**: 1.0

