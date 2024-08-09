package nunet

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
)

// NewNuNetValidator creates a new validator for the NuNet configuration.
func NewNuNetValidator() validate.Validator {
	return validate.NewValidator(
		map[tree.Path]validate.ValidatorFunc{
			"":                    ValidateSpec,
			"jobs.[]":             ValidateJob,
			"jobs.**.children.[]": ValidateJob,
		},
	)
}

// ValidateSpec checks the root configuration for consistency.
func ValidateSpec(root *map[string]any, data any, path tree.Path) error {
	spec, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid spec configuration: %v", data)
	}
	// Check if the jobs list is present and not empty.
	if spec["jobs"] == nil || len(spec["jobs"].([]any)) == 0 {
		return fmt.Errorf("jobs list is required")
	}
	return nil
}

// ValidateJob checks the job configuration.
func ValidateJob(root *map[string]any, data any, path tree.Path) error {
	job, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid job configuration: %v", data)
	}
	// Check if the job has either children or an execution.
	if job["children"] == nil || len(job["children"].([]any)) == 0 {
		if job["execution"] == nil {
			return fmt.Errorf("job must have either children or an execution")
		}
	}
	return nil
}
