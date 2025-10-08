// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package translator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/translator/types"
)

// MockSuccessTranslator implements types.Translator for successful translation tests
type MockSuccessTranslator struct {
	expectedInput []byte
	result        *types.Translation
}

func (m *MockSuccessTranslator) Translate(input []byte) (*types.Translation, error) {
	if m.expectedInput != nil && string(input) != string(m.expectedInput) {
		return nil, errors.New("unexpected input")
	}
	return m.result, nil
}

// MockErrorTranslator implements types.Translator for error testing
type MockErrorTranslator struct {
	errorMessage string
}

func (m *MockErrorTranslator) Translate(_ []byte) (*types.Translation, error) {
	return nil, errors.New(m.errorMessage)
}

type TranslatorTestSuite struct {
	suite.Suite
	originalRegistry *Registry
}

func (s *TranslatorTestSuite) SetupTest() {
	// Save original registry
	s.originalRegistry = registry

	// Create new registry for testing
	registry = &Registry{
		translators: make(map[SpecType]types.Translator),
	}
}

func (s *TranslatorTestSuite) TearDownTest() {
	// Restore original registry
	registry = s.originalRegistry
}

func TestTranslatorTestSuite(t *testing.T) {
	suite.Run(t, new(TranslatorTestSuite))
}

func (s *TranslatorTestSuite) TestTranslateSuccess() {
	// Setup mock translator
	expectedConfig := &jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"test": {
					DNSName: "test-service",
				},
			},
		},
	}
	expectedWarnings := []string{"warning1", "warning2"}

	mockTranslator := &MockSuccessTranslator{
		result: &types.Translation{
			Config:   expectedConfig,
			Warnings: expectedWarnings,
		},
	}

	// Register mock translator
	testSpecType := SpecType("test-spec")
	registry.RegisterTranslator(testSpecType, mockTranslator)

	// Test translation
	input := []byte("test input")
	result, err := Translate(testSpecType, input)

	// Verify results
	s.NoError(err)
	s.NotNil(result)
	s.Equal(expectedConfig, result.Config)
	s.Equal(expectedWarnings, result.Warnings)
}

func (s *TranslatorTestSuite) TestTranslateWithDockerCompose() {
	// Test with actual docker-compose spec type
	dockerComposeInput := []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`)

	// Register a mock translator for docker-compose
	mockTranslator := &MockSuccessTranslator{
		result: &types.Translation{
			Config: &jobtypes.EnsembleConfig{
				V1: &jobtypes.EnsembleConfigV1{
					Allocations: map[string]jobtypes.AllocationConfig{
						"web": {
							DNSName: "web",
						},
					},
				},
			},
			Warnings: []string{},
		},
	}
	registry.RegisterTranslator(SpecTypeDockerCompose, mockTranslator)

	// Test translation
	result, err := Translate(SpecTypeDockerCompose, dockerComposeInput)

	// Verify results
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Config)
	s.Contains(result.Config.V1.Allocations, "web")
}

func (s *TranslatorTestSuite) TestTranslateTranslatorNotFound() {
	nonExistentSpec := SpecType("non-existent-spec")
	input := []byte("test input")

	// Test translation with non-existent spec type
	result, err := Translate(nonExistentSpec, input)

	// Verify error
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "translator for spec type non-existent-spec not found")
}

func (s *TranslatorTestSuite) TestTranslateTranslatorError() {
	// Setup mock translator that returns error
	mockTranslator := &MockErrorTranslator{
		errorMessage: "translation failed",
	}

	testSpecType := SpecType("error-spec")
	registry.RegisterTranslator(testSpecType, mockTranslator)

	// Test translation
	input := []byte("test input")
	result, err := Translate(testSpecType, input)

	// Verify error
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "translation failed")
}

func (s *TranslatorTestSuite) TestTranslateEmptyInput() {
	// Setup mock translator
	mockTranslator := &MockSuccessTranslator{
		expectedInput: []byte(""),
		result: &types.Translation{
			Config:   &jobtypes.EnsembleConfig{},
			Warnings: []string{},
		},
	}

	testSpecType := SpecType("empty-input-spec")
	registry.RegisterTranslator(testSpecType, mockTranslator)

	// Test translation with empty input
	result, err := Translate(testSpecType, []byte(""))

	// Verify results
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Config)
}

func (s *TranslatorTestSuite) TestTranslateNilInput() {
	// Setup mock translator that handles nil input
	mockTranslator := &MockSuccessTranslator{
		result: &types.Translation{
			Config:   &jobtypes.EnsembleConfig{},
			Warnings: []string{"nil input provided"},
		},
	}

	testSpecType := SpecType("nil-input-spec")
	registry.RegisterTranslator(testSpecType, mockTranslator)

	// Test translation with nil input
	result, err := Translate(testSpecType, nil)

	// Verify results
	s.NoError(err)
	s.NotNil(result)
}

func (s *TranslatorTestSuite) TestTranslateMultipleSpecTypes() {
	// Setup multiple mock translators
	mockTranslator1 := &MockSuccessTranslator{
		result: &types.Translation{
			Config: &jobtypes.EnsembleConfig{
				V1: &jobtypes.EnsembleConfigV1{
					Allocations: map[string]jobtypes.AllocationConfig{
						"service1": {DNSName: "service1"},
					},
				},
			},
			Warnings: []string{},
		},
	}

	mockTranslator2 := &MockSuccessTranslator{
		result: &types.Translation{
			Config: &jobtypes.EnsembleConfig{
				V1: &jobtypes.EnsembleConfigV1{
					Allocations: map[string]jobtypes.AllocationConfig{
						"service2": {DNSName: "service2"},
					},
				},
			},
			Warnings: []string{},
		},
	}

	specType1 := SpecType("spec-type-1")
	specType2 := SpecType("spec-type-2")

	registry.RegisterTranslator(specType1, mockTranslator1)
	registry.RegisterTranslator(specType2, mockTranslator2)

	// Test both translations
	input := []byte("test input")

	result1, err1 := Translate(specType1, input)
	s.NoError(err1)
	s.Contains(result1.Config.V1.Allocations, "service1")

	result2, err2 := Translate(specType2, input)
	s.NoError(err2)
	s.Contains(result2.Config.V1.Allocations, "service2")
}

func (s *TranslatorTestSuite) TestSpecTypeConstants() {
	// Test that spec type constants are defined correctly
	s.Equal("docker-compose", string(SpecTypeDockerCompose))
	s.NotEmpty(SpecTypeDockerCompose)
}

func (s *TranslatorTestSuite) TestTranslateWithWarnings() {
	// Setup mock translator with warnings
	expectedWarnings := []string{
		"Feature X is not supported",
		"Configuration Y will be ignored",
		"Using default value for Z",
	}

	mockTranslator := &MockSuccessTranslator{
		result: &types.Translation{
			Config: &jobtypes.EnsembleConfig{
				V1: &jobtypes.EnsembleConfigV1{
					Allocations: map[string]jobtypes.AllocationConfig{
						"web": {DNSName: "web"},
					},
				},
			},
			Warnings: expectedWarnings,
		},
	}

	testSpecType := SpecType("warning-spec")
	registry.RegisterTranslator(testSpecType, mockTranslator)

	// Test translation
	input := []byte("test input with unsupported features")
	result, err := Translate(testSpecType, input)

	// Verify results
	s.NoError(err)
	s.NotNil(result)
	s.Equal(expectedWarnings, result.Warnings)
	s.Len(result.Warnings, 3)
}
