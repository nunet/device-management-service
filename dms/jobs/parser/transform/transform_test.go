// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package transform

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/utils"
)

type TransformerTestSuite struct {
	suite.Suite
	transformers []map[tree.Path]TransformerFunc
}

// Mock transformer function
func mockTransformer(_ *map[string]interface{}, _ interface{}, _ tree.Path) (any, error) {
	return "transformed", nil
}

func TestTransformerTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerTestSuite))
}

func (s *TransformerTestSuite) SetupTest() {
	transformers := []map[tree.Path]TransformerFunc{
		{
			tree.NewPath("mockPath"):       mockTransformer,
			tree.NewPath("**", "mockPath"): mockTransformer,
		},
	}

	s.transformers = transformers
}

func (s *TransformerTestSuite) TestNewTransformer() {
	transformer := NewTransformer(s.transformers)
	s.NotNil(transformer)
}

func (s *TransformerTestSuite) TestTransform() {
	transformer := NewTransformer(s.transformers)

	rawConfig := map[string]interface{}{
		"mockPath": "value",
	}
	expectedConfig := map[string]interface{}{
		"mockPath": "transformed",
	}

	result, err := transformer.Transform(&rawConfig)
	s.NoError(err)
	s.Equal(utils.Normalize(expectedConfig), result)
}

func (s *TransformerTestSuite) TestTransformWithNestedData() {
	transformer := NewTransformer(s.transformers)

	rawConfig := map[string]interface{}{
		"mockPath": "value",
		"nested": map[string]interface{}{
			"mockPath": "nestedValue",
		},
	}
	expectedConfig := map[string]interface{}{
		"mockPath": "transformed",
		"nested": map[string]interface{}{
			"mockPath": "transformed",
		},
	}

	result, err := transformer.Transform(&rawConfig)
	s.NoError(err)
	s.Equal(utils.Normalize(expectedConfig), result)
}

func (s *TransformerTestSuite) TestTransformWithDifferentOrder() {
	transformer := NewTransformer(s.transformers)

	rawConfig := map[string]interface{}{
		"jobs": []interface{}{
			map[string]interface{}{
				"name":     "job1",
				"mockPath": "value1",
			},
			map[string]interface{}{
				"name":     "job2",
				"mockPath": "value2",
			},
		},
	}

	expectedConfig := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{
				"name":     "job2",
				"mockPath": "transformed",
			},
			{
				"name":     "job1",
				"mockPath": "transformed",
			},
		},
	}

	result, err := transformer.Transform(&rawConfig)
	s.NoError(err)
	s.Equal(utils.Normalize(expectedConfig), utils.Normalize(result))
}
