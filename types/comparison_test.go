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

	comparisons := []Comparison{Better, Worse, Equal, Error}
	tests := []testCase{
		{
			name:     "Better checks",
			c:        Better,
			expected: []Comparison{Better, Worse, Better, Error},
		},
		{
			name:     "Worse checks",
			c:        Worse,
			expected: []Comparison{Worse, Worse, Worse, Error},
		},
		{
			name:     "Equal checks",
			c:        Equal,
			expected: []Comparison{Better, Worse, Equal, Error},
		},
		{
			name:     "Error checks",
			c:        Error,
			expected: []Comparison{Error, Error, Error, Error},
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
			name:     "All comparisons are errors",
			c:        ComplexComparison{"a": Error, "b": Error, "c": Error},
			expected: Error,
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
