//go:build acceptance || !unit

package acceptance

import (
	"testing"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

func TestSubnet(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/subnet.feature"}

	suite := godog.TestSuite{
		Name:                "subnet",
		Options:             &o,
		ScenarioInitializer: steps.Subnet,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
