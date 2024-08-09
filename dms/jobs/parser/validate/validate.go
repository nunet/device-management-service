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
