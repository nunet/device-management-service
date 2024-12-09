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
