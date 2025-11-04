//go:build acceptance || !unit

package acceptance

import (
	"testing"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

func TestDeploymentOperations(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/deployment_operations.feature"}

	suite := godog.TestSuite{
		Name:                "deployment_operations",
		Options:             &o,
		ScenarioInitializer: steps.DeploymentOperations,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
