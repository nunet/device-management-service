package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs"
)

type MockTransformer struct {
	mock.Mock
}

func (m *MockTransformer) Transform(data *map[string]any) (interface{}, error) {
	args := m.Called(data)
	return args.Get(0).(map[string]any), args.Error(1)
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(data *map[string]any) error {
	args := m.Called(data)
	return args.Error(0)
}

type ParserTestSuite struct {
	suite.Suite
	parser       Impl[jobs.JobSpec]
	transformer  *MockTransformer
	validator    *MockValidator
	rawYAMLData  []byte
	rawJSONData  []byte
	expectedData map[string]any
}

func (suite *ParserTestSuite) SetupTest() {
	suite.transformer = new(MockTransformer)
	suite.validator = new(MockValidator)
	suite.parser = Impl[jobs.JobSpec]{
		transformer: suite.transformer,
		validator:   suite.validator,
	}

	suite.rawYAMLData = []byte(`
version: "1.0"
jobs: []
`)

	suite.rawJSONData = []byte(`{
	"version": "1.0",
	"jobs": []
}`)

	suite.expectedData = map[string]any{
		"version": "1.0",
		"jobs":    []any{},
	}
}

func (suite *ParserTestSuite) TestParseYAMLSuccess() {
	suite.transformer.On("Transform", &suite.expectedData).Return(suite.expectedData, nil)
	suite.validator.On("Validate", &suite.expectedData).Return(nil)

	result, err := suite.parser.Parse(suite.rawYAMLData)
	suite.NoError(err)
	suite.Equal("1.0", result.Version)
	suite.Equal([]*jobs.Job{}, result.Jobs)

	suite.transformer.AssertExpectations(suite.T())
	suite.validator.AssertExpectations(suite.T())
}

func (suite *ParserTestSuite) TestParseJSONSuccess() {
	suite.transformer.On("Transform", &suite.expectedData).Return(suite.expectedData, nil)
	suite.validator.On("Validate", &suite.expectedData).Return(nil)

	result, err := suite.parser.Parse(suite.rawJSONData)
	suite.NoError(err)
	suite.Equal("1.0", result.Version)
	suite.Equal([]*jobs.Job{}, result.Jobs)

	suite.transformer.AssertExpectations(suite.T())
	suite.validator.AssertExpectations(suite.T())
}

func (suite *ParserTestSuite) TestParseYAMLError() {
	invalidData := []byte(`
id: testJob
name: "Test Job
`)

	result, err := suite.parser.Parse(invalidData)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse config")
	suite.Empty(result)
}

func (suite *ParserTestSuite) TestParseJSONError() {
	invalidData := []byte(`{
	"id": "testJob",
	"name": "Test Job
}`)

	result, err := suite.parser.Parse(invalidData)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse config")
	suite.Empty(result)
}

func (suite *ParserTestSuite) TestTransformError() {
	suite.transformer.On("Transform", &suite.expectedData).Return(map[string]interface{}{}, errors.New("transform error"))

	result, err := suite.parser.Parse(suite.rawYAMLData)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to transform config")
	suite.Empty(result)

	suite.transformer.AssertExpectations(suite.T())
}

func (suite *ParserTestSuite) TestValidateError() {
	suite.transformer.On("Transform", &suite.expectedData).Return(suite.expectedData, nil)
	suite.validator.On("Validate", &suite.expectedData).Return(errors.New("validation error"))

	result, err := suite.parser.Parse(suite.rawYAMLData)
	suite.Error(err)
	suite.Contains(err.Error(), "validation error")
	suite.Empty(result)

	suite.transformer.AssertExpectations(suite.T())
	suite.validator.AssertExpectations(suite.T())
}

func TestParserTestSuite(t *testing.T) {
	suite.Run(t, new(ParserTestSuite))
}
