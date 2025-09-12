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
	"strings"
)

const (
	configPathSeparator        = "."
	configPathMatchAny         = "*"
	configPathMatchAnyMultiple = "**"
	configPathList             = "[]"
)

// Path is a custom type for representing paths in the configuration
type Path string

func NewPath(path ...string) Path {
	return Path(strings.Join(path, configPathSeparator))
}

// Parts returns the parts of the path
func (p Path) Parts() []string {
	return strings.Split(string(p), configPathSeparator)
}

// Parent returns the parent path
func (p Path) Parent() Path {
	parts := p.Parts()
	if len(parts) > 1 {
		return Path(strings.Join(parts[:len(parts)-1], configPathSeparator))
	}
	return ""
}

// Next returns the next part of the path
func (p Path) Next(path string) Path {
	if path == "" {
		return p
	}
	if p == "" {
		return Path(path)
	}
	return Path(string(p) + configPathSeparator + path)
}

// Last returns the last part of the path
func (p Path) Last() string {
	parts := p.Parts()
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Matches checks if the path matches a given pattern
func (p Path) Matches(pattern Path) bool {
	pathParts := p.Parts()
	patternParts := pattern.Parts()
	return matchParts(pathParts, patternParts)
}

// FindParentWithKey returns the first parent path that has the specified key
func (p Path) FindParentWithKey(key string) Path {
	if p == "" || key == "" {
		return ""
	}

	parts := p.Parts()
	for i := len(parts); i > 0; i-- {
		// Check if current part matches the key
		if parts[i-1] == key {
			return Path(strings.Join(parts[:i], configPathSeparator))
		}
	}
	return ""
}

// String returns the string representation of the path
func (p Path) String() string {
	return string(p)
}

// matchParts checks if the path parts match the pattern parts
func matchParts(pathParts, patternParts []string) bool {
	// If the pattern is longer than the path, it can't match
	if len(pathParts) < len(patternParts) {
		return false
	}
	for i, part := range patternParts {
		switch part {
		case configPathMatchAnyMultiple:
			// if it is the last part of the pattern, it matches
			if i == len(patternParts)-1 {
				return true
			}
			// Otherwise, try to match the rest of the path
			for j := i; j < len(pathParts); j++ {
				if matchParts(pathParts[j:], patternParts[i+1:]) {
					return true
				}
			}
		case configPathList:
			// check if pathParts[i] is inclosed by []
			if pathParts[i][0] != '[' || pathParts[i][len(pathParts[i])-1] != ']' {
				return false
			}
		default:
			// If the part doesn't match, it doesn't match
			if part != configPathMatchAny && !strings.EqualFold(part, pathParts[i]) {
				return false
			}
		}
		// If it is the last part of the pattern and the path is longer, it doesn't match
		if i == len(patternParts)-1 && i < len(pathParts)-1 {
			return false
		}
	}
	return true
}

// WalkFunc is a function applied to each node in the data structure.
type WalkFunc func(node *any, path Path) error

// Walk recursively traverses a generic data structure and applies a function to each node.
func Walk(data *any, path Path, fn WalkFunc) error {
	// Apply the function to the current node first.
	if err := fn(data, path); err != nil {
		return err
	}

	switch v := (*data).(type) {
	case map[string]any:
		for key, val := range v {
			// A temporary variable is needed to pass the address correctly.
			tempVal := val
			if err := Walk(&tempVal, path.Next(key), fn); err != nil {
				return err
			}
			v[key] = tempVal
		}
	case []any:
		for i, val := range v {
			tempVal := val
			next := path.Next(fmt.Sprintf("[%d]", i))
			if err := Walk(&tempVal, next, fn); err != nil {
				return err
			}
			v[i] = tempVal
		}
	}
	return nil
}
