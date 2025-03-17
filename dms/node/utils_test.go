package node

import (
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
)

// createKey creates a key in the keystore.
func createKey(t *testing.T, fs afero.Fs, basePath, contextKey, passphrase string) {
	t.Helper()

	keyStoreDir := filepath.Join(basePath, KeystoreDir)
	ks, err := keystore.New(fs, keyStoreDir)
	require.NoError(t, err)

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)

	rawPriv, err := crypto.MarshalPrivateKey(priv)
	require.NoError(t, err)

	_, err = ks.Save(
		contextKey,
		rawPriv,
		passphrase,
	)
	require.NoError(t, err)
}
