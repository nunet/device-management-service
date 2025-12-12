// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto

import (
	"bytes"
	"crypto/ed25519"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
)

type CardanoPublicKey struct {
	key *ed25519.PublicKey
}

var _ PubKey = (*CardanoPublicKey)(nil)

func UnmarshalCardanoPublicKey(data []byte) (_k PubKey, err error) {
	if len(data) == ed25519.PublicKeySize {
		pubKey := ed25519.PublicKey(data)
		return &CardanoPublicKey{key: &pubKey}, nil
	}

	// Try to unmarshal as COSE Key
	var coseKey map[int]interface{}
	if err := cbor.Unmarshal(data, &coseKey); err != nil {
		return nil, fmt.Errorf("invalid cardano public key: %w", err)
	}

	// Check kty = 1 (OKP)
	if kty, ok := coseKey[1].(uint64); !ok || kty != 1 {
		return nil, fmt.Errorf("invalid COSE Key: kty must be 1 (OKP)")
	}

	// Check crv = 6 (Ed25519)
	// crv is label -1. In CBOR negative integers are distinct types.
	// map[int]interface{} might not work if the key is negative?
	// cbor library handles map keys.
	// fxamacker/cbor supports map[int]interface{}. Negative integers are just negative ints in Go.

	// crv (-1)
	if crv, ok := coseKey[-1].(uint64); !ok || crv != 6 {
		return nil, fmt.Errorf("invalid COSE Key: crv must be 6 (Ed25519)")
	}

	// x (-2)
	x, ok := coseKey[-2].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid COSE Key: missing x parameter")
	}

	if len(x) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid COSE Key: x parameter length %d", len(x))
	}

	pubKey := ed25519.PublicKey(x)
	return &CardanoPublicKey{key: &pubKey}, nil
}

func (k *CardanoPublicKey) Verify(data []byte, sigBytes []byte) (success bool, err error) {
	dec := cbor.NewDecoder(bytes.NewReader(sigBytes))
	var coseSign1 []interface{}
	if err := dec.Decode(&coseSign1); err != nil {
		return false, fmt.Errorf("failed to decode COSE_Sign1: %w", err)
	}

	if len(coseSign1) != 4 {
		return false, fmt.Errorf("unexpected COSE_Sign1 length: %d", len(coseSign1))
	}

	protected, ok := coseSign1[0].([]byte)
	if !ok {
		return false, fmt.Errorf("invalid protected headers")
	}

	signature, ok := coseSign1[3].([]byte)
	if !ok {
		return false, fmt.Errorf("invalid signature bytes")
	}

	sigStruct := []interface{}{
		"Signature1",
		protected,
		[]byte{}, // external AAD empty
		data,
	}

	sigStructBytes, err := cbor.Marshal(sigStruct)
	if err != nil {
		return false, fmt.Errorf("failed to marshal SigStructure: %w", err)
	}

	valid := ed25519.Verify(*k.key, sigStructBytes, signature)

	return valid, nil
}

func (k *CardanoPublicKey) Raw() (res []byte, err error) {
	return []byte(*k.key), nil
}

func (k *CardanoPublicKey) Type() pb.KeyType {
	return Cardano
}

func (k *CardanoPublicKey) Equals(o Key) bool {
	sk, ok := o.(*CardanoPublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return k.key.Equal(*sk.key)
}
