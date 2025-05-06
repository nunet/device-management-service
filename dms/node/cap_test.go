package node

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestCap(t *testing.T) {
	t.Parallel()

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

		basePath := t.TempDir()
		afs := afero.Afero{Fs: afero.NewOsFs()}
		contextKey := "context"
		passphrase := "passphrase"

		createKey(t, afs.Fs, basePath, contextKey, passphrase)

		trustCtx, privKey, err := CreateTrustContextFromKeyStore(
			afs, contextKey, passphrase, basePath)
		require.NoError(t, err)
		require.NotNil(t, trustCtx)
		require.NotNil(t, privKey)

		// Get the trust context
		savedTrustCtx, err := GetTrustContext(afs, contextKey, passphrase, basePath)
		require.NoError(t, err)
		require.NotNil(t, savedTrustCtx)
	})

	t.Run("must be able to save and load capability context", func(t *testing.T) {
		t.Parallel()

		basePath := t.TempDir()
		afs := afero.Afero{Fs: afero.NewOsFs()}
		contextKey := "context"
		passphrase := "passphrase"

		createKey(t, afs.Fs, basePath, contextKey, passphrase)

		trustCtx, privKey, err := CreateTrustContextFromKeyStore(
			afs, contextKey, passphrase, basePath)
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

		err = SaveCapabilityContext(capCtx, basePath)
		require.NoError(t, err)

		savedCapCtx, err := LoadCapabilityContext(trustCtx, contextKey, basePath)
		require.NoError(t, err)
		require.NotNil(t, savedCapCtx)
		require.Equal(t, capCtx, savedCapCtx)
	})
}
