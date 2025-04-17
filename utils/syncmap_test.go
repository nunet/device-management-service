// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncMapFromMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected map[string]int
	}{
		{
			name:     "empty map",
			input:    map[string]int{},
			expected: map[string]int{},
		},
		{
			name: "map with values",
			input: map[string]int{
				"one":   1,
				"two":   2,
				"three": 3,
			},
			expected: map[string]int{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.input)

			// Verify all keys and values were transferred
			for k, expectedVal := range tt.expected {
				val, ok := syncMap.Get(k)
				assert.True(t, ok, "Key %s should exist", k)
				assert.Equal(t, expectedVal, val, "Value for key %s should match", k)
			}

			// Verify no extra keys exist
			count := 0
			syncMap.Iter(func(key string, _ int) bool {
				count++
				_, exists := tt.expected[key]
				assert.True(t, exists, "Unexpected key %s in SyncMap", key)
				return true
			})
			assert.Equal(t, len(tt.expected), count, "SyncMap should have same number of entries as input map")
		})
	}
}

func TestSyncMap_Get(t *testing.T) {
	tests := []struct {
		name        string
		initialMap  map[string]int
		key         string
		expectedVal int
		shouldExist bool
	}{
		{
			name:        "get existing key",
			initialMap:  map[string]int{"key1": 100, "key2": 200},
			key:         "key1",
			expectedVal: 100,
			shouldExist: true,
		},
		{
			name:        "get non-existent key",
			initialMap:  map[string]int{"key1": 100, "key2": 200},
			key:         "key3",
			expectedVal: 0, // zero value for int
			shouldExist: false,
		},
		{
			name:        "get from empty map",
			initialMap:  map[string]int{},
			key:         "any",
			expectedVal: 0,
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.initialMap)
			val, exists := syncMap.Get(tt.key)

			assert.Equal(t, tt.shouldExist, exists, "Existence check should match expected")
			if tt.shouldExist {
				assert.Equal(t, tt.expectedVal, val, "Value should match expected")
			}
		})
	}
}

func TestSyncMap_Put(t *testing.T) {
	tests := []struct {
		name       string
		initialMap map[string]int
		key        string
		value      int
	}{
		{
			name:       "add new key",
			initialMap: map[string]int{"existing": 100},
			key:        "new",
			value:      200,
		},
		{
			name:       "update existing key",
			initialMap: map[string]int{"existing": 100},
			key:        "existing",
			value:      300,
		},
		{
			name:       "add to empty map",
			initialMap: map[string]int{},
			key:        "first",
			value:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.initialMap)

			// Put the new key-value pair
			syncMap.Put(tt.key, tt.value)

			// Verify the key exists with correct value
			val, exists := syncMap.Get(tt.key)
			assert.True(t, exists, "Key should exist after Put")
			assert.Equal(t, tt.value, val, "Value should match what was put")

			// Verify other keys weren't affected
			for k, expectedVal := range tt.initialMap {
				if k != tt.key {
					val, exists := syncMap.Get(k)
					assert.True(t, exists, "Original key %s should still exist", k)
					assert.Equal(t, expectedVal, val, "Value for key %s should be unchanged", k)
				}
			}
		})
	}
}

func TestSyncMap_Iter(t *testing.T) {
	tests := []struct {
		name       string
		initialMap map[string]int
		stopAfter  int // stop iteration after this many items
	}{
		{
			name:       "iterate all items",
			initialMap: map[string]int{"a": 1, "b": 2, "c": 3},
			stopAfter:  -1, // don't stop early
		},
		{
			name:       "stop after 2 items",
			initialMap: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4},
			stopAfter:  2,
		},
		{
			name:       "empty map",
			initialMap: map[string]int{},
			stopAfter:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.initialMap)

			visited := make(map[string]bool)
			count := 0

			syncMap.Iter(func(key string, value int) bool {
				// Record that we visited this key
				visited[key] = true
				count++

				// Verify the value matches
				expectedVal, exists := tt.initialMap[key]
				assert.True(t, exists, "Iterated key %s should exist in original map", key)
				assert.Equal(t, expectedVal, value, "Value for key %s should match", key)

				// Determine whether to continue
				if tt.stopAfter > 0 && count >= tt.stopAfter {
					return false // stop iteration
				}
				return true // continue iteration
			})

			// Check if we visited the expected number of items
			if tt.stopAfter > 0 && tt.stopAfter < len(tt.initialMap) {
				assert.Equal(t, tt.stopAfter, count, "Should have stopped after %d items", tt.stopAfter)
			} else {
				assert.Equal(t, len(tt.initialMap), count, "Should have visited all items")
			}

			// If we didn't stop early, verify we visited all keys
			if tt.stopAfter < 0 || tt.stopAfter >= len(tt.initialMap) {
				for k := range tt.initialMap {
					assert.True(t, visited[k], "Key %s should have been visited", k)
				}
			}
		})
	}
}

func TestSyncMap_Keys(t *testing.T) {
	tests := []struct {
		name       string
		initialMap map[string]int
	}{
		{
			name:       "map with multiple keys",
			initialMap: map[string]int{"a": 1, "b": 2, "c": 3},
		},
		{
			name:       "map with one key",
			initialMap: map[string]int{"single": 100},
		},
		{
			name:       "empty map",
			initialMap: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.initialMap)

			keys := syncMap.Keys()

			// Sort both slices for comparison since map iteration order is non-deterministic
			sort.Strings(keys)

			expectedKeys := make([]string, 0, len(tt.initialMap))
			for k := range tt.initialMap {
				expectedKeys = append(expectedKeys, k)
			}
			sort.Strings(expectedKeys)

			assert.Equal(t, expectedKeys, keys, "Keys should match expected")
		})
	}
}

func TestSyncMap_String(t *testing.T) {
	tests := []struct {
		name       string
		initialMap map[string]int
		contains   []string // substrings that should be in the result
	}{
		{
			name:       "map with values",
			initialMap: map[string]int{"a": 1, "b": 2},
			contains:   []string{"a=1", "b=2", "{", "}"},
		},
		{
			name:       "empty map",
			initialMap: map[string]int{},
			contains:   []string{"{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncMap := SyncMapFromMap(tt.initialMap)

			result := syncMap.String()

			// Check that the result contains all expected substrings
			for _, substr := range tt.contains {
				assert.True(t, strings.Contains(result, substr),
					"String representation should contain '%s', got: %s", substr, result)
			}

			// For empty map, check exact match
			if len(tt.initialMap) == 0 {
				assert.Equal(t, "{}", result, "Empty map should be represented as '{}'")
			}
		})
	}
}

func TestSyncMap_Concurrency(t *testing.T) {
	// This is a basic concurrency test to ensure no race conditions
	// For more thorough testing, consider using the race detector

	syncMap := &SyncMap[string, int]{}

	// Add some initial data
	countInitial := 2
	for i := range countInitial {
		syncMap.Put(fmt.Sprintf("key%d", i), i*100)
	}

	// Test concurrent reads and writes
	done := make(chan bool)

	// Start multiple goroutines to read and write
	countRoutines := 10
	countReads := 100

	for i := range countRoutines {
		go func(id int) {
			// Do some reads
			for range countReads {
				syncMap.Get("key1")
				syncMap.Get("key2")
			}

			// Do some writes
			for j := range countReads {
				syncMap.Put(string(rune('a'+id)), id*100+j)
			}

			// Signal completion
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range countRoutines {
		<-done
	}

	// Verify the map has the expected number of entries
	keys := syncMap.Keys()
	require.Equal(t, countInitial+countRoutines, len(keys), "Map should have %d keys after concurrent operations", countInitial+countRoutines)
}
