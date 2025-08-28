//go:build acceptance || !unit

package acceptance

import (
	"testing"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

func TestDeploymentUpdate(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/deployment_update.feature"}

	suite := godog.TestSuite{
		Name:                "deployment_update",
		Options:             &o,
		ScenarioInitializer: steps.DeploymentUpdate,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
