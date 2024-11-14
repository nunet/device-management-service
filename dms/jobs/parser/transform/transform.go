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
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/utils"
)

// TransformerFunc is a function that transforms a part of the configuration.
// It modifies the data to conform to the expected structure and returns the transformed data.
// It takes the root configuration, the data to transform and the current path in the tree.
type TransformerFunc func(*map[string]interface{}, interface{}, tree.Path) (any, error)

// Transformer is a configuration transformer.
type Transformer interface {
	Transform(*map[string]interface{}) (interface{}, error)
}

// TransformerImpl is the implementation of the Transformer interface.
type TransformerImpl struct {
	transformers []map[tree.Path]TransformerFunc
}

// NewTransformer creates a new transformer with the given transformers.
func NewTransformer(transformers []map[tree.Path]TransformerFunc) Transformer {
	return TransformerImpl{
		transformers: transformers,
	}
}

// Transform applies the transformers to the configuration.
func (t TransformerImpl) Transform(rawConfig *map[string]interface{}) (interface{}, error) {
	data := any(*rawConfig)
	var err error
	for _, transformers := range t.transformers {
		data, err = t.transform(rawConfig, data, tree.NewPath(), transformers)
		if err != nil {
			return nil, err
		}
	}
	// return Normalize(data), nil
	return data, nil
}

// transform is a recursive function that applies the transformers to the configuration.
func (t TransformerImpl) transform(root *map[string]interface{}, data any, path tree.Path, transformers map[tree.Path]TransformerFunc) (interface{}, error) {
	var err error
	// Apply transformers that match the current path.
	for pattern, transformer := range transformers {
		if path.Matches(pattern) {
			data, err = transformer(root, data, path)
			if err != nil {
				return nil, err
			}
		}
	}
	// Recursively apply transformers to children.
	if result, ok := data.(map[string]interface{}); ok {
		for key, value := range result {
			next := path.Next(key)
			result[key], err = t.transform(root, value, next, transformers)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	} else if result, err := utils.ToAnySlice(data); err == nil {
		for i, value := range result {
			next := path.Next(fmt.Sprintf("[%d]", i))
			result[i], err = t.transform(root, value, next, transformers)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	return data, nil
}
