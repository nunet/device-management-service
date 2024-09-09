package keystore

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
)

func TestNewKey(t *testing.T) {
	testID := "id123"
	testData := []byte("hello world")
	key, err := NewKey(testID, testData)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.NotEmpty(t, key.ID)
	assert.Equal(t, testID, key.ID)
}

func TestPrivKey(t *testing.T) {
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

// fixme: wrong stringfied jsons
// func TestUnmarshalKey(t *testing.T) {
// 	t.Parallel()
// 	cases := map[string]struct {
// 		keyData    string
// 		passphrase string
// 		expErr     string
// 	}{
// 		"empty passphrase": {
// 			expErr: ErrEmptyPassphrase.Error(),
// 		},
// 		"empty key": {
// 			passphrase: "123",
// 			expErr:     "failed to unmarshal key data: unexpected end of JSON input",
// 		},
// 		"invalid version": {
// 			passphrase: "123",
// 			keyData:    `{}`,
// 			expErr:     ErrVersionMismatch.Error(),
// 		},
// 		"invalid cipher": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3}`,
// 			expErr:     ErrCipherMismatch.Error(),
// 		},
// 		"invalid mac": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "0"}}`,
// 			expErr:     "failed to decode mac: encoding/hex: odd length hex string",
// 		},
// 		"invalid cipherParams": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "cipherparams":{ "iv":"0" }}}`,
// 			expErr:     "failed to decode cipher params iv: encoding/hex: odd length hex string",
// 		},
// 		"invalid salt": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "kdfparams": {"salt": "0"}, "cipherparams":{ "iv":"1232" }}}`,
// 			expErr:     "failed to decode salt: encoding/hex: odd length hex string",
// 		},
// 		"invalid cipherText": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "ciphertext": "0", "kdfparams": {"salt": "1232"}, "cipherparams":{ "iv":"1232" }}}`,
// 			expErr:     "failed to decode cipher text: encoding/hex: odd length hex string",
// 		},
// 		"failed to derive key": {
// 			passphrase: "123",
// 			keyData:    `{"version": 3, "crypto":{"cipher":"aes-256-ctr", "mac": "1232", "ciphertext": "1232", "kdfparams": {"salt": "1232"}, "cipherparams":{ "iv":"1232" }}}`,
// 			expErr:     "failed to derive key: scrypt: N must be > 1 and a power of 2",
// 		},
// 		"mac mismatch": {
// 			passphrase: "1234",
// 			keyData:    `{ "crypto": { "cipher": "aes-256-ctr", "ciphertext": "", "cipherparams": { "iv": "2e250214b665831ad7a5ed84508445e2" }, "kdf": "scrypt", "kdfparams": { "n": 262144, "r": 8, "p": 1, "dklen": 32, "salt": "cf2a22196d7865aaa23fea7d6eea03a93270edf23bebe8c9297d0b20db82d39d" }, "mac": "ba035e813a993cfdcf621915b6b55e5470bbfb67e0ecaa30d286cd6e7fe34b69" }, "id": "252841d8-393e-42ea-a793-9f9860cb32d3", "version": 3 }`,
// 			expErr:     ErrMACMismatch.Error(),
// 		},
// 		"success": {
// 			passphrase: "1234",
// 			keyData:    `{"version":3,"id":"id123","crypto":{"cipher":"aes-256-ctr","ciphertext":"fb62c5e577779c11c1632e","cipherparams":{"iv":"34355762a7ba935d7ebcf62e20ea49fb"},"kdf":"scrypt","kdfparams":{"n":262144,"r":8,"p":1,"dklen":64,"salt":"13c31c8edba13966ca18a1fd8610e0bc86708cd2d7f51f450c67f30ba3f5a2d772933d17c31a12993a9977293863bca795101fac16201dcf6fd16362b134d4"},"mac":"8215d7206726fc3088f2c2be47d090674a4fb09450c8649d438d513dc6fbcd9e"}}`,
// 		},
// 	}
//
// 	for name, tt := range cases {
// 		tt := tt
// 		t.Run(name, func(t *testing.T) {
// 			t.Parallel()
// 			key, err := UnmarshalKey([]byte(tt.keyData), tt.passphrase)
// 			if tt.expErr != "" {
// 				assert.Nil(t, key)
// 				assert.EqualError(t, err, tt.expErr)
// 			} else {
// 				assert.NoError(t, err)
// 				assert.NotNil(t, key)
// 			}
// 		})
// 	}
// }
