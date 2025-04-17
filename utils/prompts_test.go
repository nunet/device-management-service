// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: PromptForPassphrase needs mock for term.ReadPassword

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedResult bool
		expectedError  bool
	}{
		{
			name:           "yes response",
			input:          "yes\n",
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "y response",
			input:          "y\n",
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "no response",
			input:          "no\n",
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "n response",
			input:          "n\n",
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "uppercase YES response",
			input:          "YES\n",
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "mixed case No response",
			input:          "No\n",
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "invalid then valid response",
			input:          "invalid\ny\n",
			expectedResult: true,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}

			result, err := PromptYesNo(in, out, "Test prompt")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify the prompt was written to output
			assert.Contains(t, out.String(), "Test prompt")
		})
	}
}

func TestPromptReonboard(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError bool
		errorContains string
	}{
		{
			name:          "user confirms reonboard",
			input:         "y\n",
			expectedError: false,
		},
		{
			name:          "user declines reonboard",
			input:         "n\n",
			expectedError: true,
			errorContains: "reonboarding aborted by user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}

			err := PromptReonboard(in, out)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify the prompt was written to output
			assert.Contains(t, out.String(), reonboardPrompt)
		})
	}
}
