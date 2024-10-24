// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
)

// NewNuNetValidator creates a new validator for the NuNet configuration.
func NewEnsembleV1Validator() validate.Validator {
	return validate.NewValidator(
		map[tree.Path]validate.ValidatorFunc{
			"V1":               ValidateSpec,
			"V1.allocations.*": ValidateAllocation,
		},
	)
}

// ValidateSpec checks the root configuration for consistency.
func ValidateSpec(_ *map[string]any, data any, _ tree.Path) error {
	spec, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid spec configuration: %v", data)
	}

	// TODO: Specify and complete validation - Dawit Abate

	// Check if the allocations map is present and not empty.
	if spec["allocations"] == nil || len(spec["allocations"].(map[string]any)) == 0 {
		return fmt.Errorf("allocations list is required")
	}
	return nil
}

// ValidateAllocation checks the allocation configuration.
func ValidateAllocation(_ *map[string]any, data any, _ tree.Path) error {
	allocation, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid allocation configuration: %v", data)
	}

	// TODO: Specify and complete validation - Dawit Abate

	// Check if the allocation has an execution.
	if allocation["execution"] == nil {
		return fmt.Errorf("allocation must have an execution")
	}
	return nil
}
