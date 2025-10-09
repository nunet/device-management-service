// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package translator

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.com/nunet/device-management-service/dms/translator/types"
)

// MockTranslator implements the types.Translator interface for testing
type MockTranslator struct {
	name string
}

func (m *MockTranslator) Translate(_ []byte) (*types.Translation, error) {
	return &types.Translation{
		Config:   nil, // Mock implementation
		Warnings: []string{},
	}, nil
}

type RegistryTestSuite struct {
	suite.Suite
	registry *Registry
}

func (s *RegistryTestSuite) SetupTest() {
	s.registry = &Registry{
		translators: make(map[SpecType]types.Translator),
	}
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) TestRegisterTranslator() {
	mockTranslator := &MockTranslator{name: "test"}
	specType := SpecType("test-spec")

	// Register translator
	s.registry.RegisterTranslator(specType, mockTranslator)

	// Verify it was registered
	translator, exists := s.registry.GetTranslator(specType)
	s.True(exists)
	s.Equal(mockTranslator, translator)
}

func (s *RegistryTestSuite) TestRegisterTranslatorOverwrite() {
	specType := SpecType("test-spec")
	mockTranslator1 := &MockTranslator{name: "test1"}
	mockTranslator2 := &MockTranslator{name: "test2"}

	// Register first translator
	s.registry.RegisterTranslator(specType, mockTranslator1)

	// Register second translator (should overwrite)
	s.registry.RegisterTranslator(specType, mockTranslator2)

	// Verify second translator is registered
	translator, exists := s.registry.GetTranslator(specType)
	s.True(exists)
	s.Equal(mockTranslator2, translator)
}

func (s *RegistryTestSuite) TestGetTranslatorNotFound() {
	specType := SpecType("non-existent")

	// Try to get non-existent translator
	translator, exists := s.registry.GetTranslator(specType)
	s.False(exists)
	s.Nil(translator)
}

func (s *RegistryTestSuite) TestGetTranslatorMultiple() {
	mockTranslator1 := &MockTranslator{name: "test1"}
	mockTranslator2 := &MockTranslator{name: "test2"}
	specType1 := SpecType("test-spec-1")
	specType2 := SpecType("test-spec-2")

	// Register multiple translators
	s.registry.RegisterTranslator(specType1, mockTranslator1)
	s.registry.RegisterTranslator(specType2, mockTranslator2)

	// Verify both are registered correctly
	translator1, exists1 := s.registry.GetTranslator(specType1)
	s.True(exists1)
	s.Equal(mockTranslator1, translator1)

	translator2, exists2 := s.registry.GetTranslator(specType2)
	s.True(exists2)
	s.Equal(mockTranslator2, translator2)
}

func (s *RegistryTestSuite) TestRegistryConcurrency() {
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Both register and get operations

	// Concurrent registration
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				specType := SpecType("test-spec-" + string(rune(id)))
				mockTranslator := &MockTranslator{name: "test-" + string(rune(id))}
				s.registry.RegisterTranslator(specType, mockTranslator)
			}
		}(i)
	}

	// Concurrent retrieval
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				specType := SpecType("test-spec-" + string(rune(id)))
				s.registry.GetTranslator(specType)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify registry is still functional
	testSpec := SpecType("final-test")
	testTranslator := &MockTranslator{name: "final"}
	s.registry.RegisterTranslator(testSpec, testTranslator)

	translator, exists := s.registry.GetTranslator(testSpec)
	s.True(exists)
	s.Equal(testTranslator, translator)
}

func (s *RegistryTestSuite) TestRegistryEmptyInitialization() {
	emptyRegistry := &Registry{
		translators: make(map[SpecType]types.Translator),
	}

	// Should not find any translators in empty registry
	translator, exists := emptyRegistry.GetTranslator(SpecTypeDockerCompose)
	s.False(exists)
	s.Nil(translator)
}

func (s *RegistryTestSuite) TestRegistryNilTranslator() {
	specType := SpecType("nil-test")

	// Register nil translator (should be allowed)
	s.registry.RegisterTranslator(specType, nil)

	// Should be able to retrieve nil translator
	translator, exists := s.registry.GetTranslator(specType)
	s.True(exists)
	s.Nil(translator)
}
