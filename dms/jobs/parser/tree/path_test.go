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
