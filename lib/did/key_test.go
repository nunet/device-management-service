// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestKeyDIDEd25519(t *testing.T) {
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	pubDID := FromPublicKey(pubk)
	require.Equal(t, pubDID.Method(), "key", "did method")

	pubk2, err := PublicKeyFromDID(pubDID)
	require.NoError(t, err, "public key from DID")
	require.Equal(t, pubk2, pubk, "public keys are the same")

	anchor, err := AnchorFromPublicKey(pubk)
	require.NoError(t, err, "anchor from public key")
	require.Equal(t, anchor.DID(), pubDID, "compare anchor and pubk DID")

	provider, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")
	require.Equal(t, provider.DID(), pubDID, "compare provider and pubk DID")

	data := []byte("this is a test")
	sig, err := provider.Sign(data)
	require.NoError(t, err, "provider sign")
	require.NoError(t, anchor.Verify(data, sig), "anchor verify")

	junk := []byte("junk")
	require.Error(t, anchor.Verify(data, junk), "anchor verify")
}

func TestKeyDIDSecp256k1(t *testing.T) {
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Secp256k1)
	require.NoError(t, err, "generate key")

	pubDID := FromPublicKey(pubk)
	require.Equal(t, pubDID.Method(), "key", "did method")

	pubk2, err := PublicKeyFromDID(pubDID)
	require.NoError(t, err, "public key from DID")
	require.Equal(t, pubk2, pubk, "public keys are the same")

	anchor, err := AnchorFromPublicKey(pubk)
	require.NoError(t, err, "anchor from public key")
	require.Equal(t, anchor.DID(), pubDID, "compare anchor and pubk DID")

	provider, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")
	require.Equal(t, provider.DID(), pubDID, "compare provider and pubk DID")

	data := []byte("this is a test")
	sig, err := provider.Sign(data)
	require.NoError(t, err, "provider sign")
	require.NoError(t, anchor.Verify(data, sig), "anchor verify")

	junk := []byte("junk")
	require.Error(t, anchor.Verify(data, junk), "anchor verify")
}
