package parser

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
	yaml "gopkg.in/yaml.v3"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
)

type SpecType string

const (
	specTypeNuNet SpecType = "nunet"
	specTypeNomad SpecType = "nomad"
	specTypeK8s   SpecType = "k8s"
)

const DefaultTagName = "json"

type Parser[T any] interface {
	Parse(data []byte) (T, error)
}

// nolint:revive
type ParserImpl[T any] struct {
	validator   validate.Validator
	transformer transform.Transformer
}

func NewParser[T any](transformer transform.Transformer, validator validate.Validator) Parser[T] {
	return ParserImpl[T]{
		transformer: transformer,
		validator:   validator,
	}
}

func (p ParserImpl[T]) Parse(data []byte) (T, error) {
	var rawConfig map[string]any
	var config T

	// Try to unmarshal as YAML first
	err := yaml.Unmarshal(data, &rawConfig)
	if err != nil {
		// If YAML fails, try JSON
		err = json.Unmarshal(data, &rawConfig)
		if err != nil {
			return config, fmt.Errorf("failed to parse config: %v", err)
		}
	}

	// Apply transformers
	transformed, err := p.transformer.Transform(&rawConfig)
	if err != nil {
		return config, fmt.Errorf("failed to transform config: %v", err)
	}

	// Validate the transformed configuration
	if err = p.validator.Validate(&rawConfig); err != nil {
		return config, err
	}

	// Create a new mapstructure decoder
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &config,
		TagName: DefaultTagName,
	})
	if err != nil {
		return config, fmt.Errorf("failed to create decoder: %v", err)
	}

	// Decode the transformed configuration
	err = decoder.Decode(transformed)
	if err != nil {
		return config, fmt.Errorf("failed to decode config: %v", err)
	}

	return config, err
}
