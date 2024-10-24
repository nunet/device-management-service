// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// Sample validator functions
func sampleValidatorFunc1(_ *map[string]any, data any, _ tree.Path) error {
	if data == "invalid1" {
		return errors.New("invalid data for sampleValidatorFunc1")
	}
	return nil
}

func sampleValidatorFunc2(_ *map[string]any, data any, _ tree.Path) error {
	if data == "invalid2" {
		return errors.New("invalid data for sampleValidatorFunc2")
	}
	return nil
}

type ValidatorTestSuite struct {
	suite.Suite
	validators map[tree.Path]ValidatorFunc
}

func TestValidatorTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorTestSuite))
}

func (s *ValidatorTestSuite) SetupTest() {
	validators := map[tree.Path]ValidatorFunc{
		tree.NewPath("a.b"):    sampleValidatorFunc1,
		tree.NewPath("c.d"):    sampleValidatorFunc2,
		tree.NewPath("a.b.[]"): sampleValidatorFunc2,
	}

	s.validators = validators
}

func (s *ValidatorTestSuite) TestNewValidator() {
	v := NewValidator(s.validators)
	s.NotNil(v)
}

func (s *ValidatorTestSuite) TestValidate() {
	rawConfig := map[string]any{
		"a": map[string]any{
			"b": "valid",
		},
		"c": map[string]any{
			"d": "valid",
		},
	}

	v := NewValidator(s.validators)
	err := v.Validate(&rawConfig)
	s.NoError(err)

	rawConfigInvalid1 := map[string]any{
		"a": map[string]any{
			"b": "invalid1",
		},
	}

	err = v.Validate(&rawConfigInvalid1)
	s.Error(err)
	s.Equal("invalid data for sampleValidatorFunc1", err.Error())

	rawConfigInvalid2 := map[string]any{
		"c": map[string]any{
			"d": "invalid2",
		},
	}

	err = v.Validate(&rawConfigInvalid2)
	s.Error(err)
	s.Equal("invalid data for sampleValidatorFunc2", err.Error())
}

func (s *ValidatorTestSuite) TestValidateSlice() {
	rawConfig := map[string]any{
		"a": map[string]any{
			"b": []any{"valid", "invalid2"},
		},
	}

	v := NewValidator(s.validators)
	err := v.Validate(&rawConfig)
	s.Error(err)
	s.Equal("invalid data for sampleValidatorFunc2", err.Error())

	rawConfigValid := map[string]any{
		"a": map[string]any{
			"b": []any{"valid", "valid"},
		},
	}

	err = v.Validate(&rawConfigValid)
	s.NoError(err)
}
