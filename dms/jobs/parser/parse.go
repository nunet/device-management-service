package parser

import (
	"fmt"
)

func Parse(specType SpecType, data []byte, result any) error {
	parser, exists := registry.GetParser(specType)
	if !exists {
		return fmt.Errorf("parser for spec type %s not found", specType)
	}

	err := parser.Parse(data, result)
	if err != nil {
		return err
	}

	return nil
}
