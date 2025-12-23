package presets

import (
	"fmt"
	"os"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

var deploymentArgs DeploymentArgs

func init() {
	register("deployment", Deployment, nil)
}

type DeploymentArgs struct {
	DeploymentID string `help:"Allocation ID or Orchestrator ID" arg:"--deployment-id,required"`
}

func Deployment(args types.Args, files []shared.LogFile, logs [][]*shared.LogLine) (
	[]shared.LogFile, [][]*shared.LogLine,
) {
	// parse args
	parsePresetArgs(args, "deployment", &deploymentArgs)

	// preset logic
	var err error
	id := deploymentArgs.DeploymentID
	query := fmt.Sprintf(`(.orchestratorID == "%s") or ((.allocationID // "") | startswith("%s"))`, id, id)
	for i, lines := range logs {
		logs[i], err = shared.JSONQuery(lines, query)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			os.Exit(1)
		}
	}

	return files, logs
}
