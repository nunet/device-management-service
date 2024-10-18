package parser

import (
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	yaml "gopkg.in/yaml.v3"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
)

type SpecType string

const (
	SpecTypeEnsembleV1 SpecType = "ensembleV1"
)

const DefaultTagName = "json"

type Parser interface {
	Parse(data []byte, dest any) error
}

type BasicParser struct {
	validator   validate.Validator
	transformer transform.Transformer
}

func NewBasicParser(transformer transform.Transformer, validator validate.Validator) Parser {
	return BasicParser{
		transformer: transformer,
		validator:   validator,
	}
}

func (p BasicParser) Parse(data []byte, result any) error {
	var rawConfig map[string]any

	// Try to unmarshal as YAML first
	err := yaml.Unmarshal(data, &rawConfig)
	if err != nil {
		// If YAML fails, try JSON
		err = json.Unmarshal(data, &rawConfig)
		if err != nil {
			return fmt.Errorf("failed to parse config: %v", err)
		}
	}

	// Apply transformers
	transformed, err := p.transformer.Transform(&rawConfig)
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

	// Create a new mapstructure decoder
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  result,
		TagName: DefaultTagName,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %v", err)
	}

	// Decode the transformed configuration
	err = decoder.Decode(transformed)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	return err
}
