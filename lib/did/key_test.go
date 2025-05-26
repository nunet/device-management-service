// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
	"github.com/multiformats/go-multibase"
	varint "github.com/multiformats/go-varint"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

type bogusKey struct{}

func (bogusKey) Verify([]byte, []byte) (bool, error) { return false, nil }
func (bogusKey) Raw() ([]byte, error)                { return []byte{1, 2, 3}, nil }
func (bogusKey) Type() pb.KeyType                    { return pb.KeyType(0xaa) } // unknown
func (bogusKey) Equals(crypto.Key) bool              { return false }

// stub key whose Raw() errors → FormatKeyURI must return empty string
type badRawKey struct{}

func (badRawKey) Verify([]byte, []byte) (bool, error) { return false, nil }
func (badRawKey) Raw() ([]byte, error)                { return nil, fmt.Errorf("raw failure") }
func (badRawKey) Type() pb.KeyType                    { return pb.KeyType_Ed25519 }
func (badRawKey) Equals(crypto.Key) bool              { return false }

func TestKeyDIDRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		genFunc func() (crypto.PrivKey, crypto.PubKey, error)
	}{
		{
			name: "Ed25519",
			genFunc: func() (crypto.PrivKey, crypto.PubKey, error) {
				return crypto.GenerateKeyPair(crypto.Ed25519)
			},
		},
		{
			name: "Secp256k1",
			genFunc: func() (crypto.PrivKey, crypto.PubKey, error) {
				return crypto.GenerateKeyPair(crypto.Secp256k1)
			},
		},
	}

	for _, tc := range cases {
		tc := tc // pin loop var
		t.Run(tc.name, func(t *testing.T) {
			privk, pubk, err := tc.genFunc()
			require.NoError(t, err)

			did := FromPublicKey(pubk)
			require.Equal(t, "key", did.Method())

			recovered, err := PublicKeyFromDID(did)
			require.NoError(t, err)
			require.Equal(t, pubk, recovered)

			anchor, err := AnchorFromPublicKey(pubk)
			require.NoError(t, err)

			prov, err := ProviderFromPrivateKey(privk)
			require.NoError(t, err)

			msg := []byte("round-trip-test")
			sig, err := prov.Sign(msg)
			require.NoError(t, err)
			require.NoError(t, anchor.Verify(msg, sig))
		})
	}
}

// unexpected multibase prefix (“u” means base64)
func TestParseKeyURIInvalidMultibase(t *testing.T) {
	_, err := ParseKeyURI("did:key:uSGVsbG8")
	require.ErrorContains(t, err, "unexpected multibase")
}

// unsupported codec → ErrInvalidKeyType
func TestParseKeyURIUnsupportedCodec(t *testing.T) {
	unknownCodec := uint64(0x99)
	raw := make([]byte, varint.UvarintSize(unknownCodec)+32)
	n := varint.PutUvarint(raw, unknownCodec)

	keyBytes, err := hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	require.NoError(t, err)
	copy(raw[n:], keyBytes)

	enc, err := multibase.Encode(multibase.Base58BTC, raw)
	require.NoError(t, err)

	_, err = ParseKeyURI("did:key:" + enc)
	require.ErrorIs(t, err, ErrInvalidKeyType)
}

// Eth-codec coverage (GenerateKeyPair doesn’t support Eth, so we craft it)
func TestKeyDIDEth(t *testing.T) {
	// 1) secp256k1 private key (Ethereum curve)
	sk, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)

	// 2) Wrap its public part into EthPublicKey
	pubBytes := sk.PubKey().SerializeCompressed()
	pubk, err := crypto.UnmarshalEthPublicKey(pubBytes)
	require.NoError(t, err)

	// 3) DID ↔ PubKey round-trip
	did := FromPublicKey(pubk)
	require.Equal(t, "key", did.Method())

	recovered, err := PublicKeyFromDID(did)
	require.NoError(t, err)
	require.Equal(t, pubk, recovered)

	// 4) Build anchor and create an Ethereum-style signature
	anchor, err := AnchorFromPublicKey(pubk)
	require.NoError(t, err)

	msg := []byte("nunet-eth-test")
	const ethMagic = "\x19Ethereum Signed Message:\n"

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(ethMagic))
	fmt.Fprintf(hasher, "%d", len(msg))
	hasher.Write(msg)
	hash := hasher.Sum(nil)

	sig := secpECDSA.Sign(sk, hash) // one return value
	require.NoError(t, anchor.Verify(msg, sig.Serialize()))
	require.Error(t, anchor.Verify([]byte("tamper"), sig.Serialize()))
}

// FromID → DID → PubKey round-trip *plus* invalid-ID branch.
func TestFromIDRoundTripAndFailure(t *testing.T) {
	// happy path
	_, pubk, _ := crypto.GenerateKeyPair(crypto.Ed25519)
	id, _ := crypto.IDFromPublicKey(pubk)

	didFromID, err := FromID(id)
	require.NoError(t, err)

	recovered, err := PublicKeyFromDID(didFromID)
	require.NoError(t, err)
	require.Equal(t, pubk, recovered)

	// failure path: garbage bytes that cannot unmarshal to a key
	badID := crypto.ID{PublicKey: []byte("not-a-valid-key")}
	_, err = FromID(badID)
	require.Error(t, err)
}

// missing key bytes after a valid codec varint → varint succeeds, Unmarshal fails
func TestParseKeyURITruncatedPayload(t *testing.T) {
	// varint for Ed25519 codec only (0xed) – no key bytes appended
	const edCodec = 0xed
	buf := make([]byte, varint.UvarintSize(edCodec))
	varint.PutUvarint(buf, edCodec)

	enc, _ := multibase.Encode(multibase.Base58BTC, buf) // z<...>
	_, err := ParseKeyURI("did:key:" + enc)

	require.Error(t, err, "expected failure for truncated payload")
}

// unsupported key type in FormatKeyURI → should return ""
func TestFormatKeyURIUnsupportedType(t *testing.T) {
	uri := FormatKeyURI(bogusKey{})
	require.Equal(t, "", uri, "unsupported key type must yield empty URI")
}

// ParseKeyURI must fail if the URI doesn’t start with "did:key:"
func TestParseKeyURIMissingPrefix(t *testing.T) {
	_, err := ParseKeyURI("did:web:xyz")
	require.Error(t, err, "expected failure for wrong DID method")
}

// PublicKeyFromDID returns ErrInvalidDID when method ≠ "key".
func TestPublicKeyFromDIDWrongMethod(t *testing.T) {
	did, err := FromString("did:web:example.com")
	require.NoError(t, err)

	_, err = PublicKeyFromDID(did)
	require.ErrorIs(t, err, ErrInvalidDID)
}

// invalid base58 string ➜ ParseKeyURI should error while decoding multibase
func TestParseKeyURIInvalidBase58(t *testing.T) {
	_, err := ParseKeyURI("did:key:z!@#$") // '!' breaks base58btc decoding
	require.Error(t, err, "expected multibase decode failure")
	require.Contains(t, err.Error(), "decoding multibase")
}

func TestFormatKeyURIErrorOnRaw(t *testing.T) {
	uri := FormatKeyURI(badRawKey{})
	require.Equal(t, "", uri, "FormatKeyURI should return empty string on Raw() error")
}
