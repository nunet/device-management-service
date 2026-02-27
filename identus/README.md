# Identus SDK Key Generation

This project serves as a utility and demonstration for using the **Hyperledger Identus SDK** (formerly Prism) in a Node.js environment. Its primary purpose is to generate cryptographic key pairs and export them in formats suitable for integration with the **Device Management Service (DMS)**.

## Context and Overview

**Hyperledger Identus** is an open-source decentralized identity framework. This repository provides a simplified interface to interact with the Identus SDK's cryptographic primitives, specifically for creating keys that will eventually be used by devices managed by the DMS.

The **Device Management Service (DMS)** is written in **Go**. To allow seamless interoperability between keys generated in a JavaScript/Node.js environment (using Identus) and the Go-based DMS, this tool bridges the gap by exporting keys in standard formats like **JWK (JSON Web Key)** and raw hexadecimal strings.

### Key Features
*   Generates **Secp256k1** key pairs (compatible with DMS).
*   Uses the `Apollo` class from `@hyperledger/identus-sdk`.
*   Exports keys in **JWK** format (RFC 7517).
*   Exports raw key material (Hex encoded) for direct import into Go applications.

---

## Prerequisites

*   **Node.js**: A recent version (LTS recommended, e.g., v18 or v20).
*   **npm**: Included with Node.js.

## Setup and Installation

1.  **Install Dependencies**:
    Run the following command to install the required `@hyperledger/identus-sdk`:
    ```bash
    npm install
    ```

2.  **Run the Script**:
    Execute the demo script:
    ```bash
    node index.mjs
    ```

---

## How It Works (Deep Dive)

The `index.mjs` script performs a sequence of operations to ensure a secure key is generated and correctly formatted for export.

### 1. Initialization
We import the `Apollo` class from the SDK. `Apollo` is the main entry point for cryptographic operations in Identus.
```javascript
import { Apollo } from '@hyperledger/identus-sdk';
const apollo = new Apollo();
```

### 2. Seed Generation
We generate a cryptographically secure random seed. The SDK requires a seed to deterministically derivation keys if needed, though here we use it for fresh generation.
```javascript
// Generates a 64-byte random seed
const seed = new Int8Array(crypto.randomBytes(64));
```

### 3. Key Creation
We instruct Apollo to create a generic **EC (Elliptic Curve)** key using the **secp256k1** curve.
```javascript
const privateKey = apollo.createPrivateKey({
    type: 'EC',
    curve: 'secp256k1',
    seed: seed
});
```
*Note: Identus supports `Ed25519` `secp256k1` and `X25519`. Dms supports the first 2 key types.

### 4. Serialization (The Tricky Part)

The internal `privateKey` object in the SDK uses complex types like `BigInt`, `Map`, and `Uint8Array/Int8Array` which standard `JSON.stringify` **cannot handle** by default.

The script includes a custom serializer to ensure you get a readable JSON output:
*   **BigInt** → Converted to String.
*   **Map** → Converted to Object `Object.fromEntries()`.
*   **TypedArray (Uint8Array)** → Converted to Hex String.

---

## Output Formats

When you run the script, you will see two main outputs in your console.

### 1. JSON Web Key (JWK)
This is the standard format for representing keys on the web.
```json
{
  "kty": "EC",
  "crv": "secp256k1",
  "x": "<public_key_x_coordinate>",
  "y": "<public_key_y_coordinate>",
  "d": "<private_key_scalar>"
}
```

### 2. Raw Private Key Object
This output shows the internal structure of the key. **This is crucial for the Go integration.**
Look for the `raw` field:
```json
{
  "raw": "39648c978d04f6fd399501435b2b8143c31ae66fcd00169c001446316e7b6918",
  "keySpecification": { "curve": "secp256k1" }
}
```
*   **`raw`**: This is your private key in hexadecimal format (32 bytes / 64 hex characters).

---

## DMS Integration (Go)

The ultimate goal is to import these keys into the DMS.

### Current Status (Milestone 2)

We are currently implementing a full JWK importer in the DMS. The interface looks like this:

```go
// importSecp256k1PrivateKeyFromJWK imports a secp256k1 private key from JWK
func importSecp256k1PrivateKeyFromJWK(_ JWK) (crypto.PrivKey, error) {
    return nil, fmt.Errorf("JWK private key import not yet implemented - use raw key format")
}
```

### Interim Solution: Raw Hex Import
Until the JWK importer is finalized, **use the raw hex string** from the output above. The following Go code demonstrates how to import this key using `libp2p/go-libp2p/core/crypto`.

#### Go Code Example
```go
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
    // 1. Copy the "raw" value from the node script output
	rawHex := "39648c978d04f6fd399501435b2b8143c31ae66fcd00169c001446316e7b6918"

    // 2. Decode hex string to bytes
	privKeyBytes, err := hex.DecodeString(rawHex)
	if err != nil {
		log.Fatalf("Failed to decode hex: %v", err)
	}

    // 3. Unmarshal into a libp2p Crypto Private Key
	privKey, err := crypto.UnmarshalSecp256k1PrivateKey(privKeyBytes)
	if err != nil {
		log.Fatalf("Failed to unmarshal key: %v", err)
	}

	fmt.Printf("Successfully imported Secp256k1 key.\n")
	
    // 4. (Optional) Sign data to verify it works
	data := []byte("Hello, Identus!")
	signature, err := privKey.Sign(data)
	if err != nil {
		log.Fatalf("Failed to sign data: %v", err)
	}
	fmt.Printf("Signature: %x\n", signature)

    // 5. Store in DMS Keystore
    // keystore.Import("my-key-alias", "my-passphrase", privKeyBytes)
}
```

## Troubleshooting

*   **`SyntaxError: Cannot use import statement outside a module`**: ensure your `package.json` does NOT contain `"type": "module"` if you are renaming to `.js`, OR keep the file extension as `.mjs` (which it is by default). The script is designed as an ES Module.
*   **`Error: Identus SDK not found`**: Make sure you ran `npm install` and that `.npmrc` is configured correctly if you are using a private registry (though `@hyperledger/identus-sdk` should be public).
