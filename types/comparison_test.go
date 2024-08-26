package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComparisonAnd(t *testing.T) {
	comparisons := []Comparison{Better, Worse, Equal, Error}

	comp := Better
	expected := []Comparison{Better, Worse, Better, Error}

	actual := make([]Comparison, len(comparisons))
	for i, c := range comparisons {
		actual[i] = comp.And(c)
	}

	for i := range comparisons {
		assert.Equal(t, expected[i], actual[i])
	}

	comp = Worse
	expected = []Comparison{Worse, Worse, Worse, Error}

	actual = make([]Comparison, len(comparisons))
	for i, c := range comparisons {
		actual[i] = comp.And(c)
	}

	for i := range comparisons {
		assert.Equal(t, expected[i], actual[i])
	}

	comp = Equal
	expected = []Comparison{Better, Worse, Equal, Error}

	actual = make([]Comparison, len(comparisons))
	for i, c := range comparisons {
		actual[i] = comp.And(c)
	}

	for i := range comparisons {
		assert.Equal(t, expected[i], actual[i])
	}

	comp = Error
	expected = []Comparison{Error, Error, Error, Error}

	actual = make([]Comparison, len(comparisons))
	for i, c := range comparisons {
		actual[i] = comp.And(c)
	}

	for i := range comparisons {
		assert.Equal(t, expected[i], actual[i])
	}
}
