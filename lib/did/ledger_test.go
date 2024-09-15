//go:build ledger
// +build ledger

package did

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerProvider(t *testing.T) {
	prov, err := NewLedgerWalletProvider(0)
	require.NoError(t, err)

	data := []byte("i am a walrus")
	sig, err := prov.Sign(data)
	require.NoError(t, err)

	anchor := prov.Anchor()
	err = anchor.Verify(data, sig)
	require.NoError(t, err)
}

func TestLedgerDID(t *testing.T) {
	prov, err := NewLedgerWalletProvider(0)
	require.NoError(t, err)

	didStr := prov.DID().String()
	did, err := FromString(didStr)
	require.NoError(t, err)

	pubk, err := PublicKeyFromDID(did)
	require.NoError(t, err)

	anchor, err := AnchorFromPublicKey(pubk)
	require.NoError(t, err)

	data := []byte("i am a walrus")
	sig, err := prov.Sign(data)
	require.NoError(t, err)
	err = anchor.Verify(data, sig)
	require.NoError(t, err)
}
