// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package transform

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

type CommonTransformerTestSuite struct {
	suite.Suite
}

func TestCommonTransformerTestSuite(t *testing.T) {
	suite.Run(t, new(CommonTransformerTestSuite))
}

// Test ToSpecConfigTransformer
func (s *CommonTransformerTestSuite) TestToSpecConfigTransformer() {
	transformer := ToSpecConfigTransformer("test")
	path := tree.NewPath("test")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			hasError: false,
		},
		{
			name: "valid map with type and params",
			input: map[string]any{
				"type":   "docker",
				"image":  "nginx",
				"ports":  []string{"80:80"},
				"memory": "1GB",
			},
			expected: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image":  "nginx",
					"ports":  []string{"80:80"},
					"memory": "1GB",
				},
			},
			hasError: false,
		},
		{
			name: "map with only type",
			input: map[string]any{
				"type": "docker",
			},
			expected: map[string]any{
				"type":   "docker",
				"params": map[string]any{},
			},
			hasError: false,
		},
		{
			name: "map without type",
			input: map[string]any{
				"image": "nginx",
				"ports": []string{"80:80"},
			},
			expected: map[string]any{
				"type": nil,
				"params": map[string]any{
					"image": "nginx",
					"ports": []string{"80:80"},
				},
			},
			hasError: false,
		},
		{
			name:     "invalid input type",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid input number",
			input:    123,
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid test configuration")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test FlattenSpecConfigTransformer
func (s *CommonTransformerTestSuite) TestFlattenSpecConfigTransformer() {
	transformer := FlattenSpecConfigTransformer("test")
	path := tree.NewPath("test")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			hasError: false,
		},
		{
			name: "valid spec config",
			input: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image":  "nginx",
					"ports":  []string{"80:80"},
					"memory": "1GB",
				},
			},
			expected: map[string]any{
				"type":   "docker",
				"image":  "nginx",
				"ports":  []string{"80:80"},
				"memory": "1GB",
			},
			hasError: false,
		},
		{
			name: "spec config with only type",
			input: map[string]any{
				"type":   "docker",
				"params": map[string]any{},
			},
			expected: map[string]any{
				"type": "docker",
			},
			hasError: false,
		},
		{
			name:     "invalid input type",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test MapToNamedSliceTransformer
func (s *CommonTransformerTestSuite) TestMapToNamedSliceTransformer() {
	transformer := MapToNamedSliceTransformer("volumes")
	path := tree.NewPath("volumes")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			hasError: false,
		},
		{
			name: "valid map of maps",
			input: map[string]any{
				"vol1": map[string]any{
					"size": "10GB",
					"type": "local",
				},
				"vol2": map[string]any{
					"size": "20GB",
					"type": "nfs",
				},
			},
			expected: []any{
				map[string]any{
					"name": "vol1",
					"size": "10GB",
					"type": "local",
				},
				map[string]any{
					"name": "vol2",
					"size": "20GB",
					"type": "nfs",
				},
			},
			hasError: false,
		},
		{
			name: "map with nil values",
			input: map[string]any{
				"vol1": nil,
				"vol2": map[string]any{
					"size": "20GB",
				},
			},
			expected: []any{
				map[string]any{
					"name": "vol1",
				},
				map[string]any{
					"name": "vol2",
					"size": "20GB",
				},
			},
			hasError: false,
		},
		// NOTE: Not sure this should be the expected behavior
		{
			name: "map with non-map values",
			input: map[string]any{
				"vol1": "string_value",
				"vol2": 123,
			},
			expected: []any{
				"string_value",
				123,
			},
			hasError: false,
		},
		{
			name:     "invalid input type",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
			} else {
				s.NoError(err)
				// For slice comparisons, we need to check length and contents
				if tt.expected == nil {
					s.Nil(result)
				} else {
					s.IsType([]any{}, result)
					resultSlice := result.([]any)
					expectedSlice := tt.expected.([]any)
					s.Len(resultSlice, len(expectedSlice))
					s.ElementsMatch(resultSlice, expectedSlice)
				}
			}
		})
	}
}

// Test NamedSliceToMapTransformer
func (s *CommonTransformerTestSuite) TestNamedSliceToMapTransformer() {
	transformer := NamedSliceToMapTransformer("volumes")
	path := tree.NewPath("volumes")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			hasError: false,
		},
		{
			name: "valid slice of maps with names",
			input: []any{
				map[string]any{
					"name": "vol1",
					"size": "10GB",
					"type": "local",
				},
				map[string]any{
					"name": "vol2",
					"size": "20GB",
					"type": "nfs",
				},
			},
			expected: map[string]any{
				"vol1": map[string]any{
					"size": "10GB",
					"type": "local",
				},
				"vol2": map[string]any{
					"size": "20GB",
					"type": "nfs",
				},
			},
			hasError: false,
		},
		{
			name: "slice with maps without names",
			input: []any{
				map[string]any{
					"size": "10GB",
					"type": "local",
				},
				map[string]any{
					"size": "20GB",
					"type": "nfs",
				},
			},
			expected: map[string]any{
				"volumes_1": map[string]any{
					"size": "10GB",
					"type": "local",
				},
				"volumes_2": map[string]any{
					"size": "20GB",
					"type": "nfs",
				},
			},
			hasError: false,
		},
		{
			name: "slice with nil values",
			input: []any{
				nil,
				map[string]any{
					"name": "vol2",
					"size": "20GB",
				},
			},
			expected: map[string]any{
				"volumes_1": map[string]any{},
				"vol2": map[string]any{
					"size": "20GB",
				},
			},
			hasError: false,
		},
		{
			name: "slice with empty name",
			input: []any{
				map[string]any{
					"name": "",
					"size": "10GB",
				},
			},
			expected: map[string]any{
				"volumes_1": map[string]any{
					"name": "",
					"size": "10GB",
				},
			},
			hasError: false,
		},
		{
			name: "slice with non-string name",
			input: []any{
				map[string]any{
					"name": 123,
					"size": "10GB",
				},
			},
			expected: map[string]any{
				"volumes_1": map[string]any{
					"name": 123,
					"size": "10GB",
				},
			},
			hasError: false,
		},
		{
			name:     "invalid input type",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
		{
			name: "slice with non-map elements",
			input: []any{
				"string_value",
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid volumes configuration")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test ParseWithDefaultUnit
func (s *CommonTransformerTestSuite) TestParseWithDefaultUnit() {
	transformer := ParseWithDefaultUnit("memory", "MB")
	path := tree.NewPath("memory")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "valid string with unit",
			input:    "1GB",
			expected: float64(1e+09),
			hasError: false,
		},
		{
			name:     "valid string without unit uses default",
			input:    "512",
			expected: float64(5.12e+08),
			hasError: false,
		},
		{
			name:     "valid number",
			input:    1024,
			expected: float64(1.024e+09),
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid memory value")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test ParseBytesWithDefaultUnit
func (s *CommonTransformerTestSuite) TestParseBytesWithDefaultUnit() {
	transformer := ParseBytesWithDefaultUnit("storage", "GB")
	path := tree.NewPath("storage")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "valid string with unit",
			input:    "1TB",
			expected: uint64(1000000000000),
			hasError: false,
		},
		{
			name:     "valid string without unit uses default",
			input:    "10",
			expected: uint64(10000000000),
			hasError: false,
		},
		{
			name:     "valid number",
			input:    1024,
			expected: uint64(1024000000000),
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid storage value")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test ToBytesWithDefaultUnit
func (s *CommonTransformerTestSuite) TestToBytesWithDefaultUnit() {
	transformer := ToBytesWithDefaultUnit("disk", "GB")
	path := tree.NewPath("disk")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "valid string with unit",
			input:    "2TB",
			expected: uint64(2000000000000),
			hasError: false,
		},
		{
			name:     "valid string without unit uses default",
			input:    "5",
			expected: uint64(5000000000),
			hasError: false,
		},
		{
			name:     "valid number",
			input:    512,
			expected: uint64(512000000000),
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid disk value")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test ToBytesFormat
func (s *CommonTransformerTestSuite) TestToBytesFormat() {
	transformer := ToBytesFormat("memory")
	path := tree.NewPath("memory")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "valid bytes value",
			input:    int64(1073741824),
			expected: "1.1 GB",
			hasError: false,
		},
		{
			name:     "valid small bytes value",
			input:    int64(1048576),
			expected: "1.0 MB",
			hasError: false,
		},
		{
			name:     "valid int value",
			input:    1024,
			expected: "1.0 kB",
			hasError: false,
		},
		{
			name:     "zero value",
			input:    0,
			expected: "0 B",
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid memory value")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}

// Test ToSIFormatWithUnit
func (s *CommonTransformerTestSuite) TestToSIFormatWithUnit() {
	transformer := ToSIFormatWithUnit("cpu", "Hz")
	path := tree.NewPath("cpu")

	tests := []struct {
		name     string
		input    any
		expected any
		hasError bool
	}{
		{
			name:     "valid large value",
			input:    int64(2000000000),
			expected: "2 GHz",
			hasError: false,
		},
		{
			name:     "valid medium value",
			input:    int64(500000000),
			expected: "500 MHz",
			hasError: false,
		},
		{
			name:     "valid small value",
			input:    int64(1000),
			expected: "1 kHz",
			hasError: false,
		},
		{
			name:     "valid int value",
			input:    1000000,
			expected: "1 MHz",
			hasError: false,
		},
		{
			name:     "zero value",
			input:    0,
			expected: "0 Hz",
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := transformer(nil, tt.input, path)
			if tt.hasError {
				s.Error(err)
				s.Contains(err.Error(), "invalid cpu value")
			} else {
				s.NoError(err)
				s.Equal(tt.expected, result)
			}
		})
	}
}
