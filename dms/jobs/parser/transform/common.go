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

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// SpecConfigTransformer converts a map to a map with a "type" field and a "params" field.
func SpecConfigTransformer(specName string) TransformerFunc {
	return func(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
		if data == nil {
			return nil, nil
		}
		spec, ok := data.(map[string]any)
		result := map[string]any{}

		if !ok {
			return nil, fmt.Errorf("invalid %s configuration: %v", specName, data)
		}
		params := map[string]any{}
		for k, v := range spec {
			if k != "type" {
				params[k] = v
			}
		}
		result["type"] = spec["type"]
		result["params"] = params

		return result, nil
	}
}

// MapToNamedSliceTransformer converts a map of maps to a slice of maps and assigns the key to the "name" field.
func MapToNamedSliceTransformer(name string) TransformerFunc {
	return func(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
		if data == nil {
			return nil, nil
		}
		maps, ok := data.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s configuration: %v", name, data)
		}

		result := []any{}
		for k, v := range maps {
			if v == nil {
				v = map[string]any{}
			}
			if e, ok := v.(map[string]any); ok {
				e["name"] = k
			}
			result = append(result, v)
		}
		return result, nil
	}
}
