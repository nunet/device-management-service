package node

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestCap(t *testing.T) {
	t.Parallel()

	userDir := "/tmp/dms/user"

	t.Run("must be able to identify a ledger context", func(t *testing.T) {
		tests := []struct {
			name     string
			context  string
			expected bool
		}{
			{
				name:     "ledger context",
				context:  "ledger:context",
				expected: true,
			},
			{
				name:     "not ledger context",
				context:  "context",
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				actual := IsLedgerContext(tt.context)
				if actual != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, actual)
				}
			})
		}
	})

	t.Run("must be able to get the context key", func(t *testing.T) {
		tests := []struct {
			name     string
			context  string
			expected string
		}{
			{
				name:     "ledger context",
				context:  "ledger:context",
				expected: "context",
			},
			{
				name:     "not ledger context",
				context:  "notledger:context",
				expected: "context",
			},
			{
				name:     "without prefix",
				context:  "context",
				expected: "context",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				actual := GetContextKey(tt.context)
				if actual != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, actual)
				}
			})
		}
	})

	t.Run("must be able to create and get trust context", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		keysDir := filepath.Join(userDir, KeystoreDir)

		contextKey := "context"
		passphrase := "passphrase"

		createKey(t, fs, keysDir, contextKey, passphrase)

		trustCtx, privKey, err := CreateTrustContextFromKeyStore(fs, contextKey, passphrase, keysDir)
		require.NoError(t, err)
		require.NotNil(t, trustCtx)
		require.NotNil(t, privKey)

		// Get the trust context
		savedTrustCtx, err := GetTrustContext(fs, contextKey, passphrase, keysDir)
		require.NoError(t, err)
		require.NotNil(t, savedTrustCtx)
	})

	t.Run("must be able to save and load capability context", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		keysDir := filepath.Join(userDir, KeystoreDir)
		capsDir := filepath.Join(userDir, CapstoreDir)

		contextKey := "context"
		passphrase := "passphrase"

		createKey(t, fs, keysDir, contextKey, passphrase)

		trustCtx, privKey, err := CreateTrustContextFromKeyStore(
			fs, contextKey, passphrase, keysDir)
		require.NoError(t, err)
		require.NotNil(t, trustCtx)
		require.NotNil(t, privKey)

		capCtx, err := ucan.NewCapabilityContextWithName(contextKey,
			trustCtx,
			did.FromPublicKey(privKey.GetPublic()),
			nil,
			ucan.TokenList{},
			ucan.TokenList{},
			ucan.TokenList{},
		)
		require.NoError(t, err)

		err = SaveCapabilityContext(capCtx, fs, capsDir)
		require.NoError(t, err)

		savedCapCtx, err := LoadCapabilityContext(trustCtx, fs, contextKey, capsDir)
		require.NoError(t, err)
		require.NotNil(t, savedCapCtx)
		require.Equal(t, capCtx, savedCapCtx)
	})
}
