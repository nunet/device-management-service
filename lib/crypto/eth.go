// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto

import (
	"crypto/subtle"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
	"golang.org/x/crypto/sha3"
)

var ethSignMagic = []byte(
	"\x19Ethereum Signed Message:\n",
)

type EthPublicKey struct {
	key *secp256k1.PublicKey
}

var _ PubKey = (*EthPublicKey)(nil)

func UnmarshalEthPublicKey(data []byte) (_k PubKey, err error) {
	k, err := secp256k1.ParsePubKey(data)
	if err != nil {
		return nil, err
	}

	return &EthPublicKey{key: k}, nil
}

func (k *EthPublicKey) Verify(data []byte, sigStr []byte) (success bool, err error) {
	sig, err := ecdsa.ParseDERSignature(sigStr)
	if err != nil {
		return false, err
	}

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(ethSignMagic)
	hasher.Write([]byte(fmt.Sprintf("%d", len(data))))
	hasher.Write(data)

	hash := hasher.Sum(nil)
	return sig.Verify(hash, k.key), nil
}

func (k *EthPublicKey) Raw() (res []byte, err error) {
	return k.key.SerializeCompressed(), nil
}

func (k *EthPublicKey) Type() pb.KeyType {
	return Eth
}

func (k *EthPublicKey) Equals(o Key) bool {
	sk, ok := o.(*EthPublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return k.key.IsEqual(sk.key)
}

func basicEquals(k1, k2 Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
