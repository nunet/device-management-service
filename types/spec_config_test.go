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
)

func TestNewSpecConfig(t *testing.T) {
	spec := NewSpecConfig("docker")

	if spec.Type != "docker" {
		t.Errorf("Expected Type to be 'docker', got %s", spec.Type)
	}

	if spec.Params == nil {
		t.Error("Expected Params to be initialized, got nil")
	}

	if len(spec.Params) != 0 {
		t.Errorf("Expected empty Params, got %d items", len(spec.Params))
	}
}

func TestSpecConfig_WithParam(t *testing.T) {
	spec := NewSpecConfig("docker")

	// Test adding a single parameter
	result := spec.WithParam("image", "ubuntu:latest")
	if result != spec {
		t.Error("WithParam should return the same instance")
	}

	if val, ok := spec.Params["image"]; !ok || val != "ubuntu:latest" {
		t.Errorf("Expected Params['image'] to be 'ubuntu:latest', got %v", val)
	}

	// Test adding multiple parameters
	spec.WithParam("memory", 1024).WithParam("cpu", 2)

	if len(spec.Params) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(spec.Params))
	}

	// Test with nil Params
	spec = &SpecConfig{Type: "docker", Params: nil}
	spec.WithParam("image", "ubuntu:latest")

	if spec.Params == nil {
		t.Error("WithParam should initialize Params if nil")
	}

	if val, ok := spec.Params["image"]; !ok || val != "ubuntu:latest" {
		t.Errorf("Expected Params['image'] to be 'ubuntu:latest', got %v", val)
	}
}

func TestSpecConfig_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    *SpecConfig
		expected *SpecConfig
	}{
		{
			name:     "nil spec",
			input:    nil,
			expected: nil,
		},
		{
			name: "trim type",
			input: &SpecConfig{
				Type:   "  docker  ",
				Params: map[string]any{"image": "ubuntu"},
			},
			expected: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{"image": "ubuntu"},
			},
		},
		{
			name: "nil params",
			input: &SpecConfig{
				Type:   "docker",
				Params: nil,
			},
			expected: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
		},
		{
			name: "empty params",
			input: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
			expected: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()

			if tt.input == nil {
				if tt.expected != nil {
					t.Errorf("Expected nil, got non-nil")
				}
				return
			}

			if tt.input.Type != tt.expected.Type {
				t.Errorf("Expected Type to be '%s', got '%s'", tt.expected.Type, tt.input.Type)
			}

			if (tt.input.Params == nil) != (tt.expected.Params == nil) {
				t.Errorf("Params nil state mismatch: expected %v, got %v",
					tt.expected.Params == nil, tt.input.Params == nil)
			}

			if len(tt.input.Params) != len(tt.expected.Params) {
				t.Errorf("Expected %d params, got %d", len(tt.expected.Params), len(tt.input.Params))
			}
		})
	}
}

func TestSpecConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		spec        *SpecConfig
		expectError bool
	}{
		{
			name:        "nil spec",
			spec:        nil,
			expectError: true,
		},
		{
			name: "empty type",
			spec: &SpecConfig{
				Type:   "",
				Params: map[string]any{},
			},
			expectError: true,
		},
		{
			name: "whitespace type",
			spec: &SpecConfig{
				Type:   "   ",
				Params: map[string]any{},
			},
			expectError: true,
		},
		{
			name: "valid spec",
			spec: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
			expectError: false,
		},
		{
			name: "valid spec with params",
			spec: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{"image": "ubuntu"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSpecConfig_IsType(t *testing.T) {
	tests := []struct {
		name     string
		spec     *SpecConfig
		typeStr  string
		expected bool
	}{
		{
			name:     "nil spec",
			spec:     nil,
			typeStr:  "docker",
			expected: false,
		},
		{
			name: "exact match",
			spec: &SpecConfig{
				Type: "docker",
			},
			typeStr:  "docker",
			expected: true,
		},
		{
			name: "case insensitive match",
			spec: &SpecConfig{
				Type: "Docker",
			},
			typeStr:  "docker",
			expected: true,
		},
		{
			name: "with whitespace in input",
			spec: &SpecConfig{
				Type: "docker",
			},
			typeStr:  "  docker  ",
			expected: true,
		},
		{
			name: "no match",
			spec: &SpecConfig{
				Type: "docker",
			},
			typeStr:  "firecracker",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.IsType(tt.typeStr)

			if result != tt.expected {
				t.Errorf("IsType() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSpecConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		spec     *SpecConfig
		expected bool
	}{
		{
			name:     "nil spec",
			spec:     nil,
			expected: true,
		},
		{
			name: "empty type and params",
			spec: &SpecConfig{
				Type:   "",
				Params: map[string]any{},
			},
			expected: true,
		},
		{
			name: "whitespace type and empty params",
			spec: &SpecConfig{
				Type:   "   ",
				Params: map[string]any{},
			},
			expected: true,
		},
		{
			name: "empty type with params",
			spec: &SpecConfig{
				Type:   "",
				Params: map[string]any{"image": "ubuntu"},
			},
			expected: false,
		},
		{
			name: "non-empty type with empty params",
			spec: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{},
			},
			expected: false,
		},
		{
			name: "non-empty type and params",
			spec: &SpecConfig{
				Type:   "docker",
				Params: map[string]any{"image": "ubuntu"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.IsEmpty()

			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
