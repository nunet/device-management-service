// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
)

func TestNewKey(t *testing.T) {
	t.Parallel()
	testID := "id123"
	testData := []byte("hello world")
	key, err := NewKey(testID, testData)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.NotEmpty(t, key.ID)
	assert.Equal(t, testID, key.ID)
}

func TestPrivKey(t *testing.T) {
	t.Parallel()
	generatedPrivkey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	rawPriv, err := crypto.MarshalPrivateKey(generatedPrivkey)
	assert.NoError(t, err)

	key, err := NewKey("pvkey", rawPriv)
	assert.NoError(t, err)

	pvkey, err := key.PrivKey()
	assert.NoError(t, err)
	assert.Equal(t, pvkey, generatedPrivkey)
}

func TestKeyMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	testID := "id123"
	testData := []byte("hello world")

	key, err := NewKey(testID, testData)
	assert.NoError(t, err)

	// empty passphrase
	data, err := key.MarshalToJSON("")
	assert.ErrorIs(t, err, ErrEmptyPassphrase)
	assert.Nil(t, data)

	// valid passphrase
	data, err = key.MarshalToJSON("1234")
	assert.NoError(t, err)
	assert.NotNil(t, key)

	// unmarshal key with wrong passphrase
	derivedKey, err := UnmarshalKey(data, "222")
	assert.ErrorIs(t, err, ErrMACMismatch)
	assert.Nil(t, derivedKey)

	// unmarshal key with valid passphrase
	derivedKey, err = UnmarshalKey(data, "1234")
	assert.NoError(t, err)
	assert.Equal(t, key, derivedKey)
}

func TestUnmarshalKey(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		keyData     string
		passphrase  string
		expectedErr error
	}{
		"empty passphrase": {
			expectedErr: ErrEmptyPassphrase,
		},
		"empty key": {
			passphrase:  "123",
			expectedErr: ErrDecodeKey,
		},
		"invalid version": {
			passphrase:  "123",
			keyData:     `{}`,
			expectedErr: ErrVersionMismatch,
		},
		"invalid cipher": {
			passphrase:  "123",
			keyData:     `{"version": 3}`,
			expectedErr: ErrCipherMismatch,
		},
		"invalid mac": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "0"}}`,
			expectedErr: ErrDecodeKey,
		},
		"invalid cipherParams": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "cipherparams":{ "iv":"0" }}}`,
			expectedErr: ErrDecodeKey,
		},
		"invalid salt": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "kdfparams": {"salt": "0"}, "cipherparams":{ "iv":"1232" }}}`,
			expectedErr: ErrDecodeKey,
		},
		"invalid cipherText": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "ciphertext": "0", "kdfparams": {"salt": "1232"}, "cipherparams":{ "iv":"1232" }}}`,
			expectedErr: ErrDecodeKey,
		},
		"failed to derive key": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "ciphertext": "1232", "kdfparams": {"salt": "1232"}, "cipherparams":{ "iv":"1232" }}}`,
			expectedErr: ErrKeyProcessing,
		},
		"mac mismatch error with simple data": {
			passphrase:  "123",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1234", "ciphertext": "1234", "kdfparams": {"salt": "1234", "n": 2, "r": 8, "p": 1, "dklen": 64}, "cipherparams":{ "iv":"1234" }}}`,
			expectedErr: ErrMACMismatch,
		},
		"mac mismatch error with hexadecimal data": {
			passphrase:  "correctpassword",
			keyData:     `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a", "ciphertext": "0a0a0a0a0a0a0a0a", "kdfparams": {"salt": "0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a", "n": 2, "r": 8, "p": 1, "dklen": 64}, "cipherparams":{ "iv":"0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a" }}}`,
			expectedErr: ErrMACMismatch,
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			key, err := UnmarshalKey([]byte(tt.keyData), tt.passphrase)
			if tt.expectedErr != nil {
				assert.Nil(t, key)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
			}
		})
	}
}
