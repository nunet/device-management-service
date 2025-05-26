package did

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/lib/crypto"
)

// happy-path: a key-based DID produces a working Anchor
func TestGetAnchorForDIDKey(t *testing.T) {
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	did := FromPublicKey(pubk)

	anchor, err := GetAnchorForDID(did)
	require.NoError(t, err)
	require.Equal(t, did, anchor.DID())
	require.Equal(t, pubk, anchor.PublicKey())

	// verify that signatures created with the private key validate
	msg := []byte("nunet-test")
	sig, err := privk.Sign(msg)
	require.NoError(t, err)
	require.NoError(t, anchor.Verify(msg, sig))
}

// unsupported DID method should raise ErrNoAnchorMethod
func TestGetAnchorForDIDUnsupportedMethod(t *testing.T) {
	did, err := FromString("did:web:example.com")
	require.NoError(t, err)

	_, err = GetAnchorForDID(did)
	require.ErrorIs(t, err, ErrNoAnchorMethod)
}

// makeKeyAnchor should fail when the identifier isn't a valid key encoding
func TestMakeKeyAnchorInvalidInputs(t *testing.T) {
	cases := []string{
		"did:key:notBase58",  // non-base58 chars
		"did:key:z!!invalid", // base58 “z” prefix but invalid body
	}

	for _, uri := range cases {
		uri := uri // pin loop variable
		t.Run(uri, func(t *testing.T) {
			_, err := makeKeyAnchor(DID{URI: uri})
			require.Error(t, err)
		})
	}
}

func TestGetAnchorForDIDWithInjectedCustomMethod(t *testing.T) {
	// save and restore the original map so we don't break other tests
	orig := anchorMethods
	t.Cleanup(func() { anchorMethods = orig })

	// inject a fake method handler
	anchorMethods["foo"] = func(did DID) (Anchor, error) {
		return NewAnchor(did, nil), nil
	}

	did, err := FromString("did:foo:bar")
	require.NoError(t, err)

	anchor, err := GetAnchorForDID(did)
	require.NoError(t, err)
	require.Equal(t, did, anchor.DID())
}
