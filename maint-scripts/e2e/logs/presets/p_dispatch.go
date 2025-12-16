package presets

import (
	"fmt"
	"os"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	register("dispatch", Dispatch, nil)
}

func Dispatch(_ types.Args, files []shared.LogFile, logs [][]*shared.LogLine) (
	[]shared.LogFile, [][]*shared.LogLine,
) {
	// preset logic
	var err error
	for i, lines := range logs {
		logs[i], err = shared.JSONQuery(lines, `.msg == "dispatching_message"`)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			os.Exit(1)
		}
	}

	return files, logs
}
