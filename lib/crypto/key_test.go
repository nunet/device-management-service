package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestAllowedKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		keyType  int
		expected bool
	}{
		{"Ed25519 allowed", crypto.Ed25519, true},
		{"Secp256k1 allowed", crypto.Secp256k1, true},
		{"Unknown not allowed", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := crypto.AllowedKey(tt.keyType)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateKeyPair(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		keyType  int
		hasError bool
	}{
		{"Generate Ed25519", crypto.Ed25519, false},
		{"Generate Secp256k1", crypto.Secp256k1, false},
		{"Invalid Key Type", 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			priv, pub, err := crypto.GenerateKeyPair(tt.keyType)
			if tt.hasError {
				// For unsupported key types, expect an error.
				require.Error(t, err, "expected an error for invalid key type")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, priv)
			require.NotNil(t, pub)

			// Verify that the derived public key from the private key matches the returned public key.
			derivedPub := priv.GetPublic()
			require.True(t, pub.Equals(derivedPub), "Public key should match derived public key from private key")
		})
	}
}

func TestKeyEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		keyType int
	}{
		{"Ed25519 encoding", crypto.Ed25519},
		{"Secp256k1 encoding", crypto.Secp256k1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			priv, pub, err := crypto.GenerateKeyPair(tt.keyType)
			require.NoError(t, err)

			// Encode and decode private key.
			privBytes, err := crypto.PrivateKeyToBytes(priv)
			require.NoError(t, err)
			decodedPriv, err := crypto.BytesToPrivateKey(privBytes)
			require.NoError(t, err)
			// Verify that the decoded private key matches the original private key.
			require.True(t, priv.Equals(decodedPriv), "Private key should match decoded private key")

			// Encode and decode public key.
			pubBytes, err := crypto.PublicKeyToBytes(pub)
			require.NoError(t, err)
			decodedPub, err := crypto.BytesToPublicKey(pubBytes)
			require.NoError(t, err)
			// Verify that the decoded public key matches the original public key.
			require.True(t, pub.Equals(decodedPub), "Public key should match decoded public key")
		})
	}
}

func TestInvalidKeyDecoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		isPublic bool
	}{
		{"Invalid private key bytes", []byte("not-a-valid-key"), false},
		{"Invalid public key bytes", []byte("not-a-valid-key"), true},
		{"Empty private key", nil, false},
		{"Empty public key", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tt.isPublic {
				_, err = crypto.BytesToPublicKey(tt.input)
			} else {
				_, err = crypto.BytesToPrivateKey(tt.input)
			}
			require.Error(t, err)
		})
	}
}

func TestIDConversion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		keyType int
	}{
		{"Ed25519 to/from ID conversion", crypto.Ed25519},
		{"Secp256k1 to/from ID conversion", crypto.Secp256k1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, pub, err := crypto.GenerateKeyPair(tt.keyType)
			require.NoError(t, err)

			// Convert public key to ID.
			id, err := crypto.IDFromPublicKey(pub)
			require.NoError(t, err)

			// Convert ID back to public key.
			pub2, err := crypto.PublicKeyFromID(id)
			require.NoError(t, err)

			// Verify that the public key matches the public key from ID.
			require.True(t, pub.Equals(pub2), "Public key should match public key from ID")
		})
	}
}
