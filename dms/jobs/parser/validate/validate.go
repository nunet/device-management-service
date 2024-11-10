// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// ValidatorFunc is a function that validates a part of the configuration.
// It takes the root configuration, the data to validate and the current path in the tree.
type ValidatorFunc func(*map[string]any, any, tree.Path) error

// Validator is a configuration validator.
// It contains a map of patterns to paths to functions that validate the configuration.

type Validator interface {
	Validate(*map[string]any) error
}

// ValidatorImpl is the implementation of the Validator interface.
type ValidatorImpl struct {
	validators map[tree.Path]ValidatorFunc
}

// NewValidator creates a new validator with the given validators.
func NewValidator(validators map[tree.Path]ValidatorFunc) Validator {
	return ValidatorImpl{
		validators: validators,
	}
}

// Validate applies the validators to the configuration.
func (v ValidatorImpl) Validate(rawConfig *map[string]any) error {
	data := any(*rawConfig)
	return v.validate(rawConfig, data, tree.NewPath(), v.validators)
}

// validate is a recursive function that applies the validators to the configuration.
func (v ValidatorImpl) validate(root *map[string]interface{}, data any, path tree.Path, validators map[tree.Path]ValidatorFunc) error {
	// Apply validators that match the current path.
	for pattern, validator := range validators {
		if path.Matches(pattern) {
			if err := validator(root, data, path); err != nil {
				return err
			}
		}
	}
	// Recursively apply validators to children.
	switch data := data.(type) {
	case map[string]interface{}:
		for key, value := range data {
			next := path.Next(key)
			if err := v.validate(root, value, next, validators); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, value := range data {
			next := path.Next(fmt.Sprintf("[%d]", i))
			if err := v.validate(root, value, next, validators); err != nil {
				return err
			}
		}
	}
	return nil
}
