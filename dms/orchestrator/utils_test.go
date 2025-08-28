// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"errors"
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

func TestAggregateErrors(t *testing.T) {
	// Define test error strings
	testError1 := "error 1"
	testError2 := "error 2"
	testCombinedErrors := testError1 + "\n" + testError2

	t.Run("no errors", func(t *testing.T) {
		errCh := make(chan error)
		close(errCh)

		err := aggregateErrors(errCh)
		require.NoError(t, err)
	})

	t.Run("single error", func(t *testing.T) {
		errCh := make(chan error, 1)
		errCh <- errors.New(testError1)
		close(errCh)

		err := aggregateErrors(errCh)
		require.Error(t, err)
		require.Equal(t, testError1, err.Error())
	})

	t.Run("multiple errors", func(t *testing.T) {
		errCh := make(chan error, 2)
		errCh <- errors.New(testError1)
		errCh <- errors.New(testError2)
		close(errCh)

		err := aggregateErrors(errCh)
		require.Error(t, err)
		require.Equal(t, testCombinedErrors, err.Error())
	})
}
