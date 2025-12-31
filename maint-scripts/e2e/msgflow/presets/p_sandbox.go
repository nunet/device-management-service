package presets

import (
	"fmt"
	"os"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/msgflow/types"
)

var sandboxArgs SandboxArgs

func init() {
	register("sandbox", Sandbox, nil)
}

type SandboxArgs struct {
	types.Args
	PresetSandboxFlag bool `help:"Binary flag" default:"true" arg:"--preset-sandbox-flag"`
}

func Sandbox(args types.Args, files []shared.LogFile, lines []*shared.LogLine) (
	[]shared.LogFile, []*shared.LogLine,
) {
	// parse args
	parsePresetArgs(args, "sandbox", &sandboxArgs)

	// JSON query filter
	var err error
	lines, err = shared.JSONQuery(lines, `.msg == "dispatching_message"`)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	// manual filter
	filtered := make([]*shared.LogLine, 0, len(lines))
	for i, l := range lines {
		// filter based on a condition
		if l.MsgFrom == nil {
			continue
		}

		// add to output
		l.RawJSON = `{"sandbox": true, ` + l.RawJSON[1:]

		filtered = append(filtered, lines[i])
	}
	lines = filtered

	return files, lines
}
