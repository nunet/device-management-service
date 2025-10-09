// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v3"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
	"gitlab.com/nunet/device-management-service/lib/env"
)

type Options struct {
	Env        env.EnvironmentProvider
	Fs         afero.Afero
	WorkingDir string
}

type Parser interface {
	Decode(data []byte, dest any, opts *Options) error
	Encode(data any) ([]byte, error)
}

const DefaultTagName = "json"

type resolveFunc func(data *any, options *Options) error

type BasicParser struct {
	format    string
	resolveFn resolveFunc
	decoder   transform.Transformer
	encoder   transform.Transformer
	validator validate.Validator
}

func NewBasicParser(format string, resolveFn resolveFunc, decoder, encoder transform.Transformer, validator validate.Validator) BasicParser {
	return BasicParser{
		format:    format,
		resolveFn: resolveFn,
		decoder:   decoder,
		encoder:   encoder,
		validator: validator,
	}
}

func (p BasicParser) unmarshal(data []byte, result any) error {
	switch p.format {
	case "json":
		return json.Unmarshal(data, result)
	case "yaml":
		return yaml.Unmarshal(data, result)
	default:
		return fmt.Errorf("invalid format: %s", p.format)
	}
}

func (p BasicParser) marshal(data any) ([]byte, error) {
	switch p.format {
	case "json":
		return json.Marshal(data)
	case "yaml":
		return yaml.Marshal(data)
	default:
		return nil, fmt.Errorf("invalid format: %s", p.format)
	}
}

func (p BasicParser) Decode(data []byte, result any, opts *Options) error {
	var rawConfig map[string]any

	if err := p.unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	// Resolve files and environment variables
	var rawConfigAny any = rawConfig
	err := p.resolveFn(&rawConfigAny, opts)
	if err != nil {
		return fmt.Errorf("failed to resolve config: %v", err)
	}
	rawConfig = rawConfigAny.(map[string]any)

	// Apply transformers
	transformed, err := p.decoder.Transform(&rawConfig)
	if err != nil {
		return fmt.Errorf("failed to transform config: %v", err)
	}

	transformedMap, ok := transformed.(map[string]any)
	if !ok {
		return fmt.Errorf("transformed config is not a map: %v", transformed)
	}

	// Validate the transformed configuration
	if err = p.validator.Validate(&transformedMap); err != nil {
		return err
	}

	if err := decodeWithDefaultTagName(transformedMap, &result); err != nil {
		return fmt.Errorf("failed to decode spec: %v", err)
	}

	return err
}

func decodeWithDefaultTagName(input any, result any) error {
	var m mapstructure.Metadata

	ms, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: &m,
		Result:   result,
		TagName:  DefaultTagName,
	})
	if err != nil {
		return err
	}
	return ms.Decode(input)
}

func (p BasicParser) Encode(data any) ([]byte, error) {
	// Convert struct -> map[string]any via JSON roundtrip so that
	// nested maps of structs (e.g., Allocations, Nodes) are converted
	// into map[string]any values instead of remaining as structs.
	raw, err := p.marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode spec: %v", err)
	}

	var config map[string]any
	if err := p.unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("failed to encode spec: %v", err)
	}

	transformed, err := p.encoder.Transform(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to transform config: %v", err)
	}

	return p.marshal(transformed)
}
