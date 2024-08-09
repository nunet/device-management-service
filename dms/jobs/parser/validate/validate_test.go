package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// Sample validator functions
func sampleValidatorFunc1(root *map[string]any, data any, path tree.Path) error {
	if data == "invalid1" {
		return errors.New("invalid data for sampleValidatorFunc1")
	}
	return nil
}

func sampleValidatorFunc2(root *map[string]any, data any, path tree.Path) error {
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
		tree.NewPath("a.b"):      sampleValidatorFunc1,
		tree.NewPath("c.d"):      sampleValidatorFunc2,
		tree.NewPath("a.b.[]"):   sampleValidatorFunc2,
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

