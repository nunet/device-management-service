// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto_test

import (
	"fmt"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func generateEthKeyPair(t *testing.T) (*secp256k1.PrivateKey, *secp256k1.PublicKey, error) {
	t.Helper()

	sk, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, nil, err
	}
	pk := sk.PubKey()
	return sk, pk, nil
}

func TestUnmarshalEthPublicKey(t *testing.T) {
	t.Parallel()
	_, pub, err := generateEthKeyPair(t)
	require.NoError(t, err)

	// convert public key to bytes
	pubKeyBytes := pub.SerializeCompressed()

	// unmarshal public key
	ethPubKey, err := crypto.UnmarshalEthPublicKey(pubKeyBytes)
	require.NoError(t, err)

	// convert public key back to bytes
	ethPubKeyBytes, err := ethPubKey.Raw()
	require.NoError(t, err)

	// compare public key bytes
	require.Equal(t, pubKeyBytes, ethPubKeyBytes)
}

func TestEthPublicKeyVerify(t *testing.T) {
	t.Parallel()
	// Generate a keypair for signing
	priv, pub, err := generateEthKeyPair(t)
	require.NoError(t, err)

	// Create an EthPublicKey from the secp256k1 public key
	ethPubKey, err := crypto.UnmarshalEthPublicKey(
		pub.SerializeCompressed())
	require.NoError(t, err)

	message := []byte("test message")

	// Create proper Ethereum signature
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte("\x19Ethereum Signed Message:\n"))
	fmt.Fprintf(hasher, "%d", len(message))
	hasher.Write(message)
	hash := hasher.Sum(nil)

	// Sign the hashed message
	signature := ecdsa.Sign(priv, hash)
	sigBytes := signature.Serialize()

	// Test invalid signature
	invalidSig := []byte("invalid-signature")

	tests := []struct {
		name     string
		data     []byte
		sig      []byte
		expected bool
		hasError bool
	}{
		{"Valid signature", message, sigBytes, true, false},
		{"Invalid signature format", message, invalidSig, false, true},
		{"Empty signature", message, []byte{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			success, err := ethPubKey.Verify(tt.data, tt.sig)

			if tt.hasError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, success)
		})
	}
}

func TestEthPublicKeyEquals(t *testing.T) {
	t.Parallel()
	// Create two different keys
	privateKey1, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	privateKey2, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)

	pubKey1Bytes := privateKey1.PubKey().SerializeCompressed()
	pubKey2Bytes := privateKey2.PubKey().SerializeCompressed()

	ethPubKey1, err := crypto.UnmarshalEthPublicKey(pubKey1Bytes)
	require.NoError(t, err)
	ethPubKey1Copy, err := crypto.UnmarshalEthPublicKey(pubKey1Bytes)
	require.NoError(t, err)
	ethPubKey2, err := crypto.UnmarshalEthPublicKey(pubKey2Bytes)
	require.NoError(t, err)

	// Create a non-Eth key for comparison
	_, edPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	tests := []struct {
		name     string
		key1     crypto.Key
		key2     crypto.Key
		expected bool
	}{
		{"Same key", ethPubKey1, ethPubKey1, true},
		{"Same key content", ethPubKey1, ethPubKey1Copy, true},
		{"Different keys", ethPubKey1, ethPubKey2, false},
		{"Different key types", ethPubKey1, edPubKey, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.key1.Equals(tt.key2)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBasicEquals(t *testing.T) {
	t.Parallel()
	// Create keys of different types
	_, edPubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	_, secpPubKey, err := crypto.GenerateKeyPair(crypto.Secp256k1)
	require.NoError(t, err)

	tests := []struct {
		name     string
		key1     crypto.Key
		key2     crypto.Key
		expected bool
	}{
		{"Different key types", edPubKey, secpPubKey, false},
		{"Same key", edPubKey, edPubKey, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// We're testing the basicEquals function, but it's private
			// so we'll test it through the public Equals method
			result := tt.key1.Equals(tt.key2)
			require.Equal(t, tt.expected, result)
		})
	}
}
