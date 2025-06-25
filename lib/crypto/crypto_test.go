// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestRandomEntropy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		length   int
		hasError bool
	}{
		{"Valid length", 32, false},
		{"Zero length", 0, false},
		{"Negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entropy, err := crypto.RandomEntropy(tt.length)

			if tt.hasError {
				require.Error(t, err)
				require.Nil(t, entropy)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, entropy)
			assert.Equal(t, tt.length, len(entropy))

			if tt.length > 0 {
				// For lengths > 0, consecutive calls should produce different outputs
				entropy2, err := crypto.RandomEntropy(tt.length)
				require.NoError(t, err)
				assert.False(t, bytes.Equal(entropy, entropy2), "Two random entropy calls should produce different values")
			}
		})
	}
}

func TestSha3(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    [][]byte
		expected string // hex encoded hash
	}{
		{
			name:     "Single input",
			input:    [][]byte{[]byte("hello")},
			expected: "3338be694f50c5f338814986cdf0686453a888b84f424d792af4b9202398f392",
		},
		{
			name:     "Multiple inputs",
			input:    [][]byte{[]byte("foo"), []byte("bar")},
			expected: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:     "Empty input",
			input:    [][]byte{},
			expected: "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash, err := crypto.Sha3(tt.input...)
			require.NoError(t, err)

			require.Equal(t, tt.expected, hex.EncodeToString(hash))
		})
	}
}
