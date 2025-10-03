//go:build acceptance || !unit

package acceptance

import (
	"testing"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

func TestContract(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/contract.feature"}

	suite := godog.TestSuite{
		Name:                "contract",
		Options:             &o,
		ScenarioInitializer: steps.Contract,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
