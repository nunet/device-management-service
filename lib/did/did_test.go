package did

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDID(t *testing.T) {
	validDIDString := "did:example:123456789abcdefghi"
	invalidDIDString := "invalid:did"
	emptyPartDIDString := "did::invalid"
	tooManyPartsDIDString := "did:example:123:456"

	validMethod := "example"
	validIdentifier := "123456789abcdefghi"

	// Test FromString
	_, err := FromString(validDIDString)
	assert.NoError(t, err, "FromString failed for valid DID")

	_, err = FromString(invalidDIDString)
	assert.ErrorIs(t, err, ErrInvalidDID)

	// Test Equal
	did1 := DID{URI: validDIDString}
	did2 := DID{URI: validDIDString}
	did3 := DID{URI: "did:example:987654321ihgfedcba"}

	assert.True(t, did1.Equal(did2), "Equal failed for identical DIDs")
	assert.False(t, did1.Equal(did3), "Equal should have failed for different DIDs")

	// Test Empty
	emptyDID := DID{}
	assert.True(t, emptyDID.Empty(), "Empty failed for empty DID")
	assert.False(t, did1.Empty(), "Empty should have failed for non-empty DID")

	// Test String
	assert.Equal(t, validDIDString, did1.String(), "String failed to return correct URI")

	// Test Method
	assert.Equal(t, validMethod, did1.Method(), "Method failed to return correct value")

	// Test Identifier
	assert.Equal(t, validIdentifier, did1.Identifier(), "Identifier failed to return correct value")

	// Test invalid DID
	_, err = FromString(emptyPartDIDString)
	assert.Error(t, err, "FromString should have failed for DID with empty parts")

	// Test DID with more than 3 parts
	_, err = FromString(tooManyPartsDIDString)
	assert.Error(t, err, "FromString should have failed for DID with more than 3 parts")
}
