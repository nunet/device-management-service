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
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
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
			name: "allocation default values are set",
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
							"dns_name":         "alloc1",
							"failure_recovery": defaultAllocationFailureStrategy,
						},
					},
				},
			},
		},
		{
			name: "node default values are set",
			input: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
				},
			},
			expectError: false,
			expected: map[string]any{
				"V1": map[string]any{
					"nodes": map[string]any{
						"node1": map[string]any{
							"failure_recovery": defautNodeFailureStrategy,
							"redundancy":       0,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name        string
		root        *map[string]any
		input       any
		path        tree.Path
		expectError bool
		expected    map[string]any
	}{
		{
			name: "string reference inherits values",
			root: &map[string]any{
				"resources": []any{
					map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": 4,
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": 8,
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
					"cores": 4,
					"arch":  "x86_64",
				},
				"ram": map[string]any{
					"size": uint64(8589934592), // 8 GiB in bytes
				},
			},
		},
		{
			name: "map format with default units",
			input: map[string]any{
				"cpu": map[string]any{
					"cores":       4,
					"clock_speed": 2.4, // defaults to GHz
				},
				"ram": map[string]any{
					"size":        16,  // defaults to GiB
					"clock_speed": 3.2, // defaults to GHz
				},
				"disk": map[string]any{
					"size": 500, // defaults to GiB
				},
				"gpu": []any{
					map[string]any{
						"vram": 8, // defaults to GiB
					},
				},
			},
			expectError: false,
			expected: map[string]any{
				"cpu": map[string]any{
					"cores":       4,
					"clock_speed": float64(2.4e9),
				},
				"ram": map[string]any{
					"size":        uint64(17179869184), // 16 GiB in bytes
					"clock_speed": float64(3.2e9),
				},
				"disk": map[string]any{
					"size": uint64(536870912000), // 500 GiB in bytes
				},
				"gpu": []any{
					map[string]any{
						"vram": uint64(8589934592), // 8 GiB in bytes
					},
				},
			},
		},
		{
			name: "explicit units",
			input: map[string]any{
				"cpu": map[string]any{
					"clock_speed": "3.6GHz",
				},
				"ram": map[string]any{
					"size":        "32GiB",
					"clock_speed": "3600MHz",
				},
				"disk": map[string]any{
					"size": "1TB",
				},
				"gpu": []any{
					map[string]any{
						"vram": "16GB",
					},
				},
			},
			expectError: false,
			expected: map[string]any{
				"cpu": map[string]any{
					"clock_speed": float64(3.6e9),
				},
				"ram": map[string]any{
					"size":        uint64(34359738368), // 32 GiB in bytes
					"clock_speed": float64(3.6e9),
				},
				"disk": map[string]any{
					"size": uint64(1000000000000), // 1 TB in bytes
				},
				"gpu": []any{
					map[string]any{
						"vram": uint64(16000000000), // 16 GB in bytes
					},
				},
			},
		},
		{
			name: "invalid cpu clock_speed unit",
			input: map[string]any{
				"cpu": map[string]any{
					"clock_speed": "3.5 invalid",
				},
			},
			expectError: true,
		},
		{
			name: "invalid ram size unit",
			input: map[string]any{
				"ram": map[string]any{
					"size": "16 invalid",
				},
			},
			expectError: true,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	input := map[string]any{
		"version": "V1",
		"resources": map[string]any{
			"rc1": map[string]any{
				"cpu": map[string]any{
					"cores": 2,
					"arch":  "x86_64",
				},
				"ram": map[string]any{
					"size": 2,
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
				"dns_name":         "node1",
				"failure_recovery": defaultAllocationFailureStrategy,
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
		"nodes": map[string]any{
			"node1": map[string]any{
				"allocations": []any{"alloc1"},
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
						"cores": 2,
						"arch":  "x86_64",
					},
					"ram": map[string]any{
						"size": uint64(2305843009213693952),
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
							"cores": 2,
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": uint64(2305843009213693952),
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
					"dns_name":         "node1",
					"failure_recovery": defaultAllocationFailureStrategy,
				},
				"alloc2": map[string]any{
					"resources": map[string]any{
						"name": "rc1",
						"cpu": map[string]any{
							"cores": 2,
							"arch":  "x86_64",
						},
						"ram": map[string]any{
							"size": uint64(2305843009213693952),
						},
					},
					"executor": "docker",
					"execution": map[string]any{
						"type": "docker",
						"params": map[string]any{
							"image": "redis",
						},
					},
					"dns_name":         "node2",
					"failure_recovery": defaultAllocationFailureStrategy,
				},
			},
			"nodes": map[string]any{
				"node1": map[string]any{
					"allocations":      []any{"alloc1"},
					"redundancy":       0,
					"failure_recovery": defautNodeFailureStrategy,
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
