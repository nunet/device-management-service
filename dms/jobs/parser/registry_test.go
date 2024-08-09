package parser

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.com/nunet/device-management-service/dms/jobs"
)


type RegistryTestSuite struct {
	suite.Suite
	registry *RegistryImpl[jobs.JobSpec]
}

func (suite *RegistryTestSuite) SetupTest() {
	suite.registry = &RegistryImpl[jobs.JobSpec]{
		parsers: make(map[SpecType]Parser[jobs.JobSpec]),
		mu:      sync.RWMutex{},
	}
}

func (suite *RegistryTestSuite) TestRegisterParser() {
	specType := SpecType("testSpec")
	parser := &MockParser{}

	suite.registry.RegisterParser(specType, parser)

	retrievedParser, exists := suite.registry.GetParser(specType)
	suite.True(exists)
	suite.Equal(parser, retrievedParser)
}

func (suite *RegistryTestSuite) TestGetParserNotFound() {
	specType := SpecType("unknownSpec")

	_, exists := suite.registry.GetParser(specType)
	suite.False(exists)
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
