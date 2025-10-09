// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package parser

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/types"
)

type SpecType string

type Options types.Options

const (
	SpecTypeEnsembleV1 SpecType = "ensembleV1"
)

func Decode(specType SpecType, data []byte, result any, opts *Options) error {
	parser, exists := registry.GetParser(specType)
	if !exists {
		return fmt.Errorf("parser for spec type %s not found", specType)
	}

	err := parser.Decode(data, result, (*types.Options)(opts))
	if err != nil {
		return err
	}

	return nil
}

func Encode(specType SpecType, data any) ([]byte, error) {
	parser, exists := registry.GetParser(specType)
	if !exists {
		return nil, fmt.Errorf("parser for spec type %s not found", specType)
	}

	return parser.Encode(data)
}
