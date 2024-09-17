package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs"
)

// Mock objects for the registry and parser
type MockRegistry struct {
	mock.Mock
}

type MockParser struct {
	mock.Mock
}

func (m *MockRegistry) GetParser(specType SpecType) (Parser[jobs.JobSpec], bool) {
	args := m.Called(specType)
	return args.Get(0).(Parser[jobs.JobSpec]), args.Bool(1)
}

func (m *MockRegistry) RegisterParser(specType SpecType, parser Parser[jobs.JobSpec]) {
	m.Called(specType, parser)
}

func (m *MockParser) Parse(data []byte) (jobs.JobSpec, error) {
	args := m.Called(data)
	return args.Get(0).(jobs.JobSpec), args.Error(1)
}

type ParseTestSuite struct {
	suite.Suite
	registry *MockRegistry
	parser   *MockParser
}

func TestParseTestSuite(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}

func (s *ParseTestSuite) SetupTest() {
	s.registry = new(MockRegistry)
	s.parser = new(MockParser)

	registry = s.registry
}

func (s *ParseTestSuite) TestParseSuccess() {
	specType := SpecType("validSpec")
	data := []byte("validData")
	expectedJobSpec := jobs.JobSpec{Version: "1.0"}

	s.registry.On("GetParser", specType).Return(s.parser, true)
	s.parser.On("Parse", data).Return(expectedJobSpec, nil)

	result, err := Parse(specType, data)
	s.NoError(err)
	s.Equal(expectedJobSpec, result)

	s.registry.AssertExpectations(s.T())
	s.parser.AssertExpectations(s.T())
}

func (s *ParseTestSuite) TestParseParserNotFound() {
	specType := SpecType("invalidSpec")
	data := []byte("data")

	s.registry.On("GetParser", specType).Return(s.parser, false)

	result, err := Parse(specType, data)
	s.Error(err)
	s.Contains(err.Error(), "parser for spec type invalidSpec not found")
	s.Equal(jobs.JobSpec{}, result)

	s.registry.AssertExpectations(s.T())
}

func (s *ParseTestSuite) TestParseError() {
	specType := SpecType("errorSpec")
	data := []byte("errorData")
	expectedError := errors.New("parse error")

	s.registry.On("GetParser", specType).Return(s.parser, true)
	s.parser.On("Parse", data).Return(jobs.JobSpec{}, expectedError)

	result, err := Parse(specType, data)
	s.Error(err)
	s.Equal(expectedError, err)
	s.Equal(jobs.JobSpec{}, result)

	s.registry.AssertExpectations(s.T())
	s.parser.AssertExpectations(s.T())
}
