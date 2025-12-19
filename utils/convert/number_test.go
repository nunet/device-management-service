// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringToFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		fieldName string
		want      float64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid float",
			input:     "123.45",
			fieldName: "test_field",
			want:      123.45,
			wantErr:   false,
		},
		{
			name:      "valid integer as float",
			input:     "100",
			fieldName: "test_field",
			want:      100.0,
			wantErr:   false,
		},
		{
			name:      "empty string",
			input:     "",
			fieldName: "test_field",
			want:      0,
			wantErr:   true,
			errMsg:    "test_field is required and cannot be empty",
		},
		{
			name:      "invalid format",
			input:     "not_a_number",
			fieldName: "test_field",
			want:      0,
			wantErr:   true,
			errMsg:    "invalid test_field",
		},
		{
			name:      "negative number",
			input:     "-50.5",
			fieldName: "test_field",
			want:      -50.5,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToFloat64(tt.input, tt.fieldName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}

func TestStringToInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		fieldName string
		want      int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid integer",
			input:     "123",
			fieldName: "test_field",
			want:      123,
			wantErr:   false,
		},
		{
			name:      "zero",
			input:     "0",
			fieldName: "test_field",
			want:      0,
			wantErr:   false,
		},
		{
			name:      "empty string",
			input:     "",
			fieldName: "test_field",
			want:      0,
			wantErr:   true,
			errMsg:    "test_field is required and cannot be empty",
		},
		{
			name:      "invalid format",
			input:     "not_a_number",
			fieldName: "test_field",
			want:      0,
			wantErr:   true,
			errMsg:    "invalid test_field",
		},
		{
			name:      "float string",
			input:     "123.45",
			fieldName: "test_field",
			want:      0,
			wantErr:   true,
			errMsg:    "invalid test_field",
		},
		{
			name:      "negative number",
			input:     "-50",
			fieldName: "test_field",
			want:      -50,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToInt(tt.input, tt.fieldName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
