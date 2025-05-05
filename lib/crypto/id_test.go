package crypto_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func generateRandomID(t *testing.T) crypto.ID {
	t.Helper()
	_, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	id, err := crypto.IDFromPublicKey(pubKey)
	require.NoError(t, err)
	return id
}

func TestID_Equal(t *testing.T) {
	t.Parallel()

	// Generate two random public keys for testing.
	id1 := generateRandomID(t)
	id2 := generateRandomID(t)

	id1Copy := crypto.ID{PublicKey: id1.PublicKey}

	// Test Equal method of ID.
	tests := []struct {
		name     string
		id1, id2 crypto.ID
		expected bool
	}{
		{"Same ID", id1, id1, true},
		{"Same ID values", id1, id1Copy, true},
		{"Different IDs", id1, id2, false},
		{"Empty ID and non-empty ID", crypto.ID{}, id1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.id1.Equal(tt.id2)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestID_Empty(t *testing.T) {
	t.Parallel()

	id := generateRandomID(t) // non-empty ID

	tests := []struct {
		name     string
		id       crypto.ID
		expected bool
	}{
		{"Empty ID", crypto.ID{}, true},
		{"Non-empty ID", id, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.id.Empty()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestID_String(t *testing.T) {
	t.Parallel()

	id := generateRandomID(t)

	idStr := id.String()
	require.NotEmpty(t, idStr, "ID string representation should not be empty")

	// Convert string back to ID
	decodedID, err := crypto.IDFromString(idStr)
	require.NoError(t, err)

	// Verify that the decoded ID matches the original ID
	require.True(t, id.Equal(decodedID), "Original ID and ID recreated from string should be equal")
}

func TestID_FromStringError(t *testing.T) {
	t.Parallel()

	// Test with invalid base32 string
	_, err := crypto.IDFromString("invalid base32!")
	require.Error(t, err, "Should return error for invalid base32 string")
}

func TestID_Marshaling(t *testing.T) {
	t.Parallel()
	id := generateRandomID(t)

	// Test marshaling
	jsonData, err := json.Marshal(id)
	require.NoError(t, err)
	require.NotEmpty(t, jsonData, "Marshaled JSON should not be empty")

	// Test unmarshaling
	var decodedID crypto.ID
	err = json.Unmarshal(jsonData, &decodedID)
	require.NoError(t, err)

	// Verify that the unmarshaled ID matches the original ID
	require.True(t, id.Equal(decodedID), "Original ID and unmarshaled ID should be equal")

	// Test with empty ID
	emptyID := crypto.ID{}
	emptyJSONData, err := json.Marshal(emptyID)
	require.NoError(t, err)
	require.NotEmpty(t, emptyJSONData)

	// Test unmarshaling empty JSON
	err = json.Unmarshal(emptyJSONData, &decodedID)
	require.NoError(t, err)
	require.True(t, emptyID.Equal(decodedID), "Original empty ID and unmarshaled empty ID should be equal")
}

func TestID_JSONUnmarshaling_Error(t *testing.T) {
	t.Parallel()
	// Test with invalid JSON
	var id crypto.ID
	err := json.Unmarshal([]byte(`{"pub": "invalid base32!"}`), &id)
	require.Error(t, err, "Should return error for invalid pub value")

	// Test with malformed JSON
	err = json.Unmarshal([]byte(`{malformed`), &id)
	require.Error(t, err, "Should return error for malformed JSON")
}

func TestID_RoundTrip(t *testing.T) {
	t.Parallel()
	// Tests a full round trip: key generation -> ID -> JSON -> ID -> public key

	// Generate key pair
	_, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	// Create ID from public key
	id, err := crypto.IDFromPublicKey(pubKey)
	require.NoError(t, err)

	// Marshal ID to JSON
	jsonData, err := json.Marshal(id)
	require.NoError(t, err)

	// Unmarshal JSON back to ID
	var decodedID crypto.ID
	err = json.Unmarshal(jsonData, &decodedID)
	require.NoError(t, err)

	// Get public key from decoded ID
	pubKeyFromID, err := crypto.PublicKeyFromID(decodedID)
	require.NoError(t, err)

	// Verify that the public key from ID matches the original public key
	originalKeyBytes, err := crypto.PublicKeyToBytes(pubKey)
	require.NoError(t, err)

	pubKeyFromIDBytes, err := crypto.PublicKeyToBytes(pubKeyFromID)
	require.NoError(t, err)

	require.Equal(t, originalKeyBytes, pubKeyFromIDBytes, "Original and public key from ID should match")
}
