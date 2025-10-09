// Original Copyright 2020 The Compose Specification Authors; Modified Copyright 2024, Nunet;
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tree

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PathTestSuite struct {
	suite.Suite
}

func TestPathTestSuite(t *testing.T) {
	suite.Run(t, new(PathTestSuite))
}

func (s *PathTestSuite) TestNewPath() {
	tests := []struct {
		input    []string
		expected Path
	}{
		{[]string{"a", "b", "c"}, "a.b.c"},
		{[]string{"root", "child"}, "root.child"},
		{[]string{"one"}, "one"},
		{[]string{}, ""},
	}

	for _, test := range tests {
		s.Equal(test.expected, NewPath(test.input...))
	}
}

func (s *PathTestSuite) TestParts() {
	tests := []struct {
		input    Path
		expected []string
	}{
		{"a.b.c", []string{"a", "b", "c"}},
		{"root.child", []string{"root", "child"}},
		{"one", []string{"one"}},
		{"", []string{""}},
	}

	for _, test := range tests {
		s.Equal(test.expected, test.input.Parts())
	}
}

func (s *PathTestSuite) TestParent() {
	tests := []struct {
		input    Path
		expected Path
	}{
		{"a.b.c", "a.b"},
		{"root.child", "root"},
		{"one", ""},
		{"", ""},
	}

	for _, test := range tests {
		s.Equal(test.expected, test.input.Parent())
	}
}

func (s *PathTestSuite) TestNext() {
	tests := []struct {
		input    Path
		next     string
		expected Path
	}{
		{"a.b", "c", "a.b.c"},
		{"root", "child", "root.child"},
		{"one", "", "one"},
		{"", "root", "root"},
	}

	for _, test := range tests {
		s.Equal(test.expected, test.input.Next(test.next))
	}
}

func (s *PathTestSuite) TestLast() {
	tests := []struct {
		input    Path
		expected string
	}{
		{"a.b.c", "c"},
		{"root.child", "child"},
		{"one", "one"},
		{"", ""},
	}

	for _, test := range tests {
		s.Equal(test.expected, test.input.Last())
	}
}

func (s *PathTestSuite) TestMatches() {
	tests := []struct {
		input    Path
		pattern  Path
		expected bool
	}{
		{"a.b.c", "a.b.c", true},
		{"a.b.c", "a.*.c", true},
		{"a.b.c", "a.**", true},
		{"a.b.c.d", "a.b.**", true},
		{"a.b.c.d", "a.**.d", true},
		{"a.b.c", "a.b.d", false},
		{"a.b.c", "a.**.e", false},
		{"a.b.c.d", "a.b.c", false},
		{"a.b.[0]", "a.b.[]", true},
		{"a.b.c.[0]", "a.**.c.[]", true},
	}

	for _, test := range tests {
		s.Equalf(test.expected, test.input.Matches(test.pattern), "%v doesn't match %v", test.input, test.pattern)
	}
}

func (s *PathTestSuite) TestFindParentWithKey() {
	tests := []struct {
		name     string
		path     Path
		key      string
		expected Path
	}{
		{
			name:     "direct parent has key",
			path:     "V1.allocations.alloc1.resources",
			key:      "allocations",
			expected: "V1.allocations",
		},
		{
			name:     "ancestor has key",
			path:     "V1.allocations.alloc1.resources.cpu.cores",
			key:      "allocations",
			expected: "V1.allocations",
		},
		{
			name:     "key is root",
			path:     "V1.allocations.alloc1",
			key:      "V1",
			expected: "V1",
		},
		{
			name:     "key not found",
			path:     "V1.allocations.alloc1",
			key:      "nonexistent",
			expected: "",
		},
		{
			name:     "key is empty",
			path:     "V1.allocations.alloc1",
			key:      "",
			expected: "",
		},
		{
			name:     "path is empty",
			path:     "",
			key:      "allocations",
			expected: "",
		},
		{
			name:     "key appears multiple times",
			path:     "V1.keys.alloc1.keys.value",
			key:      "keys",
			expected: "V1.keys.alloc1.keys", // should find the closest parent
		},
		{
			name:     "key matches full path component",
			path:     "V1.resources.alloc1",
			key:      "resources",
			expected: "V1.resources",
		},
		{
			name:     "key is partial match",
			path:     "V1.resources_config.alloc1",
			key:      "resources",
			expected: "", // should not match partial strings
		},
		{
			name:     "single component path with matching key",
			path:     "resources",
			key:      "resources",
			expected: "resources",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			result := test.path.FindParentWithKey(test.key)
			s.Equal(test.expected, result, "Test case: %s", test.name)
		})
	}
}

func (s *PathTestSuite) TestString() {
	tests := []struct {
		input    Path
		expected string
	}{
		{"a.b.c", "a.b.c"},
		{"root.child", "root.child"},
		{"one", "one"},
		{"", ""},
		{"complex.path.with.many.parts", "complex.path.with.many.parts"},
	}

	for _, test := range tests {
		s.Equal(test.expected, test.input.String())
	}
}

func (s *PathTestSuite) TestMatchParts() {
	tests := []struct {
		name         string
		pathParts    []string
		patternParts []string
		expected     bool
	}{
		{
			name:         "exact match",
			pathParts:    []string{"a", "b", "c"},
			patternParts: []string{"a", "b", "c"},
			expected:     true,
		},
		{
			name:         "single wildcard match",
			pathParts:    []string{"a", "b", "c"},
			patternParts: []string{"a", "*", "c"},
			expected:     true,
		},
		{
			name:         "multiple wildcard match",
			pathParts:    []string{"a", "b", "c", "d"},
			patternParts: []string{"a", "**"},
			expected:     true,
		},
		{
			name:         "multiple wildcard in middle",
			pathParts:    []string{"a", "b", "c", "d", "e"},
			patternParts: []string{"a", "**", "e"},
			expected:     true,
		},
		{
			name:         "list pattern match",
			pathParts:    []string{"a", "b", "[0]"},
			patternParts: []string{"a", "b", "[]"},
			expected:     true,
		},
		{
			name:         "list pattern no match",
			pathParts:    []string{"a", "b", "c"},
			patternParts: []string{"a", "b", "[]"},
			expected:     false,
		},
		{
			name:         "pattern longer than path",
			pathParts:    []string{"a", "b"},
			patternParts: []string{"a", "b", "c"},
			expected:     false,
		},
		{
			name:         "path longer than pattern without wildcard",
			pathParts:    []string{"a", "b", "c", "d"},
			patternParts: []string{"a", "b", "c"},
			expected:     false,
		},
		{
			name:         "no match different parts",
			pathParts:    []string{"a", "b", "c"},
			patternParts: []string{"a", "x", "c"},
			expected:     false,
		},
		{
			name:         "empty path and pattern",
			pathParts:    []string{},
			patternParts: []string{},
			expected:     true,
		},
		{
			name:         "empty path with pattern",
			pathParts:    []string{},
			patternParts: []string{"a"},
			expected:     false,
		},
		{
			name:         "complex multiple wildcard",
			pathParts:    []string{"services", "web", "ports", "80", "protocol"},
			patternParts: []string{"**", "ports", "**"},
			expected:     true,
		},
		{
			name:         "list with different index",
			pathParts:    []string{"items", "[5]", "value"},
			patternParts: []string{"items", "[]", "value"},
			expected:     true,
		},
		{
			name:         "malformed list path",
			pathParts:    []string{"items", "5]", "value"},
			patternParts: []string{"items", "[]", "value"},
			expected:     false,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			result := matchParts(test.pathParts, test.patternParts)
			s.Equal(test.expected, result, "Test case: %s", test.name)
		})
	}
}

func (s *PathTestSuite) TestWalk() {
	s.Run("simple map traversal", func() {
		data := map[string]any{
			"key1": "value1",
			"key2": map[string]any{
				"nested": "nestedValue",
			},
		}

		var visitedPaths []string
		walkFunc := func(_ *any, path Path) error {
			visitedPaths = append(visitedPaths, path.String())
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "", walkFunc)
		s.NoError(err)

		expectedPaths := []string{"", "key1", "key2", "key2.nested"}
		s.ElementsMatch(expectedPaths, visitedPaths)
	})

	s.Run("array traversal", func() {
		data := []any{
			"item1",
			map[string]any{
				"key": "value",
			},
			"item3",
		}

		var visitedPaths []string
		walkFunc := func(_ *any, path Path) error {
			visitedPaths = append(visitedPaths, path.String())
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "root", walkFunc)
		s.NoError(err)

		expectedPaths := []string{"root", "root.[0]", "root.[1]", "root.[1].key", "root.[2]"}
		s.ElementsMatch(expectedPaths, visitedPaths)
	})

	s.Run("complex nested structure", func() {
		data := map[string]any{
			"allocations": map[string]any{
				"web": map[string]any{
					"resources": map[string]any{
						"cpu": map[string]any{
							"cores":       2,
							"clock_speed": "2GHz",
						},
						"ram": map[string]any{
							"size": "8GB",
						},
					},
				},
			},
		}

		var visitedPaths []string
		walkFunc := func(_ *any, path Path) error {
			visitedPaths = append(visitedPaths, path.String())
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "", walkFunc)
		s.NoError(err)

		expectedPaths := []string{
			"",
			"allocations",
			"allocations.web",
			"allocations.web.resources",
			"allocations.web.resources.cpu",
			"allocations.web.resources.cpu.cores",
			"allocations.web.resources.cpu.clock_speed",
			"allocations.web.resources.ram",
			"allocations.web.resources.ram.size",
		}
		s.ElementsMatch(expectedPaths, visitedPaths)
	})

	s.Run("walk function returns error", func() {
		data := map[string]any{
			"key1": "value1",
			"key2": "value2",
		}

		walkFunc := func(_ *any, path Path) error {
			if path.String() == "key1" {
				return fmt.Errorf("test error")
			}
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "", walkFunc)
		s.Error(err)
		s.Contains(err.Error(), "test error")
	})

	s.Run("walk modifies data", func() {
		data := map[string]any{
			"key1": "value1",
			"nested": map[string]any{
				"key2": "value2",
			},
		}

		walkFunc := func(node *any, _ Path) error {
			if str, ok := (*node).(string); ok && str == "value1" {
				*node = "modified_value1"
			}
			if str, ok := (*node).(string); ok && str == "value2" {
				*node = "modified_value2"
			}
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "", walkFunc)
		s.NoError(err)

		modifiedData := dataAny.(map[string]any)
		s.Equal("modified_value1", modifiedData["key1"])
		nestedData := modifiedData["nested"].(map[string]any)
		s.Equal("modified_value2", nestedData["key2"])
	})

	s.Run("walk with primitive types", func() {
		data := "simple string"

		var visitedPaths []string
		walkFunc := func(_ *any, path Path) error {
			visitedPaths = append(visitedPaths, path.String())
			return nil
		}

		dataAny := any(data)
		err := Walk(&dataAny, "root", walkFunc)
		s.NoError(err)

		expectedPaths := []string{"root"}
		s.Equal(expectedPaths, visitedPaths)
	})
}
