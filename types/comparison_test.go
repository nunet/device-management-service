// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComparison_And(t *testing.T) {
	type testCase struct {
		name     string
		c        Comparison
		expected []Comparison
	}

	comparisons := []Comparison{Better, Worse, Equal, None}
	tests := []testCase{
		{
			name:     "Better checks",
			c:        Better,
			expected: []Comparison{Better, Worse, Better, None},
		},
		{
			name:     "Worse checks",
			c:        Worse,
			expected: []Comparison{Worse, Worse, Worse, Worse},
		},
		{
			name:     "Equal checks",
			c:        Equal,
			expected: []Comparison{Better, Worse, Equal, None},
		},
		{
			name:     "None checks",
			c:        None,
			expected: []Comparison{None, None, None, None},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := make([]Comparison, len(comparisons))
			for i, c := range comparisons {
				actual[i] = test.c.And(c)
			}

			for i := range comparisons {
				assert.Equal(t, test.expected[i], actual[i])
			}
		})
	}
}

func TestComplexComparison_Result(t *testing.T) {
	type testCase struct {
		name     string
		c        ComplexComparison
		expected Comparison
	}

	tests := []testCase{
		{
			name:     "All comparisons are equal",
			c:        ComplexComparison{"a": Equal, "b": Equal, "c": Equal},
			expected: Equal,
		},
		{
			name:     "All comparisons are worse",
			c:        ComplexComparison{"a": Worse, "b": Worse, "c": Worse},
			expected: Worse,
		},
		{
			name:     "All comparisons are better",
			c:        ComplexComparison{"a": Better, "b": Better, "c": Better},
			expected: Better,
		},
		{
			name:     "All comparisons are none",
			c:        ComplexComparison{"a": None, "b": None, "c": None},
			expected: None,
		},
		{
			name:     "Mixed comparisons",
			c:        ComplexComparison{"a": Better, "b": Worse, "c": Equal},
			expected: Worse,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.c.Result()
			assert.Equal(t, test.expected, actual)
		})
	}
}
