// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/scrypt"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

var (
	// KDF parameters
	nameKDF      = "scrypt"
	scryptKeyLen = 64

	// Default scrypt parameters (production values)
	defaultScryptN = 1 << 18
	defaultScryptR = 8
	defaultScryptP = 1

	// Current scrypt parameters (can be overridden for testing)
	scryptN = defaultScryptN
	scryptR = defaultScryptR
	scryptP = defaultScryptP

	ksVersion = 3
	ksCipher  = "aes-256-ctr"
)

// SetTestScryptParams allows overriding scrypt parameters for testing purposes.
// This significantly speeds up key generation/derivation but reduces security.
// IMPORTANT: Only use this in test environments.
func SetTestScryptParams(n, r, p int) {
	scryptN = n
	scryptR = r
	scryptP = p
}

// ResetScryptParamsToDefaults restores scrypt parameters to their original production values.
func ResetScryptParamsToDefaults() {
	scryptN = defaultScryptN
	scryptR = defaultScryptR
	scryptP = defaultScryptP
}

// Key represents a keypair to be stored in a keystore
type Key struct {
	ID   string
	Data []byte
}

// NewKey creates new Key
func NewKey(id string, data []byte) (*Key, error) {
	return &Key{
		ID:   id,
		Data: data,
	}, nil
}

// PrivKey acts upon a Key which its `Data` is a private key.
// The method unmarshals the raw pvkey bytes.
func (key *Key) PrivKey() (crypto.PrivKey, error) {
	priv, err := libp2p_crypto.UnmarshalPrivateKey(key.Data)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal private key: %v", err)
	}
	return priv, nil
}

// MarshalToJSON encrypts and marshals a key to json byte array.
func (key *Key) MarshalToJSON(passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, ErrEmptyPassphrase
	}
	salt, err := crypto.RandomEntropy(64)
	if err != nil {
		return nil, err
	}
	dk, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, err
	}
	iv, err := crypto.RandomEntropy(aes.BlockSize)
	if err != nil {
		return nil, err
	}
	enckey := dk[:32]

	aesBlock, err := aes.NewCipher(enckey)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	cipherText := make([]byte, len(key.Data))
	stream.XORKeyStream(cipherText, key.Data)

	mac, err := crypto.Sha3(dk[32:64], cipherText)
	if err != nil {
		return nil, err
	}
	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	sp := ScryptParams{
		N:          scryptN,
		R:          scryptR,
		P:          scryptP,
		DKeyLength: scryptKeyLen,
		Salt:       hex.EncodeToString(salt),
	}

	keyjson := cryptoJSON{
		Cipher:       ksCipher,
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          nameKDF,
		KDFParams:    sp,
		MAC:          hex.EncodeToString(mac),
	}

	encjson := encryptedKeyJSON{
		Crypto:  keyjson,
		ID:      key.ID,
		Version: ksVersion,
	}
	data, err := json.MarshalIndent(&encjson, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

// UnmarshalKey decrypts and unmarhals the private key
func UnmarshalKey(data []byte, passphrase string) (*Key, error) {
	if passphrase == "" {
		return nil, ErrEmptyPassphrase
	}
	encjson := encryptedKeyJSON{}
	if err := json.Unmarshal(data, &encjson); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeKey, err)
	}
	if encjson.Version != ksVersion {
		return nil, ErrVersionMismatch
	}
	if encjson.Crypto.Cipher != ksCipher {
		return nil, ErrCipherMismatch
	}
	mac, err := hex.DecodeString(encjson.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("%w: mac: %w", ErrDecodeKey, err)
	}
	iv, err := hex.DecodeString(encjson.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher params: %w", ErrDecodeKey, err)
	}
	salt, err := hex.DecodeString(encjson.Crypto.KDFParams.Salt)
	if err != nil {
		return nil, fmt.Errorf("%w: salt: %w", ErrDecodeKey, err)
	}
	ciphertext, err := hex.DecodeString(encjson.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher text: %w", ErrDecodeKey, err)
	}
	dk, err := scrypt.Key([]byte(passphrase), salt, encjson.Crypto.KDFParams.N, encjson.Crypto.KDFParams.R, encjson.Crypto.KDFParams.P, encjson.Crypto.KDFParams.DKeyLength)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyProcessing, err)
	}
	hash, err := crypto.Sha3(dk[32:64], ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyProcessing, err)
	}
	if !bytes.Equal(hash, mac) {
		return nil, ErrMACMismatch
	}
	aesBlock, err := aes.NewCipher(dk[:32])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeKey, err)
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outputkey := make([]byte, len(ciphertext))
	stream.XORKeyStream(outputkey, ciphertext)

	return &Key{
		ID:   encjson.ID,
		Data: outputkey,
	}, nil
}

func removeFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}
