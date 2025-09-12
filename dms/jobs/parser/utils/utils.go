// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// Sentinel errors to enable precise upstream handling
var (
	ErrKeyNotFound       = errors.New("key not found")
	ErrInvalidIndex      = errors.New("invalid index")
	ErrIndexOutOfRange   = errors.New("index out of range")
	ErrInvalidTypeAtPath = errors.New("invalid type at path")
	ErrCycleDetected     = errors.New("cycle detected")
)

// GetConfigAtPath retrieves a part of the configuration at a given path
func GetConfigAtPath(config any, path tree.Path) (any, error) {
	current := config
	for _, key := range path.Parts() {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[key]
			if !ok {
				if val, ok = v[strings.ToLower(key)]; !ok {
					return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
				}
			}
			current = val
		case []any:
			if len(key) < 3 || key[0] != '[' || key[len(key)-1] != ']' {
				return nil, fmt.Errorf("%w: %s (expected [n])", ErrInvalidIndex, key)
			}
			i, err := strconv.Atoi(key[1 : len(key)-1])
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidIndex, key)
			}
			if i < 0 || i >= len(v) {
				return nil, fmt.Errorf("%w: %d (len=%d)", ErrIndexOutOfRange, i, len(v))
			}
			current = v[i]
		case []map[string]any:
			if len(key) < 3 || key[0] != '[' || key[len(key)-1] != ']' {
				return nil, fmt.Errorf("%w: %s (expected [n])", ErrInvalidIndex, key)
			}
			i, err := strconv.Atoi(key[1 : len(key)-1])
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidIndex, key)
			}
			if i < 0 || i >= len(v) {
				return nil, fmt.Errorf("%w: %d (len=%d)", ErrIndexOutOfRange, i, len(v))
			}
			current = v[i]
		default:
			return nil, fmt.Errorf("%w at %q: %T", ErrInvalidTypeAtPath, key, current)
		}
	}
	return current, nil
}

// CreateAdjencyList creates an adjacency list from a map
func CreateAdjencyList[T comparable](m map[T]any, path tree.Path) map[T][]T {
	adjencyList := make(map[T][]T)
	for key, value := range m {
		if val, err := GetConfigAtPath(value, path); err == nil {
			switch v := val.(type) {
			case []T:
				adjencyList[key] = v
			case []any:
				for _, v := range v {
					if k, ok := v.(T); ok {
						adjencyList[key] = append(adjencyList[key], k)
					}
				}
			case T:
				adjencyList[key] = []T{v}
			}
		}
	}
	return adjencyList
}

func hasCycle[T comparable](adjencyList map[T][]T, node T, visited, recursionStack map[T]bool) bool {
	// If the node is already in the current recursion stack, we found a cycle
	if recursionStack[node] {
		return true
	}
	// If we've already fully visited this node (and its descendants), skip
	if visited[node] {
		return false
	}
	// Mark as visited and add to the recursion stack
	visited[node] = true
	recursionStack[node] = true
	// Recurse on all neighbors
	for _, neighbor := range adjencyList[node] {
		if hasCycle(adjencyList, neighbor, visited, recursionStack) {
			return true
		}
	}
	// Remove from recursion stack when backtracking
	recursionStack[node] = false
	return false
}

func DetectCycles[T comparable](adjencyList map[T][]T) bool {
	visited := make(map[T]bool)
	recursionStack := make(map[T]bool)
	for node := range adjencyList {
		if !visited[node] {
			if hasCycle(adjencyList, node, visited, recursionStack) {
				return true
			}
		}
	}
	return false
}
