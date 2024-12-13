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

	"github.com/stretchr/testify/assert"
)

func TestToPositiveFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		field     string
		want      float64
		wantError bool
		errorMsg  string
	}{
		{
			name:      "float64",
			input:     float64(42.5),
			field:     "test",
			want:      42.5,
			wantError: false,
		},
		{
			name:      "float32",
			input:     float32(42.5),
			field:     "test",
			want:      42.5,
			wantError: false,
		},
		{
			name:      "int",
			input:     42,
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "int32",
			input:     int32(42),
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "int64",
			input:     int64(42),
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "uint",
			input:     uint(42),
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "uint32",
			input:     uint32(42),
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "uint64",
			input:     uint64(42),
			field:     "test",
			want:      42,
			wantError: false,
		},
		{
			name:      "string valid",
			input:     "42.5",
			field:     "test",
			want:      42.5,
			wantError: false,
		},
		{
			name:      "negative float64",
			input:     float64(-42.5),
			field:     "test",
			wantError: true,
			errorMsg:  "test must be positive",
		},
		{
			name:      "zero float64",
			input:     float64(0),
			field:     "test",
			wantError: true,
			errorMsg:  "test must be positive",
		},
		{
			name:      "invalid type",
			input:     []int{1, 2, 3},
			field:     "test",
			wantError: true,
			errorMsg:  "test must be a valid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToPositiveFloat64(tt.input, tt.field)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Equal(t, tt.errorMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
