// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import "testing"

func TestNumericComparator(t *testing.T) {
	t.Parallel()

	t.Run("int checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			left     int
			right    int
			expected Comparison
		}{
			{
				name:     "left is equal to right",
				left:     1,
				right:    1,
				expected: Equal,
			},
			{
				name:     "left is less than right",
				left:     1,
				right:    2,
				expected: Worse,
			},
			{
				name:     "left is greater than right",
				left:     2,
				right:    1,
				expected: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				actual := NumericComparator(tt.left, tt.right)
				if actual != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, actual)
				}
			})
		}
	})

	t.Run("float64 checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			left     float64
			right    float64
			expected Comparison
		}{
			{
				name:     "left is equal to right",
				left:     1.0,
				right:    1.0,
				expected: Equal,
			},
			{
				name:     "left is less than right",
				left:     1.0,
				right:    2.0,
				expected: Worse,
			},
			{
				name:     "left is greater than right",
				left:     2.0,
				right:    1.0,
				expected: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				actual := NumericComparator(tt.left, tt.right)
				if actual != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, actual)
				}
			})
		}
	})

	// TODO: add checks for other numeric types
}

func TestLiteralComparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		left     string
		right    string
		expected Comparison
	}{
		{
			name:     "left is equal to right",
			left:     "test",
			right:    "test",
			expected: Equal,
		},
		{
			name:     "left is not equal to right",
			left:     "test",
			right:    "test2",
			expected: None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := LiteralComparator(tt.left, tt.right)
			if actual != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TODO: maybe in 575?
func TestComplexCompare(t *testing.T) {
	t.Skipf("not implemented")
}
