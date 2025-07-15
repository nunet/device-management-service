//go:build acceptance || !unit

package acceptance

import (
	"flag"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

var opts = godog.Options{
	Output:        colors.Colored(os.Stdout),
	Concurrency:   4,
	Format:        "pretty",
	StopOnFailure: true,
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opts)
}

func TestDeployment(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/deployment.feature"}

	suite := godog.TestSuite{
		Name:                "deployment",
		Options:             &o,
		ScenarioInitializer: steps.Deployment,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
