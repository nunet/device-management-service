// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

func TestNormalize(t *testing.T) {
	data := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{"name": "job2"},
			{"name": "job1"},
		},
	}

	expected := map[string]interface{}{
		"jobs": []interface{}{
			map[string]interface{}{
				"name": "job1",
			},
			map[string]interface{}{
				"name": "job2",
			},
		},
	}

	result := Normalize(data)
	assert.Equal(t, expected, result)
}

func TestToAnySlice(t *testing.T) {
	data := []string{"a", "b", "c"}
	expected := []any{"a", "b", "c"}

	result, err := ToAnySlice(data)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetConfigAtPath(t *testing.T) {
	data := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{"name": "job1"},
			{"name": "job2"},
		},
	}

	expected := map[string]interface{}{
		"name": "job1",
	}

	result, err := GetConfigAtPath(data, "jobs.[0]")
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	_, err = GetConfigAtPath(data, "jobs.[x]")
	assert.Error(t, err)
}

func TestCreateAdjencyList(t *testing.T) {
	tt := []struct {
		name        string
		input       map[string]any
		path        tree.Path
		expected    map[string][]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "multiple edges",
			input: map[string]any{
				"entity1": map[string]any{"edges": []string{"entity2", "entity3"}},
				"entity2": map[string]any{"edges": []string{"entity3"}},
				"entity3": map[string]any{},
			},
			path: "edges",
			expected: map[string][]string{
				"entity1": {"entity2", "entity3"},
				"entity2": {"entity3"},
			},
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			path:     "edges",
			expected: map[string][]string{},
		},
		{
			name: "mismatched type",
			input: map[string]any{
				"entity1": map[string]any{"edges": []int{1, 2}},
			},
			path:     "edges",
			expected: map[string][]string{},
		},
		{
			name: "empty path",
			input: map[string]any{
				"entity1": map[string]any{"edges": []string{"entity2", "entity3"}},
				"entity2": map[string]any{"edges": []string{"entity3"}},
				"entity3": map[string]any{},
			},
			path:     "",
			expected: map[string][]string{},
		},
		{
			name: "single edges",
			input: map[string]any{
				"entity1": map[string]any{"edges": "entity2"},
				"entity2": map[string]any{},
			},
			path: "edges",
			expected: map[string][]string{
				"entity1": {"entity2"},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			result := CreateAdjencyList(tc.input, tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectCycles(t *testing.T) {
	tt := []struct {
		name     string
		input    map[string][]string
		expected bool
	}{
		{
			name: "cycle detected",
			input: map[string][]string{
				"entity1": {"entity2", "entity3"},
				"entity2": {"entity3"},
				"entity3": {"entity1"},
			},
			expected: true,
		},
		{
			name: "cycle not detected",
			input: map[string][]string{
				"entity1": {"entity2", "entity3"},
				"entity2": {"entity3"},
				"entity3": {"entity4"},
			},
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectCycles(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
