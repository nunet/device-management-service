package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalCardanoPublicKey(t *testing.T) {
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cardanoKey, err := UnmarshalCardanoPublicKey(pubKey)
	require.NoError(t, err)
	assert.NotNil(t, cardanoKey)
	assert.Equal(t, pb.KeyType(Cardano), cardanoKey.Type())

	raw, err := cardanoKey.Raw()
	require.NoError(t, err)
	assert.Equal(t, []byte(pubKey), raw)
}

func TestCardanoPublicKey_Verify(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cardanoKey, err := UnmarshalCardanoPublicKey(pubKey)
	require.NoError(t, err)

	data := []byte("hello world")
	protected := []byte{0xa1, 0x01, 0x27} // Example protected header

	// Construct Sig_structure to sign
	// Sig_structure = [
	//   context: "Signature1",
	//   body_protected: bstr,
	//   external_aad: bstr,
	//   payload: bstr
	// ]
	sigStruct := []interface{}{
		"Signature1",
		protected,
		[]byte{}, // external AAD empty
		data,
	}

	sigStructBytes, err := cbor.Marshal(sigStruct)
	require.NoError(t, err)

	signature := ed25519.Sign(privKey, sigStructBytes)

	// Construct COSE_Sign1
	// COSE_Sign1 = [
	//   protected: bstr,
	//   unprotected: map,
	//   payload: bstr / nil,
	//   signature: bstr
	// ]
	coseSign1 := []interface{}{
		protected,
		map[interface{}]interface{}{},
		data,
		signature,
	}

	sigBytes, err := cbor.Marshal(coseSign1)
	require.NoError(t, err)

	valid, err := cardanoKey.Verify(data, sigBytes)
	require.NoError(t, err)
	assert.True(t, valid)

	// Test invalid signature
	invalidSignature := make([]byte, len(signature))
	copy(invalidSignature, signature)
	invalidSignature[0] ^= 0xFF
	coseSign1Invalid := []interface{}{
		protected,
		map[interface{}]interface{}{},
		data,
		invalidSignature,
	}
	sigBytesInvalid, err := cbor.Marshal(coseSign1Invalid)
	require.NoError(t, err)

	valid, err = cardanoKey.Verify(data, sigBytesInvalid)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestCardanoPublicKey_Equals(t *testing.T) {
	pubKey1, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	pubKey2, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	k1, err := UnmarshalCardanoPublicKey(pubKey1)
	require.NoError(t, err)
	k2, err := UnmarshalCardanoPublicKey(pubKey2)
	require.NoError(t, err)
	k1Copy, err := UnmarshalCardanoPublicKey(pubKey1)
	require.NoError(t, err)

	assert.True(t, k1.Equals(k1Copy))
	assert.False(t, k1.Equals(k2))
}

func TestEternlSignature(t *testing.T) {
	sigHex := "845846a201276761646472657373583900b1411a4480087f2f3095f6936877315087ee08a7e91fc0cd349e98bcff10f17d16e937ba8671a2d7e6258e98c405f7fd6f91deeb7409b8b6a166686173686564f44a6765745f7075626b657958406f98987538f5e667719315cbd9bed373af4f1bd9c26ae1a636eefe52f3ee1c793fd1f6ecf91e5b8955fa0ced7d40c99e80d1070c7c62cbffd6283770d56ff505"
	keyHex := "a40101032720062158203472bf94f1431735449e5aa537d2d5157a92a275ca8c6886d22da580deffa410"

	sigBytes, err := hex.DecodeString(sigHex)
	require.NoError(t, err)

	keyBytes, err := hex.DecodeString(keyHex)
	require.NoError(t, err)

	cardanoKey, err := UnmarshalCardanoPublicKey(keyBytes)
	require.NoError(t, err)

	var coseSign1 []interface{}
	err = cbor.Unmarshal(sigBytes, &coseSign1)
	require.NoError(t, err)
	require.Len(t, coseSign1, 4)

	payload, ok := coseSign1[2].([]byte)
	require.True(t, ok, "payload should be present")
	assert.Equal(t, []byte("get_pubkey"), payload)

	valid, err := cardanoKey.Verify(payload, sigBytes)
	require.NoError(t, err)
	assert.True(t, valid)
}
