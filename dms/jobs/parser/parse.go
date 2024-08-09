package parser

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs"
)

func Parse(specType SpecType, data []byte) (jobs.JobSpec, error) {
	result := jobs.JobSpec{}

    parser, exists := registry.GetParser(specType)
    if !exists {
        return result, fmt.Errorf("parser for spec type %s not found", specType)
    }

	result, err := parser.Parse(data)
    if err != nil {
        return result, err
    }

    return result, nil
}
