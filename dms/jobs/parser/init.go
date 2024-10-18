package parser

import (
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/ensemblev1"
)

var registry *Registry

func init() {
	registry = &Registry{
		parsers: make(map[SpecType]Parser),
	}

	// Register Nunet parser.
	ensembleV1Parser := NewBasicParser(
		ensemblev1.NewEnsemblev1Transformer(),
		ensemblev1.NewEnsembleV1Validator(),
	)
	registry.RegisterParser(SpecTypeEnsembleV1, ensembleV1Parser)
}
