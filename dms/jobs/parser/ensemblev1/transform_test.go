// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

func TestTransformStringToBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectError bool
		expected    []byte
	}{
		{
			name:        "valid string",
			input:       "test script",
			expectError: false,
			expected:    []byte("test script"),
		},
		{
			name:        "empty string",
			input:       "",
			expectError: false,
			expected:    []byte(""),
		},
		{
			name:        "invalid type",
			input:       123,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TransformStringToBytes(nil, tt.input, "")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTransformSpec(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectError bool
		expected    map[string]any
	}{
		{
			name: "valid spec with edge_constraints",
			input: map[string]any{
				"edge_constraints": []any{
					map[string]any{"edges": []any{"node1", "node2"}},
				},
				"other_field": "value",
			},
			expectError: false,
			expected: map[string]any{
				"V1": map[string]any{
					"edges": []any{
						map[string]any{"edges": []any{"node1", "node2"}},
					},
					"other_field": "value",
				},
			},
		},
		{
			name: "dns_name defaults to allocation name",
			input: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
			},
			expectError: false,
			expected: map[string]any{
				"V1": map[string]any{
					"allocations": map[string]any{
						"alloc1": map[string]any{
							"dns_name": "alloc1",
						},
					},
				},
			},
		},
		{
			name: "valid spec without edge_constraints",
			input: map[string]any{
				"field": "value",
			},
			expectError: false,
			expected: map[string]any{
				"V1": map[string]any{
					"field": "value",
				},
			},
		},
		{
			name:        "invalid type",
			input:       "not a map",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TransformSpec(nil, tt.input, "")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTransformEdgeConstraint(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectError bool
		expected    map[string]any
	}{
		{
			name: "valid edge constraint",
			input: map[string]any{
				"edges": []any{"node1", "node2"},
				"type":  "dependency",
			},
			expectError: false,
			expected: map[string]any{
				"S":    "node1",
				"T":    "node2",
				"type": "dependency",
			},
		},
		{
			name: "invalid edges length",
			input: map[string]any{
				"edges": []any{"node1"},
			},
			expectError: true,
		},
		{
			name: "invalid edges type",
			input: map[string]any{
				"edges": "not an array",
			},
			expectError: true,
		},
		{
			name:        "invalid type",
			input:       "not a map",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TransformEdgeConstraint(nil, tt.input, "")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTransformVolume(t *testing.T) {
	tests := []struct {
		name        string
		root        *map[string]any
		input       any
		path        tree.Path
		expectError bool
		expected    map[string]any
	}{
		{
			name:        "string format",
			input:       "vol1:/mnt/data",
			expectError: false,
			expected: map[string]any{
				"name":       "vol1",
				"mountpoint": "/mnt/data",
			},
		},
		{
			name: "map format",
			input: map[string]any{
				"name":       "vol1",
				"mountpoint": "/mnt/data",
				"type":       "bind",
			},
			expectError: false,
			expected: map[string]any{
				"name":       "vol1",
				"mountpoint": "/mnt/data",
				"type":       "bind",
			},
		},
		{
			name: "inherit from parent",
			root: &map[string]any{
				"volumes": []any{
					map[string]any{
						"name":   "vol1",
						"type":   "bind",
						"source": "/host/path",
					},
				},
			},
			input: map[string]any{
				"name":       "vol1",
				"mountpoint": "/mnt/data",
			},
			path:        tree.NewPath("allocations", "alloc1", "volumes", "[0]"),
			expectError: false,
			expected: map[string]any{
				"name":       "vol1",
				"mountpoint": "/mnt/data",
				"type":       "bind",
				"source":     "/host/path",
			},
		},
		{
			name:        "invalid type",
			input:       123,
			expectError: true,
		},
		{
			name:        "invalid string format",
			input:       "invalid_format",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TransformVolume(tt.root, tt.input, tt.path)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTransformResources(t *testing.T) {
	tests := []struct {
		name        string
		root        *map[string]any
		input       any
		path        tree.Path
		expectError bool
		expected    map[string]any
	}{
		{
			name: "string reference",
			root: &map[string]any{
				"resources": []any{
					map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": float64(1),
						},
						"ram": map[string]any{
							"size": float64(1024),
						},
					},
				},
			},
			input:       "rc1",
			path:        tree.NewPath("allocations", "alloc1", "resources"),
			expectError: false,
			expected: map[string]any{
				"name": "rc1",
				"cpu": map[string]any{
					"cores": float64(1),
				},
				"ram": map[string]any{
					"size": float64(1024),
				},
			},
		},
		{
			name: "map format",
			input: map[string]any{
				"cpu": map[string]any{
					"cores": float64(2),
				},
				"ram": map[string]any{
					"size": float64(2048),
				},
			},
			expectError: false,
			expected: map[string]any{
				"cpu": map[string]any{
					"cores": float64(2),
				},
				"ram": map[string]any{
					"size": float64(2048),
				},
			},
		},
		{
			name: "inherit and override",
			root: &map[string]any{
				"resources": []any{
					map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": float64(1),
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": float64(1024),
						},
					},
				},
			},
			input:       "rc1",
			path:        tree.NewPath("allocations", "alloc1", "resources"),
			expectError: false,
			expected: map[string]any{
				"name": "rc1",
				"cpu": map[string]any{
					"cores": float64(1),
					"arch":  "x86_64",
				},
				"ram": map[string]any{
					"size": float64(1024),
				},
			},
		},
		{
			name:        "invalid type",
			input:       123,
			expectError: true,
		},
		{
			name: "reference not found",
			root: &map[string]any{
				"resources": []any{},
			},
			input:       "nonexistent",
			path:        tree.NewPath("allocations", "alloc1", "resources"),
			expectError: false,
			expected:    map[string]any{"name": "nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TransformResources(tt.root, tt.input, tt.path)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewEnsemblev1Transformer(t *testing.T) {
	input := map[string]any{
		"version": "V1",
		"resources": map[string]any{
			"rc1": map[string]any{
				"cpu": map[string]any{
					"cores": float64(2),
					"arch":  "x86_64",
				},
				"ram": map[string]any{
					"size": float64(2048),
				},
			},
		},
		"volumes": map[string]any{
			"vol1": map[string]any{
				"type":   "bind",
				"source": "/host/path",
			},
		},
		"allocations": map[string]any{
			"alloc1": map[string]any{
				"resources": "rc1",
				"volumes": map[string]any{
					"vol1": map[string]any{
						"mountpoint": "/mnt/data",
					},
				},
				"executor": "docker",
				"execution": map[string]any{
					"type":  "docker",
					"image": "nginx",
				},
				"dns_name": "node1",
			},
			"alloc2": map[string]any{
				"resources": "rc1",
				"executor":  "docker",
				"execution": map[string]any{
					"type":  "docker",
					"image": "redis",
				},
				"dns_name": "node2",
			},
		},
		"edge_constraints": []any{
			map[string]any{
				"edges": []any{"node1", "node2"},
				"type":  "dependency",
			},
		},
	}

	expected := map[string]any{
		"V1": map[string]any{
			"version": "V1",
			"resources": []any{
				map[string]any{
					"name": "rc1",
					"cpu": map[string]any{
						"cores": float64(2),
						"arch":  "x86_64",
					},
					"ram": map[string]any{
						"size": float64(2048),
					},
				},
			},
			"volumes": []any{
				map[string]any{
					"name":       "vol1",
					"type":       "bind",
					"source":     "/host/path",
					"mountpoint": "/mnt/data",
				},
			},
			"allocations": map[string]any{
				"alloc1": map[string]any{
					"resources": map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": float64(2),
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": float64(2048),
						},
					},
					"volumes": []any{
						map[string]any{
							"name":       "vol1",
							"type":       "bind",
							"source":     "/host/path",
							"mountpoint": "/mnt/data",
						},
					},
					"executor": "docker",
					"execution": map[string]any{
						"type": "docker",
						"params": map[string]any{
							"image": "nginx",
						},
					},
					"dns_name": "node1",
				},
				"alloc2": map[string]any{
					"resources": map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": float64(2),
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": float64(2048),
						},
					},
					"executor": "docker",
					"execution": map[string]any{
						"type": "docker",
						"params": map[string]any{
							"image": "redis",
						},
					},
					"dns_name": "node2",
				},
			},
			"edges": []any{
				map[string]any{
					"S":    "node1",
					"T":    "node2",
					"type": "dependency",
				},
			},
		},
	}

	transformer := NewEnsemblev1Transformer()
	result, err := transformer.Transform(&input)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
