package did

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestTrustContext(t *testing.T) {
	ctx := NewTrustContext()

	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	pubDID := FromPublicKey(pubk)
	anchor, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor.DID(), pubDID, "compare anchor DID with pubk DID")

	anchor2, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor2, anchor, "cached anchor must equal the initial")

	provider, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")
	ctx.AddProvider(provider)

	provider2, err := ctx.GetProvider(provider.DID())
	require.NoError(t, err, "get key provider")
	require.Equal(t, provider2, provider, "cached provider must equal the initial")

	require.Equal(t, ctx.Anchors(), []DID{pubDID}, "anchor list")
	require.Equal(t, ctx.Providers(), []DID{pubDID}, "provider list")
}
