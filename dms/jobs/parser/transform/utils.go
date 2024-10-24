// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package transform

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// getConfigAtPath retrieves a part of the configuration at a given path
func GetConfigAtPath(config map[string]interface{}, path tree.Path) (any, error) {
	current := any(config)
	for _, key := range path.Parts() {
		switch v := current.(type) {
		case map[string]any:
			current = v[key]
		case []any, []map[string]any:
			i, err := strconv.Atoi(key[1 : len(key)-1])
			if err != nil {
				return nil, fmt.Errorf("invalid index: %v", key)
			}
			switch v := v.(type) {
			case []any:
				current = v[i]
			case []map[string]any:
				current = v[i]
			}
		default:
			return nil, fmt.Errorf("invalid data type: %v", current)
		}
	}
	return current, nil
}

// Generic function to convert any slice to []any
func ToAnySlice(slice any) ([]any, error) {
	value := reflect.ValueOf(slice)

	// Check if the input is a slice
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not a slice. type: %T", slice)
	}

	length := value.Len()
	anySlice := make([]any, length)

	for i := 0; i < length; i++ {
		anySlice[i] = value.Index(i).Interface()
	}

	return anySlice, nil
}

func normalizeMap(m interface{}) interface{} {
	v := reflect.ValueOf(m)

	switch v.Kind() {
	case reflect.Map:
		// Create a new map to hold normalized values
		newMap := reflect.MakeMap(reflect.MapOf(v.Type().Key(), reflect.TypeOf((*interface{})(nil)).Elem()))
		for _, key := range v.MapKeys() {
			newValue := normalizeMap(v.MapIndex(key).Interface())
			newMap.SetMapIndex(key, reflect.ValueOf(newValue))
		}
		return newMap.Interface()

	case reflect.Slice:
		// Create a new []interface{} slice to hold normalized values
		newSlice := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			newSlice[i] = normalizeMap(v.Index(i).Interface())
		}

		// Sort the slice if it's sortable
		sort.Slice(newSlice, func(i, j int) bool {
			return fmt.Sprint(newSlice[i]) < fmt.Sprint(newSlice[j])
		})

		return newSlice

	default:
		// For other types, return as is
		return m
	}
}

// Normalize is the exported function that users will call
func Normalize(m any) interface{} {
	return normalizeMap(m)
}
