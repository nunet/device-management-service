package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderByDependency(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		vertices := map[string][]string{}
		order, err := orderByDependency(vertices)
		require.NoError(t, err)
		require.Empty(t, order)
	})

	t.Run("single item no dependencies", func(t *testing.T) {
		vertices := map[string][]string{
			"standalone": {},
		}
		order, err := orderByDependency(vertices)
		require.NoError(t, err)
		require.Len(t, order, 1)
		require.Equal(t, []string{"standalone"}, order[0])
	})

	t.Run("multiple independent items", func(t *testing.T) {
		vertices := map[string][]string{
			"item1": {},
			"item2": {},
			"item3": {},
		}
		order, err := orderByDependency(vertices)
		require.NoError(t, err)
		require.Len(t, order, 1)
		require.ElementsMatch(t, []string{"item1", "item2", "item3"}, order[0])
	})

	t.Run("non-existent dependency", func(t *testing.T) {
		vertices := map[string][]string{
			"item1": {"missing"},
		}
		_, err := orderByDependency(vertices)
		require.Error(t, err, "non-existent dependency")
	})

	t.Run("detects cycles", func(t *testing.T) {
		vertices := map[string][]string{
			"vertex1": {"vertex2"},
			"vertex2": {"vertex3"},
			"vertex3": {"vertex1"},
		}

		_, err := orderByDependency(vertices)
		require.Error(t, err, "cycle detected in dependencies")
	})

	t.Run("returns correct order", func(t *testing.T) {
		vertices := map[string][]string{
			"vertex1": {"vertex2"},
			"vertex2": {"vertex3"},
			"vertex3": {},
			"vertex4": {"vertex3"},
		}

		order, err := orderByDependency(vertices)
		require.NoError(t, err)
		require.Len(t, order, 3)
		require.ElementsMatch(t, order[0], []string{"vertex3"})
		require.ElementsMatch(t, order[1], []string{"vertex2", "vertex4"})
		require.ElementsMatch(t, order[2], []string{"vertex1"})
	})
	t.Run("returns correct order with less dependencies", func(t *testing.T) {
		vertices := map[string][]string{
			"vertex1": {"vertex3"},
			"vertex2": {},
			"vertex3": {},
			"vertex4": {},
		}

		order, err := orderByDependency(vertices)
		require.NoError(t, err)
		require.Len(t, order, 2)
		require.ElementsMatch(t, order[0], []string{"vertex2", "vertex3", "vertex4"})
		require.ElementsMatch(t, order[1], []string{"vertex1"})
	})
}
