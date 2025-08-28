// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package env

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMockEnvironment(t *testing.T) {
	t.Parallel()
	env := NewMockEnvironment()

	require.NotNil(t, env, "NewMockEnvironment() should not return nil")
	require.NotNil(t, env.vars, "MockEnvironment vars map should be initialized")
	require.Empty(t, env.vars, "vars map should be empty initially")
}

func TestMockEnvironmentGetenv(t *testing.T) {
	t.Parallel()
	env := NewMockEnvironment()

	// Test getting non-existent key
	val := env.Getenv("NON_EXISTENT_KEY")
	require.Empty(t, val, "Getting non-existent key should return empty string")

	// Set value and test getting it
	testKey := "TEST_KEY"
	testValue := "test_value"
	env.vars[testKey] = testValue

	val = env.Getenv(testKey)
	require.Equal(t, testValue, val, "Getenv should return the correct value")
}

func TestMockEnvironmentSetenv(t *testing.T) {
	t.Parallel()
	env := NewMockEnvironment()

	// Set a value
	testKey := "TEST_KEY"
	testValue := "test_value"
	err := env.Setenv(testKey, testValue)
	require.NoError(t, err, "Setenv should not return an error")

	// Verify it was set correctly
	v, ok := env.vars[testKey]
	require.True(t, ok, "Key should be present in the internal map")
	require.Equal(t, testValue, v, "Value should be set correctly in the internal map")

	// Test getting the value
	val := env.Getenv(testKey)
	require.Equal(t, testValue, val, "Getenv should return the value set by Setenv")

	// Update the value
	newValue := "new_value"
	err = env.Setenv(testKey, newValue)
	require.NoError(t, err, "Updating a value should not return an error")

	// Verify it was updated
	val = env.Getenv(testKey)
	require.Equal(t, newValue, val, "Getenv should return the updated value")
}

func TestMockEnvironmentConcurrency(t *testing.T) {
	t.Parallel()
	env := NewMockEnvironment()
	var wg sync.WaitGroup

	// Number of concurrent goroutines
	concurrency := 10
	// Operations per goroutine
	iterations := 10

	// Set up a barrier for all goroutines to start at the same time
	var startBarrier, endBarrier sync.WaitGroup
	startBarrier.Add(1)
	endBarrier.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer endBarrier.Done()

			// Wait for start signal
			startBarrier.Wait()

			// Set and get values concurrently
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("KEY_%d", id)
				value := fmt.Sprintf("VALUE_%d_%d", id, j)

				err := env.Setenv(key, value)
				if err != nil {
					t.Errorf("Error in goroutine %d setting %s=%s: %v", id, key, value, err)
				}

				val := env.Getenv(key)
				if val != value {
					t.Errorf("Goroutine %d expected %q for key %q, got %q", id, value, key, val)
				}
			}
		}(i)
	}

	// Start all goroutines simultaneously
	startBarrier.Done()

	// Wait for all goroutines to complete their operations
	endBarrier.Wait()
	wg.Wait()

	// Verify final state of keys set by the first goroutine
	key := "KEY_0" // First goroutine has ID 0
	expectedValue := fmt.Sprintf("VALUE_0_%d", (iterations - 1))
	val := env.Getenv(key)
	require.Equal(t, expectedValue, val, "After concurrent operations, value should match expected")
}
