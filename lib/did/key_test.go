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
