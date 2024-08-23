package parser

import (
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/nunet"
)

var registry Registry[jobs.JobSpec]

func init() {
	registry = &RegistryImpl[jobs.JobSpec]{
		parsers: make(map[SpecType]Parser[jobs.JobSpec]),
	}

	// Register Nunet parser.
	nunetParser := NewParser[jobs.JobSpec](
		nunet.NewNuNetTransformer(),
		nunet.NewNuNetValidator(),
	)
	registry.RegisterParser(specTypeNuNet, nunetParser)
}
