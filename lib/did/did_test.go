// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test that an *empty string* is accepted and yields a zero-value DID.
func TestDIDFromStringEmptyString(t *testing.T) {
	d, err := FromString("")
	require.NoError(t, err, "empty string must not error")
	assert.True(t, d.Empty(), "zero DID should be empty")
	assert.Empty(t, d.String())
	assert.Equal(t, "", d.Method())
	assert.Equal(t, "", d.Identifier())
}

// Verify Method() and Identifier() gracefully fall back to empty strings for
// various malformed URIs that bypass FromString validation.
func TestDIDMethodIdentifierInvalidURIs(t *testing.T) {
	broken := []string{
		"did:key", // missing identifier
		"did::",   // empty method + identifier
		"notaDID", // not even a DID
	}

	for _, uri := range broken {
		d := DID{URI: uri}
		assert.Equalf(t, "", d.Method(), "Method should be blank for %q", uri)
		assert.Equalf(t, "", d.Identifier(), "Identifier should be blank for %q", uri)
	}
}

// Extra checks on Equal(): reflexive, symmetric, and negative cases.
func TestDIDEqualSymmetricAndReflexive(t *testing.T) {
	a := DID{URI: "did:key:abc"}
	b := DID{URI: "did:key:abc"}
	c := DID{URI: "did:key:def"}
	//nolint:gocritic // intentional: testing reflexivity of Equal()
	assert.True(t, a.Equal(a), "reflexivity failed")
	assert.True(t, a.Equal(b) && b.Equal(a), "symmetry failed for identical URIs")
	assert.False(t, a.Equal(c))
	assert.False(t, c.Equal(a))
}
