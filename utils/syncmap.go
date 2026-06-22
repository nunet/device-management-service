// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"strings"
	"sync"
)

// A SyncMap is a concurrency-safe sync.Map that uses strongly-typed
// method signatures to ensure the types of its stored data are known.
type SyncMap[K comparable, V any] struct {
	sync.Map
}

// SyncMapFromMap converts a standard Go map to a concurrency-safe SyncMap.
func SyncMapFromMap[K comparable, V any](m map[K]V) *SyncMap[K, V] {
	ret := &SyncMap[K, V]{}
	for k, v := range m {
		ret.Put(k, v)
	}
	return ret
}

// Get retrieves the value associated with the given key from the map.
// It returns the value and a boolean indicating whether the key was found.
func (m *SyncMap[K, V]) Get(key K) (V, bool) {
	value, ok := m.Load(key)
	if !ok {
		var empty V
		return empty, false
	}
	return value.(V), true
}

// Put inserts or updates a key-value pair in the map.
func (m *SyncMap[K, V]) Put(key K, value V) {
	m.Store(key, value)
}

// Iter iterates over each key-value pair in the map, executing the provided function on each pair.
// The iteration stops if the provided function returns false.
func (m *SyncMap[K, V]) Iter(ranger func(key K, value V) bool) {
	m.Range(func(key, value any) bool {
		k := key.(K)
		v := value.(V)
		return ranger(k, v)
	})
}

// Keys returns a slice containing all the keys present in the map.
func (m *SyncMap[K, V]) Keys() []K {
	keys := make([]K, 0)
	m.Iter(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

// String provides a string representation of the map, listing all key-value pairs.
func (m *SyncMap[K, V]) String() string {
	// Use a strings.Builder for efficient string concatenation.
	var sb strings.Builder
	sb.WriteString("{")

	first := true
	m.Range(func(key, value any) bool {
		if !first {
			sb.WriteString(" ")
		}
		first = false
		// Properly format the key-value pair
		fmt.Fprintf(&sb, "%v=%v", key, value)
		return true
	})

	sb.WriteString("}")
	return sb.String()
}
